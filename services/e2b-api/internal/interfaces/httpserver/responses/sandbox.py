"""Sandbox response DTOs."""

from pydantic import BaseModel, Field


class SandboxHealthResponse(BaseModel):
    """Response from sandbox health check."""

    sandbox_id: str = Field(description="Sandbox public ID")
    status: str = Field(description="Sandbox status (running, stopped, etc.)")
    healthy: bool = Field(description="Whether sandbox is healthy and reachable")
    ping_ms: int | None = Field(default=None, description="Ping latency in milliseconds")
    error: str | None = Field(default=None, description="Error message if unhealthy")
