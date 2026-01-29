"""Sandbox domain entities."""

from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from typing import Any


class SandboxStatus(str, Enum):
    """Sandbox status enumeration."""

    CREATED = "created"  # Initial state, not yet started
    RUNNING = "running"  # Actively running
    PAUSED = "paused"  # Paused (can be resumed within 30 days)
    STOPPED = "stopped"  # Stopped (needs restart)
    ERROR = "error"  # Error state
    NOT_FOUND = "not_found"  # Not found (for API responses)

    def can_pause(self) -> bool:
        """Check if sandbox can be paused."""
        return self == SandboxStatus.RUNNING


@dataclass
class SandboxRecord:
    """Database-backed sandbox record with lifecycle tracking."""

    id: int | None = None
    public_id: str = ""
    user_id: str = ""
    e2b_sandbox_id: str | None = None
    status: SandboxStatus = SandboxStatus.CREATED
    view_url: str | None = None  # View-only VNC URL (no interaction)
    control_url: str | None = None  # Full control VNC URL (takeover)
    sandbox_url: str | None = None  # External URL for MCP proxy access
    started_at: datetime | None = None
    timeout_at: datetime | None = None  # When sandbox auto-stops (24h max)
    paused_at: datetime | None = None
    pause_expires_at: datetime | None = None  # When paused sandbox expires (30 days)
    error_message: str | None = None
    created_at: datetime | None = None
    updated_at: datetime | None = None


@dataclass
class Workspace:
    """Conversation-scoped workspace within a sandbox."""

    id: int | None = None
    public_id: str = ""
    sandbox_id: int = 0
    conversation_id: str = ""
    workspace_path: str = ""  # e.g., /workspace/conv_xxx/
    created_at: datetime | None = None
    updated_at: datetime | None = None


@dataclass
class Sandbox:
    """Sandbox entity representing an E2B sandbox instance (runtime state)."""

    sandbox_id: str
    status: SandboxStatus
    view_url: str | None = None  # View-only VNC URL (no interaction)
    control_url: str | None = None  # Full control VNC URL (takeover)
    sandbox_url: str | None = None  # External URL for MCP proxy access
    started_at: datetime | None = None  # When sandbox started (from E2B)
    end_at: datetime | None = None  # When sandbox will timeout (from E2B)


@dataclass
class CommandResult:
    """Result of a shell command execution."""

    stdout: str | None = None
    stderr: str | None = None
    exit_code: int | None = None
    error: str | None = None

    def is_success(self) -> bool:
        """Check if command executed successfully."""
        return self.error is None and (self.exit_code is None or self.exit_code == 0)


@dataclass
class FileInfo:
    """Information about a file in the sandbox."""

    name: str
    is_dir: bool
    size: int = 0


@dataclass
class FileContent:
    """Content of a file read from the sandbox."""

    path: str
    content: str
    encoding: str = "base64"


@dataclass
class FileWriteResult:
    """Result of a file write operation."""

    path: str
    status: str
    bytes_written: int = 0


@dataclass
class Screenshot:
    """Screenshot of the sandbox desktop."""

    image: str  # base64 encoded
    format: str = "png"
    encoding: str = "base64"


@dataclass
class BrowserResult:
    """Result of a browser operation."""

    url: str | None = None
    status: str = ""


@dataclass
class InputResult:
    """Result of an input operation (click, type)."""

    status: str
    x: int | None = None
    y: int | None = None
    button: str | None = None
    text: str | None = None


@dataclass
class MCPResult:
    """Result of an MCP call."""

    result: Any | None = None
    error: dict[str, Any] | None = None


@dataclass
class CodeExecuteResult:
    """Result of code execution."""

    stdout: str = ""
    stderr: str = ""
    exit_code: int = 0
    status: str = "ok"  # "ok" or "error"
    error: str | None = None
    duration_ms: int = 0

    def is_success(self) -> bool:
        """Check if code executed successfully."""
        return self.status == "ok" and self.exit_code == 0 and self.error is None


@dataclass
class PackageInstallResult:
    """Result of package installation (pip or npm)."""

    packages: list[str]
    package_manager: str = "pip"  # "pip" or "npm"
    status: str = "success"  # "success" or "error"
    output: str = ""  # pip or npm output
    exit_code: int = 0
    error: str | None = None
    duration_ms: int = 0

    def is_success(self) -> bool:
        """Check if installation succeeded."""
        return self.status == "success" and self.exit_code == 0
