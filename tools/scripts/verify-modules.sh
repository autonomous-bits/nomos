#!/usr/bin/env bash
# verify-modules.sh
# Verification script for module builds and tests as specified in FEATURE-56-1
# This script validates that all library modules under libs/ can build and test successfully.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Module paths to verify
MODULES=(
  "libs/compiler"
  "libs/parser"
  "libs/provider-proto"
)

echo "======================================"
echo "Module Verification Script"
echo "======================================"
echo ""

# Change to repo root
cd "${REPO_ROOT}"

# Track failures
FAILED_MODULES=()

# Verify each module
for module in "${MODULES[@]}"; do
  echo -e "${YELLOW}Verifying module: ${module}${NC}"
  echo "--------------------------------------"
  
  # Check if module directory exists
  if [ ! -d "${module}" ]; then
    echo -e "${RED}✗ Module directory not found: ${module}${NC}"
    FAILED_MODULES+=("${module}")
    echo ""
    continue
  fi
  
  # Check if go.mod exists
  if [ ! -f "${module}/go.mod" ]; then
    echo -e "${RED}✗ go.mod not found in ${module}${NC}"
    FAILED_MODULES+=("${module}")
    echo ""
    continue
  fi
  
  # Change to module directory
  cd "${REPO_ROOT}/${module}"
  
  # Run go build
  echo "Running: go build ./..."
  if go build ./... &>/dev/null; then
    echo -e "${GREEN}✓ Build successful${NC}"
  else
    echo -e "${RED}✗ Build failed${NC}"
    echo "Build output:"
    go build ./... 2>&1 | sed 's/^/  /'
    FAILED_MODULES+=("${module}")
    cd "${REPO_ROOT}"
    echo ""
    continue
  fi
  
  # Run go test
  echo "Running: go test ./..."
  if go test ./... &>/dev/null; then
    echo -e "${GREEN}✓ Tests passed${NC}"
  else
    echo -e "${RED}✗ Tests failed${NC}"
    echo "Test output:"
    go test ./... 2>&1 | sed 's/^/  /'
    FAILED_MODULES+=("${module}")
  fi
  
  # Return to repo root
  cd "${REPO_ROOT}"
  echo ""
done

# Verify go.work
echo -e "${YELLOW}Verifying go.work${NC}"
echo "--------------------------------------"
if [ -f "go.work" ]; then
  echo "Running: go work sync"
  if go work sync &>/dev/null; then
    echo -e "${GREEN}✓ go.work sync successful${NC}"
  else
    echo -e "${RED}✗ go.work sync failed${NC}"
    echo "Sync output:"
    go work sync 2>&1 | sed 's/^/  /'
    FAILED_MODULES+=("go.work")
  fi
else
  echo -e "${RED}✗ go.work not found${NC}"
  FAILED_MODULES+=("go.work")
fi

echo ""
echo "======================================"
echo "Verification Complete"
echo "======================================"

# Summary
if [ ${#FAILED_MODULES[@]} -eq 0 ]; then
  echo -e "${GREEN}✓ All modules verified successfully!${NC}"
  exit 0
else
  echo -e "${RED}✗ Verification failed for:${NC}"
  for failed in "${FAILED_MODULES[@]}"; do
    echo -e "  ${RED}- ${failed}${NC}"
  done
  exit 1
fi
