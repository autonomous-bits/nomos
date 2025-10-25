#!/usr/bin/env bash
#
# work-sync.sh - Create or update Go workspace for local development
#
# This script ensures that a go.work file exists at the repository root
# and includes all required modules for local development.

set -e  # Exit on error

# Get the repository root (parent of tools/scripts)
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "$REPO_ROOT"

echo "Nomos Go Workspace Sync"
echo "======================="
echo "Repository root: $REPO_ROOT"
echo ""

# Check if go.work exists
if [ -f "go.work" ]; then
    echo "✓ Found existing go.work file"
else
    echo "Creating new go.work file..."
    go work init
    echo "✓ Created go.work"
fi

# Add all required modules
echo ""
echo "Adding modules to workspace..."

MODULES=(
    "./apps/command-line"
    "./libs/compiler"
    "./libs/parser"
)

for module in "${MODULES[@]}"; do
    if [ -d "$module" ] && [ -f "$module/go.mod" ]; then
        echo "  Adding $module"
        go work use "$module" 2>/dev/null || true
    else
        echo "  Warning: Module $module not found or missing go.mod"
    fi
done

# Sync dependencies
echo ""
echo "Syncing workspace dependencies..."
go work sync

echo ""
echo "✓ Workspace sync complete!"
echo ""
echo "Verification:"
echo "============="

# Show workspace status
echo "Modules in workspace:"
grep -E "^\s*use " go.work | sed 's/^\s*use /  - /' || echo "  (none)"

echo ""
echo "To test the workspace:"
echo "  go test ./..."
echo "  go test -race ./..."
echo "  go build ./..."
