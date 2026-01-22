"""Alembic migration environment configuration."""

import os
from logging.config import fileConfig

from alembic import context
from sqlalchemy import create_engine, pool, text

# Import models to register with metadata
from internal.infrastructure.database.models import Base

# Alembic Config object for access to .ini file values
config = context.config

# Set up Python logging from config file
if config.config_file_name is not None:
    fileConfig(config.config_file_name)

# Target metadata for autogenerate support
target_metadata = Base.metadata


def get_url() -> str:
    """Get database URL from environment variable."""
    url = os.getenv("DB_POSTGRESQL_WRITE_DSN")
    if not url:
        # Fallback to constructing from individual components
        user = os.getenv("DB_USER", "postgres")
        password = os.getenv("DB_PASSWORD", "postgres")
        host = os.getenv("DB_HOST", "localhost")
        port = os.getenv("DB_PORT", "5432")
        name = os.getenv("DB_NAME", "jan")
        url = f"postgresql://{user}:{password}@{host}:{port}/{name}"

    # Ensure we use psycopg2 (sync driver) for migrations
    if url.startswith("postgresql+asyncpg://"):
        url = url.replace("postgresql+asyncpg://", "postgresql://")

    return url


def run_migrations_offline() -> None:
    """Run migrations in 'offline' mode.

    This configures the context with just a URL and not an Engine,
    though an Engine is acceptable here as well. By skipping the Engine
    creation we don't even need a DBAPI to be available.

    Calls to context.execute() here emit the given string to the
    script output.
    """
    url = get_url()
    context.configure(
        url=url,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
        version_table_schema="e2b_api",
    )

    with context.begin_transaction():
        context.run_migrations()


def run_migrations_online() -> None:
    """Run migrations in 'online' mode.

    In this scenario we need to create an Engine and associate a
    connection with the context.
    """
    connectable = create_engine(
        get_url(),
        poolclass=pool.NullPool,
    )

    with connectable.connect() as connection:
        # Create schema first so alembic_version table can be created in it
        connection.execute(text("CREATE SCHEMA IF NOT EXISTS e2b_api"))
        connection.commit()

        context.configure(
            connection=connection,
            target_metadata=target_metadata,
            version_table_schema="e2b_api",
        )

        with context.begin_transaction():
            context.run_migrations()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
