"""E2B API - FastAPI application entry point."""

import logging
import sys
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from internal.config import settings
from internal.infrastructure.database.database import close_db, init_db, run_migrations
from internal.interfaces.httpserver.error_handlers import register_exception_handlers
from internal.interfaces.httpserver.routes import router as api_router

from .wire import create_container

# Configure logging
logging.basicConfig(
    level=getattr(logging, settings.log_level.upper()),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan - startup and shutdown logic.

    This is the composition root where DI container is created.
    Similar to main() in Go services that calls wire.
    """
    # Startup
    logger.info(f"Starting E2B API on port {settings.e2b_service_port}")
    logger.info(f"E2B Template ID: {settings.e2b_template_id}")

    if not settings.e2b_api_key:
        logger.warning("E2B_API_KEY not configured - sandbox operations will fail")

    # Run database migrations if AUTO_MIGRATE is enabled (like Go services)
    if settings.auto_migrate:
        try:
            run_migrations()
        except Exception as e:
            logger.error(f"Failed to run migrations: {e}")
            # Continue anyway - migrations might already be applied

    # Initialize database connection
    try:
        await init_db()
        logger.info("Database connection initialized")
    except Exception as e:
        logger.error(f"Failed to initialize database: {e}")
        # Continue anyway - database may not be needed for all operations

    # Create DI container (composition root)
    container = create_container()
    app.state.container = container
    logger.info("DI container initialized")

    yield  # Application runs here

    # Shutdown
    logger.info("Shutting down E2B API")

    # Close database connections
    try:
        await close_db()
        logger.info("Database connections closed")
    except Exception as e:
        logger.error(f"Error closing database: {e}")

    # Clean up any running sandboxes
    gateway = container.sandbox_gateway
    for sandbox_id in list(gateway._sandboxes.keys()):
        try:
            gateway.stop_sandbox(sandbox_id)
            logger.info(f"Stopped sandbox {sandbox_id} during shutdown")
        except Exception as e:
            logger.error(f"Error stopping sandbox {sandbox_id}: {e}")


# Create FastAPI app with lifespan
app = FastAPI(
    title="E2B API",
    description="Sandbox management API for Jan Server using E2B",
    version="1.0.0",
    lifespan=lifespan,
)

# Register exception handlers (DDD error handling)
register_exception_handlers(app)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include API router
app.include_router(api_router)


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "e2b-api"}


@app.get("/readyz")
async def readiness_check():
    """Readiness check endpoint."""
    if not settings.e2b_api_key:
        return {"status": "not_ready", "reason": "E2B_API_KEY not configured"}
    return {"status": "ready", "service": "e2b-api"}


def main():
    """Run the server."""
    import uvicorn

    uvicorn.run(
        "cmd.server.main:app",
        host="0.0.0.0",
        port=settings.e2b_service_port,
        reload=True,
    )


if __name__ == "__main__":
    main()
