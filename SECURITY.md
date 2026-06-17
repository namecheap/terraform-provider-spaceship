# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in this project, please report it
responsibly. **Do not open a public GitHub issue.**

Email **opensource@namecheap.com** with:

- A description of the vulnerability
- Steps to reproduce or a proof-of-concept
- Any relevant tool versions or environment details

We will acknowledge your report within 3 business days and aim to provide a
fix or mitigation plan within 30 days, depending on complexity.

## Scope

This policy covers the Terraform provider code in this repository. For
vulnerabilities in the Spaceship platform or API, please contact
[Spaceship support](https://www.spaceship.com/about/contact-us/).

## Supported Versions

Security fixes are applied to the latest release only. We recommend always
using the most recent version of the provider.

## Automated Security Scanning

Every push and pull request runs a set of security gates in
[`.github/workflows/ci.yml`](.github/workflows/ci.yml):

- **Trivy** (`security` job) — vulnerability (SCA), IaC misconfiguration,
  secret, and license scanning. The gate blocks merge on `CRITICAL,HIGH`
  findings with an available fix (`ignore-unfixed: true`). The license denylist
  lives in [`trivy.yaml`](trivy.yaml) (copyleft / source-available licenses we
  cannot ship). A CycloneDX SBOM is generated and uploaded as the
  `sbom-cyclonedx` artifact.
- **govulncheck** (`govulncheck` job) — Go-native call-graph **reachability**
  analysis against <https://vuln.go.dev> and the standard library. Fails the
  build only when a vulnerable symbol is actually reachable from the provider's
  code; imported-but-uncalled vulnerabilities are reported but do not gate.
- **CodeQL** ([`codeql.yml`](.github/workflows/codeql.yml)) — static analysis
  with the `security-and-quality` query suite.

### PR security summary

The `security-report` job consolidates all of the above into a single Markdown
report, surfaced three ways: a **sticky pull-request comment** (one comment,
updated in place per push), the run's **job summary**, and the
**`security-summary`** workflow artifact. It reports Trivy findings by severity,
govulncheck reachable-vs-imported counts, open CodeQL alerts, and a dependency
overview (SBOM component count, `go.mod` direct/indirect split, license
breakdown).

The summary is **reporting only and never gates** — the per-scan pass/fail it
shows is taken verbatim from the gate jobs above, so a green comment can never
mask a red gate. Secret findings list the rule and location only, never the
matched value. The comment is skipped for pull requests from forks (whose token
is read-only); the job summary and artifact are still produced.
