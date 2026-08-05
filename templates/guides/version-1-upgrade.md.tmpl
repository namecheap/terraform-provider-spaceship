---
page_title: "Upgrading to version 1.0"
description: |-
  How to upgrade from a 0.x release to version 1.0 of the Spaceship provider.
---

# Upgrading to version 1.0

Version 1.0.0 is the first stable release of the Spaceship provider.

**There are no breaking changes.** Version 1.0.0 is functionally identical to
0.6.0: no resource or data source schemas changed, no attributes were renamed
or removed, and no state migration is required. Existing configurations and
state files continue to work as-is — upgrading is a version-constraint change
and nothing more.

## Upgrade steps

Update the provider version constraint in your configuration:

```terraform
terraform {
  required_providers {
    spaceship = {
      source  = "namecheap/spaceship"
      version = "~> 1.0"
    }
  }
}
```

Then upgrade the locked provider version and verify:

```shell
terraform init -upgrade
terraform plan
```

`terraform plan` should report no changes for a configuration that was
in sync before the upgrade.

## Stability and testing

The 1.0.0 release was validated end-to-end against the live Spaceship API:
every resource and data source was exercised through its full
plan / apply / import / destroy lifecycle, including rate-limit handling,
record matching edge cases, and out-of-band drift detection. The release ships
only after that validation reported no functional defects.

## Semantic versioning commitment

From 1.0.0 onward the provider follows [Semantic Versioning](https://semver.org/):

- **Patch releases** (`1.0.x`) contain only bug fixes.
- **Minor releases** (`1.x.0`) add functionality in a backwards-compatible way.
- **Breaking changes** ship only in a new **major release**, with an upgrade
  guide like this one.

The `~> 1.0` constraint above therefore tracks all 1.x releases safely while
protecting you from a future major version until you opt in.
