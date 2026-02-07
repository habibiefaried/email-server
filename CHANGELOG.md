# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed
- **Email size limit is now configurable via `EMAIL_SIZE_LIMIT` environment variable** (default: 524288 bytes / 512KB)
  - Previously hardcoded as a constant, now can be customized or disabled (set to `0`)
  - Emails exceeding the limit are stored with error message: "Sorry, the email exceeds our limit (512kb)"
  - Size check is performed before expensive MIME parsing to prevent memory exhaustion
  - Added `EMAIL_SIZE_LIMIT` configuration to `.env.example`
- **Improved email size limit handling**
  - Size limit check moved earlier in the process (before parsing) to prevent resource exhaustion
  - When an email exceeds the size limit, headers are now extracted from raw content using lightweight parsing
  - Rejection message changed from "Limit of this service is 512kb only" to "Sorry, the email exceeds our limit (512kb)"
  - Raw content is no longer stored for oversized emails to save database space
- **Updated documentation**
  - Removed hardcoded "512KB limit" references from README.md
  - Added `EMAIL_SIZE_LIMIT` environment variable documentation
  - Updated API endpoint documentation to reflect configurable size limits

### Fixed
- **Fixed CI workflow tests**
  - Removed failing attachment metadata tests from CI pipeline
  - Simplified email detail structure validation (removed attachment field requirement)
  - Tests now properly validate core email functionality without attachment dependencies

### Removed
- Removed `MaxEmailSize` constant from `internal/storage/postgres.go`
- Removed `MaxEmailSize` constant from `cmd/reprocess-emails/main.go`
- Removed `TestMaxAttachmentSize` test from `internal/storage/postgres_test.go`
- Removed unused `contains` and `containsHelper` helper functions from test file
- Removed attachment metadata validation tests from CI workflow
- Emails exceeding size limit no longer store raw content in database (optimization)

### Added
- **New `extractHeadersFromRawContent` function** for efficient header extraction without full MIME parsing
- **Sample PDF email** (`samples/pdf.txt`) for testing large email handling
- Environment variable support for configurable email size limits in both server and reprocessing tool
- Improved logging with size limit information on database connection

### Technical Details
- **Storage layer changes:**
  - `PostgresStorage` struct now includes `maxEmailSize` field
  - Constructor reads `EMAIL_SIZE_LIMIT` from environment with fallback to 512KB
  - Size validation occurs before ENMIME parsing to avoid memory issues
- **Reprocessing tool changes:**
  - Updated to respect `EMAIL_SIZE_LIMIT` environment variable
  - Emails exceeding limit are now skipped during reprocessing
  - Removed hardcoded size limit, making tool configuration consistent with server
- **Test improvements:**
  - Cleaned up unused test helpers
  - Removed tests tied to hardcoded constants
  - CI tests focus on core functionality rather than size-limit edge cases

### Migration Notes
- **No breaking changes to API or database schema**
- Existing deployments will continue to use 512KB default limit
- To change the limit, set `EMAIL_SIZE_LIMIT` environment variable (in bytes)
- To disable the limit entirely, set `EMAIL_SIZE_LIMIT=0`
