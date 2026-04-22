# Release Process

This document describes how changes merged to `master` become published
binaries on the [Terraform Registry](https://registry.terraform.io/providers/namecheap/spaceship).

The intent is captured in the three workflows under
[`.github/workflows/`](.github/workflows/); this file elaborates on the
operator-facing pieces that are not obvious from the YAML alone.

## Overview

The project uses a **semi-automated, maintainer-gated** release flow. Changes
merged to `master` do **not** ship immediately. Instead they accumulate in a
long-lived "Release PR" maintained by [release-please](https://github.com/googleapis/release-please),
and a binary is published only when a maintainer merges that PR.

Three workflows participate:

| Workflow | Trigger | Role |
|---|---|---|
| [`ci.yml`](.github/workflows/ci.yml) | push to `master`, PRs | Lint, unit tests, coverage threshold, doc validation, optional acceptance tests |
| [`versioning.yml`](.github/workflows/versioning.yml) | `ci.yml` success on `master`, manual dispatch | Runs release-please to open/update the Release PR |
| [`release.yml`](.github/workflows/release.yml) | push of a `v*` tag | Runs GoReleaser to build, sign, and publish artifacts |

Supporting configuration:

- [`pr-title.yml`](.github/workflows/pr-title.yml) — enforces
  [Conventional Commits](https://www.conventionalcommits.org/) on PR titles;
  release-please consumes these to compute version bumps.
- [`.release-please-config.json`](.release-please-config.json) /
  [`.release-please-manifest.json`](.release-please-manifest.json) —
  release-please state. The manifest is the source of truth for the current
  version.
- [`.goreleaser.yml`](.goreleaser.yml) — cross-compile matrix, archive naming,
  checksum, and GPG signing.
- [`terraform-registry-manifest.json`](terraform-registry-manifest.json) —
  bundled into every release so the Terraform Registry knows the provider's
  supported protocol versions.

## Step-by-step flow

### 1. PR is opened and merged

- PR title must follow Conventional Commits. `pr-title.yml` fails the PR
  otherwise.
- PRs are typically squash-merged, so the PR title becomes the commit
  message that release-please classifies.
- `ci.yml` runs on the PR and again on the resulting merge commit.

At this point nothing is released — the commit simply sits on `master`.

### 2. release-please opens or updates the Release PR

- `versioning.yml` triggers on a successful `ci.yml` run against `master`
  (`workflow_run` with `conclusion == 'success'`).
- release-please walks commits since the last tag, classifies them, and:
  - computes the next SemVer bump (`fix:` → patch, `feat:` → minor,
    `feat!:` or `BREAKING CHANGE:` → major),
  - updates `CHANGELOG.md` with a generated entry grouped by type,
  - updates `.release-please-manifest.json` with the new version,
  - opens (or updates) a PR titled `chore(master): release X.Y.Z`.
- Authentication uses a dedicated GitHub App
  (`SPS_RELEASE_APP_ID`, `SPS_RELEASE_PRIVATE_KEY`), **not** the default
  `GITHUB_TOKEN`. This is required: events authored by `GITHUB_TOKEN` do
  not re-trigger workflows, so a token-authored Release PR would never run
  CI. App-authored events do.

The Release PR is long-lived. As more PRs merge to `master`, release-please
keeps appending to the same PR, updating the CHANGELOG and (if the bump
type changes) the target version.

### 3. Maintainer decides when to cut a release

This is the semi-manual step.

Changes accumulate in the Release PR indefinitely. A release happens only
when a maintainer:

1. Reviews the computed version bump and CHANGELOG entries in the Release PR.
2. Merges the Release PR when the bundle of included changes is worth
   shipping.

Merging the Release PR:

- commits the version bump and regenerated CHANGELOG to `master`,
- creates a `vX.Y.Z` git tag on that merge commit (release-please-action
  creates both the tag and a GitHub Release draft).

There is no fixed cadence. Merge when there is enough change to justify a
new binary — a single critical `fix:`, a batch of `feat:` entries, etc.

### 4. GoReleaser publishes the binary

- `release.yml` triggers on the pushed `v*` tag.
- It runs GoReleaser against `.goreleaser.yml`, which:
  - cross-compiles the provider for `freebsd`, `windows`, `linux`, `darwin`
    across `amd64`, `386`, `arm`, `arm64` (skipping the invalid
    `darwin/386`),
  - produces `terraform-provider-spaceship_<version>_<os>_<arch>.zip`
    archives,
  - generates `terraform-provider-spaceship_<version>_SHA256SUMS` and
    bundles `terraform-provider-spaceship_<version>_manifest.json`
    alongside it,
  - signs the checksum file with GPG using `GPG_PRIVATE_KEY` /
    `PASSPHRASE` (fingerprint exported as `GPG_FINGERPRINT`). The
    Terraform Registry verifies this signature against the maintainer's
    published public key,
  - attaches archives, checksums, signatures, and the manifest to the
    GitHub Release for the tag.

`changelog.disable: true` in `.goreleaser.yml` is deliberate:
release-please already wrote the CHANGELOG during the Release PR, so
GoReleaser must not overwrite it.

### 5. Terraform Registry ingestion

The Terraform Registry watches GitHub releases and ingests new versions
automatically. Most of the time, a release appears within minutes of the
GitHub Release going live.

> **Note:** Occasionally a new release does not surface in the Terraform
> Registry within a reasonable window (typically under an hour). When this
> happens, the publisher account must trigger a manual resync from the
> provider's page on <https://registry.terraform.io/>. This is a known
> operational quirk and is expected to be needed from time to time, at least
> for now.

## Required secrets

| Secret | Used by | Purpose |
|---|---|---|
| `SPS_RELEASE_APP_ID` | `versioning.yml` | GitHub App ID for release-please |
| `SPS_RELEASE_PRIVATE_KEY` | `versioning.yml` | GitHub App private key |
| `GPG_PRIVATE_KEY` | `release.yml` | Signing key for release artifacts |
| `PASSPHRASE` | `release.yml` | GPG key passphrase |
| `SPACESHIP_API_KEY` / `SPACESHIP_API_SECRET` / `SPACESHIP_TEST_DOMAIN` | `ci.yml` | Acceptance tests (optional; skipped if missing) |
| `CODECOV_TOKEN` | `ci.yml` | Upload coverage to Codecov |

## Versioning

- Current version lives in [`.release-please-manifest.json`](.release-please-manifest.json).
- [Semantic Versioning](https://semver.org/), derived automatically from
  Conventional Commit types since the previous tag.
- Pre-1.0.0: minor versions may contain breaking changes. After 1.0.0,
  standard SemVer rules apply.

## Manual / emergency release

Prefer the normal flow. In rare cases (release-please unavailable, out-of-band
hotfix, etc.) a release can be cut by hand:

1. Bump the version in `.release-please-manifest.json` and add the
   corresponding entry to `CHANGELOG.md`. Commit to `master`.
2. Tag the commit: `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`.
3. `release.yml` runs on the pushed tag and publishes binaries as usual.
4. The next release-please run will reconcile its state with the updated
   manifest.
