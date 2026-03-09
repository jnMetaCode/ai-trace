#!/bin/bash
# AI-Trace Release Script
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh v0.2.0

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if version is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Version not provided${NC}"
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.2.0"
    exit 1
fi

VERSION=$1

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    echo -e "${RED}Error: Invalid version format${NC}"
    echo "Expected: vX.Y.Z or vX.Y.Z-suffix"
    echo "Example: v0.2.0, v1.0.0-beta.1"
    exit 1
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}AI-Trace Release Script${NC}"
echo -e "${GREEN}Version: ${VERSION}${NC}"
echo -e "${GREEN}========================================${NC}"

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${RED}Error: You have uncommitted changes${NC}"
    echo "Please commit or stash your changes before releasing."
    exit 1
fi

# Ensure we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${YELLOW}Warning: Not on main branch (currently on: ${CURRENT_BRANCH})${NC}"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Pull latest changes
echo ""
echo -e "${YELLOW}Pulling latest changes...${NC}"
git pull origin $CURRENT_BRANCH

# Run tests
echo ""
echo -e "${YELLOW}Running tests...${NC}"
go test -v -race ./...

# Run linter
echo ""
echo -e "${YELLOW}Running linter...${NC}"
if command -v golangci-lint &> /dev/null; then
    golangci-lint run ./...
else
    echo "golangci-lint not found, skipping..."
fi

# Build binaries to verify
echo ""
echo -e "${YELLOW}Building binaries...${NC}"
go build -ldflags="-w -s -X main.Version=${VERSION}" -o bin/ai-trace ./cmd/ai-trace

# Update version in files
echo ""
echo -e "${YELLOW}Updating version in files...${NC}"

# Update version in version.go if exists
if [ -f "internal/version/version.go" ]; then
    sed -i.bak "s/Version = \".*\"/Version = \"${VERSION}\"/" internal/version/version.go
    rm -f internal/version/version.go.bak
fi

# Update version in Python SDK
PYTHON_VERSION=${VERSION#v}  # Remove 'v' prefix for Python
sed -i.bak "s/version = \".*\"/version = \"${PYTHON_VERSION}\"/" sdk/python/pyproject.toml
rm -f sdk/python/pyproject.toml.bak
sed -i.bak "s/__version__ = \".*\"/__version__ = \"${PYTHON_VERSION}\"/" sdk/python/ai_trace/__init__.py
rm -f sdk/python/ai_trace/__init__.py.bak

# Check if CHANGELOG needs updating
echo ""
echo -e "${YELLOW}Checking CHANGELOG.md...${NC}"
if ! grep -q "\[${VERSION}\]" CHANGELOG.md 2>/dev/null; then
    echo -e "${YELLOW}Warning: ${VERSION} not found in CHANGELOG.md${NC}"
    echo "Please update CHANGELOG.md before releasing."
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Commit version changes
echo ""
echo -e "${YELLOW}Committing version changes...${NC}"
git add -A
if ! git diff-index --quiet HEAD --; then
    git commit -m "chore: bump version to ${VERSION}"
fi

# Create and push tag
echo ""
echo -e "${YELLOW}Creating git tag...${NC}"
git tag -a ${VERSION} -m "Release ${VERSION}"

echo ""
echo -e "${YELLOW}Pushing to origin...${NC}"
git push origin $CURRENT_BRANCH
git push origin ${VERSION}

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Release ${VERSION} created successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Next steps:"
echo "1. GitHub Actions will automatically:"
echo "   - Build binaries for all platforms"
echo "   - Publish Docker image to ghcr.io"
echo "   - Create GitHub release"
echo "   - Publish Python SDK to PyPI"
echo ""
echo "2. Monitor the release at:"
echo "   https://github.com/ai-trace/ai-trace/actions"
echo ""
echo "3. After release, update documentation if needed"
