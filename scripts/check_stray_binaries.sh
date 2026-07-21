#!/bin/bash
#
# check_stray_binaries.sh — Detect (and optionally remove) compiled
# binaries accidentally left at the repo root.
#
# Usage:
#   ./scripts/check_stray_binaries.sh          # detect and fail
#   ./scripts/check_stray_binaries.sh --clean  # detect and remove
#
# Exit codes:
#   0  No stray binaries found (or --clean removed them all)
#   1  Stray binaries detected (without --clean)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Find Mach-O and ELF binaries at repo root (skip the gray binary itself)
stray=$(find "$PROJECT_ROOT" -maxdepth 1 -type f ! -name 'gray' ! -name '.*' \
    -exec file {} \; | grep -E 'Mach-O|ELF' | cut -d: -f1)

if [ -z "$stray" ]; then
    echo "No stray binaries at repo root."
    exit 0
fi

count=$(echo "$stray" | wc -l | tr -d ' ')

if [ "$1" = "--clean" ]; then
    echo "$stray" | xargs rm
    echo "Removed $count stray binary(ies) from repo root."
    exit 0
fi

echo "ERROR: $count stray compiled binary(ies) found at repo root:"
echo "$stray" | head -10
if [ "$count" -gt 10 ]; then
    echo "  ... and $((count - 10)) more"
fi
echo ""
echo "Run: ./scripts/check_stray_binaries.sh --clean"
exit 1
