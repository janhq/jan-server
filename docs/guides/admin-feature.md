# Admin Features Documentation - Jan Server Platform

> Comprehensive documentation of all admin features in the Jan Server platform application (`apps/platform`).

---

## Table of Contents

1. [Overview](#1-overview)
2. [Admin Dashboard](#2-admin-dashboard)
3. [User Management](#3-user-management)
4. [Group Management](#4-group-management)
5. [Feature Flags Management](#5-feature-flags-management)
6. [Model Management Overview](#6-model-management-overview)
7. [Model Providers Management](#7-model-providers-management)
8. [Provider Models Management](#8-provider-models-management)
9. [Model Catalogs Management](#9-model-catalogs-management)
10. [Prompt Templates Management](#10-prompt-templates-management)
11. [MCP Tools Management](#11-mcp-tools-management)
12. [API Reference](#12-api-reference)
13. [Data Models](#13-data-models)
14. [Admin Workflows](#14-admin-workflows)

---

## 1. Overview

The Jan Server platform includes an extensive admin panel accessible at `/admin` with role-based access control. All admin features require admin authentication and are protected by the admin layout.

### Access Control

**File**: `apps/platform/src/app/admin/layout.tsx`

**Access Check Flow**:
1. Verify user is authenticated
2. Call `adminClient.checkIsAdmin()`
3. Check multiple admin indicators:
   - `is_admin === true`
   - `role === 'admin'`
   - `roles.includes('admin')`
4. Redirect to `/docs` if not admin
5. Show loading state during verification

### Navigation Structure

The admin panel provides sidebar navigation with the following main sections:
- Dashboard (Overview)
- Users (User & Group Management)
- Models (Providers, Models, Catalogs)
- Prompt Templates
- MCP Tools

---

## 2. Admin Dashboard

**File**: `apps/platform/src/app/admin/page.tsx`

**URL**: `/admin`

**Purpose**: Central hub displaying key metrics and quick access to all admin features.

### Dashboard Statistics

| Metric | Description | API Call |
|--------|-------------|----------|
| Total Users | Count of all registered users | `adminClient.users.listUsers({ offset: 0, limit: 1 })` |
| Active Users | Count of enabled users | `adminClient.users.listUsers({ enabled: true, offset: 0, limit: 1 })` |
| Total Models | Count of all provider models | `adminClient.providerModels.listProviderModels({ limit: 1 })` |
| Active Models | Count of active provider models | `adminClient.providerModels.listProviderModels({ active: true, limit: 1 })` |
| Providers | Total number of model providers | `adminClient.providers.listProviders({ limit: 1 })` |
| Model Usage | Percentage of active models | Calculated from active/total |

### Quick Action Cards

| Card | Description | Link |
|------|-------------|------|
| Manage Users | Access user management | `/admin/users` |
| Model Providers | Configure providers | `/admin/models/providers` |
| Provider Models | Manage individual models | `/admin/models/provider-models` |
| Model Catalogs | Model metadata & templates | `/admin/models/catalogs` |

### System Status Indicators

| Status | Description |
|--------|-------------|
| API Status | Operational indicator |
| Database Status | Connection status |
| Authentication Status | Auth service status |

---

## 3. User Management

**File**: `apps/platform/src/app/admin/users/page.tsx`

**URL**: `/admin/users`

**Purpose**: Manage all users, their groups, permissions, and feature flags.

### User List Display

**Table Columns**:
| Column | Description |
|--------|-------------|
| User | Profile picture, name |
| Email | User email address |
| Username | Unique username |
| Status | Enabled/Disabled badge |
| Groups | Group memberships with count |
| Feature Flags | Active flags with count |
| Role | Admin/User badge |
| Actions | Dropdown menu |

### Filtering & Search

| Filter | Type | Options |
|--------|------|---------|
| Search | Text input | Email or username |
| Status | Dropdown | All / Enabled / Disabled |
| Hide Guests | Toggle | Show/hide guest users |

### Pagination

- **Page Size**: 20 users per page
- **Navigation**: Previous/Next buttons
- **Display**: Current page indicator

### User Actions (Dropdown Menu)

| Action | Description | API Endpoint |
|--------|-------------|--------------|
| Manage Groups | Add/remove group memberships | Multiple group APIs |
| View Feature Flags | See active feature flags | `GET /v1/admin/users/{userId}` |
| Activate User | Enable user account | `POST /v1/admin/users/{userId}/activate` |
| Deactivate User | Disable user account | `POST /v1/admin/users/{userId}/deactivate` |
| Assign Admin Role | Grant admin privileges | `POST /v1/admin/users/{userId}/roles/admin` |

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List Users | GET | `/v1/admin/users` |
| Create User | POST | `/v1/admin/users` |
| Get User | GET | `/v1/admin/users/{userId}` |
| Update User | PATCH | `/v1/admin/users/{userId}` |
| Activate User | POST | `/v1/admin/users/{userId}/activate` |
| Deactivate User | POST | `/v1/admin/users/{userId}/deactivate` |
| Assign Admin | POST | `/v1/admin/users/{userId}/roles/admin` |

---

## 4. Group Management

**File**: `apps/platform/src/app/admin/users/page.tsx` (Modal component)

**Purpose**: Create and manage user groups for organization and permission assignment.

### Group List Features

| Feature | Description |
|---------|-------------|
| Group Name | Display name of the group |
| Delete Button | Remove group (with confirmation) |
| Scrollable List | Handle many groups |

### Create Group Modal

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Group Name | Text input | Yes | Unique group name |

### User Group Assignment Modal

| Feature | Description |
|---------|-------------|
| Current Groups | List with remove buttons |
| Available Groups | Dropdown selector |
| Add Button | Assign selected group |

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List Groups | GET | `/v1/admin/groups` |
| Create Group | POST | `/v1/admin/groups` |
| Delete Group | DELETE | `/v1/admin/groups/{groupId}` |
| Add User to Group | POST | `/v1/admin/users/{userId}/groups/{groupId}` |
| Remove User from Group | DELETE | `/v1/admin/users/{userId}/groups/{groupId}` |

---

## 5. Feature Flags Management

**File**: `apps/platform/src/app/admin/users/feature-flags/page.tsx`

**URL**: `/admin/users/feature-flags`

**Purpose**: Create and manage feature flags for gradual feature rollout and access control.

### Feature Flag List

**Table Columns**:
| Column | Description |
|--------|-------------|
| Key | Unique identifier |
| Name | Display name |
| Description | Optional description |
| Created At | Creation timestamp |
| Actions | Edit/Delete buttons |

### Create/Edit Feature Flag Modal

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Key | Text input | Yes | Unique identifier (immutable after creation) |
| Name | Text input | Yes | Human-readable name |
| Description | Textarea | No | Detailed description |
| Auto-enabled for new groups | Checkbox | No | Default assignment behavior |

### Group Feature Flags Assignment

| Feature | Description |
|---------|-------------|
| Group Selection | Choose target group |
| Flag Checkboxes | Enable/disable flags per group |
| Batch Save | Update all flags at once |

### User Feature Flags Modal

| Feature | Description |
|---------|-------------|
| Active Flags | Flags inherited from groups |
| All Available Flags | Complete flag list |
| Group Membership | Groups and their flags |

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List Feature Flags | GET | `/v1/admin/feature-flags` |
| Create Feature Flag | POST | `/v1/admin/feature-flags` |
| Update Feature Flag | PATCH | `/v1/admin/feature-flags/{flagId}` |
| Delete Feature Flag | DELETE | `/v1/admin/feature-flags/{flagId}` |
| Get Group Flags | GET | `/v1/admin/groups/{groupId}/feature-flags` |
| Set Group Flags | PATCH | `/v1/admin/groups/{groupId}/feature-flags` |

---

## 6. Model Management Overview

**File**: `apps/platform/src/app/admin/models/page.tsx`

**URL**: `/admin/models`

**Purpose**: Central overview for managing model providers, individual models, and catalog entries.

### Dashboard Statistics

| Metric | Description |
|--------|-------------|
| Total Providers | Number of configured providers |
| Active Providers | Enabled providers |
| Total Models | All synced models |
| Active Models | Enabled models |
| Model Catalogs | Catalog entries |

### Quick Navigation Tips

- How to sync models from providers
- How to activate/deactivate models
- How to view model capabilities
- How to use filters

---

## 7. Model Providers Management

**File**: `apps/platform/src/app/admin/models/providers/page.tsx`

**URL**: `/admin/models/providers`

**Purpose**: Manage external model providers (OpenAI, Anthropic, Google, etc.).

### Provider Grid Display

| Field | Description |
|-------|-------------|
| Provider Name | Name with vendor |
| Status | Active/Inactive badge |
| Model Counts | Total and active models |
| Provider ID | Copyable public ID |
| Endpoints | First 3 endpoints shown |
| Updated At | Last modification time |

### Filtering

| Filter | Type | Options |
|--------|------|---------|
| Provider Type | Dropdown | openrouter, openai, anthropic, google, local |
| Status | Dropdown | All / Active / Inactive |

### Create Provider Modal

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Name | Text input | Yes | Provider display name |
| Vendor | Text input | Yes | Provider vendor (openai, anthropic, etc.) |
| Category | Dropdown | Yes | llm or image |
| Base URL | Text input | No | Fallback URL |
| Endpoints | Textarea | No | Comma-separated or JSON array |
| Image Edit Path | Text input | No | Path for image editing |
| API Key | Password input | No | Provider API key |
| Description | Textarea | No | Provider description |
| Active | Checkbox | No | Enable/disable provider |
| Default for Image Generation | Checkbox | No | Use as default image provider |
| Default for Image Editing | Checkbox | No | Use as default edit provider |
| Auto-enable New Models | Checkbox | No | Auto-activate synced models |

### Edit Provider Modal

Same fields as Create, plus:
- Read-only provider ID
- Model count statistics

### Provider Actions

| Action | Description | Confirmation |
|--------|-------------|--------------|
| Edit | Modify provider settings | None |
| Sync Models | Fetch models from provider | Confirmation dialog |
| Delete | Remove provider and all models | Text input "Approved" |

### Sync Models Feature

| Option | Description |
|--------|-------------|
| Auto-enable New Models | Automatically activate newly synced models |
| Success Message | Shows count of synced models |

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List Providers | GET | `/v1/admin/providers` |
| Get Provider | GET | `/v1/admin/providers/{publicId}` |
| Create Provider | POST | `/v1/admin/providers` |
| Update Provider | PATCH | `/v1/admin/providers/{publicId}` |
| Delete Provider | DELETE | `/v1/admin/providers/{publicId}` |
| Sync Models | POST | `/v1/admin/providers/{publicId}/sync` |

---

## 8. Provider Models Management

**File**: `apps/platform/src/app/admin/models/provider-models/page.tsx`

**URL**: `/admin/models/provider-models`

**Purpose**: Manage individual models synced from providers.

### Model Table

| Column | Description |
|--------|-------------|
| Model | Display name with public ID (code formatted) |
| Catalog Link | Link to model catalog entry |
| Provider | Provider name |
| Status | Active/Inactive badge |
| Category | Category badge |
| Pricing | Input/Output per 1M tokens |
| Actions | Edit, toggle active |

### Filtering & Search

| Filter | Type | Options |
|--------|------|---------|
| Search | Text input | Model ID, display name, or provider |
| Provider | Dropdown | List of all providers |
| Status | Dropdown | All / Active / Inactive |
| Image Support | Checkbox | Filter by image capability |

### Bulk Operations

| Operation | Description |
|-----------|-------------|
| Select All | Select all visible models |
| Deselect All | Clear selection |
| Bulk Activate | Enable selected models |
| Bulk Deactivate | Disable selected models |
| Selection Counter | Shows count of selected items |

### Pagination

- **Page Size**: 20 models per page
- **Display**: Total model count
- **Navigation**: Previous/Next buttons

### Edit Model Modal

**Basic Settings**:
| Field | Type | Description |
|-------|------|-------------|
| Display Name | Text input | Model display name |
| Category | Dropdown | jan, legacy |
| Category Order | Number | Sort order within category |
| Model Order | Number | Sort order within models |

**Pricing**:
| Field | Type | Description |
|-------|------|-------------|
| Prompt Price | Number | Cost per 1M input tokens |
| Completion Price | Number | Cost per 1M output tokens |

**Token Limits**:
| Field | Type | Description |
|-------|------|-------------|
| Context Length | Number | Maximum context window |
| Max Completion Tokens | Number | Maximum output tokens |

**Capabilities** (Checkboxes):
| Capability | Description |
|------------|-------------|
| Vision (supports_images) | Can process images |
| Audio (supports_audio) | Can process audio |
| Video (supports_video) | Can process video |
| Reasoning (supports_reasoning) | Extended thinking capability |
| Embeddings (supports_embeddings) | Vector embeddings support |

**Additional**:
| Field | Type | Description |
|-------|------|-------------|
| Instruct Model Backup | Dropdown | Fallback model for reasoning |
| Active | Toggle | Enable/disable model |

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List Models | GET | `/v1/admin/models/provider-models` |
| Get Model | GET | `/v1/admin/models/provider-models/{publicId}` |
| Update Model | PATCH | `/v1/admin/models/provider-models/{publicId}` |
| Batch Update | POST | `/v1/admin/models/provider-models/bulk-toggle` |

---

## 9. Model Catalogs Management

**File**: `apps/platform/src/app/admin/models/catalogs/page.tsx`

**URL**: `/admin/models/catalogs`

**Purpose**: Manage model catalog entries with metadata and prompt template assignments.

### Catalog Display

| Field | Description |
|-------|-------------|
| Model Name | Display name |
| Description | Model description |
| Status | Active/Inactive badge |
| Moderation | Moderation status |
| Capabilities | Capability icons/badges |
| Feature Flag | Required flag (if any) |
| Active Toggle | Quick activation |

### Filtering

| Filter | Type | Options |
|--------|------|---------|
| Search | Text input | Model name |
| Family | Dropdown | Model family |
| Status | Dropdown | Active / Inactive |
| Experimental | Checkbox | Show experimental only |
| Feature Flag | Dropdown | Required flag filter |

**Capability Filters**:
| Capability | Description |
|------------|-------------|
| Supports Embeddings | Vector embedding support |
| Supports Images/Vision | Image processing |
| Supports Reasoning | Extended thinking |
| Supports Audio | Audio processing |
| Supports Video | Video processing |
| Supports Tools | Tool/function calling |
| Supports Browser | Web browsing capability |

### Edit Catalog Modal

**Basic Information**:
| Field | Type | Description |
|-------|------|-------------|
| Display Name | Text input | Model display name |
| Description | Textarea | Model description |
| Family | Text input | Model family |
| Status | Dropdown | Status setting |
| Moderation Status | Dropdown | Moderation level |
| Experimental | Checkbox | Mark as experimental |
| Feature Flag Requirement | Dropdown | Required feature flag |
| Tags | Tag input | Model tags |
| Context Length | Number | Context window size |
| Notes | Textarea | Additional notes |

**Architecture Details**:
| Field | Type | Description |
|-------|------|-------------|
| Modality | Text input | Input/output modality |
| Input Modalities | Array | Accepted input types |
| Output Modalities | Array | Generated output types |
| Tokenizer | Text input | Tokenizer type |
| Instruct Type | Text input | Instruction format |

**Supported Parameters**:
- JSON configuration for model parameters

**Capabilities** (Checkboxes):
Same as Provider Models capabilities

### Model Prompt Templates Tab

**File**: `apps/platform/src/app/admin/models/catalogs/components/ModelPromptTemplatesTab.tsx`

**Purpose**: Assign system/tool/reasoning prompts to specific models.

**Features**:
| Feature | Description |
|---------|-------------|
| Template Assignments | List of assigned templates |
| Assign Template | Add new template assignment |
| Update Assignment | Modify priority, active status |
| Remove Assignment | Unassign template |
| Effective Templates | View resolved template chain |

**Assignment Fields**:
| Field | Type | Description |
|-------|------|-------------|
| Template Key | Dropdown | Select template to assign |
| Priority | Number | Template priority (higher = preferred) |
| Is Active | Checkbox | Enable/disable assignment |

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List Catalogs | GET | `/v1/admin/models/catalogs` |
| Get Catalog | GET | `/v1/admin/models/catalogs/{publicId}` |
| Update Catalog | PATCH | `/v1/admin/models/catalogs/{publicId}` |
| Batch Toggle | POST | `/v1/admin/models/catalogs/bulk-toggle` |
| List Model Templates | GET | `/v1/admin/models/prompt-templates/list/{modelId}` |
| Assign Template | POST | `/v1/admin/models/prompt-templates/assign/{modelId}` |
| Update Assignment | PATCH | `/v1/admin/models/prompt-templates/update/{templateKey}/{modelId}` |
| Unassign Template | DELETE | `/v1/admin/models/prompt-templates/unassign/{templateKey}/{modelId}` |
| Get Effective Templates | GET | `/v1/admin/models/prompt-templates/effective/{modelId}` |

---

## 10. Prompt Templates Management

**File**: `apps/platform/src/app/admin/prompt-templates/page.tsx`

**URL**: `/admin/prompt-templates`

**Purpose**: Manage system prompts, tool instructions, and reasoning templates used by models.

### Template Table

| Column | Description |
|--------|-------------|
| Name | Template name |
| Category | Category badge (orchestration, system, tool, reasoning) |
| Description | Template description |
| Status | Active/Inactive badge |
| Template Key | Unique identifier |
| System | Lock icon for system templates |
| Created At | Creation timestamp |
| Actions | Menu with options |

### Filtering & Search

| Filter | Type | Options |
|--------|------|---------|
| Search | Text input | Name or content |
| Category | Dropdown | orchestration, system, tool, reasoning |
| Status | Dropdown | All / Active / Inactive |

### Create/Edit Template Modal

**File**: `apps/platform/src/app/admin/prompt-templates/components/prompt-template-modal.tsx`

**Basic Tab**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Name | Text input | Yes | Template name |
| Template Key | Text input | Yes | Unique key (read-only on edit) |
| Category | Dropdown | Yes | Template category |
| Active | Toggle | No | Enable/disable template |

**Content Tab**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Content | Markdown editor | Yes | Template content |

**Variables Tab**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Variables | JSON array | No | Variable names used in template |

**Metadata Tab**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Metadata | JSON object | No | Additional metadata |

### Preview Modal

**File**: `apps/platform/src/app/admin/prompt-templates/components/preview-modal.tsx`

| Feature | Description |
|---------|-------------|
| Rendered Content | Display template content |
| Variables List | Show available variables |
| Metadata Display | Show metadata |

### Template Actions

| Action | Description | Availability |
|--------|-------------|--------------|
| Edit | Modify template | All templates |
| Duplicate | Clone template | All templates |
| Activate | Enable template | Inactive templates |
| Deactivate | Disable template | Active templates |
| Delete | Remove template | Non-system templates only |
| Preview | View rendered template | All templates |

### Special Rules

- **System templates** cannot be deleted, only edited
- **Template keys** are unique within categories
- **Duplication** creates a new template with optional name suffix

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List Templates | GET | `/v1/admin/prompt-templates` |
| Get Template | GET | `/v1/admin/prompt-templates/{publicId}` |
| Get by Key | GET | `/v1/admin/prompt-templates/key/{templateKey}` |
| Create Template | POST | `/v1/admin/prompt-templates` |
| Update Template | PATCH | `/v1/admin/prompt-templates/{publicId}` |
| Delete Template | DELETE | `/v1/admin/prompt-templates/{publicId}` |
| Duplicate Template | POST | `/v1/admin/prompt-templates/{publicId}/duplicate` |

---

## 11. MCP Tools Management

**File**: `apps/platform/src/app/admin/mcp-tools/page.tsx`

**URL**: `/admin/mcp-tools`

**Purpose**: Manage Model Context Protocol (MCP) tools like search, web scraping, and code execution.

### Tools Table

| Column | Description |
|--------|-------------|
| Icon | Category icon |
| Name | Tool name |
| Category | Tool category badge |
| Status | Active/Inactive badge |
| Description | First 60 characters |
| Actions | Edit menu |

### Categories

| Category | Description |
|----------|-------------|
| search | Web search tools |
| scrape | Web scraping tools |
| file_search | File system search |
| code_execution | Code runner tools |
| memory | Semantic memory tools |

### Filtering & Search

| Filter | Type | Options |
|--------|------|---------|
| Search | Text input | Name or description |
| Category | Dropdown | All categories listed above |
| Status | Dropdown | All / Active / Inactive |

### Edit MCP Tool Modal

**File**: `apps/platform/src/app/admin/mcp-tools/components/mcp-tool-modal.tsx`

**General Tab**:
| Field | Type | Editable | Description |
|-------|------|----------|-------------|
| Tool Name | Text | Read-only | Tool display name |
| Tool Key | Text | Read-only | Unique identifier |
| Description | Textarea | Yes | Tool description |
| Category | Dropdown | Yes | Tool category |
| Active | Toggle | Yes | Enable/disable tool |

**Advanced Tab**:
| Field | Type | Description |
|-------|------|-------------|
| Disallowed Keywords | Textarea | One regex pattern per line |

**Disallowed Keywords Features**:
- Supports Go regex patterns
- Inline flags supported (e.g., `(?i)` for case-insensitive)
- Pattern validation on save
- Blocks tool execution when input matches patterns

### CRUD Operations

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List MCP Tools | GET | `/v1/admin/mcp-tools` |
| Get MCP Tool | GET | `/v1/admin/mcp-tools/{publicId}` |
| Update MCP Tool | PATCH | `/v1/admin/mcp-tools/{publicId}` |

---

## 12. API Reference

### API Client Architecture

**File**: `apps/platform/src/lib/admin/api.ts`

### AdminAPIClient (Main Entry Point)

```typescript
export class AdminAPIClient {
  users: UserManagementAPI;
  providers: ProviderManagementAPI;
  providerModels: ProviderModelManagementAPI;
  modelCatalogs: ModelCatalogManagementAPI;
  promptTemplates: PromptTemplateManagementAPI;
  mcpTools: MCPToolManagementAPI;

  checkIsAdmin(): Promise<boolean>;
}
```

### UserManagementAPI Methods

| Method | Description |
|--------|-------------|
| `getMe()` | Get current user profile |
| `listUsers(params)` | List all users with filters |
| `createUser(data)` | Create new user |
| `getUser(userId)` | Get specific user |
| `updateUser(userId, data)` | Update user |
| `deactivateUser(userId)` | Deactivate user account |
| `activateUser(userId)` | Reactivate user |
| `assignAdminRole(userId)` | Make user admin |
| `listGroups()` | List all groups |
| `createGroup(name)` | Create new group |
| `deleteGroup(groupId)` | Delete group |
| `addUserToGroup(userId, groupId)` | Add user to group |
| `removeUserFromGroup(userId, groupId)` | Remove from group |
| `getGroupFeatureFlags(groupId)` | Get group flags |
| `setGroupFeatureFlags(groupId, flags)` | Set group flags |
| `listFeatureFlags()` | List all feature flags |
| `createFeatureFlag(data)` | Create flag |
| `updateFeatureFlag(flagId, data)` | Update flag |
| `deleteFeatureFlag(flagId)` | Delete flag |

### ProviderManagementAPI Methods

| Method | Description |
|--------|-------------|
| `listProviders(params)` | List all providers |
| `getProvider(publicId)` | Get specific provider |
| `createProvider(data)` | Create new provider |
| `updateProvider(publicId, data)` | Update provider config |
| `deleteProvider(publicId)` | Delete provider |
| `syncProviderModels(publicId, autoEnableNewModels)` | Sync models |

### ProviderModelManagementAPI Methods

| Method | Description |
|--------|-------------|
| `listProviderModels(params)` | List all models |
| `getProviderModel(publicId)` | Get specific model |
| `updateProviderModel(publicId, data)` | Update model config |
| `activateModel(publicId)` | Enable model |
| `deactivateModel(publicId)` | Disable model |
| `batchUpdateActive(params)` | Bulk activate/deactivate |

### ModelCatalogManagementAPI Methods

| Method | Description |
|--------|-------------|
| `listModelCatalogs(params)` | List all catalogs |
| `getModelCatalog(publicId)` | Get specific catalog |
| `updateModelCatalog(publicId, data)` | Update catalog |
| `batchToggle(params)` | Bulk toggle |
| `listModelPromptTemplates(modelId)` | List assignments |
| `assignPromptTemplate(modelId, data)` | Assign template |
| `updatePromptTemplateAssignment(modelId, templateKey, data)` | Update assignment |
| `unassignPromptTemplate(modelId, templateKey)` | Remove assignment |
| `getEffectiveTemplates(modelId)` | Get resolved templates |

### PromptTemplateManagementAPI Methods

| Method | Description |
|--------|-------------|
| `listPromptTemplates(params)` | List all templates |
| `getPromptTemplate(publicId)` | Get specific template |
| `getPromptTemplateByKey(templateKey)` | Get by key |
| `createPromptTemplate(data)` | Create new template |
| `updatePromptTemplate(publicId, data)` | Update template |
| `deletePromptTemplate(publicId)` | Delete template |
| `duplicatePromptTemplate(publicId, data)` | Clone template |
| `activateTemplate(publicId)` | Activate template |
| `deactivateTemplate(publicId)` | Deactivate template |

### MCPToolManagementAPI Methods

| Method | Description |
|--------|-------------|
| `listMCPTools(params)` | List all tools |
| `getMCPTool(publicId)` | Get specific tool |
| `updateMCPTool(publicId, data)` | Update tool settings |
| `activateTool(publicId)` | Enable tool |
| `deactivateTool(publicId)` | Disable tool |

### Base URL Configuration

```typescript
const JAN_BASE_URL = process.env.NEXT_PUBLIC_JAN_BASE_URL || 'http://localhost:8000'
```

### Authentication

- All requests use Bearer token from auth service
- Token validation handled by `getValidAccessToken()`
- Credentials included in requests

---

## 13. Data Models

### User Types

```typescript
interface UserProfile {
  id: string;
  email?: string;
  username?: string;
  first_name?: string;
  last_name?: string;
  name?: string;
  picture?: string;
  object: string;
  role?: string;
  is_admin?: boolean;
  enabled?: boolean;
  active?: boolean;
  created_at?: string;
  updated_at?: string;
  groups?: Group[];
  roles?: string[];
}

interface Group {
  id: string;
  name: string;
  path?: string;
  feature_flags?: string[];
  created_at?: string;
  updated_at?: string;
}

interface FeatureFlag {
  id: string;
  key: string;
  name: string;
  description?: string;
  enabled?: boolean;
  created_at?: string;
  updated_at?: string;
}
```

### Provider Types

```typescript
interface Provider {
  id: string;
  name: string;
  vendor: string;
  base_url?: string;
  endpoints?: Endpoint[];
  active: boolean;
  category?: string;
  default_provider_image_generate?: boolean;
  default_provider_image_edit?: boolean;
  model_count?: number;
  model_active_count?: number;
  metadata?: Record<string, any>;
  created_at?: string;
  updated_at?: string;
}

interface Endpoint {
  url: string;
  weight?: number;
  priority?: number;
  healthy?: boolean;
}
```

### Model Types

```typescript
interface ProviderModel {
  id: string;
  provider_id: string;
  model_display_name: string;
  model_id: string;
  model_public_id: string;
  pricing: {
    prompt?: number;
    completion?: number;
    image?: number;
    request?: number;
  };
  category: string;
  category_order_number?: number;
  model_order_number?: number;
  active: boolean;
  created_at?: string;
  updated_at?: string;
  catalog?: ModelCatalog;
  supports_audio?: boolean;
  supports_embeddings?: boolean;
  supports_images?: boolean;
  supports_reasoning?: boolean;
  supports_instruct?: boolean;
  supports_video?: boolean;
  supports_tools?: boolean;
  supports_browser?: boolean;
  token_limits?: {
    context_length?: number;
    max_completion_tokens?: number;
  };
  instruct_model_public_id?: string;
}

interface ModelCatalog {
  id: string;
  model_display_name?: string;
  description?: string;
  supported_parameters?: any;
  architecture?: string | ArchitectureDetails;
  supports_images?: boolean;
  supports_embeddings?: boolean;
  supports_reasoning?: boolean;
  supports_instruct?: boolean;
  supports_audio?: boolean;
  supports_video?: boolean;
  supports_tools?: boolean;
  supports_browser?: boolean;
  family?: string;
  status?: string;
  is_moderated?: boolean;
  active?: boolean;
  experimental?: boolean;
  requires_feature_flag?: string | null;
  notes?: string;
  context_length?: number;
  tags?: string[];
  created_at?: string;
  updated_at?: string;
}
```

### Template Types

```typescript
interface PromptTemplate {
  id: string;
  public_id: string;
  name: string;
  description?: string;
  category: string;
  template_key: string;
  content: string;
  variables?: string[];
  metadata?: Record<string, any>;
  is_active: boolean;
  is_system: boolean;
  version: number;
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
}

interface ModelPromptTemplate {
  id: string;
  model_catalog_id: string;
  template_key: string;
  prompt_template_id: string;
  priority: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  prompt_template?: PromptTemplate;
}
```

### MCP Tool Types

```typescript
interface MCPTool {
  id: string;
  public_id: string;
  tool_key: string;
  name: string;
  description: string;
  category: string;
  is_active: boolean;
  metadata?: Record<string, any>;
  disallowed_keywords?: string[];
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
}
```

---

## 14. Admin Workflows

### Workflow 1: Add New Model Provider

1. Navigate to `/admin/models/providers`
2. Click "Add Provider" button
3. Fill in provider details:
   - Name and vendor (required)
   - Category (llm/image)
   - Endpoints or base URL
   - API key (if required)
4. Set active status and defaults
5. Submit form → Creates provider
6. Click "Sync Models" → Fetches available models
7. Models appear in Provider Models list

### Workflow 2: Manage User Permissions via Groups

1. Navigate to `/admin/users`
2. Search and find target user
3. Click menu → "Manage Groups"
4. View current group memberships
5. Add groups from dropdown
6. Remove groups with X button
7. Groups automatically grant feature flags
8. User gains access to gated features

### Workflow 3: Set Up Model Prompt Template

1. Navigate to `/admin/prompt-templates`
2. Create or edit a template:
   - Set name and key
   - Choose category (system/tool/reasoning)
   - Write template content
   - Define variables (optional)
3. Navigate to `/admin/models/catalogs`
4. Find target model in list
5. Click "Prompt Templates" tab
6. Click "Assign Template"
7. Select template key and set priority
8. Model now uses custom prompts

### Workflow 4: Enable/Disable MCP Tool

1. Navigate to `/admin/mcp-tools`
2. Search for tool (e.g., "web_search")
3. Click menu → Edit
4. Toggle Active checkbox
5. Optionally configure disallowed keywords:
   - Add regex patterns (one per line)
   - Patterns block matching inputs
6. Save changes
7. Tool is now enabled/disabled globally

### Workflow 5: Feature-Gate a Model

1. Navigate to `/admin/users/feature-flags`
2. Create new feature flag:
   - Key: `model_beta_access`
   - Name: "Beta Model Access"
3. Navigate to `/admin/models/catalogs`
4. Find model to gate
5. Edit catalog
6. Set "Requires Feature Flag" to `model_beta_access`
7. Save changes
8. Navigate to `/admin/users`
9. Manage groups for target users
10. Assign flag to groups
11. Only users in those groups can access the model

### Workflow 6: Bulk Model Management

1. Navigate to `/admin/models/provider-models`
2. Use filters to find target models:
   - Filter by provider
   - Filter by status
   - Search by name
3. Select models using checkboxes
4. Use bulk actions:
   - "Activate Selected" to enable all
   - "Deactivate Selected" to disable all
5. Changes apply immediately

---

## File Structure Summary

```
apps/platform/src/
├── app/admin/
│   ├── page.tsx                                    # Dashboard
│   ├── layout.tsx                                  # Admin layout & auth check
│   ├── users/
│   │   ├── page.tsx                                # User management
│   │   └── feature-flags/
│   │       └── page.tsx                            # Feature flags management
│   ├── models/
│   │   ├── page.tsx                                # Model overview
│   │   ├── providers/
│   │   │   └── page.tsx                            # Provider management
│   │   ├── provider-models/
│   │   │   └── page.tsx                            # Provider models management
│   │   └── catalogs/
│   │       ├── page.tsx                            # Model catalogs management
│   │       └── components/
│   │           └── ModelPromptTemplatesTab.tsx     # Template assignments
│   ├── prompt-templates/
│   │   ├── page.tsx                                # Template list & management
│   │   └── components/
│   │       ├── prompt-template-modal.tsx           # Create/edit templates
│   │       └── preview-modal.tsx                   # Template preview
│   └── mcp-tools/
│       ├── page.tsx                                # MCP tools list & management
│       └── components/
│           └── mcp-tool-modal.tsx                  # Edit MCP tools
├── lib/admin/
│   └── api.ts                                      # Admin API client
└── store/
    └── auth-store.ts                               # Auth state management
```

---

## Summary Statistics

| Category | Count |
|----------|-------|
| Admin Pages | 10 |
| Modal Components | 5 |
| API Client Classes | 6 |
| Total API Endpoints | 45+ |
| Data Models | 10+ |
| Filtering Options | 30+ |
| Bulk Operations | 4 |
