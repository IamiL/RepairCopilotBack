#!/bin/bash
# Cleanup script - removes binary files before git pull

echo "Removing binary files..."

# Remove all versioned api-gateway binaries
api_gateway_count=$(ls api-gateway-app-v* 2>/dev/null | wc -l)
if [ "$api_gateway_count" -gt 0 ]; then
    rm api-gateway-app-v*
    echo "✓ Removed $api_gateway_count api-gateway binary(ies)"
else
    echo "- No api-gateway binaries found"
fi

# Remove all versioned tz-bot binaries
tz_bot_count=$(ls tz-bot-app-v* 2>/dev/null | wc -l)
if [ "$tz_bot_count" -gt 0 ]; then
    rm tz-bot-app-v*
    echo "✓ Removed $tz_bot_count tz-bot binary(ies)"
else
    echo "- No tz-bot binaries found"
fi

user_service_count=$(ls user-app-v* 2>/dev/null | wc -l)
if [ "$user_service_count" -gt 0 ]; then
    rm user-app-v*
    echo "✓ Removed $user_service_count user-service binary(ies)"
else
    echo "- No tz-bot binaries found"
fi

echo "Cleanup completed!"