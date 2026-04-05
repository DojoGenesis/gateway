# Dojo Skill Marketplace Policy

**Version:** 1.0.0
**Effective:** Before first community publish
**Last Updated:** 2026-04-05
**ADR:** 020-plugin-trust-tiers-marketplace

---

## 1. Publishing Requirements

### All Tiers
- **SemVer required.** Every published skill version must follow Semantic Versioning (MAJOR.MINOR.PATCH). CalVer, date-based, and arbitrary versioning are rejected by the registry.
- **License field required.** SPDX license identifier must be present in the manifest. Unlicensed skills are not publishable.
- **Description required.** The `description` field in the manifest must be non-empty.
- **Name rules.** Skill names must be:
  - At least 4 characters
  - Alphanumeric plus hyphens only (`[a-z0-9-]+`)
  - No leading or trailing hyphens
  - No consecutive hyphens
  - No Unicode homoglyphs (confusable characters are rejected)
  - No reserved names (see Section 6)

### Tier 0: Community
- Unsigned. Install shows a warning: "This skill is unsigned and unverified."
- WASM sandbox required for any executable content.
- Publisher must have a verified email address.
- New publishers: 5 skill publishes per day for the first 30 days.

### Tier 1: Verified
- Cosign OIDC keyless signature required (GitHub Actions or compatible OIDC provider).
- Signature verified against Rekor transparency log.
- Source repository URL must be present in OCI annotations.
- WASM sandbox required for any executable content.

### Tier 2: Official
- Signed with the Dojo Platform organization key.
- Curated by the Dojo maintainers.
- WASM sandbox optional (trusted code path).
- Listed in the official skill index.

---

## 2. Immutability and Yanking

### Published versions are immutable.
Once a version is published, its content cannot be modified. To fix a bug, publish a new version.

### Yank, never delete.
- `dojo skill yank <name>@<version> --reason "<reason>"` marks a version as yanked.
- Yanked versions:
  - Are **blocked** from new installs (`dojo skill install` returns an error with the yank reason).
  - **Remain available** in CAS stores that already have them (no remote deletion).
  - Show a **YANKED** badge in `dojo skill info` and `dojo skill list`.
  - Preserve **reproducibility** — existing lockfiles and CAS stores are not broken.
- Yanking is **reversible** by the original publisher or marketplace administrators.
- Deletion is **never performed** — even for security vulnerabilities. Deletion provides false security (the artifact is already distributed) while breaking reproducibility.

### Yank reasons
Yank reasons should be categorized:
- `security: <CVE or description>` — security vulnerability
- `broken: <description>` — non-functional release
- `superseded: <new-version>` — replaced by a better version
- `policy: <description>` — violates marketplace policy

---

## 3. Abuse Reporting

- `dojo skill report <name> --reason "<reason>"` files an abuse report.
- Reports are reviewed by marketplace administrators.
- Grounds for action:
  - Malware or malicious code
  - Name squatting (registering names with no intent to publish meaningful content)
  - Slopsquatting (registering AI-hallucinated names to confuse users)
  - Typosquatting (registering names similar to popular skills)
  - License violations
  - Spam or low-quality bulk publishing

---

## 4. Publisher Accounts

- Email verification required before first publish.
- Publishers are identified by their OIDC identity (e.g., GitHub username) for Tier 1+.
- Account suspension:
  - 3+ sustained abuse reports → temporary publish suspension pending review
  - Confirmed malware → permanent ban
- Transfer: skill ownership can be transferred between verified publishers.

---

## 5. WASM Sandbox Requirements

All executable content in Tier 0 and Tier 1 skills MUST run inside the WASM sandbox (Wazero). This includes:
- Custom scoring functions
- Data transformation scripts
- Channel adapter hooks
- Workflow node processors

Capability grants (filesystem, network, environment) follow the deny-by-default model from Era 1 (ADR-011). Skills must declare required capabilities in their manifest.

---

## 6. Reserved Names

The following name categories are reserved and cannot be registered by community publishers:

### Official skill names (44)
All skills currently packaged in the Dojo Platform distribution are reserved. See `dojo skill list` for the current list.

### Platform names
`dojo`, `dojo-platform`, `dojo-core`, `dojo-cli`, `dojo-gateway`, `dojo-runtime`, `dojo-sdk`, `dojo-api`, `dojo-admin`, `dojo-official`, `dojo-verified`, `dojo-marketplace`

### Infrastructure names
`slack`, `discord`, `telegram`, `teams`, `whatsapp`, `email`, `sms`, `webchat`, `figma`, `github`, `gitlab`, `bitbucket`

### Slopsquatting corpus
A generated corpus of AI-hallucinated names is maintained at `reserved-names.txt` in the marketplace registry. This corpus is regenerated quarterly.

---

## 7. Versioning

- **Skills:** SemVer required. Breaking changes = major version bump.
- **Workflows:** SemVer required. Changing step order or removing steps = major version bump.
- **Plugins:** SemVer required. Removing a contained skill = major version bump.
- Dependency ranges: `>=1.0.0 <2.0.0` syntax supported in `dependencies` field.

---

## 8. Registry Configuration

### Default registry
`ghcr.io/dojo-skills` — configurable in `gateway-config.yaml`:

```yaml
marketplace:
  registry: ghcr.io/dojo-skills
  trust_minimum: 0  # 0=community, 1=verified, 2=official
  verify_signatures: true
  allow_unsigned: true  # show warning but allow install
```

### Private registries
Operators can point to any OCI-compliant registry (Harbor, ECR, Docker Hub). Air-gapped deployments use a private Harbor instance. See `deployments/harbor/` for a reference deployment.

---

## Policy Changes

This policy may be updated. Changes are announced via:
- `CHANGELOG.md` in the Dojo Platform repository
- `dojo.marketplace.policy_updated` CloudEvent on the event bus
- GitHub release notes

Substantive changes (new restrictions, tier requirement changes) take effect 30 days after announcement to allow publishers to adjust.
