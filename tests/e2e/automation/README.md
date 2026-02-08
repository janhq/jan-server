# E2E Test Automation

This directory contains end-to-end test collections for Jan Server.

## Structure

```
automation/
├── collections/              # Postman test collections
│   ├── auth.postman.json           # Authentication tests
│   ├── connectors.postman.json     # OAuth connector tests
│   ├── media.postman.json          # Media upload tests
│   ├── messages.postman.json       # Messages API tests
│   ├── model.postman.json          # Model catalog tests
│   ├── user-management.postman.json # User settings tests
│   ├── conversation/               # Conversation tests
│   │   ├── conversation.postman.json
│   │   ├── conversation-title.postman.json
│   │   └── conversation-ocr.postman.json
│   ├── mcp/                        # MCP tool tests
│   │   ├── mcp-admin.postman.json
│   │   ├── mcp-agent.postman.json
│   │   └── mcp-runtime.postman.json
│   └── response/                   # Response API tests
│       ├── response.postman.json           # Core response tests
│       ├── response-agent.postman.json     # Agent mode tests
│       ├── response-artifact.postman.json  # Artifact tests
│       └── response-mcp.postman.json       # MCP integration tests
└── README.md                       # This file
```

## Running Tests

### Postman Collections

Install Newman and run:
```bash
npm install -g newman
newman run automation/collections/auth.postman.json
```

### Environment Variables

Required variables (set in Postman or `.env`):
- `kong_url` - Kong gateway URL (default: http://localhost:8000)
- `gateway_url` - Gateway URL (default: http://localhost:8000)
- `mcp_tools_url` - MCP tools service URL
- `webhook_url` - Webhook test endpoint

### Playwright Tests

```bash
cd playwright
npm install
npx playwright test
```

## Test Categories

| Category | Coverage |
|----------|----------|
| Auth | Guest token, OAuth flows |
| Conversation | CRUD, titles, OCR |
| Response | Core, agent mode, artifacts, MCP |
| Media | Upload, resolve, jan_* IDs |
| MCP Tools | Admin, agent, runtime |
| Connectors | GitHub, Gmail, Google Drive |
