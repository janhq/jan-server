"""MCP response DTOs for JSON-RPC 2.0 protocol."""

from typing import Any

from pydantic import BaseModel, Field


class MCPToolInput(BaseModel):
    """Tool input schema definition."""

    type: str = Field(default="object")
    properties: dict[str, Any] = Field(default_factory=dict)
    required: list[str] = Field(default_factory=list)


class MCPTool(BaseModel):
    """MCP tool definition."""

    name: str = Field(description="Tool name")
    description: str = Field(description="Tool description")
    inputSchema: MCPToolInput = Field(description="Tool input schema")


class MCPInitializeResult(BaseModel):
    """Result of MCP initialize call."""

    protocolVersion: str = Field(default="2024-11-05")
    serverInfo: dict[str, str] = Field(
        default_factory=lambda: {"name": "e2b-sandbox", "version": "1.0.0"}
    )
    capabilities: dict[str, Any] = Field(
        default_factory=lambda: {"tools": {"listChanged": True}}
    )


class MCPToolsListResult(BaseModel):
    """Result of MCP tools/list call."""

    tools: list[MCPTool] = Field(default_factory=list)


class MCPToolCallResult(BaseModel):
    """Result of MCP tools/call."""

    content: list[dict[str, Any]] = Field(default_factory=list)
    isError: bool = Field(default=False)


class MCPError(BaseModel):
    """MCP JSON-RPC error."""

    code: int
    message: str
    data: Any = None


class MCPResponse(BaseModel):
    """Generic MCP JSON-RPC 2.0 response.

    Per JSON-RPC 2.0 spec, a response MUST have either `result` or `error`, not both.
    """

    jsonrpc: str = Field(default="2.0")
    id: int | str | None = None
    result: Any = Field(default=None)
    error: MCPError | None = Field(default=None)

    def model_dump(self, **kwargs) -> dict[str, Any]:
        """Override to exclude None values (result/error mutual exclusion)."""
        kwargs.setdefault("exclude_none", True)
        data = super().model_dump(**kwargs)
        # Ensure id is always present (even if None) per JSON-RPC spec
        if "id" not in data:
            data["id"] = self.id
        return data


class MCPRequest(BaseModel):
    """MCP JSON-RPC 2.0 request."""

    jsonrpc: str = Field(default="2.0")
    id: int | str | None = None
    method: str
    params: dict[str, Any] | None = None
