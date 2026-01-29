// Admin Types for User, Provider, Model, and Tool Management

// User Management Types
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

// Provider Types
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
  metadata?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
}

interface Endpoint {
  url: string;
  weight?: number;
  priority?: number;
  healthy?: boolean;
}

// Provider Model Types
interface ProviderModel {
  id: string;
  provider_id: string;
  provider_vendor: string;
  model_catalog_id?: string;
  model_public_id: string;
  provider_original_model_id: string;
  model_display_name: string;
  category: string;
  category_order_number?: number;
  model_order_number?: number;
  pricing: {
    prompt?: number;
    completion?: number;
    image?: number;
    request?: number;
  };
  token_limits?: {
    context_length?: number;
    max_completion_tokens?: number;
  };
  family?: string;
  active: boolean;
  supports_audio?: boolean;
  supports_embeddings?: boolean;
  supports_images?: boolean;
  supports_reasoning?: boolean;
  supports_instruct?: boolean;
  supports_video?: boolean;
  supports_tools?: boolean;
  supports_browser?: boolean;
  instruct_model_public_id?: string;
  created_at?: number;
  updated_at?: number;
}

// Model Catalog Types
interface ModelCatalog {
  id: string;
  public_id: string;
  model_display_name?: string;
  description?: string;
  supported_parameters?: unknown;
  architecture?:
    | string
    | {
        modality?: string;
        input_modalities?: string[] | null;
        output_modalities?: string[] | null;
        tokenizer?: string;
        instruct_type?: string | null;
      };
  tags?: string[];
  notes?: string;
  context_length?: number;
  is_moderated?: boolean;
  active?: boolean;
  extras?: Record<string, unknown>;
  status?: string;
  family?: string;
  experimental?: boolean;
  requires_feature_flag?: string | null;
  supports_images?: boolean;
  supports_embeddings?: boolean;
  supports_reasoning?: boolean;
  supports_instruct?: boolean;
  supports_audio?: boolean;
  supports_video?: boolean;
  supports_tools?: boolean;
  supports_browser?: boolean;
  last_synced_at?: number;
  created_at?: number;
  updated_at?: number;
}

// Prompt Template Types
interface PromptTemplate {
  id: string;
  public_id: string;
  name: string;
  description?: string;
  category: string;
  template_key: string;
  content: string;
  variables?: string[];
  metadata?: Record<string, unknown>;
  is_active: boolean;
  is_system: boolean;
  version: number;
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
}

interface CreatePromptTemplateRequest {
  name: string;
  description?: string;
  category: string;
  template_key: string;
  content: string;
  variables?: string[];
  metadata?: Record<string, unknown>;
  is_active?: boolean;
}

interface UpdatePromptTemplateRequest {
  name?: string;
  description?: string;
  category?: string;
  content?: string;
  variables?: string[];
  metadata?: Record<string, unknown>;
  is_active?: boolean;
}

interface DuplicatePromptTemplateRequest {
  new_name?: string;
}

// Model Prompt Template Types (Model-Specific Template Assignments)
interface ModelPromptTemplate {
  id: string;
  model_catalog_id: string;
  template_key: string;
  prompt_template_id: string;
  priority: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  prompt_template?: {
    public_id: string;
    name: string;
    description?: string;
    category: string;
    template_key: string;
    is_active: boolean;
  };
}

interface AssignTemplateRequest {
  template_key: string;
  prompt_template_id: string;
  priority?: number;
  is_active?: boolean;
}

interface UpdateAssignmentRequest {
  prompt_template_id?: string;
  priority?: number;
  is_active?: boolean;
}

interface EffectiveTemplate {
  template: PromptTemplate | null;
  source: "model_specific" | "global_default" | "hardcoded";
}

interface EffectiveTemplatesResponse {
  templates: Record<string, EffectiveTemplate>;
}

// MCP Tool Types (Admin management - different from MCP SDK tool in mcp.d.ts)
interface AdminMCPTool {
  id: string;
  public_id: string;
  tool_key: string;
  name: string;
  description: string;
  category: string;
  is_active: boolean;
  metadata?: Record<string, unknown>;
  disallowed_keywords?: string[];
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
}

interface UpdateAdminMCPToolRequest {
  description?: string;
  category?: string;
  is_active?: boolean;
  metadata?: Record<string, unknown>;
  disallowed_keywords?: string[];
}

// API Response Types
interface ListResponse<T> {
  data: T[];
  total?: number;
  limit?: number;
  offset?: number;
}

interface SyncProviderResponse {
  synced_models_count: number;
  message?: string;
}

interface BatchUpdateResponse {
  updated_count: number;
  message?: string;
}

// API Key Types
interface ApiKey {
  id: string;
  name: string;
  key?: string;
  prefix?: string;
  suffix?: string;
  created_at: string;
  expires_at?: string;
  revoked_at?: string;
  last_used?: string;
  status?: "active" | "revoked" | "expired";
}

interface CreateApiKeyRequest {
  name: string;
}

interface CreateApiKeyResponse {
  id: string;
  name: string;
  key: string;
  created_at: number;
}

interface ListApiKeysResponse {
  items: ApiKey[];
}

// Usage Types
interface UsageSummary {
  model: string;
  provider: string;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_tokens: number;
  request_count: number;
  estimated_cost_usd: string;
}

interface UsagePeriod {
  start_date: string;
  end_date: string;
}

interface UsageResponse {
  period: UsagePeriod;
  total_usage: UsageSummary;
  by_model: UsageSummary[];
  by_provider: UsageSummary[];
}

interface ActivityBucket {
  bucket_time: string;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_tokens: number;
  request_count: number;
}

interface DailyAggregate {
  date: string;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_tokens: number;
  request_count: number;
  estimated_cost_usd: string;
}
