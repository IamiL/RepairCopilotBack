#!/bin/bash
# Makes binary files executable

echo "Making binary files executable..."

# Make all api-gateway binaries executable
api_gateway_files=$(ls api-gateway-app-v* 2>/dev/null)
if [ -n "$api_gateway_files" ]; then
    chmod +x api-gateway-app-v*
    api_gateway_count=$(echo "$api_gateway_files" | wc -l)
    echo "✓ Made $api_gateway_count api-gateway binary(ies) executable"
else
    echo "- No api-gateway binaries found"
fi

# Make all tz-bot binaries executable
tz_bot_files=$(ls tz-bot-app-v* 2>/dev/null)
if [ -n "$tz_bot_files" ]; then
    chmod +x tz-bot-app-v*
    tz_bot_count=$(echo "$tz_bot_files" | wc -l)
    echo "✓ Made $tz_bot_count tz-bot binary(ies) executable1"
else
    echo "- No tz-bot binaries found"
fi

user_service_files=$(ls user-app-v* 2>/dev/null)
if [ -n "user_service_files" ]; then
    chmod +x user-app-v*
    user_service_files=$(echo "user_service_files" | wc -l)
    echo "✓ Made user_service_files user-service binary(ies) executable"
else
    echo "- No user-service binaries found"
fi

echo "Done!"