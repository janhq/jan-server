"""SQLAlchemy ORM models for e2b-api."""

from datetime import datetime

from sqlalchemy import BigInteger, DateTime, ForeignKey, String, Text, func
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    """Base class for all SQLAlchemy models."""

    pass


class SandboxModel(Base):
    """SQLAlchemy model for sandboxes table."""

    __tablename__ = "sandboxes"
    __table_args__ = {"schema": "e2b_api"}

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    public_id: Mapped[str] = mapped_column(String(32), unique=True, nullable=False)
    user_id: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    e2b_sandbox_id: Mapped[str | None] = mapped_column(String(255), nullable=True, index=True)
    status: Mapped[str] = mapped_column(
        String(32), nullable=False, server_default="created", index=True
    )
    view_url: Mapped[str | None] = mapped_column(String(512), nullable=True)  # View-only VNC
    control_url: Mapped[str | None] = mapped_column(String(512), nullable=True)  # Full control VNC
    sandbox_url: Mapped[str | None] = mapped_column(String(512), nullable=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    timeout_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    paused_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    pause_expires_at: Mapped[datetime | None] = mapped_column(
        DateTime(timezone=True), nullable=True
    )
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now(), onupdate=func.now()
    )

    # Relationship to workspaces
    workspaces: Mapped[list["WorkspaceModel"]] = relationship(
        "WorkspaceModel", back_populates="sandbox", cascade="all, delete-orphan"
    )


class WorkspaceModel(Base):
    """SQLAlchemy model for workspaces table."""

    __tablename__ = "workspaces"
    __table_args__ = {"schema": "e2b_api"}

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    public_id: Mapped[str] = mapped_column(String(32), unique=True, nullable=False)
    sandbox_id: Mapped[int] = mapped_column(
        BigInteger, ForeignKey("e2b_api.sandboxes.id", ondelete="CASCADE"), nullable=False
    )
    conversation_id: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    workspace_path: Mapped[str] = mapped_column(String(512), nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now(), onupdate=func.now()
    )

    # Relationship to sandbox
    sandbox: Mapped["SandboxModel"] = relationship("SandboxModel", back_populates="workspaces")
