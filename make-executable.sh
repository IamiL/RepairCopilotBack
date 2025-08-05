#!/bin/bash
# Makes binary files executable

echo "Making binary files executable..."

# Make api-gateway binary executable
if [ -f "api-gateway-app" ]; then
    chmod +x api-gateway-app
    echo "✓ api-gateway-app is now executable"
else
    echo "- api-gateway-app not found"
fi

# Make tz-bot binary executable
if [ -f "tz-bot-app" ]; then
    chmod +x tz-bot-app
    echo "✓ tz-bot-app is now executable"
else
    echo "- tz-bot-app not found"
fi

echo "Done!"