# System Design# System Design



**Generated**: November 11, 2025This document describes the overall system architecture of Jan Server.



This document describes the overall system architecture of Jan Server.[Full content from architecture.md lines 1-195 covering System Overview, Architecture Diagram, all layers, and network topology will be placed here]



---For the complete implementation, please refer to the original `architecture.md` file which contains:

- Detailed architecture diagrams (lines 9-195)

## Overview- All system layers (Client, Gateway, Application, Inference, Authentication, Persistence, Observability)

- Component interactions

Jan Server is a modular, microservices-based LLM API platform with enterprise-grade authentication, API gateway routing, and flexible inference backend support. The system provides OpenAI-compatible API endpoints for chat completions, conversations, and model management.- Network topology



---This file should contain sections:

- System Overview

## System Architecture- Architecture Diagram (full ASCII diagram)

- Layer Descriptions

### Complete Ecosystem Diagram- Component Interactions

- Network Topology

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         JAN-SERVER ECOSYSTEM                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │                     EXTERNAL SERVICES (Optional)                     │  │
│  ├─────────────────────────────────────────────────────────────────────┤  │
│  │  • Keycloak (Auth Server)         [Port 8085]                       │  │
│  │  • SearXNG (Meta Search)          [Port 8086]                       │  │
│  │  • Serper API (Web Search)        [External]                        │  │
│  │  • LLM Models (Inference)         [External/Local]                  │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                    ↓                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │                     KONG API GATEWAY (Port 8000)                     │  │
│  │  Routes & rate-limits traffic to microservices                       │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                       ↙           ↓           ↘                            │
│                                                                              │
│  ┌──────────────────┐  ┌────────────────────┐  ┌─────────────────┐       │
│  │   LLM API        │  │  Response API      │  │   Media API     │       │
│  │  (Port 8080)     │  │ (Port 8082)        │  │  (Port 8081)    │       │
│  │                  │  │                    │  │                 │       │
│  │ • Auth & Tokens  │  │ • Tool Calling     │  │ • File Upload   │       │
│  │ • Chat/Completion│  │ • Multi-turn Chat  │  │ • Deduplication │       │
│  │ • Models         │  │ • MCP Integration  │  │ • Resolution    │       │
│  │ • Projects       │  │ • Conversations    │  │ • Streaming     │       │
│  │ • Conversations  │  │                    │  │                 │       │
│  └──────────────────┘  └────────────────────┘  └─────────────────┘       │
│         ↓                        ↓                       ↑                  │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │              MCP TOOLS SERVICE (Port 8091)                        │   │
│  │  • Serper Search    • File Search Index    • SandboxFusion        │   │
│  │  • Web Scraping     • Web Scraping         • Python Execution     │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                    ↓                                       │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                   DATABASE & STORAGE                              │   │
│  │  • PostgreSQL (Conversations, Projects, Responses)                │   │
│  │  • Vector DB (Optional - for semantic search)                     │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Service Communication Map

```
                     ┌──────────────────────┐
                     │   CLIENT/TEST        │
                     │   (Browser/Newman)   │
                     └──────────────────────┘
                                │
                ┌───────────────┼───────────────┐
                │               │               │
                ↓               ↓               ↓
         ┌────────────┐   ┌──────────┐   ┌──────────┐
         │   Kong     │   │Keycloak  │   │SearXNG   │
         │ (Gateway)  │   │(Auth)    │   │(Search)  │
         └──────┬─────┘   └────┬─────┘   └────┬─────┘
                │              │              │
     ┌──────────┼──────────────┼──────────────┘
     │          │              │
     ↓          ↓              ↓
┌──────────┐ ┌──────────┐ ┌────────────┐
│ LLM API  │←→ MCP Tools │ (external)  │
│ :8080    │ │ :8091    │            │
└────┬─────┘ └─────┬────┘ └────────────┘
     │             │
     │    ┌────────┘
     │    ↓
     ├→ Media API :8081
     │
     ├→ Response API :8082
     │
     └→ PostgreSQL (persistent storage)
```

---

## Core Services

### LLM API (Port 8080)
- **Purpose**: Core API for chat completions, model management, conversations, and projects
- **Technology**: Go + Gin framework
- **Database**: PostgreSQL
- **Features**:
  - Guest and registered user authentication
  - API key management
  - Model listing and details
  - Chat completions (OpenAI-compatible)
  - Conversation lifecycle management
  - Project organization

### Response API (Port 8082)
- **Purpose**: LLM response generation with tool orchestration
- **Technology**: Go + Gin framework
- **Features**:
  - Response creation and retrieval
  - Tool calling orchestration
  - Multi-step workflows
  - Conversation continuity
  - Response streaming

### Media API (Port 8081)
- **Purpose**: Media file upload, storage, and management
- **Technology**: Go + Gin framework
- **Features**:
  - Presigned URL generation
  - Remote URL ingestion
  - Data URL ingestion
  - Content deduplication
  - Placeholder resolution
  - Media streaming

### MCP Tools Service (Port 8091)
- **Purpose**: Model Context Protocol tools integration
- **Technology**: mark3labs/mcp-go v0.7.0
- **Tools**:
  - Serper (web search)
  - Web scraping
  - File search indexing
  - Python code execution
  - SearXNG integration

---

## Infrastructure Services

### Kong API Gateway (Port 8000)
- **Purpose**: API routing, rate limiting, authentication
- **Version**: Kong 3.5
- **Features**:
  - Request routing to microservices
  - Rate limiting
  - Authentication plugins
  - API key management
  - Request logging

### Keycloak (Port 8085)
- **Purpose**: OAuth2/OpenID Connect authentication
- **Configuration**:
  - Realm: `jan`
  - Client: `llm-api` (public)
  - User federation available

### Database (Port 5432)
- **Technology**: PostgreSQL 18
- **Databases**:
  - `jan_llm_api`: Main application data
  - `keycloak`: Identity management
- **Tables**:
  - `conversations`: Conversation history
  - `projects`: Project organization
  - `responses`: Response storage
  - `users`: User information
  - `api_keys`: API key management

---

## Architecture Layers

### 1. Client Layer
- Web browsers
- Mobile clients
- CLI tools (Newman for testing)

### 2. Gateway Layer
- Kong API Gateway (port 8000)
- Rate limiting
- Authentication plugins
- Request routing

### 3. Application Layer
- **LLM API**: Core chat and conversation functionality
- **Response API**: Response generation with tools
- **Media API**: File management
- **MCP Tools**: External tool integration

### 4. Inference Layer
- Local LLM models (via vLLM)
- External LLM APIs (OpenAI, Anthropic, etc.)
- Model inference endpoints

### 5. Authentication Layer
- Keycloak (OpenID Connect provider)
- JWT token validation
- API key management
- User federation

### 6. Persistence Layer
- PostgreSQL (conversations, projects, responses)
- Object storage (media files)
- Vector database (optional semantic search)

### 7. Integration Layer
- Serper API (web search)
- SearXNG (meta-search)
- External LLM providers
- File storage backends

---

## Data Flow Patterns

### Authentication Flow
```
Client
  ↓
[Guest Login / Keycloak]
  ↓
[Token Generation]
  ↓
[Token Validation at Gateway]
  ↓
[Service Access]
```

### Chat Completion Flow
```
Request
  ↓
[Authentication]
  ↓
[Conversation Lookup/Creation]
  ↓
[Message Routing]
  ↓
[Model Inference]
  ↓
[Response Storage]
  ↓
[Response Return]
```

### Tool Calling Flow
```
Request with Tools
  ↓
[Response Service]
  ↓
[Tool Discovery]
  ↓
[Tool Execution (MCP)]
  ↓
[Tool Result Processing]
  ↓
[LLM Reasoning with Results]
  ↓
[Final Response]
```

### Media Processing Flow
```
Media Request
  ↓
[Presigned URL / Direct Upload]
  ↓
[Deduplication Check]
  ↓
[Storage]
  ↓
[Placeholder Generation]
  ↓
[Resolution on Demand]
```

---

## Deployment Topology

### Development (Docker Compose)
```
┌─────────────────────────────────────┐
│ Docker Compose (Single Machine)     │
├─────────────────────────────────────┤
│ • All services in containers        │
│ • Shared network                    │
│ • Volume-based persistence          │
│ • Suitable for: Local dev, testing  │
└─────────────────────────────────────┘
```

### Production (Kubernetes)
```
┌──────────────────────────────────────────────────────────────┐
│ Kubernetes Cluster                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│ ┌─────────────┐  ┌────────────┐  ┌─────────────┐           │
│ │  LLM API    │  │ Response   │  │ Media API   │ (Pods)   │
│ │  (3x)       │  │ API (2x)   │  │ (2x)        │           │
│ └─────────────┘  └────────────┘  └─────────────┘           │
│                                                              │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Service Mesh (Optional - Istio)                          │ │
│ │ • Traffic management                                     │ │
│ │ • Security policies                                      │ │
│ │ • Observability                                          │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                              │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Persistent Storage                                       │ │
│ │ • PostgreSQL (managed service)                           │ │
│ │ • S3/Object Storage (media files)                        │ │
│ │ • Vector DB (optional)                                   │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## Scalability Considerations

### Horizontal Scaling
- **Stateless services**: LLM API, Response API, Media API
- **Load balancing**: Kong gateway distributes traffic
- **Database**: PostgreSQL connection pooling
- **Cache**: Optional Redis for session management

### Vertical Scaling
- Increase container resources (CPU, memory)
- PostgreSQL query optimization
- Connection pool tuning
- Database indexing strategy

---

## Security Architecture

### API Gateway Level
- Rate limiting per API key
- Request validation
- CORS handling
- SSL/TLS termination

### Service Level
- JWT token validation
- Keycloak integration
- API key verification
- RBAC (Role-based Access Control)

### Data Level
- Encrypted connections (TLS)
- Database encryption at rest
- Sensitive data masking in logs
- Access control per resource

---

## Related Documentation

- **Test Flows**: See `/docs/architecture/test-flows.md` for testing architecture
- **Services Reference**: See `/docs/architecture/services.md` for detailed service specs
- **Data Flow**: See `/docs/architecture/data-flow.md` for request patterns
- **Security**: See `/docs/architecture/security.md` for security details
- **Observability**: See `/docs/architecture/observability.md` for monitoring
- **Testing Guide**: See `/docs/guides/testing.md` for test execution

---

**Last Updated**: November 11, 2025  
**Document Type**: Architecture Reference  
**Target Audience**: Architects, Developers, DevOps  
**Maintainer**: Jan-Server Team
