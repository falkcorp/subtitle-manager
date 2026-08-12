#!/bin/bash
# file: scripts/test_database_backends.sh
# version: 2.0.0
# guid: 3c4d5e6f-7890-1234-5678-901234567890
# last-edited: 2026-08-12

# Database Backend Testing Script
#
# There is one build configuration. SQLite arrives through the pure-Go
# modernc.org/sqlite driver, so both the SQLite and PebbleDB backends are
# available in every binary and are chosen at runtime via `db_backend`.
#
# This script used to compare a CGO build against a pure-Go build, because the
# `sqlite` build tag decided whether SQLite existed at all. That tag is gone.

set -e

echo "🚀 Database Backend Testing"
echo "==========================="
echo

echo "1. Backend selection (SQLite and Pebble both expected to work):"
echo "   Command: CGO_ENABLED=0 go test ./pkg/database -v -run TestBackendSelection"
echo
CGO_ENABLED=0 go test ./pkg/database -v -run TestBackendSelection 2>&1 | grep -E "(SQLite|Pebble|PASS|FAIL)"
echo

echo "2. SQLite availability detection:"
echo "   Command: CGO_ENABLED=0 go test ./pkg/database -v -run TestSQLiteAvailability"
echo
CGO_ENABLED=0 go test ./pkg/database -v -run TestSQLiteAvailability 2>&1 | grep -E "(SQLite support|✓|PASS)"
echo

echo "3. Pure-Go SQLite regression guard (the release-binary failure):"
echo "   Command: CGO_ENABLED=0 go test ./pkg/database -v -run TestPureGoSQLite"
echo
CGO_ENABLED=0 go test ./pkg/database -v -run TestPureGoSQLite 2>&1 | grep -E "(PASS|FAIL)"

echo
echo "✅ Database backend testing completed!"
echo
echo "Summary:"
echo "- One CGO-free build serves both backends on every platform"
echo "- SQLite comes from modernc.org/sqlite (pure Go), selected via db_backend"
echo "- Comprehensive test coverage for all supported backends"
