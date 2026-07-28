# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Local development workflow: `.env.example`, `godotenv` loading, and `make setup`.
- `make verify-setup` (`cmd/verify`) for read-only checks of Prime API access, trading wallets, balances, and Exchange price connectivity.
- `DRY_RUN` mode (default `true` locally) to log intended orders and conversions without submitting them.
- Make targets: `help`, `build`, `test`, `fmt`, `vet`, `tidy`, `run-local`, `run-local-live`, `docker-run-local`.
- Dependabot configuration for Go modules and Docker.

### Changed

- Migrated from archived `github.com/coinbase-samples/prime-sdk-go` to `github.com/coinbase/prime-sdk-go` v0.9.1.
- Raised minimum Go version to 1.25.
- Updated `golang.org/x/net` and other dependencies to address Dependabot security advisories.
- README quickstart for local development; AWS deployment sets `DRY_RUN=false` in the ECS task definition.

### Security

- Bumped `golang.org/x/net` to address medium-severity Dependabot alerts.
