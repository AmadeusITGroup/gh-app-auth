#!/bin/bash
set -e

echo "🧹 Running gh-app-auth cleanup..."

# Check if gh-app-auth is available
if ! command -v gh &> /dev/null; then
    echo "⚠️  GitHub CLI not found, skipping cleanup"
    exit 0
fi

if ! gh extension list 2>/dev/null | grep -q "app-auth"; then
    echo "⚠️  gh-app-auth extension not found, skipping cleanup"
    exit 0
fi

# Remove git credential helper configuration
echo "🗑️  Removing git credential helpers..."
gh app-auth gitconfig --clean --global 2>/dev/null || true

# Remove all app configurations (this will also remove keyring entries)
echo "🗑️  Removing app configurations..."
gh app-auth remove --all --force 2>/dev/null || true

echo "✅ Cleanup complete"
