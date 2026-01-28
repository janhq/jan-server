"""Workspace response DTOs."""

from datetime import datetime

from pydantic import BaseModel, Field


class WorkspaceResponse(BaseModel):
    """Response containing workspace information."""

    public_id: str = Field(description="Workspace public ID (e.g., ws_xxx)")
    conversation_id: str = Field(description="Conversation ID")
    workspace_path: str = Field(description="Workspace path in sandbox (e.g., /workspace/conv_xxx/)")
    created_at: datetime | None = Field(default=None, description="When workspace was created")


class SandboxRecordResponse(BaseModel):
    """Response containing sandbox record information."""

    public_id: str = Field(description="Sandbox public ID (e.g., sb_xxx)")
    user_id: str = Field(description="User who owns this sandbox")
    e2b_sandbox_id: str | None = Field(default=None, description="E2B platform sandbox ID")
    status: str = Field(description="Sandbox status (created, running, paused, stopped, error)")
    view_url: str | None = Field(default=None, description="View-only VNC URL with auth token")
    control_url: str | None = Field(default=None, description="Full control VNC URL with auth token")
    sandbox_url: str | None = Field(default=None, description="External sandbox URL for MCP proxy")
    started_at: datetime | None = Field(default=None, description="When sandbox was started")
    timeout_at: datetime | None = Field(default=None, description="When sandbox will auto-stop")
    paused_at: datetime | None = Field(default=None, description="When sandbox was paused")
    pause_expires_at: datetime | None = Field(default=None, description="When pause expires")
    error_message: str | None = Field(default=None, description="Error details if status is error")
    created_at: datetime | None = Field(default=None, description="When sandbox was created")


class WorkspaceWithSandboxResponse(BaseModel):
    """Response containing workspace and its parent sandbox."""

    sandbox: SandboxRecordResponse
    workspace: WorkspaceResponse
