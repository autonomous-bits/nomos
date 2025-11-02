#!/bin/bash
# Initial release script for tagging v0.1.0 for all three libraries
# Usage: ./tools/scripts/create-initial-tags.sh

set -e

VERSION="v0.1.0"
LIBS=("compiler" "parser" "provider-proto")

echo "🏷️  Creating initial v0.1.0 tags for all libraries..."
echo ""

# Verify we're on main and up-to-date
echo "Checking prerequisites..."
if [ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then
    echo "❌ Not on main branch. Please checkout main first."
    exit 1
fi
echo "✅ On main branch"

if [ -n "$(git status --porcelain)" ]; then
    echo "⚠️  Warning: Uncommitted changes detected. Commit them before tagging."
    git status --short
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo ""
echo "Creating tags for:"
for lib in "${LIBS[@]}"; do
    echo "  - libs/$lib/$VERSION"
done
echo ""

read -p "Create these tags? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

# Create annotated tags
for lib in "${LIBS[@]}"; do
    TAG="libs/$lib/$VERSION"
    
    # Check if tag already exists
    if git rev-parse "$TAG" >/dev/null 2>&1; then
        echo "⚠️  Tag $TAG already exists, skipping..."
        continue
    fi
    
    # Create annotated tag with release message
    echo "Creating $TAG..."
    git tag -a "$TAG" -m "libs/$lib $VERSION

Initial release of the Nomos $lib library.

See libs/$lib/CHANGELOG.md for detailed release notes."
    
    echo "✅ Created $TAG"
done

echo ""
echo "✅ All tags created successfully!"
echo ""
echo "To verify tags:"
echo "  git tag -l \"libs/*\""
echo ""
echo "To push tags to GitHub:"
echo "  git push origin --tags"
echo ""
echo "Or push individually:"
for lib in "${LIBS[@]}"; do
    echo "  git push origin libs/$lib/$VERSION"
done
echo ""
echo "After pushing, verify at: https://github.com/autonomous-bits/nomos/tags"
