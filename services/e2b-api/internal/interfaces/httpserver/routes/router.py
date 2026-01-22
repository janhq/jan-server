"""Main API router - aggregates all routes."""

from fastapi import APIRouter

from .user_sandbox import create_user_sandbox_router

# Create main API router with /api/v1 prefix
router = APIRouter(prefix="/api/v1")

# Include sub-routers
router.include_router(create_user_sandbox_router())  # User-centric routes (includes MCP endpoint)
