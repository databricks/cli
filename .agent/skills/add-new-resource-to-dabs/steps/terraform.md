# Terraform

Going forward, Terraform engine will be deprecated and new resources should not be added there.

For new resources that don't include terraform, make sure to add it to exclude lists when TF-related unit tests are failing (or seem suitable due to having a list of unuspported TF resources).

There's also logic in [validate_direct_only_resources.go](./bundle/config/mutator/validate_direct_only_resources.go) which needs to be edited.
