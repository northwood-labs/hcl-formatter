# HCL2 Formatter

## Overview

This is the code that facilitates `terraform fmt`, and was extracted from the Terraform project at [`c94a6102`](https://github.com/hashicorp/terraform/blob/c94a6102df62017766f4cc2c2a04c930c0a2c465/command/fmt.go). Only minor changes have been applied to turn this into a standalone, importable library for those of us who work with HCL outside of Terraform.

I don't expect this code to change very much, so don't freak out if there haven't been commits in years (it's not _abandoned_, it's just _complete_). It should very much still work for standard HCL2 formatting.

## License and Authorship

Authorship by the [Terraform Contributors](https://github.com/hashicorp/terraform/graphs/contributors). The contents of this file are licensed under the terms of the [Mozilla Public License 2.0](https://github.com/hashicorp/terraform/blob/master/LICENSE).
