# Internal documentation

## Why Terragrunt?

Terragrunt is used for the following reasons:

* Without terragrunt, it would be necessary to manually initialise the backends in the bucket.
* Terragrunt is able to install and manage several related Terraform-Modules uniquely in spite of having a separated terraform state for each one. The latter has the advantage to be able to separately destroy/refactor them. Concourse in the context of this terraform-project is organised in layers (called "stacks" in Terragrunt).

This degree of automation enables automatic testing and continuous delivery of this concourse and its infrastructure.