# Change Log

All notable and important changes to **Smurf Tool** are documented here.

---

## [v1.0.0] — 2026-01-20
### Added
- Add missing Smurf STF commands:
   - `smurf stf state rm`
   - `smurf stf state pull`
   - `smurf stf state push`
   - `smurf stf import`

### Fixed
- Update deploy command
- Update `smurf selm install` command
- Update kubernetes function for error handling

---

## [v0.1.3] — 2026-01-20
### Added
- Add Timeout Support to Terraform Format Command

---

## [v0.1.2] — 2026-01-20
### Added
- Added Multi-Threading Support to `smurf selm upgrade`

### Fixed
- Fix OCI Chart Loading in GitHub Actions

---

## [v0.1.0] — 2025-12-15
### Added
- Add smurf sdkr for google cloud platform(GCP)
- Add history max flag for `smurf selm`

### Fixed
- update smurf terraform provision command
- smurf stf plan `--out` flag update

---

## [v0.0.9] — 2025-11-26
### Added
- Add smurf sdkr for google cloud platform(GCP)
- Add history max flag for smurf selm
- Add GHCR repo feature
- Add `smurf deploy` command
- Add smurf selm init command

### Fixed
- Update `smurf stf format` command
- Update smurf provision GHCR logs
- Improve code suggested by Gemini AI

---

## [v0.0.8] — 2025-11-11
### Added
- Add smurf terraform
- Added commands to manage Terraform operations (plan, apply, destroy) using Smurf.

### Fixed
- Removed duplicate logic detected by SonarQube Cloud.
- Update docs with latest changes
- Improve code suggested by Gemini AI

---

## [v0.0.7] — 2025-11-04
### Added
- Introduced **GitHub Container Registry (GHCR)** support.
- Implemented new `smurf.yaml` configuration file.
- Added `smurf deploy` command for streamlined image deployment.
- Added commands for pushing Docker images to **GHCR**.
- Introduced `smurf selm init` and related Helm management features.

### Fixed
- Removed duplicate logic detected by SonarQube Cloud.
- Fixed Helm nil pointer error in test templates.

---

## [v0.0.6] — 2025-10-03
### Changed
- Refactored and optimized `smurf selm` commands for better logging and error handling.

### Fix
- Resolve `selm template` issue([#268](https://github.com/clouddrove/smurf/issues/268))

---

## [v0.0.5] - 2025-09-23
### Fix
- Resolve `smurf selm template` command timeout issue
- Update selm install log structure
- Resolve readiness issue of `selm upgrade` command

---

## [v0.0.4] — 2025-09-01
### Added
- Added `--wait` flag in `smurf selm upgrade` for improved deployment control([#243](https://github.com/clouddrove/smurf/issues/243)).

---

## [v0.0.3] — 2025-08-26
### Feat
- Enhance smurf sdkr and smurf selm logging structure ([#229](https://github.com/clouddrove/smurf/issues/229))
- added install.sh ([#140](https://github.com/clouddrove/smurf/issues/140))

### Fixed
- Add repo chart support in upgrade ([#241](https://github.com/clouddrove/smurf/issues/241))
- Fixed chart repository handling in `upgrade` command.

---

## [v0.0.2] 2025-08-21
### Feat
- Enhance smurf sdkr and smurf selm logging structure ([#229](https://github.com/clouddrove/smurf/issues/229))
- Added install.sh ([#140](https://github.com/clouddrove/smurf/issues/140))

### Fix
- fixed test in install_test.go ([#122](https://github.com/clouddrove/smurf/issues/122))

---

## [v0.0.1] — 2025-08-20
### Added
- Initial release of **Smurf CLI**.
- Features: build, push, and deploy Docker images and performing operation of Helm
- Supported registries: ACR, Docker Hub, AWS ECR, GCP
- commands
    - `smurf sdkr --help`
    - `smurf selm --help`

