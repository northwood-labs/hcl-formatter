// -----------------------------------------------------------------------------
//
// NOTICE
//
// The contents of this file were extracted from the Terraform project.
//
// Copyright (c) Terraform Contributors <https://github.com/hashicorp/terraform/graphs/contributors>
//
// The contents of this file are licensed under the terms of the Mozilla Public
// License 2.0.
//
//     <https://github.com/hashicorp/terraform/blob/master/LICENSE>
//
// The original source code can be found here:
//
//     <https://github.com/hashicorp/terraform/blob/c94a6102df62017766f4cc2c2a04c930c0a2c465/command/fmt.go>
//
// Only minor changes have been applied to turn this into a standalone,
// importable library for those of us who work with HCL outside of Terraform.
//
// -----------------------------------------------------------------------------

package hcl_formatter

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const (
	minTokenLength = 5
)

// FormatHCL is a public function which will take a byte slice, format the HCL
// into "canonical" format, and return the byte slice.
func FormatHCL(src []byte) []byte {
	f, diags := hclwrite.ParseConfig(src, "sample.tf", hcl.InitialPos)

	if diags.HasErrors() {
		fmt.Fprintf(os.Stderr, "HCL syntax error: %s", diags.Error())
	}

	formatBody(f.Body(), nil)

	return f.Bytes()
}

func formatBody(body *hclwrite.Body, inBlocks []string) {
	attrs := body.Attributes()
	for name, attr := range attrs {
		if len(inBlocks) == 1 && inBlocks[0] == "variable" && name == "type" {
			cleanedExprTokens := formatTypeExpr(attr.Expr().BuildTokens(nil))
			body.SetAttributeRaw(name, cleanedExprTokens)

			continue
		}

		cleanedExprTokens := formatValueExpr(attr.Expr().BuildTokens(nil))
		body.SetAttributeRaw(name, cleanedExprTokens)
	}

	blocks := body.Blocks()
	for _, block := range blocks {
		// Normalize the label formatting, removing any weird stuff like
		// interleaved inline comments and using the idiomatic quoted
		// label syntax.
		block.SetLabels(block.Labels())

		inBlocks = append(inBlocks, block.Type())
		formatBody(block.Body(), inBlocks)
	}
}

func formatValueExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) < minTokenLength {
		// Can't possibly be a "${ ... }" sequence without at least enough
		// tokens for the delimiters and one token inside them.
		return tokens
	}

	oQuote := tokens[0]
	oBrace := tokens[1]
	cBrace := tokens[len(tokens)-2]
	cQuote := tokens[len(tokens)-1]

	if oQuote.Type != hclsyntax.TokenOQuote ||
		oBrace.Type != hclsyntax.TokenTemplateInterp ||
		cBrace.Type != hclsyntax.TokenTemplateSeqEnd ||
		cQuote.Type != hclsyntax.TokenCQuote {
		// Not an interpolation sequence at all, then.
		return tokens
	}

	inside := tokens[2 : len(tokens)-2]

	// We're only interested in sequences that are provable to be single
	// interpolation sequences, which we'll determine by hunting inside
	// the interior tokens for any other interpolation sequences. This is
	// likely to produce false negatives sometimes, but that's better than
	// false positives and we're mainly interested in catching the easy cases
	// here.
	quotes := 0

	for _, token := range inside {
		if token.Type == hclsyntax.TokenOQuote {
			quotes++
			continue
		}

		if token.Type == hclsyntax.TokenCQuote {
			quotes--
			continue
		}

		if quotes > 0 {
			// Interpolation sequences inside nested quotes are okay, because
			// they are part of a nested expression.
			// "${foo("${bar}")}"
			continue
		}

		if token.Type == hclsyntax.TokenTemplateInterp || token.Type == hclsyntax.TokenTemplateSeqEnd {
			// We've found another template delimiter within our interior
			// tokens, which suggests that we've found something like this:
			// "${foo}${bar}"
			// That isn't unwrappable, so we'll leave the whole expression alone.
			return tokens
		}

		if token.Type == hclsyntax.TokenQuotedLit {
			// If there's any literal characters in the outermost
			// quoted sequence then it is not unwrappable.
			return tokens
		}
	}

	// If we got down here without an early return then this looks like
	// an unwrappable sequence, but we'll trim any leading and trailing
	// newlines that might result in an invalid result if we were to
	// naively trim something like this:
	// "${
	//    foo
	// }"
	return trimNewlines(inside)
}

func formatTypeExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	switch len(tokens) {
	case 1:
		kwTok := tokens[0]
		if kwTok.Type != hclsyntax.TokenIdent {
			// Not a single type keyword, then.
			return tokens
		}

		// Collection types without an explicit element type mean
		// the element type is "any", so we'll normalize that.
		switch string(kwTok.Bytes) {
		case "list", "map", "set":
			return hclwrite.Tokens{
				kwTok,
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("any"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		default:
			return tokens
		}
	default:
		return tokens
	}
}

func trimNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) == 0 {
		return nil
	}

	var start, end int

	for start = 0; start < len(tokens); start++ {
		if tokens[start].Type != hclsyntax.TokenNewline {
			break
		}
	}

	for end = len(tokens); end > 0; end-- {
		if tokens[end-1].Type != hclsyntax.TokenNewline {
			break
		}
	}

	return tokens[start:end]
}
