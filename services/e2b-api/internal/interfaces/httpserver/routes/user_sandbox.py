"""User sandbox routes - User-centric HTTP endpoints."""

from collections.abc import AsyncGenerator
from typing import Annotated

from fastapi import APIRouter, Depends, Request
from sqlalchemy.ext.asyncio import AsyncSession

from internal.domain.sandbox.service import UserSandboxService
from internal.interfaces.httpserver.handlers.user_sandbox import UserSandboxHandler
from internal.interfaces.httpserver.requests.sandbox import (
    ExtendTimeoutRequest,
    StartSandboxRequest,
)
from internal.interfaces.httpserver.responses.workspace import (
    SandboxRecordResponse,
    WorkspaceWithSandboxResponse,
)
from internal.interfaces.httpserver.responses.sandbox import SandboxHealthResponse
from internal.interfaces.httpserver.responses.mcp import MCPRequest, MCPResponse


async def get_db_session(request: Request) -> AsyncGenerator[AsyncSession, None]:
    """Get database session for request."""
    container = request.app.state.container
    async with container.session_factory() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise


async def get_handler(
    request: Request, session: AsyncSession = Depends(get_db_session)
) -> UserSandboxHandler:
    """Get user sandbox handler with database session."""
    container = request.app.state.container
    sandbox_repo = container.create_sandbox_repository(session)
    workspace_repo = container.create_workspace_repository(session)
    service = UserSandboxService(
        gateway=container.sandbox_gateway,
        sandbox_repo=sandbox_repo,
        workspace_repo=workspace_repo,
    )
    return UserSandboxHandler(
        service=service,
        gateway=container.sandbox_gateway,
        session_store=container.mcp_session_store,
    )


HandlerDep = Annotated[UserSandboxHandler, Depends(get_handler)]


def create_user_sandbox_router() -> APIRouter:
    """Create and configure user sandbox router."""
    router = APIRouter(prefix="/users", tags=["user-sandboxes"])

    @router.get("/{user_id}/sandbox", response_model=SandboxRecordResponse)
    async def get_sandbox(
        user_id: str,
        handler: HandlerDep,
    ) -> SandboxRecordResponse:
        """Get sandbox for a user."""
        return await handler.get_sandbox(user_id)

    @router.post("/{user_id}/sandbox", response_model=SandboxRecordResponse)
    async def start_sandbox(
        user_id: str,
        handler: HandlerDep,
        request: StartSandboxRequest | None = None,
    ) -> SandboxRecordResponse:
        """Create/start/resume a sandbox for a user.

        Smart start that handles all cases:
        - No sandbox exists → creates new
        - Stopped → starts fresh E2B sandbox
        - Paused → resumes
        - Running → returns current
        """
        return await handler.start_sandbox(user_id, request)

    @router.delete("/{user_id}/sandbox", response_model=SandboxRecordResponse)
    async def delete_sandbox(
        user_id: str,
        handler: HandlerDep,
    ) -> SandboxRecordResponse:
        """Delete a user's sandbox (stops E2B and removes DB record)."""
        return await handler.delete_sandbox(user_id)

    @router.post("/{user_id}/sandbox/pause", response_model=SandboxRecordResponse)
    async def pause_sandbox(
        user_id: str,
        handler: HandlerDep,
    ) -> SandboxRecordResponse:
        """Pause a user's running sandbox.

        Paused sandboxes can be resumed within 30 days using POST /sandbox.
        """
        return await handler.pause_sandbox(user_id)

    @router.post("/{user_id}/sandbox/extend", response_model=SandboxRecordResponse)
    async def extend_timeout(
        user_id: str,
        request: ExtendTimeoutRequest,
        handler: HandlerDep,
    ) -> SandboxRecordResponse:
        """Extend timeout for a running sandbox.

        Max runtime is 24 hours from start.
        """
        return await handler.extend_timeout(user_id, request)

    @router.post(
        "/{user_id}/sandbox/workspace/{conversation_id}",
        response_model=WorkspaceWithSandboxResponse,
    )
    async def get_or_create_workspace(
        user_id: str,
        conversation_id: str,
        handler: HandlerDep,
    ) -> WorkspaceWithSandboxResponse:
        """Get or create workspace for a conversation.

        This is the main entry point for workspace operations.
        - Creates a sandbox for the user if one doesn't exist
        - Starts the sandbox if it's stopped
        - Creates a workspace for the conversation if it doesn't exist
        - Returns the workspace info with its parent sandbox
        """
        return await handler.get_or_create_workspace(user_id, conversation_id)

    @router.get("/{user_id}/sandbox/health", response_model=SandboxHealthResponse)
    async def sandbox_health(
        user_id: str,
        handler: HandlerDep,
    ) -> SandboxHealthResponse:
        """Check health of a user's sandbox (ping to sandbox).

        Returns health status including whether sandbox is reachable.
        """
        return await handler.sandbox_health(user_id)

    @router.post(
        "/{user_id}/sandbox/mcp",
        response_model=MCPResponse,
        response_model_exclude_none=True,
    )
    async def mcp_endpoint(
        user_id: str,
        mcp_request: MCPRequest,
        handler: HandlerDep,
    ) -> MCPResponse:
        """MCP JSON-RPC 2.0 endpoint for a user's sandbox.

        Supports:
        - initialize: MCP protocol handshake
        - tools/list: List available tools (static E2B + dynamic from sandbox)
        - tools/call: Execute a tool (static or dynamic)

        Example request:
        ```json
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "tools/list"
        }
        ```
        """
        return await handler.handle_mcp_request(user_id, mcp_request)

    return router
