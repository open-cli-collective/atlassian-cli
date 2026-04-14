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
#   - Write access to the specified space for creating test pages
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

    echo "[$id] Processing..."

    # Step 1: Format check — skip ADF-backed pages
    local format
    format=$(cfl page view "$id" --raw --content-only 2>/dev/null | head -c1) || true
    if [[ "$format" != "<" ]]; then
        echo "[$id] SKIP: ADF-backed (not storage format)"
        ((SKIP++))
        return 0
    fi

    # Step 2: Capture original XHTML
    if ! cfl page view "$id" --raw --content-only > "$before_file" 2>/dev/null; then
        echo "[$id] FAIL: Could not fetch page"
        ((FAIL++))
        return 1
    fi

    # Step 3: Convert to markdown
    local md_content
    if ! md_content=$(cfl page view "$id" --content-only --show-macros 2>/dev/null); then
        echo "[$id] FAIL: Could not convert to markdown"
        ((FAIL++))
        return 1
    fi
    echo "$md_content" > "$golden_file"

    # Step 4: Create test page from markdown
    local new_id
    new_id=$(echo "$md_content" | cfl page create -s "$SPACE" -t "[Test] Roundtrip $id" --legacy -o json 2>/dev/null | jq -r '.id') || true
    if [[ -z "$new_id" || "$new_id" == "null" ]]; then
        echo "[$id] FAIL: Could not create test page"
        ((FAIL++))
        return 1
    fi
    CLEANUP_IDS+=("$new_id")

    # Step 5: Capture roundtripped XHTML
    if ! cfl page view "$new_id" --raw --content-only > "$after_file" 2>/dev/null; then
        echo "[$id] FAIL: Could not fetch roundtripped page"
        ((FAIL++))
        return 1
    fi

    # Step 6: Compare
    if diff -q "$before_file" "$after_file" >/dev/null 2>&1; then
        echo "[$id] PASS: Lossless roundtrip"
        ((PASS++))
    else
        echo "[$id] FAIL: Content differs (see $after_file)"
        # Show brief diff summary
        local diff_lines
        diff_lines=$(diff "$before_file" "$after_file" 2>/dev/null | wc -l | tr -d ' ')
        echo "       Diff: $diff_lines lines changed"
        ((FAIL++))
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
