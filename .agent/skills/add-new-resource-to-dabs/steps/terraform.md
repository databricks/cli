# Terraform

Going forward, Terraform engine will be deprecated and new resources should not be added there.

For direct-only resources:
- Update Terraform-related unsupported/exclude lists when relevant tests expect them.
- Update `bundle/config/mutator/validate_direct_only_resources.go` to keep direct-only validation accurate.

This step is intentionally short because Terraform support is not the default path for new resources.
