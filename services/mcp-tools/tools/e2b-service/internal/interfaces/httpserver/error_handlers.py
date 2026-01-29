"""Global exception handlers for FastAPI.

Converts domain exceptions to proper HTTP responses.
"""

import logging
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from sqlalchemy.exc import SQLAlchemyError

from internal.domain.exceptions import (
    DomainException,
    NotFoundError,
    InvalidStateError,
    ValidationError,
    E2BError,
    DatabaseError,
)

logger = logging.getLogger(__name__)


def create_error_response(
    status_code: int,
    error: str,
    code: str,
    detail: str | None = None,
    field: str | None = None,
) -> JSONResponse:
    """Create a standardized error response."""
    content = {"error": error, "code": code}
    if detail:
        content["detail"] = detail
    if field:
        content["field"] = field
    return JSONResponse(status_code=status_code, content=content)


def register_exception_handlers(app: FastAPI) -> None:
    """Register all exception handlers with the FastAPI app."""

    @app.exception_handler(NotFoundError)
    async def not_found_handler(request: Request, exc: NotFoundError) -> JSONResponse:
        logger.info(f"Not found: {exc.resource} - {exc.identifier}")
        return create_error_response(
            status_code=404,
            error=exc.message,
            code=exc.code,
        )

    @app.exception_handler(InvalidStateError)
    async def invalid_state_handler(request: Request, exc: InvalidStateError) -> JSONResponse:
        logger.warning(f"Invalid state: {exc.message}")
        return create_error_response(
            status_code=409,
            error=exc.message,
            code=exc.code,
            detail=f"Current state: {exc.current_state}",
        )

    @app.exception_handler(ValidationError)
    async def validation_handler(request: Request, exc: ValidationError) -> JSONResponse:
        logger.info(f"Validation error: {exc.message}")
        return create_error_response(
            status_code=400,
            error=exc.message,
            code=exc.code,
            field=exc.field,
        )

    @app.exception_handler(E2BError)
    async def e2b_error_handler(request: Request, exc: E2BError) -> JSONResponse:
        logger.error(f"E2B error: {exc.message}, operation: {exc.operation}")
        return create_error_response(
            status_code=502,
            error="E2B service error",
            code=exc.code,
            detail=exc.message,
        )

    @app.exception_handler(DatabaseError)
    async def database_error_handler(request: Request, exc: DatabaseError) -> JSONResponse:
        logger.error(f"Database error: {exc.message}")
        return create_error_response(
            status_code=503,
            error="Database service unavailable",
            code=exc.code,
        )

    @app.exception_handler(DomainException)
    async def domain_exception_handler(request: Request, exc: DomainException) -> JSONResponse:
        logger.warning(f"Domain error: {exc.code} - {exc.message}")
        return create_error_response(
            status_code=400,
            error=exc.message,
            code=exc.code,
        )

    @app.exception_handler(SQLAlchemyError)
    async def sqlalchemy_error_handler(request: Request, exc: SQLAlchemyError) -> JSONResponse:
        logger.error(f"SQLAlchemy error: {type(exc).__name__}: {exc}")
        # Check for common errors
        error_str = str(exc).lower()
        if "does not exist" in error_str or "undefined" in error_str:
            return create_error_response(
                status_code=503,
                error="Database schema not initialized. Run migrations first.",
                code="SCHEMA_NOT_INITIALIZED",
                detail="Run: docker exec e2b-api alembic upgrade head",
            )
        return create_error_response(
            status_code=503,
            error="Database error",
            code="DATABASE_ERROR",
        )

    @app.exception_handler(ValueError)
    async def value_error_handler(request: Request, exc: ValueError) -> JSONResponse:
        logger.warning(f"Value error: {exc}")
        return create_error_response(
            status_code=400,
            error=str(exc),
            code="VALIDATION_ERROR",
        )

    @app.exception_handler(Exception)
    async def generic_exception_handler(request: Request, exc: Exception) -> JSONResponse:
        logger.exception(f"Unhandled exception: {type(exc).__name__}: {exc}")
        return create_error_response(
            status_code=500,
            error="Internal server error",
            code="INTERNAL_ERROR",
        )
