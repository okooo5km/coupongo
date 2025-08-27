#!/bin/bash

# CouponGo Release Tagging Script
# Usage: ./scripts/tag-release.sh [version]
# Example: ./scripts/tag-release.sh 0.1.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}ℹ️  $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Check if git is available
if ! command -v git &> /dev/null; then
    print_error "Git is not installed or not in PATH"
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository"
    exit 1
fi

# Check if working directory is clean
if [[ -n $(git status --porcelain) ]]; then
    print_error "Working directory is not clean. Please commit or stash your changes."
    git status --short
    exit 1
fi

# Get version from argument or prompt
if [ -n "$1" ]; then
    VERSION="$1"
else
    print_info "Please enter the version number (e.g., 0.1.0):"
    read -r VERSION
fi

# Validate version format (semantic versioning)
if [[ ! $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    print_error "Invalid version format. Please use semantic versioning (e.g., 0.1.0)"
    exit 1
fi

TAG_NAME="v$VERSION"

# Check if tag already exists
if git tag | grep -q "^$TAG_NAME$"; then
    print_error "Tag $TAG_NAME already exists"
    exit 1
fi

print_info "Creating release for version: $VERSION"
print_info "Tag name: $TAG_NAME"

# Confirm before proceeding
print_warning "This will:"
echo "  1. Create a new git tag: $TAG_NAME"
echo "  2. Push the tag to origin"
echo "  3. Trigger GitHub Actions to build and create a release"
echo ""
read -p "Do you want to continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Aborted"
    exit 0
fi

# Create and push tag
print_info "Creating tag: $TAG_NAME"
git tag -a "$TAG_NAME" -m "Release $VERSION

- Built with GitHub Actions
- Cross-platform binaries available
- See CHANGELOG.md for detailed changes"

print_info "Pushing tag to origin..."
git push origin "$TAG_NAME"

print_success "Tag $TAG_NAME created and pushed successfully!"
print_info "GitHub Actions will now build and create the release."
print_info "Check the Actions tab in your GitHub repository for progress."
print_info "Release will be available at: https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/]*\)\.git/\1/')/releases/tag/$TAG_NAME"