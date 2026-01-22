"""Repository interfaces for sandbox domain."""

from abc import ABC, abstractmethod

from .entity import SandboxRecord, Workspace


class SandboxRepository(ABC):
    """Abstract repository interface for sandbox persistence."""

    @abstractmethod
    async def create(self, sandbox: SandboxRecord) -> SandboxRecord:
        """Create a new sandbox record."""
        ...

    @abstractmethod
    async def get_by_public_id(self, public_id: str) -> SandboxRecord | None:
        """Get sandbox by public ID (e.g., sb_xxx)."""
        ...

    @abstractmethod
    async def get_by_user_id(self, user_id: str) -> SandboxRecord | None:
        """Get sandbox by user ID (one sandbox per user)."""
        ...

    @abstractmethod
    async def update(self, sandbox: SandboxRecord) -> SandboxRecord:
        """Update an existing sandbox record."""
        ...

    @abstractmethod
    async def delete(self, sandbox_id: int) -> bool:
        """Delete a sandbox record by internal ID."""
        ...


class WorkspaceRepository(ABC):
    """Abstract repository interface for workspace persistence."""

    @abstractmethod
    async def create(self, workspace: Workspace) -> Workspace:
        """Create a new workspace record."""
        ...

    @abstractmethod
    async def get_by_conversation_id(
        self, sandbox_id: int, conversation_id: str
    ) -> Workspace | None:
        """Get workspace by conversation ID within a sandbox."""
        ...
