package routes

import (
	"net/http"

	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// notImplemented returns a handler that responds with 501 Not Implemented.
func notImplemented(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": name + " is not yet implemented",
		})
	}
}

// ============================================
// Agent Handlers
// ============================================

func listAgentsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Return list of available agents
		agents := []gin.H{
			{
				"id":          "default",
				"name":        "Default Assistant",
				"description": "General purpose AI assistant",
				"model_id":    "",
				"is_enabled":  true,
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"agents": agents,
		})
	}
}

func getAgentHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"id":          id,
			"name":        "Default Assistant",
			"description": "General purpose AI assistant",
			"is_enabled":  true,
		})
	}
}

func getAgentCapabilitiesHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"capabilities": []string{
				"chat",
				"function_calling",
				"code_execution",
				"web_search",
			},
		})
	}
}

func getAgentSchemaHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"input_schema":  gin.H{"type": "object"},
			"output_schema": gin.H{"type": "object"},
		})
	}
}

// ============================================
// Memory Handlers
// ============================================

func storeMemoryHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.MemoryEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "memory_disabled",
				"message": "Memory service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req struct {
			Content  string         `json:"content" binding:"required"`
			Metadata map[string]any `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: Store memory with embedding
		c.JSON(http.StatusCreated, gin.H{
			"id":      "mem_" + principal.ID[:8],
			"stored":  true,
			"message": "Memory stored successfully",
		})
	}
}

func searchMemoryHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.MemoryEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "memory_disabled",
				"message": "Memory service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req struct {
			Query string `json:"query" binding:"required"`
			Limit int    `json:"limit"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: Search memories with vector similarity
		c.JSON(http.StatusOK, gin.H{
			"memories": []interface{}{},
			"total":    0,
		})
	}
}

func listMemoriesHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.MemoryEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "memory_disabled",
				"message": "Memory service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"memories": []interface{}{},
			"total":    0,
		})
	}
}

func deleteMemoryHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.MemoryEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "memory_disabled",
				"message": "Memory service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

// ============================================
// Realtime Handlers
// ============================================

func createRealtimeSessionHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.RealtimeEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "realtime_disabled",
				"message": "Realtime service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// TODO: Create LiveKit session
		c.JSON(http.StatusCreated, gin.H{
			"session_id": "rt_session",
			"token":      "livekit_token",
			"url":        cfg.LiveKitURL,
		})
	}
}

func listRealtimeSessionsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.RealtimeEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "realtime_disabled",
				"message": "Realtime service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sessions": []interface{}{},
		})
	}
}

func getRealtimeSessionHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.RealtimeEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "realtime_disabled",
				"message": "Realtime service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"session_id": id,
			"status":     "active",
		})
	}
}

func deleteRealtimeSessionHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.RealtimeEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "realtime_disabled",
				"message": "Realtime service is not enabled",
			})
			return
		}

		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

// ============================================
// MCP Handler
// ============================================

func mcpHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// MCP JSON-RPC endpoint
		// This handles tool discovery and execution via the MCP protocol

		var req struct {
			JSONRPC string      `json:"jsonrpc"`
			ID      interface{} `json:"id"`
			Method  string      `json:"method"`
			Params  interface{} `json:"params"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"id":      nil,
				"error": gin.H{
					"code":    -32700,
					"message": "Parse error",
				},
			})
			return
		}

		// Handle MCP methods
		switch req.Method {
		case "initialize":
			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": gin.H{
					"protocolVersion": "2024-11-05",
					"capabilities": gin.H{
						"tools": gin.H{},
					},
					"serverInfo": gin.H{
						"name":    "jan-server",
						"version": config.Version,
					},
				},
			})

		case "tools/list":
			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": gin.H{
					"tools": []gin.H{
						{
							"name":        "web_search",
							"description": "Search the web for information",
							"inputSchema": gin.H{
								"type": "object",
								"properties": gin.H{
									"query": gin.H{
										"type":        "string",
										"description": "Search query",
									},
								},
								"required": []string{"query"},
							},
						},
						{
							"name":        "code_execute",
							"description": "Execute code in a sandbox",
							"inputSchema": gin.H{
								"type": "object",
								"properties": gin.H{
									"language": gin.H{
										"type":        "string",
										"description": "Programming language",
									},
									"code": gin.H{
										"type":        "string",
										"description": "Code to execute",
									},
								},
								"required": []string{"language", "code"},
							},
						},
					},
				},
			})

		case "tools/call":
			// TODO: Implement tool execution
			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": gin.H{
					"content": []gin.H{
						{
							"type": "text",
							"text": "Tool execution not yet implemented",
						},
					},
				},
			})

		default:
			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error": gin.H{
					"code":    -32601,
					"message": "Method not found",
				},
			})
		}
	}
}

// ============================================
// Admin Prompt Template Handlers
// ============================================

func adminListPromptTemplatesHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"templates": []interface{}{},
			"total":     0,
		})
	}
}

func adminCreatePromptTemplateHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string         `json:"name" binding:"required"`
			Description string         `json:"description"`
			Content     string         `json:"content" binding:"required"`
			Variables   []interface{}  `json:"variables"`
			Category    string         `json:"category"`
			IsPublic    bool           `json:"is_public"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":          "tmpl_new",
			"name":        req.Name,
			"description": req.Description,
			"content":     req.Content,
			"category":    req.Category,
			"is_public":   req.IsPublic,
		})
	}
}

func adminUpdatePromptTemplateHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":      id,
			"updated": true,
		})
	}
}

func adminDeletePromptTemplateHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

// ============================================
// Admin User Handlers
// ============================================

func adminListUsersHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement user listing with pagination
		c.JSON(http.StatusOK, gin.H{
			"users": []interface{}{},
			"total": 0,
		})
	}
}

func adminGetUserHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"id":       id,
			"email":    "user@example.com",
			"username": "user",
		})
	}
}

func adminUpdateUserHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":      id,
			"updated": true,
		})
	}
}

func adminDeleteUserHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

// ============================================
// Admin MCP Tools Handlers
// ============================================

func adminListMCPToolsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tools := []gin.H{
			{
				"id":          "web_search",
				"name":        "Web Search",
				"description": "Search the web",
				"enabled":     cfg.SerperEnabled || cfg.ExaEnabled || cfg.TavilyEnabled || cfg.SearXNGEnabled,
			},
			{
				"id":          "code_execute",
				"name":        "Code Execution",
				"description": "Execute code in sandbox",
				"enabled":     cfg.E2BAPIKey != "" || cfg.AIOSandboxURL != "",
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"tools": tools,
		})
	}
}

func adminUpdateMCPToolHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":      id,
			"updated": true,
		})
	}
}
