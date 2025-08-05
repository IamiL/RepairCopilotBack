#!/bin/bash
# Cleanup script - removes binary files before git pull

echo "Removing binary files..."

# Remove api-gateway binary
if [ -f "api-gateway-app" ]; then
    rm api-gateway-app
    echo "✓ Removed api-gateway-app"
else
    echo "- api-gateway-app not found"
fi

# Remove tz-bot binary
if [ -f "tz-bot-app" ]; then
    rm tz-bot-app
    echo "✓ Removed tz-bot-app"
else
    echo "- tz-bot-app not found"
fi

echo "Cleanup completed!"