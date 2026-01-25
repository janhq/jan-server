import { fetchJsonWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

function buildQueryString(params: Record<string, unknown>): string {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      query.append(key, String(value));
    }
  });
  const queryString = query.toString();
  return queryString ? `?${queryString}` : "";
}

// ============================================================================
// User Management
// ============================================================================

export const userManagementService = {
  getMe: async (): Promise<UserProfile> => {
    return fetchJsonWithAuth<UserProfile>(`${JAN_API_BASE_URL}auth/me`);
  },

  listUsers: async (params?: {
    limit?: number;
    offset?: number;
    search?: string;
    enabled?: boolean;
    exclude_guests?: boolean;
  }): Promise<ListResponse<UserProfile>> => {
    const query = params ? buildQueryString(params) : "";
    return fetchJsonWithAuth<ListResponse<UserProfile>>(
      `${JAN_API_BASE_URL}v1/admin/users${query}`,
    );
  },

  createUser: async (data: {
    email: string;
    username: string;
    first_name?: string;
    last_name?: string;
    enabled?: boolean;
  }): Promise<UserProfile> => {
    return fetchJsonWithAuth<UserProfile>(`${JAN_API_BASE_URL}v1/admin/users`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  getUser: async (userId: string): Promise<UserProfile> => {
    return fetchJsonWithAuth<UserProfile>(
      `${JAN_API_BASE_URL}v1/admin/users/${userId}`,
    );
  },

  updateUser: async (
    userId: string,
    data: Partial<UserProfile>,
  ): Promise<UserProfile> => {
    return fetchJsonWithAuth<UserProfile>(
      `${JAN_API_BASE_URL}v1/admin/users/${userId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  deactivateUser: async (userId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/users/${userId}/deactivate`,
      { method: "POST" },
    );
  },

  activateUser: async (userId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/users/${userId}/activate`,
      { method: "POST" },
    );
  },

  assignAdminRole: async (userId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/users/${userId}/roles/admin`,
      { method: "POST" },
    );
  },

  // Group operations
  listGroups: async (): Promise<ListResponse<Group>> => {
    return fetchJsonWithAuth<ListResponse<Group>>(
      `${JAN_API_BASE_URL}v1/admin/groups`,
    );
  },

  createGroup: async (name: string): Promise<Group> => {
    return fetchJsonWithAuth<Group>(`${JAN_API_BASE_URL}v1/admin/groups`, {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  },

  deleteGroup: async (groupId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/groups/${groupId}`,
      { method: "DELETE" },
    );
  },

  addUserToGroup: async (userId: string, groupId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/users/${userId}/groups/${groupId}`,
      { method: "POST" },
    );
  },

  removeUserFromGroup: async (
    userId: string,
    groupId: string,
  ): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/users/${userId}/groups/${groupId}`,
      { method: "DELETE" },
    );
  },

  // Feature flags
  getGroupFeatureFlags: async (
    groupId: string,
  ): Promise<ListResponse<FeatureFlag>> => {
    return fetchJsonWithAuth<ListResponse<FeatureFlag>>(
      `${JAN_API_BASE_URL}v1/admin/groups/${groupId}/feature-flags`,
    );
  },

  setGroupFeatureFlags: async (
    groupId: string,
    flags: string[],
  ): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/groups/${groupId}/feature-flags`,
      {
        method: "PATCH",
        body: JSON.stringify({ flags }),
      },
    );
  },

  listFeatureFlags: async (): Promise<ListResponse<FeatureFlag>> => {
    return fetchJsonWithAuth<ListResponse<FeatureFlag>>(
      `${JAN_API_BASE_URL}v1/admin/feature-flags`,
    );
  },

  createFeatureFlag: async (data: {
    key: string;
    name: string;
    description?: string;
  }): Promise<FeatureFlag> => {
    return fetchJsonWithAuth<FeatureFlag>(
      `${JAN_API_BASE_URL}v1/admin/feature-flags`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    );
  },

  updateFeatureFlag: async (
    flagId: string,
    data: Partial<FeatureFlag>,
  ): Promise<FeatureFlag> => {
    return fetchJsonWithAuth<FeatureFlag>(
      `${JAN_API_BASE_URL}v1/admin/feature-flags/${flagId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  deleteFeatureFlag: async (flagId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/feature-flags/${flagId}`,
      { method: "DELETE" },
    );
  },
};

// ============================================================================
// Provider Management
// ============================================================================

export const providerManagementService = {
  listProviders: async (params?: {
    limit?: number;
    offset?: number;
    kind?: string;
    active?: boolean;
  }): Promise<ListResponse<Provider>> => {
    const query = params ? buildQueryString(params) : "";
    const response = await fetchJsonWithAuth<Provider[] | ListResponse<Provider>>(
      `${JAN_API_BASE_URL}v1/admin/providers${query}`,
    );
    // Backend may return plain array, transform to ListResponse format
    if (Array.isArray(response)) {
      return { data: response, total: response.length };
    }
    return response;
  },

  getProvider: async (publicId: string): Promise<Provider> => {
    return fetchJsonWithAuth<Provider>(
      `${JAN_API_BASE_URL}v1/admin/providers/${publicId}`,
    );
  },

  createProvider: async (data: {
    name: string;
    vendor: string;
    category?: string;
    base_url?: string;
    url?: string;
    endpoints?: Endpoint[];
    api_key?: string;
    metadata?: Record<string, string>;
    active?: boolean;
    default_provider_image_generate?: boolean;
    default_provider_image_edit?: boolean;
  }): Promise<Provider> => {
    return fetchJsonWithAuth<Provider>(
      `${JAN_API_BASE_URL}v1/admin/providers`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    );
  },

  updateProvider: async (
    publicId: string,
    data: Partial<Provider> & { url?: string; endpoints?: Endpoint[] },
  ): Promise<Provider> => {
    return fetchJsonWithAuth<Provider>(
      `${JAN_API_BASE_URL}v1/admin/providers/${publicId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  deleteProvider: async (publicId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/providers/${publicId}`,
      { method: "DELETE" },
    );
  },

  syncProviderModels: async (
    publicId: string,
    autoEnableNewModels: boolean = false,
  ): Promise<SyncProviderResponse> => {
    return fetchJsonWithAuth<SyncProviderResponse>(
      `${JAN_API_BASE_URL}v1/admin/providers/${publicId}/sync`,
      {
        method: "POST",
        body: JSON.stringify({ auto_enable_new_models: autoEnableNewModels }),
      },
    );
  },
};

// ============================================================================
// Provider Model Management
// ============================================================================

export const providerModelService = {
  listProviderModels: async (params?: {
    limit?: number;
    offset?: number;
    provider_id?: string;
    search?: string;
    active?: boolean;
    supports_images?: boolean;
  }): Promise<ListResponse<ProviderModel>> => {
    const query = params ? buildQueryString(params) : "";
    return fetchJsonWithAuth<ListResponse<ProviderModel>>(
      `${JAN_API_BASE_URL}v1/admin/models/provider-models${query}`,
    );
  },

  getProviderModel: async (publicId: string): Promise<ProviderModel> => {
    return fetchJsonWithAuth<ProviderModel>(
      `${JAN_API_BASE_URL}v1/admin/models/provider-models/${publicId}`,
    );
  },

  updateProviderModel: async (
    publicId: string,
    data: Partial<ProviderModel>,
  ): Promise<ProviderModel> => {
    return fetchJsonWithAuth<ProviderModel>(
      `${JAN_API_BASE_URL}v1/admin/models/provider-models/${publicId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  activateModel: async (publicId: string): Promise<ProviderModel> => {
    return providerModelService.updateProviderModel(publicId, { active: true });
  },

  deactivateModel: async (publicId: string): Promise<ProviderModel> => {
    return providerModelService.updateProviderModel(publicId, {
      active: false,
    });
  },

  batchUpdateActive: async (params: {
    enable: boolean;
    provider_id?: string;
    except_models?: string[];
  }): Promise<BatchUpdateResponse> => {
    return fetchJsonWithAuth<BatchUpdateResponse>(
      `${JAN_API_BASE_URL}v1/admin/models/provider-models/bulk-toggle`,
      {
        method: "POST",
        body: JSON.stringify(params),
      },
    );
  },
};

// ============================================================================
// Model Catalog Management
// ============================================================================

export const modelCatalogService = {
  listModelCatalogs: async (params?: {
    limit?: number;
    offset?: number;
    family?: string;
    status?: string;
    is_moderated?: boolean;
    active?: boolean;
    supports_embeddings?: boolean;
    supports_images?: boolean;
    supports_reasoning?: boolean;
    supports_audio?: boolean;
    supports_video?: boolean;
    supports_tools?: boolean;
    supports_browser?: boolean;
    experimental?: boolean;
    requires_feature_flag?: string;
  }): Promise<ListResponse<ModelCatalog>> => {
    const query = params ? buildQueryString(params) : "";
    return fetchJsonWithAuth<ListResponse<ModelCatalog>>(
      `${JAN_API_BASE_URL}v1/admin/models/catalogs${query}`,
    );
  },

  getModelCatalog: async (publicId: string): Promise<ModelCatalog> => {
    return fetchJsonWithAuth<ModelCatalog>(
      `${JAN_API_BASE_URL}v1/admin/models/catalogs/${publicId}`,
    );
  },

  updateModelCatalog: async (
    publicId: string,
    data: Partial<ModelCatalog>,
  ): Promise<ModelCatalog> => {
    return fetchJsonWithAuth<ModelCatalog>(
      `${JAN_API_BASE_URL}v1/admin/models/catalogs/${publicId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  batchToggle: async (params: {
    enable: boolean;
    catalog_ids?: string[];
  }): Promise<BatchUpdateResponse> => {
    return fetchJsonWithAuth<BatchUpdateResponse>(
      `${JAN_API_BASE_URL}v1/admin/models/catalogs/bulk-toggle`,
      {
        method: "POST",
        body: JSON.stringify(params),
      },
    );
  },

  // Model Prompt Templates
  listModelPromptTemplates: async (
    modelId: string,
  ): Promise<ListResponse<ModelPromptTemplate>> => {
    return fetchJsonWithAuth<ListResponse<ModelPromptTemplate>>(
      `${JAN_API_BASE_URL}v1/admin/models/prompt-templates/list/${modelId}`,
    );
  },

  assignPromptTemplate: async (
    modelId: string,
    data: AssignTemplateRequest,
  ): Promise<ModelPromptTemplate> => {
    return fetchJsonWithAuth<ModelPromptTemplate>(
      `${JAN_API_BASE_URL}v1/admin/models/prompt-templates/assign/${modelId}`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    );
  },

  updatePromptTemplateAssignment: async (
    modelId: string,
    templateKey: string,
    data: UpdateAssignmentRequest,
  ): Promise<ModelPromptTemplate> => {
    return fetchJsonWithAuth<ModelPromptTemplate>(
      `${JAN_API_BASE_URL}v1/admin/models/prompt-templates/update/${encodeURIComponent(templateKey)}/${modelId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  unassignPromptTemplate: async (
    modelId: string,
    templateKey: string,
  ): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/models/prompt-templates/unassign/${encodeURIComponent(templateKey)}/${modelId}`,
      { method: "DELETE" },
    );
  },

  getEffectiveTemplates: async (
    modelId: string,
  ): Promise<EffectiveTemplatesResponse> => {
    return fetchJsonWithAuth<EffectiveTemplatesResponse>(
      `${JAN_API_BASE_URL}v1/admin/models/prompt-templates/effective/${modelId}`,
    );
  },
};

// ============================================================================
// Prompt Template Management
// ============================================================================

export const promptTemplateService = {
  listPromptTemplates: async (params?: {
    limit?: number;
    offset?: number;
    category?: string;
    is_active?: boolean;
    is_system?: boolean;
    search?: string;
  }): Promise<ListResponse<PromptTemplate>> => {
    const query = params ? buildQueryString(params) : "";
    return fetchJsonWithAuth<ListResponse<PromptTemplate>>(
      `${JAN_API_BASE_URL}v1/admin/prompt-templates${query}`,
    );
  },

  getPromptTemplate: async (publicId: string): Promise<PromptTemplate> => {
    return fetchJsonWithAuth<PromptTemplate>(
      `${JAN_API_BASE_URL}v1/admin/prompt-templates/${publicId}`,
    );
  },

  getPromptTemplateByKey: async (
    templateKey: string,
  ): Promise<PromptTemplate> => {
    return fetchJsonWithAuth<PromptTemplate>(
      `${JAN_API_BASE_URL}v1/prompt-templates/${templateKey}`,
    );
  },

  createPromptTemplate: async (
    data: CreatePromptTemplateRequest,
  ): Promise<PromptTemplate> => {
    return fetchJsonWithAuth<PromptTemplate>(
      `${JAN_API_BASE_URL}v1/admin/prompt-templates`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    );
  },

  updatePromptTemplate: async (
    publicId: string,
    data: UpdatePromptTemplateRequest,
  ): Promise<PromptTemplate> => {
    return fetchJsonWithAuth<PromptTemplate>(
      `${JAN_API_BASE_URL}v1/admin/prompt-templates/${publicId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  deletePromptTemplate: async (publicId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}v1/admin/prompt-templates/${publicId}`,
      { method: "DELETE" },
    );
  },

  duplicatePromptTemplate: async (
    publicId: string,
    data: DuplicatePromptTemplateRequest,
  ): Promise<PromptTemplate> => {
    return fetchJsonWithAuth<PromptTemplate>(
      `${JAN_API_BASE_URL}v1/admin/prompt-templates/${publicId}/duplicate`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    );
  },

  activateTemplate: async (publicId: string): Promise<PromptTemplate> => {
    return promptTemplateService.updatePromptTemplate(publicId, {
      is_active: true,
    });
  },

  deactivateTemplate: async (publicId: string): Promise<PromptTemplate> => {
    return promptTemplateService.updatePromptTemplate(publicId, {
      is_active: false,
    });
  },
};

// ============================================================================
// MCP Tool Management
// ============================================================================

export const mcpToolService = {
  listMCPTools: async (params?: {
    limit?: number;
    offset?: number;
    category?: string;
    is_active?: boolean;
    search?: string;
  }): Promise<ListResponse<MCPTool>> => {
    const query = params ? buildQueryString(params) : "";
    return fetchJsonWithAuth<ListResponse<MCPTool>>(
      `${JAN_API_BASE_URL}v1/admin/mcp-tools${query}`,
    );
  },

  getMCPTool: async (publicId: string): Promise<MCPTool> => {
    return fetchJsonWithAuth<MCPTool>(
      `${JAN_API_BASE_URL}v1/admin/mcp-tools/${publicId}`,
    );
  },

  updateMCPTool: async (
    publicId: string,
    data: UpdateMCPToolRequest,
  ): Promise<MCPTool> => {
    return fetchJsonWithAuth<MCPTool>(
      `${JAN_API_BASE_URL}v1/admin/mcp-tools/${publicId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  activateTool: async (publicId: string): Promise<MCPTool> => {
    return mcpToolService.updateMCPTool(publicId, { is_active: true });
  },

  deactivateTool: async (publicId: string): Promise<MCPTool> => {
    return mcpToolService.updateMCPTool(publicId, { is_active: false });
  },
};

// ============================================================================
// API Key Management
// ============================================================================

export const apiKeyService = {
  listApiKeys: async (): Promise<ListApiKeysResponse> => {
    return fetchJsonWithAuth<ListApiKeysResponse>(
      `${JAN_API_BASE_URL}auth/api-keys`,
    );
  },

  createApiKey: async (
    data: CreateApiKeyRequest,
  ): Promise<CreateApiKeyResponse> => {
    return fetchJsonWithAuth<CreateApiKeyResponse>(
      `${JAN_API_BASE_URL}auth/api-keys`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    );
  },

  deleteApiKey: async (keyId: string): Promise<void> => {
    return fetchJsonWithAuth<void>(
      `${JAN_API_BASE_URL}auth/api-keys/${keyId}`,
      { method: "DELETE" },
    );
  },
};

// ============================================================================
// Admin Check
// ============================================================================

export const adminService = {
  checkIsAdmin: async (): Promise<boolean> => {
    try {
      const profile = await userManagementService.getMe();
      return (
        profile.is_admin === true ||
        profile.role === "admin" ||
        (profile.roles && profile.roles.includes("admin")) ||
        false
      );
    } catch (error) {
      console.error("Failed to check admin status:", error);
      return false;
    }
  },
};
