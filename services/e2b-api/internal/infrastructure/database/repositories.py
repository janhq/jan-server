"""Concrete repository implementations using PostgreSQL."""

import secrets

from sqlalchemy import and_, select
from sqlalchemy.ext.asyncio import AsyncSession

from internal.domain.sandbox.entity import SandboxRecord, SandboxStatus, Workspace
from internal.domain.sandbox.repository import SandboxRepository, WorkspaceRepository

from .models import SandboxModel, WorkspaceModel


def _generate_public_id(prefix: str) -> str:
    """Generate a public ID with prefix (e.g., sb_xxx or ws_xxx)."""
    return f"{prefix}_{secrets.token_hex(12)}"


def _sandbox_model_to_entity(model: SandboxModel) -> SandboxRecord:
    """Convert SQLAlchemy model to domain entity."""
    return SandboxRecord(
        id=model.id,
        public_id=model.public_id,
        user_id=model.user_id,
        e2b_sandbox_id=model.e2b_sandbox_id,
        status=SandboxStatus(model.status),
        view_url=model.view_url,
        control_url=model.control_url,
        sandbox_url=model.sandbox_url,
        started_at=model.started_at,
        timeout_at=model.timeout_at,
        paused_at=model.paused_at,
        pause_expires_at=model.pause_expires_at,
        error_message=model.error_message,
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


def _workspace_model_to_entity(model: WorkspaceModel) -> Workspace:
    """Convert SQLAlchemy model to domain entity."""
    return Workspace(
        id=model.id,
        public_id=model.public_id,
        sandbox_id=model.sandbox_id,
        conversation_id=model.conversation_id,
        workspace_path=model.workspace_path,
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


class PostgresSandboxRepository(SandboxRepository):
    """PostgreSQL implementation of SandboxRepository."""

    def __init__(self, session: AsyncSession):
        self._session = session

    async def create(self, sandbox: SandboxRecord) -> SandboxRecord:
        """Create a new sandbox record."""
        model = SandboxModel(
            public_id=sandbox.public_id or _generate_public_id("sb"),
            user_id=sandbox.user_id,
            e2b_sandbox_id=sandbox.e2b_sandbox_id,
            status=sandbox.status.value,
            view_url=sandbox.view_url,
            control_url=sandbox.control_url,
            sandbox_url=sandbox.sandbox_url,
            started_at=sandbox.started_at,
            timeout_at=sandbox.timeout_at,
            paused_at=sandbox.paused_at,
            pause_expires_at=sandbox.pause_expires_at,
            error_message=sandbox.error_message,
        )
        self._session.add(model)
        await self._session.flush()
        await self._session.refresh(model)
        return _sandbox_model_to_entity(model)

    async def get_by_public_id(self, public_id: str) -> SandboxRecord | None:
        """Get sandbox by public ID."""
        result = await self._session.execute(
            select(SandboxModel).where(SandboxModel.public_id == public_id)
        )
        model = result.scalar_one_or_none()
        return _sandbox_model_to_entity(model) if model else None

    async def get_by_user_id(self, user_id: str) -> SandboxRecord | None:
        """Get sandbox by user ID (one sandbox per user)."""
        result = await self._session.execute(
            select(SandboxModel).where(SandboxModel.user_id == user_id)
        )
        model = result.scalar_one_or_none()
        return _sandbox_model_to_entity(model) if model else None

    async def update(self, sandbox: SandboxRecord) -> SandboxRecord:
        """Update an existing sandbox record."""
        result = await self._session.execute(
            select(SandboxModel).where(SandboxModel.id == sandbox.id)
        )
        model = result.scalar_one_or_none()
        if not model:
            raise ValueError(f"Sandbox with id {sandbox.id} not found")

        model.e2b_sandbox_id = sandbox.e2b_sandbox_id
        model.status = sandbox.status.value
        model.view_url = sandbox.view_url
        model.control_url = sandbox.control_url
        model.sandbox_url = sandbox.sandbox_url
        model.started_at = sandbox.started_at
        model.timeout_at = sandbox.timeout_at
        model.paused_at = sandbox.paused_at
        model.pause_expires_at = sandbox.pause_expires_at
        model.error_message = sandbox.error_message

        await self._session.flush()
        await self._session.refresh(model)
        return _sandbox_model_to_entity(model)

    async def delete(self, sandbox_id: int) -> bool:
        """Delete a sandbox record by internal ID."""
        result = await self._session.execute(
            select(SandboxModel).where(SandboxModel.id == sandbox_id)
        )
        model = result.scalar_one_or_none()
        if not model:
            return False
        await self._session.delete(model)
        await self._session.flush()
        return True


class PostgresWorkspaceRepository(WorkspaceRepository):
    """PostgreSQL implementation of WorkspaceRepository."""

    def __init__(self, session: AsyncSession):
        self._session = session

    async def create(self, workspace: Workspace) -> Workspace:
        """Create a new workspace record."""
        model = WorkspaceModel(
            public_id=workspace.public_id or _generate_public_id("ws"),
            sandbox_id=workspace.sandbox_id,
            conversation_id=workspace.conversation_id,
            workspace_path=workspace.workspace_path,
        )
        self._session.add(model)
        await self._session.flush()
        await self._session.refresh(model)
        return _workspace_model_to_entity(model)

    async def get_by_conversation_id(
        self, sandbox_id: int, conversation_id: str
    ) -> Workspace | None:
        """Get workspace by conversation ID within a sandbox."""
        result = await self._session.execute(
            select(WorkspaceModel).where(
                and_(
                    WorkspaceModel.sandbox_id == sandbox_id,
                    WorkspaceModel.conversation_id == conversation_id,
                )
            )
        )
        model = result.scalar_one_or_none()
        return _workspace_model_to_entity(model) if model else None
