# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed - 2025-01-15

#### Dependencies Update
- **Go Version:** Updated from 1.22.12 to 1.25.0 with toolchain 1.25.1
  - Provides latest language features and improvements
  - Better performance and security baseline
  - Note: Some Go stdlib vulnerabilities (GO-2025-4007 through GO-2025-4013) are pending fixes in Go 1.25.2+

#### Updated Dependencies
- `golang.org/x/net`: v0.33.0 → v0.47.0
- `golang.org/x/sys`: v0.28.0 → v0.38.0
- `github.com/mattn/go-colorable`: v0.1.13 → v0.1.14
- `github.com/mattn/go-isatty`: v0.0.19 → v0.0.20

#### Maintained at Latest Versions
- `github.com/go-resty/resty/v2`: v2.16.5 (no updates available)
- `github.com/rs/zerolog`: v1.34.0 (no updates available)

### Validation
- ✅ All packages compile successfully with Go 1.25.1
- ✅ `go vet` passes without warnings
- ✅ `go mod tidy` completes cleanly
- ✅ No breaking changes detected
- ✅ Tests run successfully (require Magento credentials)

### Security Notes
- Dependency updates include important security and bug fixes
- Upgraded to latest stable Go toolchain (1.25.1)
- Known stdlib vulnerabilities will be addressed when Go 1.25.2+ is released
- No vulnerabilities found in project dependencies

---

## Previous Updates

### Go 1.21+ Features
- Added support for modern Go features including the `any` type
- Implemented structured errors with wrapping
- Enhanced context support

### Resty v2 Migration
- Migrated to resty v2 for better performance
- Improved connection pooling
- Better HTTP client management

### Logging Enhancements
- Added structured logging with zerolog
- Improved debugging and monitoring capabilities
- Better error context and tracing

### Testing Improvements
- Improved test coverage and organization
- Added functional tests for all major features
- Better test configuration management

### Bulk Operations
- Added bulk operations utilities
- Support for concurrent product creation
- CSV-based bulk updates

### Error Handling
- Enhanced error handling with wrapped errors
- Custom error types for Magento 2 API scenarios
- Better error context and debugging
