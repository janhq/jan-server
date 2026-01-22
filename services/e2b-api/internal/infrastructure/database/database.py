"""Database connection and session management."""

import logging

from alembic import command
from alembic.config import Config
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from internal.config.config import settings

logger = logging.getLogger(__name__)

# Create async engine
engine = create_async_engine(
    settings.database_url,
    echo=settings.log_level == "DEBUG",
    pool_pre_ping=True,
    pool_size=5,
    max_overflow=10,
)

# Create async session factory
AsyncSessionLocal = async_sessionmaker(
    bind=engine,
    class_=AsyncSession,
    expire_on_commit=False,
    autocommit=False,
    autoflush=False,
)


async def init_db() -> None:
    """Initialize database connection.

    Note: Schema creation is handled by Alembic migrations.
    This function can be used for connection verification.
    """
    async with engine.begin() as conn:
        # Just verify connection works
        await conn.execute(text("SELECT 1"))


async def close_db() -> None:
    """Close database connections."""
    await engine.dispose()


def run_migrations() -> None:
    """Run Alembic migrations (sync, called at startup).

    Similar to Go's AutoMigrate pattern - runs pending migrations on startup.
    """
    logger.info("Running database migrations...")

    # Get sync database URL for Alembic (replace asyncpg with psycopg2)
    sync_url = settings.database_url.replace("postgresql+asyncpg://", "postgresql://")

    # Configure Alembic
    alembic_cfg = Config("alembic.ini")
    alembic_cfg.set_main_option("sqlalchemy.url", sync_url)

    try:
        # Run migrations to head
        command.upgrade(alembic_cfg, "head")
        logger.info("Database migrations completed successfully")
    except Exception as e:
        logger.error(f"Failed to run database migrations: {e}")
        raise
