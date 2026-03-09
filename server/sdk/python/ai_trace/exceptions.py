"""
Exception classes for AI-Trace SDK.
"""

from typing import Any, Dict, Optional


class AITraceError(Exception):
    """Base exception for AI-Trace SDK errors."""

    def __init__(
        self,
        message: str,
        status_code: Optional[int] = None,
        response: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.response = response or {}

    def __str__(self) -> str:
        if self.status_code:
            return f"[{self.status_code}] {self.message}"
        return self.message


class AuthenticationError(AITraceError):
    """Raised when authentication fails."""

    def __init__(
        self,
        message: str = "Authentication failed",
        status_code: int = 401,
        response: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__(message, status_code, response)


class NotFoundError(AITraceError):
    """Raised when a resource is not found."""

    def __init__(
        self,
        message: str = "Resource not found",
        status_code: int = 404,
        response: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__(message, status_code, response)


class ValidationError(AITraceError):
    """Raised when request validation fails."""

    def __init__(
        self,
        message: str = "Validation error",
        status_code: int = 400,
        response: Optional[Dict[str, Any]] = None,
        errors: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__(message, status_code, response)
        self.errors = errors or {}


class RateLimitError(AITraceError):
    """Raised when rate limit is exceeded."""

    def __init__(
        self,
        message: str = "Rate limit exceeded",
        status_code: int = 429,
        response: Optional[Dict[str, Any]] = None,
        retry_after: Optional[int] = None,
    ) -> None:
        super().__init__(message, status_code, response)
        self.retry_after = retry_after


class ServerError(AITraceError):
    """Raised when server returns an error."""

    def __init__(
        self,
        message: str = "Server error",
        status_code: int = 500,
        response: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__(message, status_code, response)


class ConnectionError(AITraceError):
    """Raised when connection to server fails."""

    def __init__(
        self,
        message: str = "Connection failed",
        response: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__(message, None, response)


class TimeoutError(AITraceError):
    """Raised when request times out."""

    def __init__(
        self,
        message: str = "Request timed out",
        response: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__(message, None, response)
