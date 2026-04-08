# Implementation Commission: Multi-Architecture Docker Images (amd64 + arm64)

**Objective:** Make the Docker image available for both `linux/amd64` and `linux/arm64` so developers on Apple Silicon (and arm64 Linux servers) can run the Gateway without Rosetta emulation.

---

## 1. Context & Grounding

**Primary Specification:**
- This commission is self-contained. The binary already builds for arm64 via Goreleaser — only the Docker image and CI pipeline need updating.

**Pattern Files (Follow these examples):**
- `.goreleaser.yml` lines 12-29 (`builds` section): Already produces `linux/amd64` and `linux/arm64` binaries with `CGO_ENABLED=0`. The Docker section at lines 72-87 only targets `linux/amd64` — this is the gap.
- `Dockerfile`: Currently builds from source inside the container. For multi-arch, we switch to copying the pre-built Goreleaser binary so each platform gets its native binary.

**Files to Read:**
- `.goreleaser.yml` — Full file. The `builds` and `dockers_v2` sections are the targets.
- `Dockerfile` — Full file. Will be restructured for multi-arch.
- `.github/workflows/ci.yml` — CI workflow. Will add a multi-arch build verification step.
- `docker-compose.yaml` — Check for any platform-specific settings (read only, likely no changes needed).

**Key Constraint:** `CGO_ENABLED=0` is already set in the Goreleaser build. The binary is statically linked. The distroless base image (`gcr.io/distroless/static-debian12`) supports both amd64 and arm64. This means multi-arch support is purely a packaging concern, not a compilation concern.

---

## 2. Detailed Requirements

### Phase 1: Update Goreleaser for Multi-Arch Docker (Steps 1-3)

1. In `.goreleaser.yml`, replace the `dockers_v2` section (lines 72-87) with two Docker image definitions — one per architecture — following the Goreleaser v2 multi-platform pattern:

   ```yaml
   dockers:
     - id: agentic-gateway-amd64
       ids: ["agentic-gateway"]
       goos: linux
       goarch: amd64
       dockerfile: Dockerfile.goreleaser
       image_templates:
         - "ghcr.io/trespies-source/agentic-gateway:v{{ .Version }}-amd64"
       build_flag_templates:
         - "--platform=linux/amd64"
       extra_files: []

     - id: agentic-gateway-arm64
       ids: ["agentic-gateway"]
       goos: linux
       goarch: arm64
       dockerfile: Dockerfile.goreleaser
       image_templates:
         - "ghcr.io/trespies-source/agentic-gateway:v{{ .Version }}-arm64"
       build_flag_templates:
         - "--platform=linux/arm64"
       extra_files: []
   ```

2. In `.goreleaser.yml`, add a `docker_manifests` section after the `dockers` section to create a multi-arch manifest list:

   ```yaml
   docker_manifests:
     - name_template: "ghcr.io/trespies-source/agentic-gateway:v{{ .Version }}"
       image_templates:
         - "ghcr.io/trespies-source/agentic-gateway:v{{ .Version }}-amd64"
         - "ghcr.io/trespies-source/agentic-gateway:v{{ .Version }}-arm64"

     - name_template: "ghcr.io/trespies-source/agentic-gateway:latest"
       image_templates:
         - "ghcr.io/trespies-source/agentic-gateway:v{{ .Version }}-amd64"
         - "ghcr.io/trespies-source/agentic-gateway:v{{ .Version }}-arm64"
   ```

3. Preserve the existing OCI labels from the original `dockers_v2` section in both `dockers` entries:
   ```yaml
   labels:
     "org.opencontainers.image.created": "{{.Date}}"
     "org.opencontainers.image.title": "{{.ProjectName}}"
     "org.opencontainers.image.revision": "{{.FullCommit}}"
     "org.opencontainers.image.version": "{{.Version}}"
     "org.opencontainers.image.source": "https://github.com/DojoGenesis/gateway"
   ```

### Phase 2: Create Goreleaser-Specific Dockerfile (Steps 4-5)

4. Create `Dockerfile.goreleaser` — a minimal Dockerfile that copies the pre-built binary instead of building from source. Goreleaser places the binary in the Docker build context automatically:

   ```dockerfile
   FROM gcr.io/distroless/static-debian12

   COPY agentic-gateway /agentic-gateway

   EXPOSE 8080

   USER 65534:65534

   ENTRYPOINT ["/agentic-gateway"]
   ```

   This Dockerfile is used exclusively by Goreleaser. The original `Dockerfile` remains unchanged for local development builds (`docker build .`).

5. Keep the existing `Dockerfile` exactly as-is. It continues to serve local development where developers run `docker build .` directly. Add a comment at the top explaining the two-Dockerfile pattern:

   ```dockerfile
   # Dockerfile — local development builds (builds from source, single-arch)
   # For production multi-arch images, Goreleaser uses Dockerfile.goreleaser
   ```

### Phase 3: CI Verification (Steps 6-7)

6. In `.github/workflows/ci.yml`, add a `docker` job that verifies the Docker image builds correctly on both architectures. This job validates the build but does NOT push to any registry:

   ```yaml
   docker:
     runs-on: ubuntu-latest
     needs: build
     steps:
       - uses: actions/checkout@v4

       - name: Set up QEMU
         uses: docker/setup-qemu-action@v3

       - name: Set up Docker Buildx
         uses: docker/setup-buildx-action@v3

       - name: Build multi-arch image (no push)
         run: |
           docker buildx build \
             --platform linux/amd64,linux/arm64 \
             -f Dockerfile \
             --tag agentic-gateway:ci-test \
             .
   ```

   This catches Dockerfile regressions (e.g., accidentally adding a platform-specific dependency) on every PR.

7. The `docker` job uses the original `Dockerfile` (source build) with `buildx` cross-compilation. This is slower than Goreleaser's approach but validates that the source build works on both architectures. The Goreleaser release workflow handles the fast path (pre-built binary copy) at release time.

---

## 3. File Manifest

**Create:**
- `Dockerfile.goreleaser` — Minimal runtime-only Dockerfile for Goreleaser multi-arch releases (~6 lines)

**Modify:**
- `.goreleaser.yml` — Replace `dockers_v2` with per-arch `dockers` + `docker_manifests`
- `.github/workflows/ci.yml` — Add `docker` build verification job
- `Dockerfile` — Add comment header only (no functional changes)

---

## 4. Success Criteria

- [ ] `goreleaser check` passes with zero warnings on the updated `.goreleaser.yml`
- [ ] `Dockerfile.goreleaser` exists and is a valid Dockerfile (no build stage, just COPY + ENTRYPOINT)
- [ ] `.goreleaser.yml` `dockers` section defines both `amd64` and `arm64` image templates
- [ ] `.goreleaser.yml` `docker_manifests` section creates `:v{version}` and `:latest` manifest lists spanning both architectures
- [ ] Original `Dockerfile` is functionally unchanged (still works for `docker build .`)
- [ ] CI workflow includes a `docker` job that runs `docker buildx build --platform linux/amd64,linux/arm64`
- [ ] OCI labels (created, title, revision, version, source) are present on both architecture images
- [ ] The `builds` section in `.goreleaser.yml` is unchanged (still produces linux+darwin × amd64+arm64)
- [ ] `docker pull ghcr.io/trespies-source/agentic-gateway:latest` resolves to the correct architecture on both amd64 and arm64 hosts (validated by manifest list structure, not live test)

---

## 5. Constraints & Boundaries

- **DO NOT** modify the existing `Dockerfile` beyond adding a comment header — it serves local development
- **DO NOT** change the `builds` section of `.goreleaser.yml` — binary compilation is already correct
- **DO NOT** add Docker push steps to CI — pushing happens only in the Goreleaser release workflow
- **DO NOT** add docker-compose changes — compose is for local development and is architecture-agnostic
- **DO NOT** introduce `docker buildx` into the Goreleaser config — Goreleaser handles platform targeting natively via the `goos`/`goarch` fields in the `dockers` section
- **DO NOT** add any new runtime dependencies to the Docker image — keep distroless

---

## 6. Integration Points

- Goreleaser release workflow (if it exists as a separate GitHub Action) will need `QEMU` and `Buildx` setup steps for arm64 image builds on amd64 CI runners. If no release workflow exists yet, this is a note for when one is created.
- The `docker-compose.yaml` service definition should work without changes because Docker Compose automatically selects the correct architecture from a manifest list.
- The existing `--health-check` flag self-contained probe works on both architectures (pure Go binary, no external tools needed).

---

## 7. Testing Requirements

**Build Verification:**
- `docker buildx build --platform linux/amd64 -f Dockerfile .` succeeds
- `docker buildx build --platform linux/arm64 -f Dockerfile .` succeeds
- `docker build -f Dockerfile.goreleaser .` succeeds (with a dummy binary in context)

**Manifest Verification:**
- `goreleaser check` passes
- The `docker_manifests` section references both architecture-specific image templates

**No unit tests needed** — this is infrastructure/packaging only.
