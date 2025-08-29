#!/bin/bash

# Script to update Homebrew formula for new commitron releases
# Usage: ./scripts/update-homebrew-formula.sh <version>
# Example: ./scripts/update-homebrew-formula.sh v0.2.0

set -e

# Check if version argument is provided
if [ $# -eq 0 ]; then
    echo "Error: Version argument is required"
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.2.0"
    exit 1
fi

VERSION=$1
FORMULA_FILE="Formula/commitron.rb"
TEMP_DIR="/tmp/commitron-update"

echo "🔄 Updating Homebrew formula for version: $VERSION"

# Create temporary directory
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

# Download the release tarball
echo "📥 Downloading release tarball..."
curl -L "https://github.com/stiliajohny/commitron/archive/refs/tags/$VERSION.tar.gz" -o "commitron-$VERSION.tar.gz"

# Calculate SHA256 checksum
echo "🔍 Calculating SHA256 checksum..."
SHA256_CHECKSUM=$(shasum -a 256 "commitron-$VERSION.tar.gz" | cut -d' ' -f1)

echo "✅ SHA256 checksum: $SHA256_CHECKSUM"

# Go back to project root
cd - > /dev/null

# Create backup of current formula
cp "$FORMULA_FILE" "$FORMULA_FILE.backup"

# Update the formula file
echo "📝 Updating formula file..."
sed -i.bak \
-e "s|url \".*\"|url \"https://github.com/stiliajohny/commitron/archive/refs/tags/$VERSION.tar.gz\"|" \
-e "s|sha256 \".*\"|sha256 \"$SHA256_CHECKSUM\"|" \
-e "s|# SHA256 checksum for .*|# SHA256 checksum for $VERSION release|" \
"$FORMULA_FILE"

# Remove backup file created by sed
rm -f "$FORMULA_FILE.bak"

# Clean up temporary files
rm -rf "$TEMP_DIR"

echo "✅ Homebrew formula updated successfully!"
echo ""
echo "📋 Changes made:"
echo "  - URL: Updated to $VERSION"
echo "  - SHA256: Updated to $SHA256_CHECKSUM"
echo ""
echo "🚀 Next steps:"
echo "  1. Review the changes: git diff $FORMULA_FILE"
echo "  2. Commit and push:"
echo "     git add $FORMULA_FILE"
echo "     git commit -m \"feat: update Homebrew formula for $VERSION\""
echo "     git push origin master"
echo ""
echo "📦 Users can then install with:"
echo "   brew tap stiliajohny/commitron https://github.com/stiliajohny/commitron.git"
echo "   brew install commitron"
