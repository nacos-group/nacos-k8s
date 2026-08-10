# AGENTS.md

This file provides guidance to AI coding agents working with the nacos-k8s repository.

## AI Contribution Guidelines

- **Do NOT post AI-generated comments** on issues or PRs
- **Discuss before implementing**: Agree on direction with maintainers first
- **Disclose AI usage**: Add `Assisted-by: Claude Code` trailer to commit messages when significant parts are AI-generated
- **Follow contribution processes**: See CONTRIBUTING.md if available

## Repository Overview

nacos-k8s provides Kubernetes deployment solutions for [Nacos](https://nacos.io/) (Dynamic Naming and Configuration Service).

**Current Version**: Nacos 3.2.3 (also supports 2.x via Helm chart version detection)

**Deployment Modes**:
- **Helm chart** (primary, actively maintained) — `helm/`
- **Quick start** — `quick-startup.sh` + `deploy/`
- **Operator** — `operator/`
- **Plain YAML** (NFS/Ceph/Ingress) — `deploy/nacos/`
- **OpenShift** — `openshift/`

## Helm Chart Structure

| File | Purpose |
|------|---------|
| `helm/templates/_helpers.tpl` | Helpers including `nacos.isV3` version detection |
| `helm/templates/statefulset.yaml` | Nacos StatefulSet with version-aware probes and conditional ports |
| `helm/templates/service.yaml` | `nacos-cs` (client) + `nacos-hs` (headless cluster) services |
| `helm/templates/configmap.yaml` | MySQL connection ConfigMap |
| `helm/templates/ingress.yaml` | Optional Ingress resource |
| `helm/values.yaml` | All configurable parameters with defaults |

**Version detection**: The chart auto-detects Nacos 2.x vs 3.x from the image tag. Override with `nacos.majorVersion` for custom tags like `latest`.

## Build & Test Commands

```bash
# Render templates locally
helm template test ./helm --set nacos.image.tag=v3.2.3

# Lint chart
helm lint ./helm

# Run CI smoke test locally (requires Kind cluster)
NACOS_VERSION=v3.2.3 MODE=standalone STORAGE=embedded ./hack/ci/helm-smoke-test.sh

# Full CI matrix runs on every PR via GitHub Actions
```

## CI Structure

- `.github/workflows/ci-helm-smoke-test.yaml` — matrix: Nacos (v2.5.3, v3.2.3) × K8s (v1.34.8, v1.36.1) × Mode (standalone, cluster)
- `hack/ci/helm-smoke-test.sh` — core test script
- `hack/ci/values-nacos-{2x,3x}.yaml` — version-specific Helm values
- `hack/ci/mysql.yaml` — lightweight MySQL for CI

## Code Style

| Type | Style |
|------|-------|
| YAML | 2 spaces indent |
| Shell | bash, `set -euo pipefail`, functions for reuse |
| Helm templates | Follow existing Go template patterns |
| Commits | Conventional Commit format: `type(scope): description` |

## PR Convention

- Target branch: `master`
- Squash merge preferred
- CI must pass (3.x cluster + embedded may be flaky — see alibaba/nacos#13762)
