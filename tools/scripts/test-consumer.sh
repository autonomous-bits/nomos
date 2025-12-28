#!/bin/bash
# Test script to verify published modules can be consumed
# This script simulates an external consumer trying to use the published libraries

set -e

echo "🧪 Testing consumer can use published Nomos libraries..."
echo ""

# Check if tags exist
echo "Step 1: Verifying tags exist..."
REQUIRED_TAGS=(
    "libs/compiler/v0.1.0"
    "libs/parser/v0.1.0"
    "libs/provider-proto/v0.1.0"
)

missing_tags=0
for tag in "${REQUIRED_TAGS[@]}"; do
    if ! git rev-parse "$tag" >/dev/null 2>&1; then
        echo "  ❌ Tag $tag not found"
        missing_tags=1
    else
        echo "  ✅ Tag $tag exists"
    fi
done

if [ $missing_tags -eq 1 ]; then
    echo ""
    echo "❌ Required tags are missing. Please create and push tags first."
    echo "   Run: ./tools/scripts/create-initial-tags.sh"
    exit 1
fi

echo ""
echo "Step 2: Testing with examples/consumer..."

# Test the existing consumer example (uses go.work)
cd examples/consumer

echo "  Building consumer example..."
if ! go build -o ../../bin/consumer-example ./cmd/consumer-example; then
    echo "  ❌ Consumer build failed"
    exit 1
fi
echo "  ✅ Consumer builds successfully"

echo "  Running consumer tests..."
if ! go test ./...; then
    echo "  ❌ Consumer tests failed"
    exit 1
fi
echo "  ✅ Consumer tests pass"

cd ../..

echo ""
echo "Step 3: Simulating external consumer (outside workspace)..."

# Create a temporary directory outside the workspace
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

cd "$TEMP_DIR"

# Create a minimal consumer that uses published versions
cat > go.mod << 'EOF'
module example.com/test-consumer

go 1.25

require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
    github.com/autonomous-bits/nomos/libs/parser v0.1.0
)
EOF

cat > main.go << 'EOF'
package main

import (
    "fmt"
    "github.com/autonomous-bits/nomos/libs/parser"
)

func main() {
    // Test that we can import and use the libraries
    fmt.Println("Testing Nomos library imports...")
    
    // Try to create a parser instance
    p := parser.NewParser()
    if p == nil {
        panic("parser is nil")
    }
    
    fmt.Println("✅ Successfully imported and used Nomos libraries!")
}
EOF

echo "  Created test consumer in $TEMP_DIR"
echo "  Running go mod download..."

# This will fail if tags aren't pushed to GitHub, which is expected
# For local testing before push, we'll handle the error gracefully
if go mod download 2>&1 | tee /tmp/go-mod-download.log; then
    echo "  ✅ go mod download succeeded"
    
    echo "  Building test consumer..."
    if go build -o test-consumer .; then
        echo "  ✅ Test consumer builds successfully"
        
        echo "  Running test consumer..."
        if ./test-consumer; then
            echo "  ✅ Test consumer runs successfully"
        else
            echo "  ❌ Test consumer failed to run"
            exit 1
        fi
    else
        echo "  ❌ Test consumer failed to build"
        exit 1
    fi
else
    # Check if error is due to unpushed tags
    if grep -q "unknown revision" /tmp/go-mod-download.log || \
       grep -q "not found" /tmp/go-mod-download.log; then
        echo "  ⚠️  Tags exist locally but are not yet pushed to GitHub"
        echo "      This is expected before running: git push origin --tags"
        echo "      Skipping external consumer test..."
    else
        echo "  ❌ go mod download failed with unexpected error"
        cat /tmp/go-mod-download.log
        exit 1
    fi
fi

cd - > /dev/null

echo ""
echo "✅ Consumer verification tests passed!"
echo ""
echo "Summary:"
echo "  - Local tags exist for all three libraries"
echo "  - examples/consumer builds and tests pass (using go.work)"
echo "  - External consumer test prepared (will work after pushing tags)"
echo ""
echo "Next steps:"
echo "  1. Commit and push CHANGELOG changes: git push origin main"
echo "  2. Push tags: git push origin --tags"
echo "  3. Verify on GitHub: https://github.com/autonomous-bits/nomos/tags"
echo "  4. Test external consumer can fetch: go get github.com/autonomous-bits/nomos/libs/compiler@v0.1.0"
