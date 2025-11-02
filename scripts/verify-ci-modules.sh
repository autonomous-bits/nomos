#!/usr/bin/env bash
#
# verify-ci-modules.sh
# Fast CI smoke test that validates workspace integrity and runs tests across all modules.
#
# Usage: ./scripts/verify-ci-modules.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo "Nomos Multi-Module CI Verification"
echo "============================================"

# Step 1: Verify go.work exists
echo ""
echo "Step 1: Checking for go.work..."
if [ ! -f go.work ]; then
    echo -e "${RED}ERROR: go.work not found at repository root${NC}"
    exit 1
fi
echo -e "${GREEN}✓ go.work found${NC}"

# Step 2: Sync workspace
echo ""
echo "Step 2: Syncing Go workspace..."
if ! go work sync; then
    echo -e "${RED}ERROR: go work sync failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Workspace synced${NC}"

# Step 3: Verify required modules are present
echo ""
echo "Step 3: Verifying required modules..."
REQUIRED_MODULES=(
    "apps/command-line"
    "libs/compiler"
    "libs/parser"
    "libs/provider-proto"
)

for module in "${REQUIRED_MODULES[@]}"; do
    if grep -q "./${module}" go.work; then
        echo -e "${GREEN}✓ Module ${module} found in go.work${NC}"
    else
        echo -e "${RED}ERROR: Module ${module} not found in go.work${NC}"
        exit 1
    fi
done

# Step 4: Run tests for each module
echo ""
echo "Step 4: Running tests for all modules..."
for module in "${REQUIRED_MODULES[@]}"; do
    echo ""
    echo -e "${YELLOW}Testing ${module}...${NC}"
    if ! (cd "${module}" && go test ./...); then
        echo -e "${RED}ERROR: Tests failed for ${module}${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Tests passed for ${module}${NC}"
done

# Step 5: Success summary
echo ""
echo "============================================"
echo -e "${GREEN}✓ All checks passed!${NC}"
echo "============================================"
echo ""
echo "Modules verified:"
for module in "${REQUIRED_MODULES[@]}"; do
    echo "  - ${module}"
done
