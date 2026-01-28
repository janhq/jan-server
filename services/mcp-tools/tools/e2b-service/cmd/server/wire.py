"""Dependency Injection Container - Similar to wire.go in Go services.

This module wires up all dependencies at application startup.
No singletons - instances are created once and passed through.
"""

from dataclasses import dataclass

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from internal.config import settings
from internal.domain.sandbox import SandboxGateway
from internal.domain.sandbox.repository import SandboxRepository, WorkspaceRepository
from internal.infrastructure.database.database import AsyncSessionLocal
from internal.infrastructure.database.repositories import (
    PostgresSandboxRepository,
    PostgresWorkspaceRepository,
)
from internal.infrastructure.e2b import E2BGateway
from internal.infrastructure.mcp.session_store import MCPSessionStore


@dataclass
class Container:
    """DI Container holding all application dependencies.

    Similar to what Wire generates in Go services.
    Created once at startup and passed to route handlers.

    Note: UserSandboxService is created per-request because it requires
    database sessions. Use the handler's dependency injection for that.
    """

    # Shared singletons
    sandbox_gateway: SandboxGateway
    mcp_session_store: MCPSessionStore
    session_factory: async_sessionmaker[AsyncSession]

    def create_sandbox_repository(self, session: AsyncSession) -> SandboxRepository:
        """Create a sandbox repository with the given session."""
        return PostgresSandboxRepository(session)

    def create_workspace_repository(self, session: AsyncSession) -> WorkspaceRepository:
        """Create a workspace repository with the given session."""
        return PostgresWorkspaceRepository(session)


def create_container() -> Container:
    """Wire up all dependencies and return the container.

    This is the composition root - the only place where
    concrete implementations are instantiated.
    """
    # Infrastructure layer
    gateway = E2BGateway(
        api_key=settings.e2b_api_key,
        template_id=settings.e2b_template_id,
    )

    # MCP session store (in-memory cache for dynamic tools)
    mcp_session_store = MCPSessionStore(default_ttl_seconds=120.0)

    return Container(
        sandbox_gateway=gateway,
        mcp_session_store=mcp_session_store,
        session_factory=AsyncSessionLocal,
    )
