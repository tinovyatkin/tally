#!/usr/bin/env bash
#
# Validate that hadolint-status.json is in sync with implemented rules.
# Checks that all rule files in internal/rules/hadolint/ are listed in the status file.
#
# Exit codes:
#   0 - All implemented rules are documented
#   1 - Missing or extra entries found

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
RULES_DIR="$ROOT_DIR/internal/rules/hadolint"
STATUS_FILE="$ROOT_DIR/internal/rules/hadolint-status.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Extract implemented rule codes from filesystem (dlXXXX.go files, excluding _test.go)
get_implemented_rules() {
    find "$RULES_DIR" -name "dl*.go" -not -name "*_test.go" -not -name "image_ref.go" | \
        xargs -n1 basename | \
        sed 's/\.go$//' | \
        tr '[:lower:]' '[:upper:]' | \
        sort
}

# Extract rule codes from hadolint-status.json with status "implemented"
get_documented_rules() {
    jq -r '.rules | to_entries[] | select(.value.status == "implemented") | .key' "$STATUS_FILE" | sort
}

main() {
    if [[ ! -f "$STATUS_FILE" ]]; then
        echo -e "${RED}Error: Status file not found: $STATUS_FILE${NC}" >&2
        exit 1
    fi

    if [[ ! -d "$RULES_DIR" ]]; then
        echo -e "${RED}Error: Rules directory not found: $RULES_DIR${NC}" >&2
        exit 1
    fi

    echo "Validating hadolint rule status..."
    echo

    # Get lists
    local impl_rules doc_rules
    impl_rules=$(get_implemented_rules)
    doc_rules=$(get_documented_rules)

    # Find missing (implemented but not documented)
    local missing
    missing=$(comm -23 <(echo "$impl_rules") <(echo "$doc_rules"))

    # Find extra (documented but not implemented)
    local extra
    extra=$(comm -13 <(echo "$impl_rules") <(echo "$doc_rules"))

    # Report
    local exit_code=0

    if [[ -n "$missing" ]]; then
        echo -e "${RED}✗ Rules implemented but not in hadolint-status.json:${NC}"
        echo "$missing" | while read -r rule; do
            echo "  - $rule"
        done
        echo
        echo "Add these to $STATUS_FILE with:"
        echo "$missing" | while read -r rule; do
            cat <<EOF
  "$rule": {
    "status": "implemented",
    "tally_rule": "hadolint/$rule"
  },
EOF
        done
        echo
        exit_code=1
    fi

    if [[ -n "$extra" ]]; then
        echo -e "${YELLOW}⚠ Rules in hadolint-status.json but not implemented:${NC}"
        echo "$extra" | while read -r rule; do
            echo "  - $rule (missing file: internal/rules/hadolint/$(echo "$rule" | tr '[:upper:]' '[:lower:]').go)"
        done
        echo
        exit_code=1
    fi

    if [[ $exit_code -eq 0 ]]; then
        local count
        count=$(echo "$impl_rules" | wc -l | tr -d ' ')
        echo -e "${GREEN}✓ All $count implemented hadolint rules are documented${NC}"
    fi

    exit $exit_code
}

main "$@"
