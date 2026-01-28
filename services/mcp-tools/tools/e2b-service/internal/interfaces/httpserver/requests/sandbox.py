"""Sandbox request DTOs."""

from pydantic import BaseModel, Field


class StartSandboxRequest(BaseModel):
    """Request to start a sandbox for a user."""

    timeout: int | None = Field(
        default=None,
        description="Sandbox timeout in seconds (default: 1800 / 30 min, min: 60, max: 86400)",
    )


class ExtendTimeoutRequest(BaseModel):
    """Request to extend sandbox timeout."""

    additional_seconds: int = Field(
        ge=60,
        le=86400,
        description="Additional seconds to add to timeout",
    )
