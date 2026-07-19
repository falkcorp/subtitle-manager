<!-- file: CHANGELOG.md -->
<!-- version: 1.0.3 -->
<!-- guid: 6a7b8c9d-0e1f-2a3b-4c5d-6e7f8a9b0c1d -->

# Changelog

All notable changes to this project will be documented in this file.

<!-- scriv-insert-here -->

<!-- scriv-end-here: releases below predate the changelog.d fragment system. -->

## [Unreleased]

### Added

- Replace local databasepb with gcommon database types

### Changed

- codex-rebase.sh now uses Codex or ChatGPT CLI for conflict resolution and
  automatically commits before force pushing

### Added

- Added unit test for /api/sync/batch invalid JSON case

### Added

- Added gRPC health service using gcommon protobufs
- Added per-IP rate limiting to search API

### Added

- Added gcommon logrus provider for structured logging
- Added config protobuf for centralized configuration
- Adopted gcommon queue proto for internal queue
- Added gRPC AuthService using gcommon protobufs
- Added protobuf definitions for database records
- Migrated cache TTL configuration to gcommon CachePolicy proto
- Added metrics proto aggregator and refactored metrics package
- Added gcommon configuration example
- Add SupportedServices helper and gRPC dial fix
- Added 'whisper start' and 'whisper stop' commands for container management
- Added CLI commands to start and stop the Whisper container
- Added GitHub project setup script
- Added caching to CLI search command
- Added config migration script for gcommon
- Added context timeouts for gRPC translation and cleaned up duplicate CLI flags
- Added create-github-projects.sh to automate project board creation
- Added docker-local make target for local multi-arch builds
- Added monitor autosync command for scheduled Sonarr/Radarr synchronization
- Added provider metadata proto messages
- Added robust safety checks to rebase scripts
- Added search result caching for CLI search command
- Added sonarr-sync command for one-time library synchronization
- Added structured logging for download API
- Added unit test for /api/sync/batch endpoint
- Added workflow to automatically fix merge conflicts using AI rebase
- Documented gcommon queue configuration in README
- Enhanced search error handling with aggregated caching
- Finalize optional Whisper container integration
- Implemented Google batch translation support
- Implemented UpdateDownloadWithResult to track download results
- Implemented manual search result caching for faster repeated queries
- Improved DirectoryChooser supports custom folder input
- Improved provider order normalization for search result caching
- Improved subtitle search handler with caching and parallelization
- Improved synchronization accuracy with median offset
- Integrated gcommon metrics and health modules
- Integrated gcommon metrics and health system
- Migrated config loader to gcommon/config with new migration script
- Replaced local metrics with gcommon implementation
- Standardized on Dockerfile.hybrid for all builds
- Switched authentication to gcommon/auth library
- Use gcommon proto messages
- Validated codex-rebase.sh conflict handling
- create-github-projects.sh validates GitHub CLI scopes
- monitor autosync supports cron schedules

### Fixed

- Implemented context timeouts for gRPC translation commands
- Add missing media_profiles table for SQLite schema
- Added @testing-library/dom to resolve missing module errors
- Codex rebase script failed due to missing origin remote
- Corrected Docker stack port mapping
- Deduplicated persistent flag definitions in root command to prevent test
  panics
- Fixed duplicate flag definitions in root command
- Fixed search cache key to ignore provider order
- Handle type field when displaying directories
- Handled missing origin remote in rebase scripts
- Normalize provider order when computing CLI search cache keys
- Skip SQLite migration test when SQLite support is not built
- use PAT secret for fix-merge-conflicts workflow

### Changed

- Switched health endpoints to use gcommon/health handlers
- Improved GitHub project setup script with auth checks
- Removed add-to-project workflow; using GitHub built-in project automation

## [0.9.0] - 2025-06-30

### Status Update

- **Project Nearing Completion**: Backend and frontend largely implemented but
  tagging, containerized Whisper, and maintenance tooling remain
- **Production Ready Backend**: Complete authentication, APIs, and provider
  support
- **Enterprise Features**: PostgreSQL, webhooks, notifications, anti-captcha,
  advanced scheduling complete
- **UI/UX Overhaul Completed**: All enhancements implemented according to the
  plan
- Code overview documentation added in docs/CODE_OVERVIEW.md

### Major Features Completed Since Last Release

- **PostgreSQL Database Backend**: Complete enterprise database support with
  full test coverage
- **Advanced Webhook System**: Sonarr/Radarr/custom webhook endpoints for
  library event integration
- **Anti-Captcha Integration**: Support for Anti-Captcha.com and 2captcha.com
  services
- **Notification Services**: Discord, Telegram, and SMTP notification providers
- **Advanced Scheduler**: Cron-based scheduling with full expression support
- **Bazarr Configuration Import**: Command-line tool for seamless Bazarr
  migration
- Complete Web UI implementation with all major pages
- Full REST API coverage for all operations
- History page with translation and download filtering
- System page with real-time logs and task monitoring
- Wanted page with subtitle search and management
- Comprehensive testing and documentation updates
- Production-ready authentication and authorization

### Planned (Optional Enhancement)

- Advanced reverse proxy base URL support for complex network setups
- Tagging system for language preferences
- Optional containerized Whisper ASR service with NVIDIA runtime
- `whisper status` command for checking local Whisper container health
- Automated database cleanup and metadata refresh tasks

### Added

- **PostgreSQL database backend** with full enterprise support and graceful test
  skipping
- **Advanced webhook system** with `/api/webhooks/sonarr`,
  `/api/webhooks/radarr`, and `/api/webhooks/custom` endpoints
- **Anti-captcha integration** supporting Anti-Captcha.com and 2captcha.com
  services
- **Notification services** with Discord, Telegram, and SMTP providers
- **Advanced cron-based scheduler** with full expression support and granular
  controls
- **Bazarr configuration import** command for seamless migration from existing
  Bazarr installations
- REST endpoint `/api/convert` for subtitle file conversion
- REST endpoint `/api/translate` for translating uploaded subtitle files
- REST endpoint `/api/download` for on-demand subtitle fetching
- Build process now runs `go generate ./webui` to embed the latest web assets in
  the binary and container image.
- Automated workflow closes duplicate issues by title
- Embedded provider now enabled by default. Other providers remain hidden until
  explicitly added or imported.
- Enhanced General Settings page with Bazarr-compatible host, proxy, update,
  logging, backup and analytics options.

## [0.4.0] - 2025-06-12

### Major Milestone: Production Ready Release

This release marks ~95% project completion with full production readiness
achieved.

### Added

- Complete Web UI implementation with all major pages:
  - History page with translation and download filtering
  - System page with real-time logs and task monitoring
  - Wanted page with subtitle search and management
- Full REST API coverage for all subtitle operations
- Enhanced documentation reflecting current implementation status
- Production-ready status with comprehensive testing coverage

### Changed

- Updated project status documentation to reflect near-completion
- Improved README with current feature set and completion status
- Enhanced TODO.md to focus on remaining optional features only

### Notes

- **Bazarr Feature Parity**: Achieved full compatibility for core operations
- **Production Ready**: Complete authentication, authorization, and monitoring
- **Remaining Work**: Only optional advanced features (5% of project scope)

## [0.3.9] - 2025-06-26

### Changed

- `GoogleTranslate` now uses the official Google Cloud SDK instead of manual
  HTTP requests.

### Added

- Initial implementation of Subtitle Manager CLI.
- Commands: `convert`, `merge`, `translate`, and `history`.
- Google Translate and ChatGPT support.
- SQLite storage for translation history.
- Component based logging with adjustable levels.
- Documentation updates and initial technical design.

## [0.1.1] - 2025-06-06

### Added

- Expanded technical design document with detailed implementation plans.
- Updated README and TODO to reference the comprehensive documentation.

## [0.1.2] - 2025-06-07

### Added

- Documented Bazarr feature set in `docs/BAZARR_FEATURES.md`
- Linked feature reference from README, TODO and TECHNICAL_DESIGN

## [0.1.3] - 2025-06-08

### Added

- Subtitle extraction from media containers using `ffmpeg`.
- New `extract` CLI command.
- React based web UI built with Vite under `webui/`.
- `web` command to serve the embedded single page application.
- Technical design and TODO updated with web front end plan.

## [0.1.4] - 2025-06-09

### Added

- OpenSubtitles provider and `fetch` CLI command.
- Provider implementation documented in README and TODO.

## [0.1.5] - 2025-06-10

### Added

- Batch translation command for concurrent processing of multiple files.
- Helper functions `TranslateFileToSRT` and `TranslateFilesToSRT` in
  `pkg/subtitles`.
- Documentation updates for the new command and concurrency model.

## [0.1.6] - 2025-06-11

### Added

- Subscene provider with `fetch` and `watch` support.
- `grpc-server` command to expose translation service.
- Customisable ffmpeg path for subtitle extraction.
- Recursive directory watching.
- `delete` command and database deletion helper.

## [0.1.7] - 2025-06-12

### Added

- Environment variable configuration via `SM_` prefix.
- GitHub Actions workflow for continuous integration.

## [0.1.8] - 2025-06-13

### Added

- Comprehensive subtitle provider list from Bazarr documented in README and
  TODO.
- Implemented Addic7ed, BetaSeries, BSplayer, Podnapisi, TVSubtitles, Titlovi,
  LegendasDivx and GreekSubs providers.

## [0.1.9] - 2025-06-14

### Added

- Implemented the remaining subtitle providers from Bazarr's list.
- Unified provider selection via a registry.

## [0.1.10] - 2025-06-15

### Added

- Dockerfile and GitHub Actions workflow for container images.
- Container images published to GitHub Container Registry.
- Documentation updates describing provider registry and Docker usage.

## [0.2.0] - 2025-06-16

### Added

- Library scanning command to automatically download and upgrade subtitles.
- Updated README and TODO for new feature.

## [0.2.1] - 2025-06-17

### Added

- Authentication system supporting password login, one time tokens, OAuth2 and
  API keys.
- Simple user manager commands for creating users and generating API keys.
- RBAC with default roles and database backed session storage.

## [0.3.0] - 2025-06-18

### Added

- Concurrent directory scanning with worker pool.
- Sonarr and Radarr integration commands.
- Initial React web UI with login page.

## [0.3.1] - 2025-06-19

### Added

- PebbleDB backend with migration command.
- Configurable database backend via `--db-backend` flag.

## [0.3.2] - 2025-06-20

### Added

- Download history stored in database with new `downloads` command.

## [0.3.3] - 2025-06-21

### Added

- GitHub OAuth2 login support with new web server endpoints.

## [0.3.4] - 2025-06-22

### Added

- Role based access control enforced on web routes.
- `user role` command to modify user permissions.

## [0.3.5] - 2025-06-23

### Added

- One time login tokens with `user token` and `login-token` commands.

## [0.3.6] - 2025-06-24

### Added

- Manual subtitle search command with `search` functionality.

## [0.3.7] - 2025-06-24

### Added

- `user list` command to display existing accounts.

## [0.3.8] - 2025-06-25

### Added

- `media_items` table to store video metadata for library scanning.
- Library scan command `scanlib` populating the `media_items` table.
- REST endpoint `/api/extract` exposing subtitle extraction from media.
- `media_items` table to store video metadata for library scanning.
- Library scan command `scanlib` populating the `media_items` table.
- REST endpoint `/api/extract` exposing subtitle extraction from media.

## [0.3.9] - 2025-06-26

### Changed

- `GoogleTranslate` now uses the official Google Cloud SDK instead of manual
  HTTP requests.

## [0.3.10] - 2025-06-27

### Added

- Documentation update verifying PR workflow.

## [0.3.11] - 2025-07-06

### Added

- `metadata fetch` supports `--id` for TMDB lookups.
- Sonarr/Radarr sync detects library conflicts and logs them.

## [0.3.12] - 2025-07-07

### Added

- `metadata pick` command for interactive TMDB selection.
- `metadata show` prints release group, alternate titles and locks.
- `monitor autosync` command for scheduled Sonarr/Radarr synchronization.

## [0.3.13] - 2025-07-09

### Added

- `radarr-sync` command for one-time library synchronization.
- Metadata refresh respects field locks set via `metadata update`.
- `metadata apply` command to write selected metadata to library items while
  respecting field locks.
