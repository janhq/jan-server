"""Configuration settings for e2b-api."""

import re

from pydantic import computed_field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # E2B Configuration
    e2b_api_key: str = ""
    e2b_access_token: str = ""  # Access token for authenticated operations
    e2b_template_id: str = "jan-browser-sandbox"

    # Service Configuration
    e2b_service_port: int = 8095
    log_level: str = "INFO"

    # Sandbox defaults
    e2b_sandbox_timeout: int = 1800  # 30 minutes default sandbox timeout
    sandbox_max_runtime: int = 86400  # 24 hours max runtime (E2B limit)
    sandbox_max_pause: int = 2592000  # 30 days max pause (E2B limit)

    # Database Configuration
    db_host: str = "api-db"
    db_port: int = 5432
    db_user: str = "sandbox"
    db_password: str = "sandbox"
    db_name: str = "e2b_sandbox"

    # Direct DSN override (takes precedence if set)
    db_postgresql_write_dsn: str = ""
    db_postgresql_read_dsn: str = ""

    # Auto-migrate on startup (like Go services)
    auto_migrate: bool = True

    @computed_field
    @property
    def database_url(self) -> str:
        """Get async database URL for SQLAlchemy."""
        if self.db_postgresql_write_dsn:
            # Convert to async driver if needed
            dsn = self.db_postgresql_write_dsn
            # Handle both postgres:// (old) and postgresql:// (new) schemes
            if dsn.startswith("postgres://"):
                return dsn.replace("postgres://", "postgresql+asyncpg://", 1)
            if dsn.startswith("postgresql://"):
                return dsn.replace("postgresql://", "postgresql+asyncpg://", 1)
            return dsn
        return f"postgresql+asyncpg://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/{self.db_name}"

    # MCP Proxy Configuration (inside sandbox)
    mcp_proxy_port: int = 17390  # HTTP port for mcp-proxy inside sandbox

    @field_validator(
        "e2b_sandbox_timeout",
        "sandbox_max_runtime",
        "sandbox_max_pause",
        mode="before",
    )
    @classmethod
    def _parse_duration_seconds(cls, value):
        if isinstance(value, (int, float)):
            return int(value)
        if isinstance(value, str):
            raw = value.strip().lower()
            if raw.isdigit():
                return int(raw)
            matches = re.findall(r"(\d+)\s*([smhd])", raw)
            if matches:
                total = 0
                units = {"s": 1, "m": 60, "h": 3600, "d": 86400}
                for number, unit in matches:
                    total += int(number) * units[unit]
                return total
        return int(value)


settings = Settings()
