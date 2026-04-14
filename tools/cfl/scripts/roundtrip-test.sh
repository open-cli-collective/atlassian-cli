#!/usr/bin/env bash
#
# roundtrip-test.sh — Validate XHTML→MD→XHTML fidelity against real Confluence pages
#
# Usage:
#   ./roundtrip-test.sh --space DEV 12345 67890
#   ./roundtrip-test.sh --space DEV < page-ids.txt
#   ROUNDTRIP_SPACE=DEV ./roundtrip-test.sh 12345 67890
#
# Requirements:
#   - cfl configured and in PATH
#   - jq for JSON parsing
#   - Write access to the specified space for creating test pages
#
# Note: Golden files are generated via the CLI pipeline (cfl page view --content-only),
# which is a thin wrapper around FromConfluenceStorageWithOptions. The Go test validates
# the library directly, so any divergence between CLI and library would cause mismatches.
#
# Output:
#   - Source fixtures: testdata/roundtrip/<id>.before.xhtml, <id>.golden.md
#   - Triage outputs: /tmp/roundtrip-<timestamp>/<id>.after.xhtml
#   - Report: stdout summary with pass/fail/skip counts

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXTURE_DIR="${SCRIPT_DIR}/../pkg/md/testdata/roundtrip"
TRIAGE_DIR="/tmp/roundtrip-$(date +%Y%m%d-%H%M%S)"
SPACE="${ROUNDTRIP_SPACE:-}"

# Counters
PASS=0
FAIL=0
SKIP=0

# Track created test pages for cleanup
declare -a CLEANUP_IDS=()

usage() {
    cat <<EOF
Usage: $(basename "$0") [--space <key>] <page-id> [page-id ...]

Validates XHTML→MD→XHTML roundtrip fidelity against real Confluence pages.

Options:
  --space <key>    Target space for test pages (required, or set ROUNDTRIP_SPACE)
  --help           Show this help message

Environment:
  ROUNDTRIP_SPACE  Default space key if --space not provided

Output:
  Committed fixtures: ${FIXTURE_DIR}/<id>.before.xhtml, <id>.golden.md
  Triage outputs:     /tmp/roundtrip-<timestamp>/<id>.after.xhtml
EOF
    exit 1
}

cleanup() {
    if [[ ${#CLEANUP_IDS[@]} -gt 0 ]]; then
        echo ""
        echo "Cleaning up ${#CLEANUP_IDS[@]} test page(s)..."
        for id in "${CLEANUP_IDS[@]}"; do
            if cfl page delete "$id" --force >/dev/null 2>&1; then
                echo "  Deleted: $id"
            else
                echo "  Failed to delete: $id (may need manual cleanup)"
            fi
        done
    fi
}

trap cleanup EXIT

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --space)
            SPACE="$2"
            shift 2
            ;;
        --help|-h)
            usage
            ;;
        -*)
            echo "Unknown option: $1" >&2
            usage
            ;;
        *)
            break
            ;;
    esac
done

# Validate space
if [[ -z "$SPACE" ]]; then
    echo "Error: --space <key> or ROUNDTRIP_SPACE required" >&2
    usage
fi

# Collect page IDs from args or stdin
PAGE_IDS=()
if [[ $# -gt 0 ]]; then
    PAGE_IDS=("$@")
elif [[ ! -t 0 ]]; then
    while IFS= read -r line; do
        # Skip empty lines and comments
        [[ -z "$line" || "$line" =~ ^# ]] && continue
        PAGE_IDS+=("$line")
    done
fi

if [[ ${#PAGE_IDS[@]} -eq 0 ]]; then
    echo "Error: No page IDs provided" >&2
    usage
fi

# Setup directories
mkdir -p "$FIXTURE_DIR" "$TRIAGE_DIR"

echo "Roundtrip Fidelity Test"
echo "======================="
echo "Space: $SPACE"
echo "Pages: ${#PAGE_IDS[@]}"
echo "Fixtures: $FIXTURE_DIR"
echo "Triage: $TRIAGE_DIR"
echo ""

process_page() {
    local id="$1"
    local before_file="$FIXTURE_DIR/${id}.before.xhtml"
    local golden_file="$FIXTURE_DIR/${id}.golden.md"
    local after_file="$TRIAGE_DIR/${id}.after.xhtml"
    local staging_md="$TRIAGE_DIR/${id}.md"

    echo "[$id] Processing..."

    # Step 1: Fetch page and check format
    # Capture raw content first to distinguish fetch failures from ADF pages
    local raw_content
    if ! raw_content=$(cfl page view "$id" --raw --content-only 2>&1); then
        echo "[$id] FAIL: Could not fetch page: $raw_content"
        FAIL=$((FAIL + 1))
        return 1
    fi

    # Check for empty content
    if [[ -z "$raw_content" ]]; then
        echo "[$id] FAIL: Empty page content"
        FAIL=$((FAIL + 1))
        return 1
    fi

    # Check if storage format (XHTML starts with <) vs ADF (JSON starts with {)
    local first_char="${raw_content:0:1}"
    if [[ "$first_char" != "<" ]]; then
        echo "[$id] SKIP: ADF-backed (not storage format)"
        SKIP=$((SKIP + 1))
        return 0
    fi

    # Step 2: Save original XHTML
    printf '%s' "$raw_content" > "$before_file"

    # Step 3: Convert to markdown (save to staging, not final location)
    local md_content
    if ! md_content=$(cfl page view "$id" --content-only --show-macros 2>&1); then
        echo "[$id] FAIL: Could not convert to markdown: $md_content"
        FAIL=$((FAIL + 1))
        return 1
    fi
    printf '%s' "$md_content" > "$staging_md"

    # Step 4: Create test page from markdown
    local new_id
    new_id=$(printf '%s' "$md_content" | cfl page create -s "$SPACE" -t "[Test] Roundtrip $id" --legacy -o json 2>/dev/null | jq -r '.id') || true
    if [[ -z "$new_id" || "$new_id" == "null" ]]; then
        echo "[$id] FAIL: Could not create test page"
        FAIL=$((FAIL + 1))
        return 1
    fi
    CLEANUP_IDS+=("$new_id")

    # Step 5: Capture roundtripped XHTML
    if ! cfl page view "$new_id" --raw --content-only > "$after_file" 2>/dev/null; then
        echo "[$id] FAIL: Could not fetch roundtripped page"
        FAIL=$((FAIL + 1))
        return 1
    fi

    # Step 6: Compare and only commit golden on success
    if diff -q "$before_file" "$after_file" >/dev/null 2>&1; then
        echo "[$id] PASS: Lossless roundtrip"
        # Only write golden file for passing tests
        cp "$staging_md" "$golden_file"
        PASS=$((PASS + 1))
    else
        echo "[$id] FAIL: Content differs (see $after_file)"
        # Show brief diff summary (diff exits 1 when files differ, so suppress error)
        local diff_lines
        diff_lines=$(diff "$before_file" "$after_file" 2>/dev/null | wc -l | tr -d ' ' || echo "?")
        echo "       Diff: $diff_lines lines changed"
        echo "       Staged MD: $staging_md (not promoted to golden)"
        FAIL=$((FAIL + 1))
    fi
}

# Process each page
for id in "${PAGE_IDS[@]}"; do
    process_page "$id" || true
done

# Summary
echo ""
echo "Summary"
echo "======="
echo "Pass: $PASS"
echo "Fail: $FAIL"
echo "Skip: $SKIP"
echo "Total: ${#PAGE_IDS[@]}"

if [[ $FAIL -gt 0 ]]; then
    echo ""
    echo "Triage outputs in: $TRIAGE_DIR"
    exit 1
fi

exit 0
