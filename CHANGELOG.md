# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- CI/CD pipeline with GitHub Actions (EP-019)
  - Automated lint, test, and build workflow
  - Coverage gate (≥80% for core packages)
  - Race detector enabled for all tests
  - Automated release workflow with checksums and changelog
  - CI report generation (ci_report.json)
- Configuration management system (EP-018)
  - YAML-based configuration with system/user merge
  - Strict validation with path-based error messages
  - `aistack config test` command for configuration validation
- Security & secrets management
  - Encrypted secret storage with NaCl secretbox
  - Strict file permissions (0600) for sensitive files
- Power management
  - Idle detection with CPU/GPU thresholds
  - Auto-suspend with systemd integration
  - Force mode (`--ignore-inhibitors`) for testing
  - TUI screen for power management configuration
- Wake-on-LAN support
  - Network interface detection
  - Magic packet sending
  - HTTP relay for remote wake-up
- Service management
  - Container orchestration (Ollama, Open WebUI, LocalAI)
  - Health checks and automatic repair
  - Update with rollback on health check failure
  - Backend switching (Ollama ↔ LocalAI)
  - Volume preservation on service removal
- Model management
  - Model listing and downloading
  - Cache statistics
  - Eviction policies
- Diagnostics
  - Diagnostic package creation (`aistack diag`)
  - Secret redaction in diagnostic outputs
  - Comprehensive system information collection
- TUI (Terminal User Interface)
  - Interactive menu system
  - Service management screens
  - Logs viewer
  - Models management
  - Power configuration
- GPU management
  - NVIDIA GPU detection with NVML
  - GPU lock for exclusive access
  - Health checks and smoke tests
- Metrics collection
  - CPU utilization and power (RAPL)
  - GPU utilization, memory, power, temperature
  - JSONL-based metrics logging

### Changed
- N/A (initial release)

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- Implemented secure secret storage with encryption
- Added file permission checks for sensitive data
- Secret redaction in diagnostic outputs

## [0.1.0-dev] - Development

Initial development version with core features implemented.

[Unreleased]: https://github.com/polygonschmiede/aistack/compare/v0.1.0...HEAD
[0.1.0-dev]: https://github.com/polygonschmiede/aistack/releases/tag/v0.1.0-dev
