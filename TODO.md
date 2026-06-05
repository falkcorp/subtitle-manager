# TODO

This file tracks remaining work and implementation status for Subtitle Manager.
**Note: Subtitle Manager is close to feature complete but still requires several
enhancements before a stable 1.0 release.**

---

## 🚀 Planned Refactor: Hybrid Protobuf + Go Types + gcommon

We are migrating to a hybrid model for all shared types and business logic:

- **Protobufs** will define the canonical data models (e.g., LanguageProfile,
  MediaItem, etc.)
- **Go types** will be generated from Protobufs for use in all Go projects
- **gcommon** will contain all shared business logic, helpers, and interface
  implementations, importing the generated types
- **Other languages** (Python, JS, etc.) can generate types from the same
  Protobufs as needed
- **All work for this migration will be done on the `gcommon-refactor` branch**
- **Main branches** will continue using Go types and type packages until the
  migration is complete

This approach will:

- Eliminate duplication and type drift
- Enable cross-language compatibility
- Centralize business logic and validation
- Allow for incremental migration and testing

---

## 🚧 Remaining Work

### 📝 Recent Updates

- [x] - [x] 🟡 **General**: Create GitHub projects for open features
- [ ] - [ ] 🟡 **General**: Investigate remote configuration for rebase script
- [ ] - [ ] 🟡 **General**: Monitor AI rebase workflow results
- [ ] - [ ] 🟡 **Search**: Add integration tests for cached provider results
- [x] - [x] 🟢 **General**: Ensure GitHub CLI has project scopes for
        create-github-projects.sh
- [ ] Add 'whisper start' and 'whisper stop' commands
- [ ] Add DB update support for download results
- [ ] Add `sonarr-sync` command for one-time library sync.
- [ ] Add configurable batch size for translations
- [ ] Add context timeouts for gRPC translations
- [ ] Add regression test for sync batch endpoint
- [ ] Add tests for DirectoryChooser directory detection
- [x] Add tests for structured logging
- [ ] Align remaining services with gcommon proto
- [ ] Deduplicated S3 and storage flags in root command
- [ ] Document cron support for monitor autosync
- [ ] Document monitor autosync usage
- [ ] Ensure SQLite integration tests run in CI with sqlite build tag
- [ ] Implement SupportedServices helper in translator package
- [ ] Implement provider fallback logic for failed searches
- [ ] Implement provider metadata usage across services
- [ ] Investigate cross-correlation method to achieve near-perfect subtitle
      alignment
- [ ] Investigate remaining web UI test failures
- [ ] Normalize provider order in CLI search cache key
- [ ] Normalize provider order when generating search cache keys
- [ ] Review directory add workflow
- [ ] Test docker-local make target
- [ ] Validate rebase scripts skip fetch when origin is absent
- [ ] Verify PAT-based merge conflict workflow
- [x] Verify Prometheus metrics via new provider
- [ ] Verify SQLite schema migration for media_profiles
- [ ] Verify built-in GitHub project automation is configured correctly
- [ ] Verify custom port mapping works in stack
- [ ] Verify new stash and remote checks in rebase scripts
- [ ] Verify root command flags are initialized only once

### 🎯 Database Backend Migration - Complete

**Full PebbleDB Implementation**: All SQLite-backed features now available in
PebbleDB for pure Go deployments.

#### Migration Summary

- **✅ Complete PebbleDB Implementation**: All SQLite features migrated to
  PebbleDB
  - Login tokens and role-based access control
  - User sessions and API key management
  - Dashboard preferences and tag associations
  - Subtitle history and metadata storage
  - Provider configuration and settings

- **✅ Pure Go Builds**: CGO-free builds using PebbleDB (`-tags nosqlite`)
  - No CGO dependencies required
  - Smaller binary size and easier deployment
  - High-performance embedded key-value store

- **✅ SQLite Builds**: Traditional CGO builds with SQLite (`-tags sqlite`)
  - Full SQL querying capabilities
  - Migration support from existing databases
  - Backward compatibility maintained

- **✅ Interface Compatibility**: Fixed ID type mapping (int64 ↔ UUID) for
  seamless operation
  - Unified interface for both database backends
  - Transparent operation switching
  - Consistent API behavior

- **✅ Comprehensive Test Suite**: All tests pass in both build modes
  - Unit tests for both database backends
  - Integration tests with consistent behavior
  - Performance benchmarks for optimization

#### Build Options

```bash
# Pure Go build (default) - uses PebbleDB
go build -tags nosqlite .

# SQLite build - requires CGO
go build -tags sqlite .

# Migration between backends
subtitle-manager migrate old.db newdir
```

Both backends provide identical functionality for all user-facing features
including authentication, session management, API keys, dashboard preferences,
tags, permissions, and subtitle history.

### ✅ Completed Major Features

#### Subtitle Quality Scoring System - Complete

**Comprehensive subtitle evaluation and automatic selection based on multiple
criteria**:

- **Provider Reliability Scoring**: Trusted providers and user reputation
  weighting
- **Release Match Quality**: Perfect release group matches and source quality
  alignment
- **Format Preferences**: SRT format prioritization with configurable format
  hierarchies
- **Metadata Quality Assessment**: Upload date recency, download popularity, and
  user ratings
- **User Preference Integration**: Hearing impaired (HI) and forced subtitle
  preferences
- **File Size Correlation**: Optimal subtitle-to-video file size ratio analysis

**CLI Integration**:

```bash
subtitle-manager fetch-scored movie.mkv en output.srt --min-score 60
subtitle-manager fetch-scored --prefer-providers opensubtitles,subscene movie.mkv en output.srt
```

**Configuration Support**: Full YAML configuration with weighted scoring
criteria and provider preferences.

#### Automatic Subtitle Synchronization - Complete

**Advanced subtitle synchronization using multiple methods for precise timing
alignment**:

- **Audio Transcription Synchronization**:
  - Whisper API integration for speech-to-text timing reference
  - Local Whisper service support for offline processing
  - Configurable model selection (base, small, medium, large)
  - Automatic language detection for optimal transcription

- **Embedded Subtitle Synchronization**:
  - Advanced track selection with audio and subtitle stream indices
  - Multi-track processing with format preservation
  - Support for all major subtitle formats (SRT, VTT, ASS, SSA)
  - Metadata and styling preservation

- **Hybrid Synchronization**:
  - Configurable weighted averaging between audio and embedded methods
  - Intelligent fallback when one method fails
  - CPU vs accuracy slider for performance tuning
  - Quality validation and sync result verification

- **Translation Integration**:
  - Synchronize and translate in single operation
  - Dual-language support with grammar structure alignment
  - Language profile integration for automatic selection

**CLI Usage**:

```bash
subtitle-manager sync movie.mkv subtitles.srt output.srt --use-audio --use-embedded --audio-weight 0.7
subtitle-manager sync movie.mkv subtitles.srt output.srt --use-audio --translate es
subtitle-manager syncbatch --config sync-config.json
```

**Web UI Integration**: Complete synchronization interface with progress
tracking, quality metrics, and batch processing.

#### Manual Subtitle Search Interface - Complete

**Comprehensive manual search interface similar to Bazarr with advanced
filtering**:

- **Multi-Provider Search**: Parallel searching across multiple providers with
  status indicators
- **Advanced Filters**: Season/episode numbers, release year, release group
  specification
- **Enhanced Results**: Sortable tables, rating systems, provider badges,
  language tags
- **Subtitle Preview**: Modal dialogs with content preview and provider
  information
- **Batch Operations**: Multi-select downloads with visual feedback
- **Search History**: Persistent history with quick replay and metadata display

**API Endpoints**: Complete REST API with `/api/search`, `/api/search/preview`,
and `/api/search/history`.

- [x] **Tagging System**: Implement tagging for language options and
      preferences.
      ([#891](https://github.com/jdfalk/subtitle-manager/issues/891))
  - [x] Add tags table to the database and expose tag management in settings.
  - [x] Apply tags to media and users to drive language selection and provider
        behavior.
  - [x] **Tag Management CLI**: List, add, and remove tags from the command
        line. ([#1288](https://github.com/jdfalk/subtitle-manager/issues/1288))
- [x] **Automated Maintenance Tasks**: Periodic database cleanup, metadata
      refresh, and disk scans. See [docs/SCHEDULING.md](docs/SCHEDULING.md).
      ([#888](https://github.com/jdfalk/subtitle-manager/issues/888))
- [x] **Azure Blob Storage Support**: Initial Azure cloud storage provider.
- [x] **Whisper Container Integration**: Optional embedded Whisper service.
      ([#1132](https://github.com/jdfalk/subtitle-manager/issues/1132))
  - [x] Launch
        [ahmetoner/whisper-asr-webservice](https://github.com/ahmetoner/whisper-asr-webservice)
        when `ENABLE_WHISPER=1` is set.
  - [x] Document requirement for NVIDIA Container Toolkit and add Docker init to
        manage the service lifecycle.
  - [x] Add `whisper status` command for checking container health.
- [x] **Automated Maintenance Tasks**: Periodic database cleanup, metadata
      refresh, and disk scans. See [docs/SCHEDULING.md](docs/SCHEDULING.md).
      ([#888](https://github.com/jdfalk/subtitle-manager/issues/888))
- [x] **Azure Blob Storage Support**: Initial Azure cloud storage provider.
- [x] **Sonarr/Radarr Sync Enhancements**: Continuous sync jobs and conflict
      resolution via new `monitor autosync` command, with conflict detection
      implemented.
      ([#889](https://github.com/jdfalk/subtitle-manager/issues/889))
  - [x] Added `monitor autosync` command to run scheduled library syncs.
  - [x] Add `radarr-sync` command for one-time library sync.
- [x] **Online Metadata Sources**: Fetch languages, ratings, and episode data
      from external APIs. `metadata fetch` command now supports `--id` for
      direct TMDB lookup.
      ([#351](https://github.com/jdfalk/subtitle-manager/issues/351),
      [#890](https://github.com/jdfalk/subtitle-manager/issues/890))
- [x] **Media Metadata Editor**: Provide manual editing interface.
      ([#1135](https://github.com/jdfalk/subtitle-manager/issues/1135))
  - [x] Allow manual metadata search and selection during import via
        `metadata pick` command.
  - [x] Support field-level locks to prevent unwanted updates via
        `metadata show` command.
  - [x] Added `metadata apply` command to write selected metadata to the
        database while honoring field locks.
  - [x] CLI interactive search and selection when fetching metadata.
- [x] **Search Result Caching**: Cache manual search results for faster
      responses.
      ([#1330](https://github.com/jdfalk/subtitle-manager/issues/1330))

### Universal Tagging System Implementation

**🎯 Core Objective**: Implement a unified tagging interface that works across
all entity types with consistent API and database schema.

#### Architecture Overview

The unified tagging system will support these entity types:

- **Media Profiles**: Content-specific tagging for movies/TV shows
- **Language Profiles**: Language preference and priority tagging
- **Movies**: Individual movie tagging for categorization
- **TV Series**: Series-level tagging and organization
- **Providers**: Provider instance tagging for selection logic
- **Users**: User preference and access control tagging

#### Implementation Plan

- [x] **Universal Tag Interface**: Create `TaggedEntity` interface that all
      taggable types implement
- [x] **Enhanced Tag Type**: Add `Type`, `EntityType`, `Color`, and
      `Description` fields to Tag struct
- [x] **Unified Database Schema**: Single `tag_associations` table with
      polymorphic relationships
- [x] **Consistent API**: Standardized REST endpoints following
      `/api/{entityType}/{id}/tags` pattern
- [x] **Provider Instance Integration**: Extend provider instances to use the
      unified tagging system
- [x] **Bulk Operations**: Tag multiple entities simultaneously with batch API
      endpoints

#### Database Schema Changes

```sql
-- Enhanced tags table with type information
CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Universal tag associations table
CREATE TABLE tag_associations (
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (tag_id, entity_type, entity_id)
);
```

#### API Standardization

All entity types will support consistent tagging operations:

- `GET /api/{entityType}/{id}/tags` - List tags for entity
- `POST /api/{entityType}/{id}/tags` - Add tag to entity
- `DELETE /api/{entityType}/{id}/tags/{tagId}` - Remove tag from entity
- `GET /api/tags?entity_type={type}` - List tags by entity type
- `POST /api/tags/bulk` - Bulk tag operations

#### Integration with Provider System

Provider instances will use the unified tagging system for:

- **Selection Logic**: Choose providers based on content tags
- **Priority Weighting**: Tag-based provider prioritization
- **Backoff Management**: Per-tag backoff strategies
- **Configuration**: Tag-driven provider configuration

#### Legacy Migration

Current tag implementations will be migrated to the unified system:

- User tags → Universal system with `entity_type='user'`
- Media tags → Universal system with `entity_type='media'`
- Provider tags → Enhanced instance system with proper associations

#### Validation and Testing

- [x] **Unit Tests**: Comprehensive test coverage for all tagging operations
- [x] **Integration Tests**: End-to-end tag workflow testing
- [x] **Migration Tests**: Verify legacy data migration integrity
- [x] **Performance Tests**: Ensure tagging operations scale appropriately

## 🧪 Testing & Quality Assurance

### E2E Test Infrastructure

- [x] **Fixed E2E test timeout issues**: Created proper `workflow.spec.js` test
      file with comprehensive workflows
  - Enhanced login test robustness in `app.spec.js`
- [x] **Add media library E2E tests**: Test file upload, scanning, and subtitle
      operations. Added scan workflow test to verify progress display.
- [x] **Add provider configuration tests**: Test subtitle provider setup and
      validation
- [x] **Add bulk operations tests**: Test batch subtitle download and processing

## 🚨 High Priority UI/UX Improvements

> **📋 Complete Implementation Plan Available**: All UI/UX improvements have
> been consolidated into `UI_UX_IMPLEMENTATION_PLAN_COMPLETE.md` with detailed
> code samples, time estimates, and step-by-step implementation guidance.
>
> **Total Project Scope**: 82-114 hours for complete UI/UX overhaul with
> actionable code samples for junior developers.

### Navigation and User Experience

- [x] **Fix user management display**: System/users shows blank usernames - need
      to properly display user data
- [x] **Move user management to settings**: Users interface should be part of
      settings, not system
- [x] **Implement working back button**: Navigation history and proper back
      button functionality
- [x] **Add sidebar pinning**: Allow users to pin/unpin the sidebar for better
      UX
- [x] **Reorganize navigation order**: Dashboard → Media Library → Wanted →
      History → Settings → System
- [x] **Restructure tools**: Move Extract/Translate/Convert to Tools section or
      integrate into System

### Settings Page Enhancements

- [x] **Enhance General Settings**: Add Bazarr-compatible settings (Host:
      Address/Port/Base URL, Proxy, Updates, Logging with filters, Backups,
      Analytics)
- [x] **Improve Database Settings**: Add comprehensive database information and
      management options
- [x] **Redesign Authentication Page**: Card-based UI for each auth method with
      enable/disable toggles
- [x] **Add OAuth2 management**: Generate/regenerate client ID/secret, reset to
      defaults
- [x] **Enhance Notifications**: Card-based interface for each notification
      method with test buttons
- [x] **Create Languages Page**: Global language settings for subtitle downloads
      (like Bazarr)
- [x] **Add Scheduler Settings**: Integration into general settings or separate
      page

### Provider System Improvements

- [x] **Fix provider configuration modals**: Proper provider selection dropdowns
      and configuration options
- [x] **Improve embedded provider config**: Working dropdown and proper
      configuration display
- [x] **Implement global language settings**: Move language settings from
      provider-level to global
- [x] **Add language profiles**: Bazarr-style language profiles for different
      content types

## 🔄 Low Priority Improvements

### Testing Enhancements

- [x] **Mock PostgreSQL tests**: Mock out PostgreSQL logic as much as possible
      to reduce dependency on local PostgreSQL installation for test coverage
  - Priority: Low (current approach works well)

### Security Enhancements

- [x] **[nitpick] Add a Referrer-Policy header** (`no-referrer`) to enhance
      privacy and reduce referrer leakage

### Development Tools

- [x] **Comprehensive Pre-commit Hooks**: Consolidated duplicate hook scripts
      into a single comprehensive pre-commit hook
  - Prevents CI failures by catching issues locally before push

---

## 📊 Complete Bazarr Feature Parity Analysis

This section provides a comprehensive comparison between Bazarr and Subtitle
Manager, cataloguing every feature from Bazarr's
[official documentation](https://wiki.bazarr.media/) and
[repository](https://github.com/morpheus65535/bazarr).

### Executive Summary: Feature Parity Status

| **Feature Category**   | **Bazarr Status**         | **Our Implementation**                                                               | **Gold Standard**             |
| ---------------------- | ------------------------- | ------------------------------------------------------------------------------------ | ----------------------------- |
| **Subtitle Providers** | 40+ providers supported   | ✅ 40+ providers ([registry.go](pkg/providers/registry.go))                          | ✅ Full parity achieved       |
| **Web Interface**      | Modern React UI           | ✅ Complete React app ([webui/src/](webui/src/))                                     | ✅ Production ready           |
| **Authentication**     | Basic auth + API keys     | ✅ Password, OAuth2, API keys, RBAC ([pkg/auth/](pkg/auth/))                         | ✅ Enterprise grade           |
| **Library Management** | Sonarr/Radarr integration | ✅ Full integration ([cmd/sonarr.go](cmd/sonarr.go), [cmd/radarr.go](cmd/radarr.go)) | ✅ Complete                   |
| **Database Support**   | SQLite + PostgreSQL       | ✅ SQLite, PebbleDB, PostgreSQL ([pkg/database/](pkg/database/))                     | ✅ Full enterprise DB support |
| **REST API**           | Comprehensive API         | ✅ Full coverage ([pkg/webserver/](pkg/webserver/))                                  | ✅ Complete                   |
| **Configuration**      | YAML + Web UI             | ✅ Viper + Web settings ([cmd/root.go](cmd/root.go))                                 | ✅ Full parity                |
| **Webhooks**           | Basic webhook support     | ✅ Sonarr/Radarr/custom endpoints ([pkg/webhooks/](pkg/webhooks/))                   | ✅ Enhanced system            |
| **Notifications**      | Apprise integration       | ✅ Discord/Telegram/SMTP ([pkg/notifications/](pkg/notifications/))                  | ✅ Multi-provider support     |
| **Anti-Captcha**       | Anti-captcha.com          | ✅ Anti-Captcha.com + 2captcha.com ([pkg/captcha/](pkg/captcha/))                    | ✅ Multi-service support      |
| **Scheduling**         | Basic cron support        | ✅ Advanced cron expressions ([pkg/scheduler/](pkg/scheduler/))                      | ✅ Enhanced scheduling        |

### Detailed Feature Analysis

#### 1. Core Subtitle Operations – Complete

| Bazarr Feature               | Implementation Status | Code Reference                       |
| ---------------------------- | --------------------- | ------------------------------------ |
| Format conversion            | ✅ Complete           | [cmd/convert.go](cmd/convert.go)     |
| Subtitle merging             | ✅ Complete           | [cmd/merge.go](cmd/merge.go)         |
| Media extraction             | ✅ Complete           | [cmd/extract.go](cmd/extract.go)     |
| Translation (Google/ChatGPT) | ✅ Complete           | [cmd/translate.go](cmd/translate.go) |
| Batch processing             | ✅ Complete           | [cmd/batch.go](cmd/batch.go)         |
| History tracking             | ✅ Complete           | [cmd/history.go](cmd/history.go)     |

#### 2. Authentication & Authorization – Complete

| Bazarr Feature            | Implementation Status | Code Reference                                   |
| ------------------------- | --------------------- | ------------------------------------------------ |
| Password authentication   | ✅ Complete           | [pkg/auth/auth.go](pkg/auth/auth.go)             |
| API key management        | ✅ Complete           | [cmd/user.go](cmd/user.go)                       |
| Session management        | ✅ Complete           | [pkg/webserver/auth.go](pkg/webserver/auth.go)   |
| Role-based access control | ✅ Complete           | [pkg/auth/rbac.go](pkg/auth/rbac.go)             |
| OAuth2 (GitHub)           | ✅ Complete           | [pkg/webserver/oauth.go](pkg/webserver/oauth.go) |
| One-time tokens           | ✅ Complete           | [cmd/user.go](cmd/user.go)                       |

#### 3. Subtitle Providers – Complete - Full Bazarr Parity

| Provider Category      | Bazarr Count | Our Implementation                                           | Status             |
| ---------------------- | ------------ | ------------------------------------------------------------ | ------------------ |
| Major providers        | ~40          | ✅ 40+ providers                                             | ✅ Parity achieved |
| OpenSubtitles variants | 3            | ✅ Complete ([opensubtitles/](pkg/providers/opensubtitles/)) | ✅                 |
| Regional providers     | ~25          | ✅ Complete (Greek, Turkish, etc.)                           | ✅                 |
| Torrent-based          | ~8           | ✅ Complete (YIFY, etc.)                                     | ✅                 |
| Embedded extraction    | 1            | ✅ Complete ([embedded/](pkg/providers/embedded/))           | ✅                 |
| Whisper transcription  | 1            | ✅ Complete ([transcribe.go](cmd/transcribe.go))             | ✅                 |

#### 4. Web Interface Pages – Complete

| Bazarr Page         | Implementation Status     | Code Reference                                     |
| ------------------- | ------------------------- | -------------------------------------------------- |
| Dashboard           | ✅ Complete               | [webui/src/Dashboard.jsx](webui/src/Dashboard.jsx) |
| Settings            | ✅ Complete               | [webui/src/Settings.jsx](webui/src/Settings.jsx)   |
| History             | ✅ Complete               | [webui/src/History.jsx](webui/src/History.jsx)     |
| Wanted              | ✅ Complete               | [webui/src/Wanted.jsx](webui/src/Wanted.jsx)       |
| System/Logs         | ✅ Complete               | [webui/src/System.jsx](webui/src/System.jsx)       |
| Providers           | ✅ Integrated in Settings | Settings page                                      |
| Subtitle extraction | ✅ Complete               | [webui/src/Extract.jsx](webui/src/Extract.jsx)     |

#### 5. Integration Features ✅ 90% Complete

| Bazarr Feature     | Implementation Status | Code Reference                                     |
| ------------------ | --------------------- | -------------------------------------------------- |
| Sonarr integration | ✅ Complete           | [cmd/sonarr.go](cmd/sonarr.go)                     |
| Radarr integration | ✅ Complete           | [cmd/radarr.go](cmd/radarr.go)                     |
| Plex integration   | ✅ Complete           | [cmd/plex.go](cmd/plex.go), [pkg/plex/](pkg/plex/) |
| Library scanning   | ✅ Complete           | [cmd/scan.go](cmd/scan.go)                         |
| Directory watching | ✅ Complete           | [cmd/watch.go](cmd/watch.go)                       |
| Webhooks           | ✅ Complete           | [pkg/webhooks](pkg/webhooks/)                      |
| Notifications      | 🔶 Planned            | [TODO] Discord/Telegram/Email                      |

#### 6. Advanced Features – Complete

| Bazarr Feature        | Implementation Status | Code Reference                            |
| --------------------- | --------------------- | ----------------------------------------- |
| PostgreSQL support    | ✅ Complete           | SQLite, PebbleDB and PostgreSQL available |
| Reverse proxy support | 🔶 Partial            | Basic configuration available             |
| Anti-captcha service  | ✅ Complete           | [pkg/captcha/](pkg/captcha/)              |
| Performance tuning    | ✅ Complete           | Concurrent workers, pools                 |
| Custom scheduling     | ✅ Complete           | [pkg/scheduler/](pkg/scheduler/)          |
| Bazarr config import  | ✅ Complete           | [cmd/import.go](cmd/import.go)            |
| Webhook system        | ✅ Complete           | [pkg/webhooks/](pkg/webhooks/)            |
| Notifications         | ✅ Complete           | [pkg/notifications/](pkg/notifications/)  |

### Complete Provider Implementation Analysis

**Reference**:
[Bazarr Providers](https://wiki.bazarr.media/Additional-Configuration/Settings/#providers)
vs [Our Registry](pkg/providers/registry.go)

#### Implemented Providers (40+ with Full Parity)

| Provider                | Bazarr | Our Implementation | Documentation                                                               |
| ----------------------- | ------ | ------------------ | --------------------------------------------------------------------------- |
| Addic7ed                | ✅     | ✅                 | [addic7ed/](pkg/providers/addic7ed/)                                        |
| AnimeKalesi             | ✅     | ✅                 | [animekalesi/](pkg/providers/animekalesi/)                                  |
| Animetosho              | ✅     | ✅                 | [animetosho/](pkg/providers/animetosho/)                                    |
| Assrt                   | ✅     | ✅                 | [assrt/](pkg/providers/assrt/)                                              |
| AvistaZ/CinemaZ         | ✅     | ✅                 | [avistaz/](pkg/providers/avistaz/)                                          |
| BetaSeries              | ✅     | ✅                 | [betaseries/](pkg/providers/betaseries/)                                    |
| BSplayer                | ✅     | ✅                 | [bsplayer/](pkg/providers/bsplayer/)                                        |
| Embedded Subtitles      | ✅     | ✅                 | [embedded/](pkg/providers/embedded/)                                        |
| Gestdown.info           | ✅     | ✅                 | [gestdown/](pkg/providers/gestdown/)                                        |
| GreekSubs               | ✅     | ✅                 | [greeksubs/](pkg/providers/greeksubs/)                                      |
| GreekSubtitles          | ✅     | ✅                 | [greeksubtitles/](pkg/providers/greeksubtitles/)                            |
| HDBits.org              | ✅     | ✅                 | [hdbits/](pkg/providers/hdbits/)                                            |
| Hosszupuska             | ✅     | ✅                 | [hosszupuska/](pkg/providers/hosszupuska/)                                  |
| Karagarga.in            | ✅     | ✅                 | [karagarga/](pkg/providers/karagarga/)                                      |
| Ktuvit                  | ✅     | ✅                 | [ktuvit/](pkg/providers/ktuvit/)                                            |
| LegendasDivx            | ✅     | ✅                 | [legendasdivx/](pkg/providers/legendasdivx/)                                |
| Legendas.net            | ✅     | ✅                 | [legendasnet/](pkg/providers/legendasnet/)                                  |
| Napiprojekt             | ✅     | ✅                 | [napiprojekt/](pkg/providers/napiprojekt/)                                  |
| Napisy24                | ✅     | ✅                 | [napisy24/](pkg/providers/napisy24/)                                        |
| Nekur                   | ✅     | ✅                 | [nekur/](pkg/providers/nekur/)                                              |
| OpenSubtitles.com       | ✅     | ✅                 | [opensubtitlescom/](pkg/providers/opensubtitlescom/)                        |
| OpenSubtitles.org (VIP) | ✅     | ✅                 | [opensubtitlesvip/](pkg/providers/opensubtitlesvip/)                        |
| Podnapisi               | ✅     | ✅                 | [podnapisi/](pkg/providers/podnapisi/)                                      |
| RegieLive               | ✅     | ✅                 | [regielive/](pkg/providers/regielive/)                                      |
| Sous-Titres.eu          | ✅     | ✅                 | [soustitres/](pkg/providers/soustitres/)                                    |
| Subdivx                 | ✅     | ✅                 | [subdivx/](pkg/providers/subdivx/)                                          |
| subf2m.co               | ✅     | ✅                 | [subf2m/](pkg/providers/subf2m/)                                            |
| Subs.sab.bz             | ✅     | ✅                 | [subssabbz/](pkg/providers/subssabbz/)                                      |
| Subs4Free               | ✅     | ✅                 | [subs4free/](pkg/providers/subs4free/)                                      |
| Subs4Series             | ✅     | ✅                 | [subs4series/](pkg/providers/subs4series/)                                  |
| Subscene                | ✅     | ✅                 | [subscene/](pkg/providers/subscene/)                                        |
| Subscenter              | ✅     | ✅                 | [subscenter/](pkg/providers/subscenter/)                                    |
| Subsunacs.net           | ✅     | ✅                 | [subsunacs/](pkg/providers/subsunacs/)                                      |
| SubSynchro              | ✅     | ✅                 | [subsynchro/](pkg/providers/subsynchro/)                                    |
| Subtitrari-noi.ro       | ✅     | ✅                 | [subtitrarinoi/](pkg/providers/subtitrarinoi/)                              |
| subtitri.id.lv          | ✅     | ✅                 | [subtitriidlv/](pkg/providers/subtitriidlv/)                                |
| Subtitulamos.tv         | ✅     | ✅                 | [subtitulamos/](pkg/providers/subtitulamos/)                                |
| Supersubtitles          | ✅     | ✅                 | [supersubtitles/](pkg/providers/supersubtitles/)                            |
| Titlovi                 | ✅     | ✅                 | [titlovi/](pkg/providers/titlovi/)                                          |
| Titrari.ro              | ✅     | ✅                 | [titrariro/](pkg/providers/titrariro/)                                      |
| Titulky.com             | ✅     | ✅                 | [titulky/](pkg/providers/titulky/)                                          |
| Turkcealtyazi.org       | ✅     | ✅                 | [turkcealtyazi/](pkg/providers/turkcealtyazi/)                              |
| TuSubtitulo             | ✅     | ✅                 | [tusubtitulo/](pkg/providers/tusubtitulo/)                                  |
| TVSubtitles             | ✅     | ✅                 | [tvsubtitles/](pkg/providers/tvsubtitles/)                                  |
| Whisper                 | ✅     | ✅                 | [whisper/](pkg/providers/whisper/) + [cmd/transcribe.go](cmd/transcribe.go) |
| Wizdom                  | ✅     | ✅                 | [wizdom/](pkg/providers/wizdom/)                                            |
| XSubs                   | ✅     | ✅                 | [xsubs/](pkg/providers/xsubs/)                                              |
| Yavka.net               | ✅     | ✅                 | [yavka/](pkg/providers/yavka/)                                              |
| YIFY Subtitles          | ✅     | ✅                 | [yifysubtitles/](pkg/providers/yifysubtitles/)                              |
| Zimuku                  | ✅     | ✅                 | [zimuku/](pkg/providers/zimuku/)                                            |

#### Bazarr Settings Comparison Analysis

**Reference**:
[Bazarr Settings](https://wiki.bazarr.media/Additional-Configuration/Settings/)

| Bazarr Setting Category      | Implementation Status | Our Location                     | Bazarr Reference                                                                                                                      |
| ---------------------------- | --------------------- | -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| **Host Settings**            | ✅ Complete           | [cmd/root.go](cmd/root.go)       | [Host](https://wiki.bazarr.media/Additional-Configuration/Settings/#host)                                                             |
| - Bind Address               | ✅ Complete           | Viper config                     | -                                                                                                                                     |
| - Port Number                | ✅ Complete           | Viper config                     | -                                                                                                                                     |
| - URL Base                   | ✅ Complete           | Reverse proxy support            | [URL Base](https://wiki.bazarr.media/Additional-Configuration/Settings/#url-base)                                                     |
| **Security Settings**        | ✅ Complete           | [pkg/auth/](pkg/auth/)           | [Security](https://wiki.bazarr.media/Additional-Configuration/Settings/#security)                                                     |
| - Authentication             | ✅ Enhanced           | Multi-mode auth                  | [Authentication](https://wiki.bazarr.media/Additional-Configuration/Settings/#authentication)                                         |
| - Username/Password          | ✅ Complete           | Hashed storage                   | -                                                                                                                                     |
| - API Key                    | ✅ Enhanced           | Multiple keys                    | [API Key](https://wiki.bazarr.media/Additional-Configuration/Settings/#api-key)                                                       |
| **Proxy Settings**           | ✅ Complete           | HTTP client config               | [Proxy](https://wiki.bazarr.media/Additional-Configuration/Settings/#proxy)                                                           |
| **Sonarr Integration**       | ✅ Complete           | [cmd/sonarr.go](cmd/sonarr.go)   | [Sonarr](https://wiki.bazarr.media/Additional-Configuration/Settings/#sonarr)                                                         |
| - Host Configuration         | ✅ Complete           | Viper config                     | -                                                                                                                                     |
| - API Key                    | ✅ Complete           | Secure storage                   | -                                                                                                                                     |
| - Path Mappings              | ✅ Complete           | Config mappings                  | [Path Mappings](https://wiki.bazarr.media/Additional-Configuration/Settings/#path-mappings)                                           |
| **Radarr Integration**       | ✅ Complete           | [cmd/radarr.go](cmd/radarr.go)   | [Radarr](https://wiki.bazarr.media/Additional-Configuration/Settings/#radarr)                                                         |
| **Subtitle Options**         | ✅ Complete           | [pkg/subtitles/](pkg/subtitles/) | [Subtitles](https://wiki.bazarr.media/Additional-Configuration/Settings/#subtitles)                                                   |
| - Subtitle Folder            | ✅ Complete           | Config option                    | -                                                                                                                                     |
| - Upgrade Logic              | ✅ Complete           | Auto-upgrade                     | [Upgrade Previously Downloaded](https://wiki.bazarr.media/Additional-Configuration/Settings/#upgrade-previously-downloaded-subtitles) |
| **Anti-Captcha**             | ✅ Basic              | [pkg/captcha/](pkg/captcha/)     | [Anti-Captcha Options](https://wiki.bazarr.media/Additional-Configuration/Settings/#anti-captcha-options)                             |
| **Performance/Optimization** | ✅ Complete           | Worker pools                     | [Performance](https://wiki.bazarr.media/Additional-Configuration/Settings/#performance-optimization)                                  |
| - Adaptive Searching         | 🔶 Basic              | Simple scheduling                | [Adaptive Searching](https://wiki.bazarr.media/Additional-Configuration/Settings/#adaptive-searching)                                 |
| - Simultaneous Search        | ✅ Complete           | Concurrent workers               | -                                                                                                                                     |
| - Embedded Subtitles         | ✅ Complete           | Full support                     | [Use Embedded Subtitles](https://wiki.bazarr.media/Additional-Configuration/Settings/#use-embedded-subtitles)                         |
| **Post-Processing**          | ✅ Complete           | UTF-8 encoding                   | [Post-Processing](https://wiki.bazarr.media/Additional-Configuration/Settings/#post-processing)                                       |
| **Languages**                | ✅ Complete           | 180+ languages                   | [Languages](https://wiki.bazarr.media/Additional-Configuration/Settings/#languages)                                                   |
| **Providers**                | ✅ Complete           | Full registry                    | [Providers](https://wiki.bazarr.media/Additional-Configuration/Settings/#providers)                                                   |
| **Notifications**            | ✅ Basic              | Infrastructure ready             | [Notifications](https://wiki.bazarr.media/Additional-Configuration/Settings/#notifications)                                           |
| **Scheduler**                | ✅ Basic              | Auto-scan available              | [Scheduler](https://wiki.bazarr.media/Additional-Configuration/Settings/#scheduler)                                                   |

### Missing Features Analysis

#### High Priority Missing (1% of project)

1. **PostgreSQL Backend** - Enterprise database support
   - Status: ✅ Complete for large deployments
   - Current: SQLite, PebbleDB and PostgreSQL fully functional
   - Reference:
     [PostgreSQL Database](https://wiki.bazarr.media/Additional-Configuration/PostgreSQL-Database/)

2. **Advanced Webhook System** - Enhanced event notifications
   - Status: ✅ Complete with Sonarr/Radarr/custom webhook endpoints implemented
   - Reference:
     [Webhooks](https://wiki.bazarr.media/Additional-Configuration/Webhooks/)

3. **Notification Services** - Discord, Telegram, Email alerts
   - Status: ✅ Complete implementation with Discord, Telegram and SMTP
     notifiers
   - Current: Full multi-provider notification system available
   - Reference:
     [Notifications](https://wiki.bazarr.media/Additional-Configuration/Settings/#notifications)

4. **Anti-Captcha Integration** - For challenging providers
   - Status: ✅ Complete with Anti-Captcha.com and 2captcha.com support
   - Current: Multi-service captcha solving available
   - Reference:
     [Anti-Captcha Options](https://wiki.bazarr.media/Additional-Configuration/Settings/#anti-captcha-options)

5. **Advanced Scheduling** - Granular scan controls
   - Status: ✅ Complete with cron-based scheduler implemented
   - Current: Supports interval or cron expression with full granular control
   - Reference:
     [Scheduler](https://wiki.bazarr.media/Additional-Configuration/Settings/#scheduler)

#### Medium Priority Missing (Convenience Features)

1. **Bazarr Settings Import** - Automated migration
   - Status: 🔶 Partial implementation
   - Current: Manual configuration transfer works
   - Reference: [docs/BAZARR_SETTINGS_SYNC.md](docs/BAZARR_SETTINGS_SYNC.md)

2. **Advanced Scheduling** - Granular scan controls
   - Status: ✅ Cron-based scheduler implemented
   - Current: Supports interval or cron expression
   - Reference:
     [Scheduler](https://wiki.bazarr.media/Additional-Configuration/Settings/#scheduler)

3. **Reverse Proxy Enhancement** - Base URL configuration
   - Status: 🔶 Basic support exists
   - Current: Works behind reverse proxies
   - Reference:
     [Reverse Proxy Help](https://wiki.bazarr.media/Additional-Configuration/Reverse-Proxy-Help/)

---

## 🎯 Completed Optional Features

### 1. Advanced Database Support

- [x] **PostgreSQL backend**: Alternative to SQLite/PebbleDB for large
      deployments
  - Location: `pkg/database/postgres.go`
  - Reference:
    [PostgreSQL Database](https://wiki.bazarr.media/Additional-Configuration/PostgreSQL-Database/)
  - Note: PostgreSQL tests require a local PostgreSQL installation and will skip
    gracefully if unavailable
  - TODO: Mock out PostgreSQL logic as much as possible for better test coverage
    (low priority)
- [x] **Database migration tools**: Enhanced migration between database types
  - Location: `cmd/migrate.go`

### 2. Advanced Integration Features

- [x] **Webhook support**: Enhanced Plex event integration
  - Location: `pkg/webhooks/` (complete implementation)
  - Reference:
    [Webhooks](https://wiki.bazarr.media/Additional-Configuration/Webhooks/)
- [x] **Anti-captcha service**: For providers requiring captcha solving
  - Location: `pkg/captcha/` (complete with Anti-Captcha.com and 2captcha.com)
- [x] **Notification services**: Discord, Telegram, Email alerts
  - Location: `pkg/notifications/` (complete implementation)
- [x] **Reverse proxy support**: Base URL configuration for proxy deployments
  - Location: `pkg/webserver/server.go` (enhance existing)
  - Reference:
    [Reverse Proxy Help](https://wiki.bazarr.media/Additional-Configuration/Reverse-Proxy-Help/)
- [x] **Advanced scheduler**: Enhanced periodic scanning with more granular
      controls
  - Location: `pkg/scheduler/` (cron support complete)

### 3. Bazarr Configuration Import (Complete)

- [x] Implement `import-bazarr` command that fetches settings from
      `/api/system/settings` using the user's API key.
  - Location: `cmd/import.go` (complete implementation)
  - Reference: `pkg/bazarr/client.go` (complete implementation)
- [x] Map Bazarr preferences for languages, providers and network options into
      the Viper configuration.
  - Location: `pkg/bazarr/mapper.go` (complete implementation)
- [x] Document the synchronization process in `docs/BAZARR_SETTINGS_SYNC.md` and
      expose it through the welcome workflow.
  - Reference: [docs/BAZARR_SETTINGS_SYNC.md](docs/BAZARR_SETTINGS_SYNC.md)
    (complete)

### 4. Three-Column Gold Standard Comparison

| **Feature**                  | **Bazarr Implementation**    | **Subtitle Manager Status**         | **Gold Standard Target**            |
| ---------------------------- | ---------------------------- | ----------------------------------- | ----------------------------------- |
| **Core Subtitle Operations** | Python-based processing      | ✅ Go with go-astisub               | ✅ **Superior performance**         |
| **Subtitle Providers**       | 40+ providers via Subliminal | ✅ 40+ native Go clients            | ✅ **Direct API integration**       |
| **Authentication**           | Basic/Forms auth             | ✅ Multi-mode + OAuth2 + RBAC       | ✅ **Enterprise grade**             |
| **Database**                 | SQLite + PostgreSQL          | ✅ SQLite, PebbleDB, PostgreSQL     | ✅ **Full enterprise DB support**   |
| **Web Interface**            | React frontend               | ✅ Modern React + Vite              | ✅ **Production ready**             |
| **API Coverage**             | Flask REST API               | ✅ Full comprehensive Go REST API   | ✅ **Type-safe & documented**       |
| **Performance**              | Single-threaded Python       | ✅ Concurrent Go workers            | ✅ **High-performance**             |
| **Configuration**            | INI files                    | ✅ YAML + Environment vars          | ✅ **Modern config**                |
| **Container Support**        | Docker available             | ✅ Multi-arch + GHCR                | ✅ **Cloud-native**                 |
| **Library Integration**      | Sonarr/Radarr webhooks       | ✅ Direct commands + basic webhooks | 🔶 **Enhanced webhook system**      |
| **Notifications**            | Apprise integration          | ✅ Basic providers                  | 🔶 **Multi-provider notifications** |
| **Anti-Captcha**             | Anti-captcha.com             | ✅ Basic implementation             | 🔶 **Optional enhancement**         |
| **Translation**              | Not available                | ✅ Google + ChatGPT                 | ✅ **Unique feature**               |
| **Transcription**            | External Whisper             | ✅ Integrated Whisper               | ✅ **Integrated solution**          |
| **Reverse Proxy**            | Full base URL support        | 🔶 Basic support                    | 🔶 **Enhanced proxy support**       |

### Summary: Bazarr Feature Parity Achievement

#### ✅ Areas Where We Exceed Bazarr

1. **Performance**: Go's concurrency vs Python's GIL limitations
2. **Authentication**: Multi-mode auth vs basic authentication only
3. **Translation**: Built-in Google/ChatGPT vs not available
4. **Transcription**: Integrated Whisper vs external service dependency
5. **Configuration**: Modern YAML + env vars vs INI files
6. **API Design**: Type-safe Go vs Flask dynamic typing
7. **Container**: Multi-arch builds vs single architecture

#### ✅ Areas With Full Parity

1. **Subtitle Providers**: 40+ providers fully implemented
2. **Web Interface**: Complete React application with all Bazarr pages
3. **Library Management**: Full Sonarr/Radarr integration
4. **Core Operations**: All subtitle operations supported
5. **Database**: SQLite support with additional PebbleDB option

#### 🔶 Areas for Enhancement (Optional)

1. **Advanced Webhooks**: Enhanced notification system
2. **Notifications**: Discord/Telegram/Email providers
3. **Anti-Captcha**: For challenging subtitle providers
4. **Advanced Scheduling**: More granular control options

**Conclusion**: Subtitle Manager is nearing completion with the majority of core
features implemented and production readiness achieved. Several advanced
capabilities are still planned, including flexible tagging for language
settings, containerized Whisper support, and automated maintenance tasks. The
project aims for complete Bazarr parity while providing superior performance and
additional functionality such as advanced audio/embedded subtitle
synchronization with translation integration.

## ✅ Completed Major Features

### Core Functionality (Complete)

- ✅ All CLI commands: `convert`, `merge`, `translate`, `history`, `extract`,
  `fetch`, `search`, `batch`, `scan`, `watch`, `delete`, `downloads`
- ✅ Configuration with Cobra & Viper including environment variables
- ✅ Component-based logging with adjustable levels

### Authentication & Authorization (Complete)

- ✅ Password authentication with hashed credentials
- ✅ One time token generation for email logins _(v0.3.5)_
- ✅ OAuth2 GitHub integration _(v0.3.3)_
- ✅ API key management with multiple keys per user
- ✅ Role based access control (admin, user, viewer) _(v0.3.4)_
- ✅ Session management with database persistence
- ✅ User management commands: `user add`, `user list`, `user role`,
  `user token`, `user apikey`

### Subtitle Processing (Complete)

- ✅ Convert between subtitle formats using go-astisub
- ✅ Merge two subtitle tracks sorted by time
- ✅ Extract subtitles from media containers via ffmpeg
- ✅ Translate subtitles through Google Translate (Cloud SDK) and ChatGPT
- ✅ Delete subtitle files and history records

### Provider Integration (Bazarr Parity Achieved)

- ✅ **40+ subtitle providers** including all major services: Addic7ed,
  OpenSubtitles, Subscene, Podnapisi, TVSubtitles, Titlovi, LegendasDivx,
  GreekSubs, BetaSeries, BSplayer, and 30+ more
- ✅ Provider registry for unified selection _(v0.1.9)_
- ✅ Manual subtitle search with `search` command _(v0.3.6)_

### Database & Storage (Complete)

- ✅ SQLite backend with full schema
- ✅ PebbleDB backend with migration support _(v0.3.1)_
- ✅ **PostgreSQL backend with enterprise support** _(v1.0.0)_
- ✅ Translation history storage and retrieval
- ✅ Download history tracking _(v0.3.2)_
- ✅ Media items table for library metadata _(v0.3.8)_

### Library Management (Complete)

- ✅ Monitor directories for new media files (`watch` command)
- ✅ Scan existing libraries (`scan` and `scanlib` commands)
- ✅ Concurrent directory scanning with worker pools _(v0.3.0)_
- ✅ Recursive directory watching
- ✅ Sonarr and Radarr integration commands _(v0.3.0)_
- ✅ **Advanced webhook system for Sonarr/Radarr/custom events** _(v1.0.0)_
- ✅ Metadata parsing with TheMovieDB integration

### Infrastructure (Complete)

- ✅ gRPC server for remote translation _(v0.1.6)_
- ✅ Docker support with automated builds _(v0.1.10)_
- ✅ GitHub Actions CI/CD pipeline _(v0.1.7)_
- ✅ Prebuilt container images on GitHub Container Registry
- ✅ **Advanced cron-based scheduler with full expression support** _(v1.0.0)_

### Enterprise Features (Complete)

- ✅ **Anti-captcha integration** with Anti-Captcha.com and 2captcha.com support
  _(v1.0.0)_
- ✅ **Notification services** with Discord, Telegram, and SMTP providers
  _(v1.0.0)_
- ✅ **Bazarr configuration import** for seamless migration _(v1.0.0)_
- ✅ **PostgreSQL database backend** for enterprise deployments _(v1.0.0)_

### Web UI (Complete) ✅

- ✅ React application with Vite build system
- ✅ Authentication flow with login page
- ✅ Dashboard with library scanning functionality
- ✅ Settings page for configuration management
- ✅ Extract page for subtitle extraction
- ✅ **History page** with translation and download history filtering
- ✅ **System page** with log viewer, task status, and system information
- ✅ **Wanted page** with search interface for missing subtitles
- ✅ Responsive design and navigation
- ✅ Complete REST API integration

## Web Front End Status

The React UI is functionally complete and includes all major functionality:

- **Authentication** – Login page with username/password and OAuth2 support
- **Dashboard** – Library scanning with progress tracking and provider selection
- **Settings** – Configuration management with live updates to YAML files
- **Extract** – Subtitle extraction from media files
- **History** – Combined view of translation and download history with language
  filtering
- **System** – Log viewer, task status, and system information
- **Wanted** – Search interface for missing subtitles with provider selection

All core pages are implemented and fully functional. The front end provides
complete feature parity with traditional subtitle management applications.

The front end is built with React and Vite under `webui/`. Run
`go generate ./webui` to build the single page application which is embedded
into the binary and served by the `web` command.

## Additional Documentation

## GCommon Refactor Plan

This refactor will replace several internal packages with the shared modules
provided by [gcommon](https://github.com/falkcorp/gcommon). The migration follows
the hybrid Docker build approach defined in `Dockerfile.hybrid` which installs
Node.js in the Go builder stage so `go generate` can embed the pre-built React
UI.

### Planned Steps

#### Phase 1: Dependency Setup

- Add `github.com/falkcorp/gcommon` to `go.mod` and run `go mod tidy`.
- Vendor the module to ensure consistent builds.
- Confirm CI passes with the new dependency.

#### Phase 2: Configuration Service

- Replace the local config loader with `gcommon/config`.
- Write a migration script to convert existing configs to the new format.
- Update CLI flags to pass configuration values through `gcommon`.
- Provide an example configuration file documenting new options.

#### Phase 3: Auth & Sessions

- Swap authentication middleware to `gcommon/auth` for JWT and OAuth2 flows.
- Configure session storage using the shared session manager.
- Add unit tests covering token validation and session creation.
- Remove deprecated local auth code.

#### Phase 4: Metrics & Health

- Integrate `gcommon/metrics` for Prometheus instrumentation.
- Replace existing health endpoints with `gcommon/health` handlers.
- Document Prometheus scraping configuration.

#### Phase 5: Queue System

- Replace the internal queue with `gcommon/queue`.
- Migrate all job types to the new system.
- Document queue configuration in the README.

#### Phase 6: Proto Updates

- Adopt gcommon protobuf messages for shared types.
- Run `protoc` to generate updated Go bindings.
- Refactor gRPC and REST handlers to use the new messages.

#### Phase 7: Docker Hybrid Build

- Standardize on `Dockerfile.hybrid` for development and CI.
- Update build scripts and documentation for the hybrid workflow.

Each step should be tracked with a dedicated issue so multiple teams can work in
parallel. The issues are organized to let developers tackle setup, auth,
metrics, and other areas independently.

For detailed architecture and design decisions, see `docs/TECHNICAL_DESIGN.md`.
The file `docs/BAZARR_FEATURES.md` enumerates all Bazarr features - parity has
been achieved for providers and core functionality.

## Automatic Subtitle Synchronization ✅ COMPLETED

~~A new subsystem will align external subtitles with media using audio analysis
and embedded subtitle tracks. The initial implementation loads existing subtitle
files and provides utilities to shift timing. Future work will integrate Whisper
transcription and multi-track comparison to automatically compute offsets.~~

**IMPLEMENTATION COMPLETED**: Automatic subtitle synchronization is now fully
implemented with:

- ✅ **Audio transcription-based sync** via Whisper API for precise timing
  alignment
- ✅ **Embedded subtitle track extraction** for reference timing using ffmpeg
- ✅ **Multiple track support** with configurable track indices for both audio
  and subtitles
- ✅ **Weighted averaging** between audio and embedded subtitle methods
  (configurable 0-1 weighting)
- ✅ **Translation integration** supporting Google Translate, ChatGPT, and gRPC
  translation services
- ✅ **Advanced CLI interface** with comprehensive flags for all sync options
- ✅ **Audio package** for extracting specific audio tracks with duration limits
- ✅ **Comprehensive test coverage** for all sync methods and edge cases
- ✅ **Smart defaults** (embedded subtitles when no method specified)
- ✅ **Translation during sync** allowing foreign language files to be aligned
  and translated simultaneously

### Usage Examples

```bash
# Sync using embedded subtitles only
subtitle-manager sync movie.mkv subs.srt output.srt --use-embedded

# Sync using audio transcription only
subtitle-manager sync movie.mkv subs.srt output.srt --use-audio

# Sync using both with 70% audio, 30% embedded weighting
subtitle-manager sync movie.mkv subs.srt output.srt --use-audio --use-embedded --audio-weight 0.7

# Sync with translation to Spanish
subtitle-manager sync movie.mkv subs.srt output.srt --use-audio --translate --translate-lang es

# Advanced: specific tracks and translation service
subtitle-manager sync movie.mkv subs.srt output.srt --use-embedded --subtitle-tracks 0,1,2 --audio-track 1 --translate --translate-service gpt
```

This demonstration showcases the advanced subtitle synchronization workflow.
Additional features such as tagging and maintenance tooling remain under
development before the project can be considered feature complete.

### Planned Advanced Sync Improvements

- CPU vs accuracy slider controlling algorithm complexity.
- Option to trade runtime for improved accuracy.
- Dual-subtitle mode for languages with different grammar.
- Experimental minimum display time mode with catch-up logic for HI subtitles.

**Note**: `pkg/audio.GetAudioTracks` now parses ffprobe JSON output for accurate
audio track details. The `splitLines` helper has been updated accordingly.

- [x] 🟢 **General**: Ensure GitHub CLI has project scopes for
      create-github-projects.sh

- [x] 🟡 **General**: Create GitHub projects for open features

- [x] **Search rate limiting**: Per-IP token bucket to prevent abuse
  - Location: `pkg/webserver/search.go`
- [x] 🟢 **General**: Ensure GitHub CLI has project scopes for
      create-github-projects.sh
- [x] 🟡 **General**: Validated codex-rebase.sh AI conflict resolution

## Remaining Work Items

- [ ] 🟡 **General**: Align remaining services with gcommon proto
- [ ] 🟡 **General**: Monitor AI rebase workflow results
- [ ] 🟡 **General**: Investigate remote configuration for rebase script
- [ ] � **Search**: Add integration tests for cached provider results

### Completed Items

- [x] Integrated gcommon/metrics for Prometheus instrumentation
- [x] Config loader migrated to gcommon/config
- [x] Adopted gcommon QueueMessage for internal queue
- [x] Auth: Implement gRPC AuthService using gcommon protobuf messages
- [x] Whisper container start/stop CLI commands implemented
- [x] Replace internal queue with gcommon/queue
- [x] Switch authentication to gcommon/auth library
- [x] Standardize on `Dockerfile.hybrid` for development and CI
- [x] Search Result Caching
- [x] Implement translation result caching
- [x] Replace existing health endpoints with gcommon/health handlers

Review /api/sync/batch test edge cases

Validate codex-rebase.sh AI conflict resolution

Verify gcommon database migration across modules
