#!/bin/bash
# Validation script for RELEASE.md documentation
# This script verifies that the release process documentation is accurate

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🔍 Validating RELEASE.md documentation..."
echo ""

# Check 1: RELEASE.md exists
echo "✓ Checking RELEASE.md exists..."
if [ ! -f "$REPO_ROOT/docs/RELEASE.md" ]; then
    echo "❌ RELEASE.md not found at docs/RELEASE.md"
    exit 1
fi
echo "  ✅ RELEASE.md found"

# Check 2: All three libraries have CHANGELOG.md
echo "✓ Checking CHANGELOG.md files exist..."
for lib in compiler parser provider-proto; do
    if [ ! -f "$REPO_ROOT/libs/$lib/CHANGELOG.md" ]; then
        echo "  ❌ CHANGELOG.md not found for libs/$lib"
        exit 1
    fi
    echo "  ✅ libs/$lib/CHANGELOG.md found"
done

# Check 3: All three libraries have go.mod
echo "✓ Checking go.mod files exist..."
for lib in compiler parser provider-proto; do
    if [ ! -f "$REPO_ROOT/libs/$lib/go.mod" ]; then
        echo "  ❌ go.mod not found for libs/$lib"
        exit 1
    fi
    
    # Verify module path
    module_path=$(grep "^module " "$REPO_ROOT/libs/$lib/go.mod" | awk '{print $2}')
    expected_path="github.com/autonomous-bits/nomos/libs/$lib"
    
    if [ "$module_path" != "$expected_path" ]; then
        echo "  ❌ Module path mismatch in libs/$lib/go.mod"
        echo "     Expected: $expected_path"
        echo "     Got: $module_path"
        exit 1
    fi
    echo "  ✅ libs/$lib/go.mod has correct module path"
done

# Check 4: RELEASE.md contains required sections
echo "✓ Checking RELEASE.md structure..."
required_sections=(
    "Tag Naming Convention"
    "Release Checklist"
    "Creating a Release"
    "Consuming Published Modules"
    "Permissions and Access"
    "Semantic Versioning Guidelines"
)

for section in "${required_sections[@]}"; do
    if ! grep -q "$section" "$REPO_ROOT/docs/RELEASE.md"; then
        echo "  ❌ Section '$section' not found in RELEASE.md"
        exit 1
    fi
done
echo "  ✅ All required sections present"

# Check 5: Makefile has release targets
echo "✓ Checking Makefile release targets..."
required_targets=(
    "release-lib"
    "release-check"
    "list-tags"
)

for target in "${required_targets[@]}"; do
    if ! grep -q "^$target:" "$REPO_ROOT/Makefile"; then
        echo "  ❌ Target '$target' not found in Makefile"
        exit 1
    fi
done
echo "  ✅ All release targets present in Makefile"

# Check 6: CHANGELOGs have comparison links
echo "✓ Checking CHANGELOG comparison links..."
for lib in compiler parser provider-proto; do
    changelog="$REPO_ROOT/libs/$lib/CHANGELOG.md"
    
    # Check for [Unreleased] link
    if ! grep -q "\[Unreleased\]:" "$changelog"; then
        echo "  ❌ [Unreleased] comparison link missing in libs/$lib/CHANGELOG.md"
        exit 1
    fi
    
    # Check for [0.1.0] link
    if ! grep -q "\[0.1.0\]:" "$changelog"; then
        echo "  ❌ [0.1.0] release link missing in libs/$lib/CHANGELOG.md"
        exit 1
    fi
    
    echo "  ✅ libs/$lib/CHANGELOG.md has comparison links"
done

echo ""
echo "✅ All validation checks passed!"
echo ""
echo "Documentation is ready for release tagging."
echo "To create tags, follow the steps in docs/RELEASE.md or use:"
echo "  make release-lib LIB=<compiler|parser|provider-proto> VERSION=v0.1.0"
