#!/bin/bash
# Release script for clink-cli

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Clink CLI Release Script${NC}"
echo ""

# Check if version is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Version required${NC}"
    echo "Usage: ./scripts/release.sh <version>"
    echo "Example: ./scripts/release.sh v0.1.0"
    exit 1
fi

VERSION=$1

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
    echo -e "${RED}Error: Invalid version format${NC}"
    echo "Version must start with 'v' followed by semantic version (e.g., v0.1.0)"
    exit 1
fi

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${YELLOW}Warning: Not on main branch (current: $CURRENT_BRANCH)${NC}"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}Warning: Uncommitted changes detected${NC}"
    git status --short
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Run tests
echo -e "${GREEN}Running tests...${NC}"
if ! go test -v ./...; then
    echo -e "${RED}Tests failed${NC}"
    exit 1
fi

# Build binaries locally to verify
echo -e "${GREEN}Building binaries locally...${NC}"
mkdir -p dist
go build -o dist/clink ./cmd/clink
go build -o dist/clink-mcp ./cmd/clink-mcp

# Create tag
echo -e "${GREEN}Creating tag $VERSION...${NC}"
git tag -a "$VERSION" -m "Release $VERSION"

# Push tag
echo -e "${GREEN}Pushing tag to origin...${NC}"
git push origin "$VERSION"

echo ""
echo -e "${GREEN}✓ Release $VERSION triggered!${NC}"
echo ""
echo "GitHub Actions will now:"
echo "  1. Run tests"
echo "  2. Build binaries for 6 platforms"
echo "  3. Create a GitHub Release"
echo ""
echo "Monitor progress at:"
echo "  https://github.com/raymondtc/clink-cli/actions"
echo ""
echo "Release will be available at:"
echo "  https://github.com/raymondtc/clink-cli/releases/tag/$VERSION"
