# E2B Desktop Sandbox Dockerfile
# This Dockerfile is used to build the E2B sandbox template with dynamic MCP tools

FROM e2bdev/desktop:latest

USER root

# Install dependencies for adding PPA
RUN apt-get update && apt-get install -y \
    software-properties-common \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Add xtradeb PPA for native Chromium .deb (Snap doesn't work in Docker)
# Reference: https://ubuntuhandbook.org/index.php/2024/05/install-chromium-ubuntu-2404/
RUN add-apt-repository -y ppa:xtradeb/apps

# Install Chromium from PPA (NOT chromium-browser which is Snap)
RUN apt-get update && apt-get install -y \
    chromium \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js LTS (for MCP server and code execution)
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Python3 with pip for code execution
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3 /usr/bin/python

# Create workspace directory for conversation workspaces (under user home)
# Use chmod 777 since user may not exist at build time
RUN mkdir -p /home/user/workspace && chmod 777 /home/user/workspace

# Copy jan-browser MCP extension (copied during build via Makefile)
COPY jan-browser-mcp /home/user/extensions/jan-browser-mcp
RUN chmod -R 755 /home/user/extensions

# Pre-install MCP server and mcp-proxy for dynamic tools
# - search-mcp-server: Provides browser automation tools via MCP
# - mcp-proxy: HTTP bridge to expose MCP server via HTTP (port 17390)
RUN npm install -g search-mcp-server@latest mcp-proxy

# Install pptxgenjs and playwright for PPTX conversion
# - pptxgenjs: PowerPoint generation (https://github.com/gitbrent/PptxGenJS)
# - playwright: Browser automation for rendering
RUN npm install -g pptxgenjs playwright
# Install Playwright browser binaries (Chromium is enough for most cases)
RUN npx playwright install chromium --with-deps

# Create systemd service for mcp-proxy wrapping search-mcp-server
# mcp-proxy exposes HTTP endpoint on port 17390
# search-mcp-server's WebSocket bridge for browser extension on port 17389
COPY mcp-proxy.service /etc/systemd/system/mcp-proxy.service
RUN systemctl enable mcp-proxy.service

# Create browser startup script
# --disable-infobars suppresses the "--no-sandbox" warning banner
COPY start-browser.sh /usr/local/bin/start-browser.sh
RUN chmod +x /usr/local/bin/start-browser.sh

# Verify extension copied
RUN ls -la /home/user/extensions/jan-browser-mcp/ && \
    cat /home/user/extensions/jan-browser-mcp/manifest.json | head -20
