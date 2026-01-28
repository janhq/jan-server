#!/bin/bash
# Start Chromium browser with jan-browser-mcp extension
#
# Flags:
#   --no-first-run: Skip first run wizard
#   --no-default-browser-check: Don't prompt to be default browser
#   --no-sandbox: Required for running in container (no root privileges)
#   --disable-infobars: Suppress the "--no-sandbox" warning banner
#   --disable-gpu: Disable GPU acceleration (not available in container)
#   --user-data-dir: Custom profile directory
#   --load-extension: Load jan-browser-mcp extension
#   --remote-debugging-port: Enable Chrome DevTools Protocol on port 9222

chromium \
    --no-first-run \
    --no-default-browser-check \
    --no-sandbox \
    --disable-infobars \
    --disable-gpu \
    --user-data-dir=/home/user/.config/chromium \
    --load-extension=/home/user/extensions/jan-browser-mcp \
    --remote-debugging-port=9222 \
    "$@" &
