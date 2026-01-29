"""Domain exceptions for e2b-api.

These exceptions represent business logic errors and are caught
by the global exception handler to return proper API responses.
"""


class DomainException(Exception):
    """Base exception for all domain errors."""

    def __init__(self, message: str, code: str = "DOMAIN_ERROR"):
        self.message = message
        self.code = code
        super().__init__(message)


class NotFoundError(DomainException):
    """Resource not found."""

    def __init__(self, resource: str, identifier: str):
        self.resource = resource
        self.identifier = identifier
        super().__init__(
            message=f"{resource} not found: {identifier}",
            code="NOT_FOUND",
        )


class SandboxNotFoundError(NotFoundError):
    """Sandbox not found for user."""

    def __init__(self, user_id: str):
        super().__init__(resource="Sandbox", identifier=f"user_id={user_id}")


class InvalidStateError(DomainException):
    """Operation not allowed in current state."""

    def __init__(self, message: str, current_state: str, allowed_states: list[str] | None = None):
        self.current_state = current_state
        self.allowed_states = allowed_states or []
        super().__init__(message=message, code="INVALID_STATE")


class SandboxStateError(InvalidStateError):
    """Sandbox is in wrong state for operation."""

    def __init__(self, operation: str, current_status: str, allowed_statuses: list[str] | None = None):
        allowed = allowed_statuses or []
        message = f"Cannot {operation} sandbox in status '{current_status}'"
        if allowed:
            message += f". Allowed: {', '.join(allowed)}"
        super().__init__(
            message=message,
            current_state=current_status,
            allowed_states=allowed,
        )


class ValidationError(DomainException):
    """Input validation failed."""

    def __init__(self, message: str, field: str | None = None):
        self.field = field
        super().__init__(message=message, code="VALIDATION_ERROR")


class E2BError(DomainException):
    """Error from E2B SDK."""

    def __init__(self, message: str, operation: str | None = None):
        self.operation = operation
        super().__init__(message=message, code="E2B_ERROR")


class DatabaseError(DomainException):
    """Database operation failed."""

    def __init__(self, message: str = "Database operation failed"):
        super().__init__(message=message, code="DATABASE_ERROR")
