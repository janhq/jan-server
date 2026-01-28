"""Sandbox domain module."""

from .entity import (
    SandboxStatus,
    SandboxRecord,
    Workspace,
    Sandbox,
    CommandResult,
    FileInfo,
    FileContent,
    FileWriteResult,
    Screenshot,
    BrowserResult,
    InputResult,
    MCPResult,
    CodeExecuteResult,
    PackageInstallResult,
)
from .gateway import SandboxGateway
from .service import UserSandboxService

__all__ = [
    # Entities
    "SandboxStatus",
    "SandboxRecord",
    "Workspace",
    "Sandbox",
    "CommandResult",
    "FileInfo",
    "FileContent",
    "FileWriteResult",
    "Screenshot",
    "BrowserResult",
    "InputResult",
    "MCPResult",
    "CodeExecuteResult",
    "PackageInstallResult",
    # Gateway interface
    "SandboxGateway",
    # Service
    "UserSandboxService",
]
