"""Sandbox domain service - Business logic layer."""

import logging
from datetime import datetime, timedelta, timezone

from internal.config import settings

logger = logging.getLogger(__name__)
from internal.domain.exceptions import (
    SandboxNotFoundError,
    SandboxStateError,
)

from .entity import (
    SandboxRecord,
    SandboxStatus,
    Workspace,
)
from .gateway import SandboxGateway
from .repository import SandboxRepository, WorkspaceRepository


class UserSandboxService:
    """User-centric sandbox service with database persistence.

    This service manages sandbox lifecycle with database tracking
    and provides user-centric operations (one sandbox per user).
    """

    # Business rule constants
    MIN_TIMEOUT = 60  # 1 minute
    MAX_TIMEOUT = settings.sandbox_max_runtime  # 24 hours (E2B limit)
    DEFAULT_TIMEOUT = settings.e2b_sandbox_timeout  # 30 minutes default
    MAX_PAUSE_DURATION = settings.sandbox_max_pause  # 30 days (E2B limit)

    def __init__(
        self,
        gateway: SandboxGateway,
        sandbox_repo: SandboxRepository,
        workspace_repo: WorkspaceRepository,
    ):
        self._gateway = gateway
        self._sandbox_repo = sandbox_repo
        self._workspace_repo = workspace_repo

    async def sync_sandbox_state(self, sandbox: SandboxRecord) -> SandboxRecord:
        """Validate actual E2B state and sync DB if needed.

        Call this before operations that require running sandbox.
        Also syncs timestamps (started_at, timeout_at) from E2B to keep DB accurate.
        Returns the updated sandbox record (or original if no sync needed).
        """
        if not sandbox.e2b_sandbox_id:
            return sandbox

        if sandbox.status not in (SandboxStatus.RUNNING, SandboxStatus.PAUSED):
            return sandbox

        # Get actual sandbox info from E2B (includes alive check + timestamps)
        e2b_info = self._gateway.get_sandbox_info(sandbox.e2b_sandbox_id)

        if not e2b_info and sandbox.status == SandboxStatus.RUNNING:
            # Sandbox died (timeout, error, etc.) - update DB
            logger.info(
                f"Sandbox {sandbox.public_id} ({sandbox.e2b_sandbox_id}) is unreachable, "
                f"syncing status from RUNNING to STOPPED"
            )
            sandbox.status = SandboxStatus.STOPPED
            sandbox.view_url = None
            sandbox.control_url = None
            sandbox.sandbox_url = None
            sandbox.error_message = "Sandbox became unreachable"
            sandbox = await self._sandbox_repo.update(sandbox)
        elif e2b_info and sandbox.status == SandboxStatus.RUNNING:
            # Sandbox is alive - sync timestamps from E2B to DB
            needs_update = False

            if e2b_info.started_at and e2b_info.started_at != sandbox.started_at:
                sandbox.started_at = e2b_info.started_at
                needs_update = True

            if e2b_info.end_at and e2b_info.end_at != sandbox.timeout_at:
                sandbox.timeout_at = e2b_info.end_at
                needs_update = True

            if needs_update:
                logger.info(
                    f"Synced timestamps for {sandbox.public_id}: "
                    f"started_at={sandbox.started_at}, timeout_at={sandbox.timeout_at}"
                )
                sandbox = await self._sandbox_repo.update(sandbox)

        return sandbox

    async def get_sandbox_by_user(self, user_id: str) -> SandboxRecord | None:
        """Get sandbox for a user."""
        return await self._sandbox_repo.get_by_user_id(user_id)

    async def get_sandbox_by_public_id(self, public_id: str) -> SandboxRecord | None:
        """Get sandbox by public ID."""
        return await self._sandbox_repo.get_by_public_id(public_id)

    async def start_sandbox(
        self, user_id: str, timeout: int | None = None
    ) -> SandboxRecord:
        """Smart start a sandbox for a user.

        Handles all cases:
        - No sandbox exists → create new
        - Stopped → start new E2B sandbox
        - Paused → resume
        - Running → return current (after sync verification)
        """
        # Validate timeout - treat None or <= 0 as "use default"
        if timeout is None or timeout <= 0:
            timeout = self.DEFAULT_TIMEOUT
        timeout = max(self.MIN_TIMEOUT, min(timeout, self.MAX_TIMEOUT))

        # Check if user already has a sandbox
        sandbox = await self._sandbox_repo.get_by_user_id(user_id)

        if sandbox is not None:
            # Sync actual E2B state before checking status
            sandbox = await self.sync_sandbox_state(sandbox)

        if sandbox is None:
            # Create new sandbox record
            sandbox = SandboxRecord(user_id=user_id, status=SandboxStatus.CREATED)
            sandbox = await self._sandbox_repo.create(sandbox)

        # Handle based on current status
        if sandbox.status == SandboxStatus.RUNNING:
            return sandbox  # Already running (verified by sync)
        elif sandbox.status == SandboxStatus.PAUSED:
            return await self._resume_paused_sandbox(sandbox, timeout)
        elif sandbox.status in (SandboxStatus.CREATED, SandboxStatus.STOPPED):
            return await self._start_new_e2b_sandbox(sandbox, timeout)
        else:
            raise ValueError(f"Cannot start sandbox in status: {sandbox.status}")

    async def _resume_paused_sandbox(
        self, sandbox: SandboxRecord, timeout: int
    ) -> SandboxRecord:
        """Resume a paused sandbox (internal helper)."""
        # Check if pause has expired
        now = datetime.now(timezone.utc)
        if sandbox.pause_expires_at and now >= sandbox.pause_expires_at:
            # Pause expired, need to start fresh
            logger.info(f"Sandbox {sandbox.public_id} pause expired, starting fresh")
            sandbox.status = SandboxStatus.STOPPED
            sandbox.error_message = "Pause duration expired"
            sandbox = await self._sandbox_repo.update(sandbox)
            return await self._start_new_e2b_sandbox(sandbox, timeout)

        # Actually resume the sandbox on E2B with proper timeout
        if not sandbox.e2b_sandbox_id:
            raise RuntimeError("No E2B sandbox ID to resume")

        resumed = self._gateway.resume_sandbox(sandbox.e2b_sandbox_id, timeout)
        if not resumed:
            # Resume failed, maybe sandbox expired on E2B side - start fresh
            logger.warning(f"Failed to resume {sandbox.public_id}, starting fresh")
            sandbox.status = SandboxStatus.STOPPED
            sandbox.error_message = "Resume failed"
            sandbox = await self._sandbox_repo.update(sandbox)
            return await self._start_new_e2b_sandbox(sandbox, timeout)

        # Update with actual E2B timestamps
        sandbox.status = SandboxStatus.RUNNING
        sandbox.started_at = resumed.started_at or now
        sandbox.timeout_at = resumed.end_at or (now + timedelta(seconds=timeout))
        sandbox.paused_at = None
        sandbox.pause_expires_at = None
        sandbox.error_message = None

        logger.info(
            f"Resumed paused sandbox {sandbox.public_id}, "
            f"timeout_at={sandbox.timeout_at}"
        )
        return await self._sandbox_repo.update(sandbox)

    async def _start_new_e2b_sandbox(
        self, sandbox: SandboxRecord, timeout: int
    ) -> SandboxRecord:
        """Start or reconnect to an E2B sandbox instance (internal helper).

        If an existing e2b_sandbox_id is present and still alive, reconnect to it.
        Otherwise, create a new E2B sandbox.
        """
        now = datetime.now(timezone.utc)

        # Check if existing sandbox is still alive
        if sandbox.e2b_sandbox_id:
            e2b_info = self._gateway.get_sandbox_info(sandbox.e2b_sandbox_id)
            if e2b_info:
                logger.info(
                    f"Reconnecting to existing E2B sandbox {sandbox.e2b_sandbox_id} "
                    f"for {sandbox.public_id}"
                )
                # Sandbox still alive, update status with actual E2B timestamps
                sandbox.status = SandboxStatus.RUNNING
                sandbox.started_at = e2b_info.started_at or now
                sandbox.timeout_at = e2b_info.end_at or (now + timedelta(seconds=timeout))
                sandbox.paused_at = None
                sandbox.pause_expires_at = None
                sandbox.error_message = None
                logger.info(
                    f"Reconnected to {sandbox.e2b_sandbox_id}: "
                    f"started_at={sandbox.started_at}, timeout_at={sandbox.timeout_at}"
                )
                return await self._sandbox_repo.update(sandbox)
            else:
                logger.info(
                    f"Existing E2B sandbox {sandbox.e2b_sandbox_id} is dead, "
                    f"creating new one for {sandbox.public_id}"
                )

        # Create new E2B sandbox
        e2b_sandbox = self._gateway.create_sandbox(timeout=timeout)

        # Update sandbox record with E2B timestamps if available
        sandbox.e2b_sandbox_id = e2b_sandbox.sandbox_id
        sandbox.status = SandboxStatus.RUNNING
        sandbox.view_url = e2b_sandbox.view_url
        sandbox.control_url = e2b_sandbox.control_url
        sandbox.sandbox_url = e2b_sandbox.sandbox_url
        # Use E2B timestamps if available, fallback to calculated values
        sandbox.started_at = e2b_sandbox.started_at or now
        sandbox.timeout_at = e2b_sandbox.end_at or (now + timedelta(seconds=timeout))
        sandbox.paused_at = None
        sandbox.pause_expires_at = None
        sandbox.error_message = None

        logger.info(
            f"Started new E2B sandbox {e2b_sandbox.sandbox_id} for {sandbox.public_id}, "
            f"started_at={sandbox.started_at}, timeout_at={sandbox.timeout_at}"
        )
        return await self._sandbox_repo.update(sandbox)

    async def pause_sandbox(self, user_id: str) -> SandboxRecord:
        """Pause a user's sandbox."""
        sandbox = await self._sandbox_repo.get_by_user_id(user_id)
        if sandbox is None:
            raise SandboxNotFoundError(user_id)

        # Validate actual E2B state before operation
        sandbox = await self.sync_sandbox_state(sandbox)

        if not sandbox.status.can_pause():
            raise SandboxStateError(
                operation="pause",
                current_status=sandbox.status.value,
                allowed_statuses=[SandboxStatus.RUNNING.value],
            )

        # Actually pause the sandbox on E2B
        if sandbox.e2b_sandbox_id:
            success = self._gateway.pause_sandbox(sandbox.e2b_sandbox_id)
            if not success:
                raise RuntimeError("Failed to pause sandbox on E2B platform")

        now = datetime.now(timezone.utc)
        sandbox.status = SandboxStatus.PAUSED
        sandbox.paused_at = now
        sandbox.pause_expires_at = now + timedelta(seconds=self.MAX_PAUSE_DURATION)

        logger.info(f"Paused sandbox {sandbox.public_id}, expires at {sandbox.pause_expires_at}")
        return await self._sandbox_repo.update(sandbox)

    async def extend_timeout(
        self, user_id: str, additional_seconds: int
    ) -> SandboxRecord:
        """Extend the timeout for a running sandbox by additional_seconds."""
        sandbox = await self._sandbox_repo.get_by_user_id(user_id)
        if sandbox is None:
            raise SandboxNotFoundError(user_id)

        # Validate actual E2B state before operation
        sandbox = await self.sync_sandbox_state(sandbox)

        if sandbox.status != SandboxStatus.RUNNING:
            raise SandboxStateError(
                operation="extend timeout",
                current_status=sandbox.status.value,
                allowed_statuses=[SandboxStatus.RUNNING.value],
            )

        now = datetime.now(timezone.utc)

        if not sandbox.e2b_sandbox_id:
            raise RuntimeError("No E2B sandbox ID to extend timeout")

        # Get actual current end_at from E2B (not from DB which might be stale)
        current_info = self._gateway.get_sandbox_info(sandbox.e2b_sandbox_id)
        if not current_info or not current_info.end_at:
            raise RuntimeError("Failed to get current sandbox info from E2B")

        current_end_at = current_info.end_at
        logger.info(f"Current E2B end_at for {sandbox.public_id}: {current_end_at}")

        # Calculate new timeout by EXTENDING from current end_at
        new_timeout_at = current_end_at + timedelta(seconds=additional_seconds)

        # Ensure we don't exceed max runtime from start
        if sandbox.started_at:
            max_timeout_at = sandbox.started_at + timedelta(seconds=self.MAX_TIMEOUT)
            if new_timeout_at > max_timeout_at:
                new_timeout_at = max_timeout_at
                logger.warning(
                    f"Capped timeout to max {self.MAX_TIMEOUT}s from start for {sandbox.public_id}"
                )

        # Calculate remaining seconds from NOW for set_timeout
        # set_timeout(X) sets timeout to X seconds from NOW
        remaining_seconds = int((new_timeout_at - now).total_seconds())
        if remaining_seconds <= 0:
            raise RuntimeError("Calculated remaining seconds is <= 0")

        # Set the new timeout on E2B
        success = self._gateway.set_timeout(sandbox.e2b_sandbox_id, remaining_seconds)
        if not success:
            raise RuntimeError("Failed to extend timeout on E2B platform")

        # Refresh actual end_at from E2B to ensure accuracy
        updated_info = self._gateway.get_sandbox_info(sandbox.e2b_sandbox_id)
        if updated_info and updated_info.end_at:
            sandbox.timeout_at = updated_info.end_at
            logger.info(
                f"Extended timeout for {sandbox.public_id}: "
                f"{current_end_at} -> {updated_info.end_at} (+{additional_seconds}s)"
            )
        else:
            sandbox.timeout_at = new_timeout_at

        return await self._sandbox_repo.update(sandbox)

    async def delete_sandbox(self, user_id: str) -> SandboxRecord:
        """Delete a user's sandbox (stops E2B sandbox and removes record)."""
        sandbox = await self._sandbox_repo.get_by_user_id(user_id)
        if sandbox is None:
            raise SandboxNotFoundError(user_id)

        # Always attempt to kill E2B sandbox if we have an ID
        # (regardless of DB status, as the sandbox might still exist on E2B)
        if sandbox.e2b_sandbox_id:
            try:
                self._gateway.stop_sandbox(sandbox.e2b_sandbox_id)
                logger.info(f"Killed E2B sandbox {sandbox.e2b_sandbox_id}")
            except Exception as e:
                logger.warning(f"Failed to kill E2B sandbox {sandbox.e2b_sandbox_id}: {e}")

        # Store the sandbox info before deletion
        deleted_sandbox = SandboxRecord(
            id=sandbox.id,
            public_id=sandbox.public_id,
            user_id=sandbox.user_id,
            e2b_sandbox_id=sandbox.e2b_sandbox_id,
            status=SandboxStatus.STOPPED,
            created_at=sandbox.created_at,
        )

        # Delete the sandbox record (cascades to workspaces)
        await self._sandbox_repo.delete(sandbox.id)

        return deleted_sandbox

    async def get_or_create_workspace(
        self, user_id: str, conversation_id: str
    ) -> tuple[SandboxRecord, Workspace]:
        """Get or create a workspace for a conversation.

        This is the main entry point for workspace operations.
        It ensures the user has a running sandbox and creates a workspace if needed.
        """
        # Ensure user has a sandbox
        sandbox = await self._sandbox_repo.get_by_user_id(user_id)
        if sandbox is None:
            # Start a new sandbox
            sandbox = await self.start_sandbox(user_id)
        else:
            # Validate actual E2B state before operation
            sandbox = await self.sync_sandbox_state(sandbox)
            if sandbox.status != SandboxStatus.RUNNING:
                # Start the existing sandbox
                sandbox = await self.start_sandbox(user_id)

        # Check if workspace exists
        workspace = await self._workspace_repo.get_by_conversation_id(
            sandbox.id, conversation_id
        )

        if workspace is None:
            # Create new workspace under user home
            workspace_path = f"/home/user/workspace/{conversation_id}"
            workspace = Workspace(
                sandbox_id=sandbox.id,
                conversation_id=conversation_id,
                workspace_path=workspace_path,
            )

            # Create the workspace directory in the sandbox FIRST
            if sandbox.e2b_sandbox_id:
                result = self._gateway.run_command(
                    sandbox.e2b_sandbox_id,
                    f"mkdir -p {workspace_path}",
                    timeout=30,
                )
                if not result.is_success():
                    logger.error(
                        f"Failed to create workspace directory {workspace_path}: "
                        f"{result.error or result.stderr}"
                    )
                    raise RuntimeError(
                        f"Failed to create workspace directory: {result.error or result.stderr}"
                    )
                logger.info(f"Created workspace directory: {workspace_path}")

            # Save to DB only after directory is created
            workspace = await self._workspace_repo.create(workspace)

        return sandbox, workspace
