"""Initial e2b_api schema with sandboxes and workspaces tables.

Revision ID: 0001
Revises:
Create Date: 2025-01-21

"""

from typing import Sequence, Union

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision: str = "0001"
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Create e2b_api schema and tables."""
    # Create schema
    op.execute("CREATE SCHEMA IF NOT EXISTS e2b_api")

    # Create sandboxes table
    op.create_table(
        "sandboxes",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "public_id",
            sa.String(length=32),
            nullable=False,
            comment="External-facing ID (e.g., sb_xxx)",
        ),
        sa.Column("user_id", sa.String(length=255), nullable=False, comment="User who owns this sandbox"),
        sa.Column(
            "e2b_sandbox_id",
            sa.String(length=255),
            nullable=True,
            comment="E2B platform sandbox ID",
        ),
        sa.Column(
            "status",
            sa.String(length=32),
            nullable=False,
            server_default="created",
            comment="Sandbox status: created, running, paused, stopped, error",
        ),
        sa.Column("view_url", sa.String(length=512), nullable=True, comment="View-only VNC URL with auth"),
        sa.Column("control_url", sa.String(length=512), nullable=True, comment="Full control VNC URL with auth"),
        sa.Column("sandbox_url", sa.String(length=512), nullable=True, comment="External sandbox URL for MCP proxy"),
        sa.Column(
            "started_at",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="When sandbox was started",
        ),
        sa.Column(
            "timeout_at",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="When sandbox will auto-stop (24h max)",
        ),
        sa.Column(
            "paused_at",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="When sandbox was paused",
        ),
        sa.Column(
            "pause_expires_at",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="When paused sandbox expires (30 day max)",
        ),
        sa.Column("error_message", sa.Text(), nullable=True, comment="Error details if status is error"),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("now()"),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("now()"),
        ),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("public_id"),
        schema="e2b_api",
    )

    # Create indexes for sandboxes
    op.create_index(
        "ix_sandboxes_user_id",
        "sandboxes",
        ["user_id"],
        unique=False,
        schema="e2b_api",
    )
    op.create_index(
        "ix_sandboxes_status",
        "sandboxes",
        ["status"],
        unique=False,
        schema="e2b_api",
    )
    op.create_index(
        "ix_sandboxes_e2b_sandbox_id",
        "sandboxes",
        ["e2b_sandbox_id"],
        unique=False,
        schema="e2b_api",
    )

    # Create workspaces table
    op.create_table(
        "workspaces",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "public_id",
            sa.String(length=32),
            nullable=False,
            comment="External-facing ID (e.g., ws_xxx)",
        ),
        sa.Column("sandbox_id", sa.BigInteger(), nullable=False, comment="FK to sandboxes"),
        sa.Column(
            "conversation_id",
            sa.String(length=255),
            nullable=False,
            comment="Conversation this workspace belongs to",
        ),
        sa.Column(
            "workspace_path",
            sa.String(length=512),
            nullable=False,
            comment="Path in sandbox filesystem (e.g., /workspace/conv_xxx/)",
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("now()"),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("now()"),
        ),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("public_id"),
        sa.ForeignKeyConstraint(
            ["sandbox_id"],
            ["e2b_api.sandboxes.id"],
            ondelete="CASCADE",
        ),
        schema="e2b_api",
    )

    # Create indexes for workspaces
    op.create_index(
        "ix_workspaces_sandbox_id",
        "workspaces",
        ["sandbox_id"],
        unique=False,
        schema="e2b_api",
    )
    op.create_index(
        "ix_workspaces_conversation_id",
        "workspaces",
        ["conversation_id"],
        unique=False,
        schema="e2b_api",
    )
    # Unique constraint: one workspace per conversation per sandbox
    op.create_index(
        "ix_workspaces_sandbox_conversation",
        "workspaces",
        ["sandbox_id", "conversation_id"],
        unique=True,
        schema="e2b_api",
    )


def downgrade() -> None:
    """Drop e2b_api schema and tables."""
    op.drop_table("workspaces", schema="e2b_api")
    op.drop_table("sandboxes", schema="e2b_api")
    op.execute("DROP SCHEMA IF EXISTS e2b_api CASCADE")
