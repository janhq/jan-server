"""MCP session store for caching dynamic tools per sandbox."""

import time
from dataclasses import dataclass, field
from typing import Any


@dataclass
class ToolInfo:
    """Information about an MCP tool."""

    name: str
    description: str
    input_schema: dict[str, Any] = field(default_factory=dict)


@dataclass
class CachedTools:
    """Cached tools for a sandbox with TTL."""

    tools: list[ToolInfo]
    cached_at: float
    ttl_seconds: float = 120.0  # 2 minutes default TTL

    def is_expired(self) -> bool:
        """Check if the cache has expired."""
        return time.time() - self.cached_at > self.ttl_seconds


class MCPSessionStore:
    """In-memory store for caching MCP tools per sandbox.

    This store caches the dynamic tools discovered from each sandbox's
    mcp-proxy server to avoid repeated discovery calls.
    """

    def __init__(self, default_ttl_seconds: float = 120.0):
        """Initialize the session store.

        Args:
            default_ttl_seconds: Default TTL for cached tools (default: 2 minutes)
        """
        self._tools_cache: dict[str, CachedTools] = {}
        self._default_ttl = default_ttl_seconds

    def get_cached_tools(self, sandbox_id: str) -> list[ToolInfo] | None:
        """Get cached tools for a sandbox.

        Args:
            sandbox_id: The sandbox public ID

        Returns:
            List of tools if cached and not expired, None otherwise
        """
        cached = self._tools_cache.get(sandbox_id)
        if cached is None:
            return None
        if cached.is_expired():
            del self._tools_cache[sandbox_id]
            return None
        return cached.tools

    def set_cached_tools(
        self,
        sandbox_id: str,
        tools: list[ToolInfo],
        ttl_seconds: float | None = None,
    ) -> None:
        """Cache tools for a sandbox.

        Args:
            sandbox_id: The sandbox public ID
            tools: List of tools to cache
            ttl_seconds: Optional TTL override
        """
        self._tools_cache[sandbox_id] = CachedTools(
            tools=tools,
            cached_at=time.time(),
            ttl_seconds=ttl_seconds or self._default_ttl,
        )

    def clear_cached_tools(self, sandbox_id: str) -> bool:
        """Clear cached tools for a sandbox.

        Args:
            sandbox_id: The sandbox public ID

        Returns:
            True if cache was cleared, False if sandbox wasn't cached
        """
        if sandbox_id in self._tools_cache:
            del self._tools_cache[sandbox_id]
            return True
        return False

    def clear_all(self) -> None:
        """Clear all cached tools."""
        self._tools_cache.clear()

    def cleanup_expired(self) -> int:
        """Remove all expired entries from the cache.

        Returns:
            Number of entries removed
        """
        expired = [
            sandbox_id
            for sandbox_id, cached in self._tools_cache.items()
            if cached.is_expired()
        ]
        for sandbox_id in expired:
            del self._tools_cache[sandbox_id]
        return len(expired)
