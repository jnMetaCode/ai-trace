"""
AI-Trace Python SDK Client.

Main entry point for interacting with the AI-Trace API.
"""

from datetime import datetime
from typing import Any, Dict, List, Optional, Union

import httpx

from ai_trace.exceptions import (
    AITraceError,
    AuthenticationError,
    NotFoundError,
    RateLimitError,
    ServerError,
    ValidationError,
)
from ai_trace.models import (
    Certificate,
    CertificateListResponse,
    ChainType,
    Event,
    EventListResponse,
    EvidenceLevel,
    Proof,
    Trace,
    TraceListResponse,
    VerificationResult,
)


class TracesAPI:
    """API client for trace operations."""

    def __init__(self, client: "AITrace") -> None:
        self._client = client

    def create(
        self,
        name: Optional[str] = None,
        tenant_id: str = "default",
        user_id: Optional[str] = None,
        session_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Trace:
        """Create a new trace.

        Args:
            name: Human-readable name for the trace
            tenant_id: Tenant identifier
            user_id: User who initiated the trace
            session_id: Session identifier
            metadata: Additional metadata

        Returns:
            Created trace object
        """
        data: Dict[str, Any] = {"tenant_id": tenant_id}
        if name:
            data["name"] = name
        if user_id:
            data["user_id"] = user_id
        if session_id:
            data["session_id"] = session_id
        if metadata:
            data["metadata"] = metadata

        response = self._client._post("/api/v1/traces", json=data)
        return Trace(**response)

    def get(self, trace_id: str) -> Trace:
        """Get a trace by ID.

        Args:
            trace_id: Trace identifier

        Returns:
            Trace object
        """
        response = self._client._get(f"/api/v1/traces/{trace_id}")
        return Trace(**response)

    def list(
        self,
        tenant_id: Optional[str] = None,
        limit: int = 20,
        offset: int = 0,
    ) -> TraceListResponse:
        """List traces with optional filtering.

        Args:
            tenant_id: Filter by tenant
            limit: Maximum number of results
            offset: Pagination offset

        Returns:
            Paginated list of traces
        """
        params: Dict[str, Any] = {"limit": limit, "offset": offset}
        if tenant_id:
            params["tenant_id"] = tenant_id

        response = self._client._get("/api/v1/traces", params=params)
        return TraceListResponse(**response)


class EventsAPI:
    """API client for event operations."""

    def __init__(self, client: "AITrace") -> None:
        self._client = client

    def add(
        self,
        trace_id: str,
        event_type: str = "custom",
        payload: Optional[Dict[str, Any]] = None,
        metadata: Optional[Dict[str, Any]] = None,
        timestamp: Optional[datetime] = None,
    ) -> Event:
        """Add an event to a trace.

        Args:
            trace_id: Parent trace identifier
            event_type: Event type (input, output, custom)
            payload: Event payload data
            metadata: Additional metadata
            timestamp: Event timestamp (defaults to now)

        Returns:
            Created event object
        """
        data: Dict[str, Any] = {
            "trace_id": trace_id,
            "event_type": event_type,
            "payload": payload or {},
            "timestamp": (timestamp or datetime.utcnow()).isoformat(),
        }
        if metadata:
            data["metadata"] = metadata

        response = self._client._post("/api/v1/events/ingest", json=data)
        return Event(**response)

    def get(self, event_id: str) -> Event:
        """Get an event by ID.

        Args:
            event_id: Event identifier

        Returns:
            Event object
        """
        response = self._client._get(f"/api/v1/events/{event_id}")
        return Event(**response)

    def list(
        self,
        trace_id: str,
        limit: int = 100,
        offset: int = 0,
    ) -> EventListResponse:
        """List events for a trace.

        Args:
            trace_id: Trace identifier
            limit: Maximum number of results
            offset: Pagination offset

        Returns:
            Paginated list of events
        """
        params = {"trace_id": trace_id, "limit": limit, "offset": offset}
        response = self._client._get("/api/v1/events/search", params=params)
        return EventListResponse(**response)


class CertsAPI:
    """API client for certificate operations."""

    def __init__(self, client: "AITrace") -> None:
        self._client = client

    def commit(
        self,
        trace_id: str,
        evidence_level: Union[EvidenceLevel, str] = EvidenceLevel.L1,
        chain_type: Optional[Union[ChainType, str]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Certificate:
        """Commit a trace to a certificate.

        Args:
            trace_id: Trace to commit
            evidence_level: Evidence level (L1, L2, L3)
            chain_type: Blockchain type for L3 (ethereum, polygon)
            metadata: Additional metadata

        Returns:
            Created certificate object
        """
        if isinstance(evidence_level, str):
            evidence_level = EvidenceLevel(evidence_level)

        data: Dict[str, Any] = {
            "trace_id": trace_id,
            "evidence_level": evidence_level.value,
        }
        if evidence_level == EvidenceLevel.L3 and chain_type:
            if isinstance(chain_type, str):
                chain_type = ChainType(chain_type)
            data["chain_type"] = chain_type.value
        if metadata:
            data["metadata"] = metadata

        response = self._client._post("/api/v1/certs/commit", json=data)
        return Certificate(**response)

    def get(self, cert_id: str, include_events: bool = False) -> Certificate:
        """Get a certificate by ID.

        Args:
            cert_id: Certificate identifier
            include_events: Whether to include event data

        Returns:
            Certificate object
        """
        params = {"include_events": include_events}
        response = self._client._get(f"/api/v1/certs/{cert_id}", params=params)
        return Certificate(**response)

    def verify(self, cert_id: str, full_verification: bool = False) -> VerificationResult:
        """Verify a certificate.

        Args:
            cert_id: Certificate to verify
            full_verification: Whether to perform full verification including blockchain

        Returns:
            Verification result
        """
        data = {"cert_id": cert_id, "full_verification": full_verification}
        response = self._client._post("/api/v1/certs/verify", json=data)
        return VerificationResult(**response)

    def list(
        self,
        tenant_id: Optional[str] = None,
        evidence_level: Optional[Union[EvidenceLevel, str]] = None,
        limit: int = 20,
        offset: int = 0,
    ) -> CertificateListResponse:
        """List certificates with optional filtering.

        Args:
            tenant_id: Filter by tenant
            evidence_level: Filter by evidence level
            limit: Maximum number of results
            offset: Pagination offset

        Returns:
            Paginated list of certificates
        """
        params: Dict[str, Any] = {"limit": limit, "offset": offset}
        if tenant_id:
            params["tenant_id"] = tenant_id
        if evidence_level:
            if isinstance(evidence_level, EvidenceLevel):
                evidence_level = evidence_level.value
            params["evidence_level"] = evidence_level

        response = self._client._get("/api/v1/certs/search", params=params)
        return CertificateListResponse(**response)

    def prove(
        self,
        cert_id: str,
        event_indices: List[int],
        disclosed_fields: Optional[List[str]] = None,
    ) -> Proof:
        """Generate a minimal disclosure proof.

        Args:
            cert_id: Certificate identifier
            event_indices: Indices of events to include in proof
            disclosed_fields: Specific fields to disclose (optional)

        Returns:
            Proof object
        """
        data: Dict[str, Any] = {
            "cert_id": cert_id,
            "event_indices": event_indices,
        }
        if disclosed_fields:
            data["disclosed_fields"] = disclosed_fields

        response = self._client._post(f"/api/v1/certs/{cert_id}/prove", json=data)
        return Proof(**response)


class AITrace:
    """Main AI-Trace SDK client.

    Example:
        client = AITrace(
            server_url="http://localhost:8006",
            api_key="your-api-key"
        )

        # Create a trace
        trace = client.traces.create(name="My Trace")

        # Add events
        client.events.add(trace_id=trace.id, event_type="input", payload={"prompt": "Hello"})

        # Commit to certificate
        cert = client.certs.commit(trace_id=trace.id, evidence_level="L2")
    """

    def __init__(
        self,
        server_url: str = "http://localhost:8006",
        api_key: Optional[str] = None,
        tenant_id: str = "default",
        timeout: float = 30.0,
        verify_ssl: bool = True,
    ) -> None:
        """Initialize the AI-Trace client.

        Args:
            server_url: Base URL of the AI-Trace server
            api_key: API key for authentication
            tenant_id: Default tenant identifier
            timeout: Request timeout in seconds
            verify_ssl: Whether to verify SSL certificates
        """
        self.server_url = server_url.rstrip("/")
        self.api_key = api_key
        self.tenant_id = tenant_id
        self.timeout = timeout
        self.verify_ssl = verify_ssl

        self._client = httpx.Client(
            base_url=self.server_url,
            timeout=timeout,
            verify=verify_ssl,
        )

        # Initialize API clients
        self.traces = TracesAPI(self)
        self.events = EventsAPI(self)
        self.certs = CertsAPI(self)

    def _get_headers(self) -> Dict[str, str]:
        """Get default request headers."""
        headers = {
            "Content-Type": "application/json",
            "X-Tenant-ID": self.tenant_id,
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        return headers

    def _handle_response(self, response: httpx.Response) -> Dict[str, Any]:
        """Handle HTTP response and raise appropriate exceptions."""
        if response.status_code == 200 or response.status_code == 201:
            return response.json()

        # Parse error response
        try:
            error_data = response.json()
            message = error_data.get("error", error_data.get("message", "Unknown error"))
        except Exception:
            message = response.text or "Unknown error"

        if response.status_code == 401:
            raise AuthenticationError(message, response.status_code, error_data)
        elif response.status_code == 404:
            raise NotFoundError(message, response.status_code, error_data)
        elif response.status_code == 400:
            raise ValidationError(message, response.status_code, error_data)
        elif response.status_code == 429:
            retry_after = response.headers.get("Retry-After")
            raise RateLimitError(
                message,
                response.status_code,
                error_data,
                int(retry_after) if retry_after else None,
            )
        elif response.status_code >= 500:
            raise ServerError(message, response.status_code, error_data)
        else:
            raise AITraceError(message, response.status_code, error_data)

    def _get(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make a GET request."""
        response = self._client.get(path, params=params, headers=self._get_headers())
        return self._handle_response(response)

    def _post(
        self,
        path: str,
        json: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make a POST request."""
        response = self._client.post(path, json=json, headers=self._get_headers())
        return self._handle_response(response)

    def _put(
        self,
        path: str,
        json: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make a PUT request."""
        response = self._client.put(path, json=json, headers=self._get_headers())
        return self._handle_response(response)

    def _delete(self, path: str) -> Dict[str, Any]:
        """Make a DELETE request."""
        response = self._client.delete(path, headers=self._get_headers())
        return self._handle_response(response)

    def health(self) -> Dict[str, Any]:
        """Check server health.

        Returns:
            Health status response
        """
        return self._get("/health")

    def close(self) -> None:
        """Close the HTTP client."""
        self._client.close()

    def __enter__(self) -> "AITrace":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()
