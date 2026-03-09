"""
AI-Trace API Client
"""

from typing import Any, Dict, List, Optional

import httpx

from .models import (
    Certificate,
    CommitCertResponse,
    Event,
    IngestEventsResponse,
    MinimalDisclosureProof,
    SearchEventsResponse,
    VerifyResult,
)


class EventsAPI:
    """Events API"""

    def __init__(self, client: "AITraceClient"):
        self._client = client

    def search(
        self,
        trace_id: Optional[str] = None,
        event_type: Optional[str] = None,
        start_time: Optional[str] = None,
        end_time: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> SearchEventsResponse:
        """Search events"""
        params: Dict[str, Any] = {"page": page, "page_size": page_size}
        if trace_id:
            params["trace_id"] = trace_id
        if event_type:
            params["event_type"] = event_type
        if start_time:
            params["start_time"] = start_time
        if end_time:
            params["end_time"] = end_time

        response = self._client._request("GET", "/events/search", params=params)
        return SearchEventsResponse(**response)

    def get(self, event_id: str) -> Event:
        """Get single event"""
        response = self._client._request("GET", f"/events/{event_id}")
        return Event(**response)

    def ingest(self, events: List[Dict[str, Any]]) -> IngestEventsResponse:
        """Ingest events"""
        response = self._client._request("POST", "/events/ingest", json={"events": events})
        return IngestEventsResponse(**response)


class CertsAPI:
    """Certificates API"""

    def __init__(self, client: "AITraceClient"):
        self._client = client

    def search(
        self,
        page: int = 1,
        page_size: int = 20,
    ) -> Dict[str, Any]:
        """Search certificates"""
        params = {"page": page, "page_size": page_size}
        return self._client._request("GET", "/certs/search", params=params)

    def get(self, cert_id: str) -> Certificate:
        """Get certificate details"""
        response = self._client._request("GET", f"/certs/{cert_id}")
        return Certificate(**response)

    def commit(
        self,
        trace_id: str,
        evidence_level: Optional[str] = None,
    ) -> CommitCertResponse:
        """Create certificate"""
        data: Dict[str, Any] = {"trace_id": trace_id}
        if evidence_level:
            data["evidence_level"] = evidence_level

        response = self._client._request("POST", "/certs/commit", json=data)
        return CommitCertResponse(**response)

    def verify(
        self,
        cert_id: Optional[str] = None,
        root_hash: Optional[str] = None,
    ) -> VerifyResult:
        """Verify certificate"""
        data: Dict[str, Any] = {}
        if cert_id:
            data["cert_id"] = cert_id
        if root_hash:
            data["root_hash"] = root_hash

        response = self._client._request("POST", "/certs/verify", json=data)
        return VerifyResult(**response)

    def generate_proof(
        self,
        cert_id: str,
        disclose_events: List[int],
        disclose_fields: Optional[List[str]] = None,
    ) -> MinimalDisclosureProof:
        """Generate minimal disclosure proof"""
        data = {
            "disclose_events": disclose_events,
            "disclose_fields": disclose_fields or [],
        }
        response = self._client._request("POST", f"/certs/{cert_id}/prove", json=data)
        return MinimalDisclosureProof(**response)


class ChatAPI:
    """Chat API (OpenAI compatible)"""

    def __init__(self, client: "AITraceClient"):
        self._client = client

    def completions(
        self,
        messages: List[Dict[str, str]],
        model: str = "gpt-3.5-turbo",
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
        """Create chat completion with tracing"""
        data: Dict[str, Any] = {
            "model": model,
            "messages": messages,
        }
        if temperature is not None:
            data["temperature"] = temperature
        if max_tokens is not None:
            data["max_tokens"] = max_tokens
        data.update(kwargs)

        return self._client._request("POST", "/chat/completions", json=data)


class AITraceClient:
    """
    AI-Trace API Client

    Example:
        client = AITraceClient(
            base_url="http://localhost:8080",
            api_key="your-api-key"
        )

        # Search events
        events = client.events.search(trace_id="trc_xxx")

        # Create certificate
        cert = client.certs.commit(trace_id="trc_xxx")

        # Verify
        result = client.certs.verify(cert_id=cert.cert_id)
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        tenant_id: str = "default",
        timeout: float = 30.0,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.tenant_id = tenant_id
        self.timeout = timeout

        self._http_client = httpx.Client(timeout=timeout)

        # API namespaces
        self.events = EventsAPI(self)
        self.certs = CertsAPI(self)
        self.chat = ChatAPI(self)

    def _get_headers(self) -> Dict[str, str]:
        """Get request headers"""
        headers = {
            "Content-Type": "application/json",
            "X-Tenant-ID": self.tenant_id,
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        return headers

    def _request(
        self,
        method: str,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        json: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make HTTP request"""
        url = f"{self.base_url}/api/v1{path}"
        headers = self._get_headers()

        response = self._http_client.request(
            method=method,
            url=url,
            headers=headers,
            params=params,
            json=json,
        )
        response.raise_for_status()
        return response.json()

    def health(self) -> Dict[str, Any]:
        """Check server health"""
        response = self._http_client.get(f"{self.base_url}/health")
        response.raise_for_status()
        return response.json()

    def close(self) -> None:
        """Close HTTP client"""
        self._http_client.close()

    def __enter__(self) -> "AITraceClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


class AsyncAITraceClient:
    """
    Async AI-Trace API Client

    Example:
        async with AsyncAITraceClient(base_url="http://localhost:8080") as client:
            events = await client.events.search(trace_id="trc_xxx")
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        tenant_id: str = "default",
        timeout: float = 30.0,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.tenant_id = tenant_id
        self.timeout = timeout

        self._http_client = httpx.AsyncClient(timeout=timeout)

        # API namespaces (async versions)
        self.events = AsyncEventsAPI(self)
        self.certs = AsyncCertsAPI(self)
        self.chat = AsyncChatAPI(self)

    def _get_headers(self) -> Dict[str, str]:
        """Get request headers"""
        headers = {
            "Content-Type": "application/json",
            "X-Tenant-ID": self.tenant_id,
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        return headers

    async def _request(
        self,
        method: str,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        json: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make async HTTP request"""
        url = f"{self.base_url}/api/v1{path}"
        headers = self._get_headers()

        response = await self._http_client.request(
            method=method,
            url=url,
            headers=headers,
            params=params,
            json=json,
        )
        response.raise_for_status()
        return response.json()

    async def health(self) -> Dict[str, Any]:
        """Check server health"""
        response = await self._http_client.get(f"{self.base_url}/health")
        response.raise_for_status()
        return response.json()

    async def close(self) -> None:
        """Close HTTP client"""
        await self._http_client.aclose()

    async def __aenter__(self) -> "AsyncAITraceClient":
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.close()


# Async API classes
class AsyncEventsAPI:
    """Async Events API"""

    def __init__(self, client: AsyncAITraceClient):
        self._client = client

    async def search(
        self,
        trace_id: Optional[str] = None,
        event_type: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> SearchEventsResponse:
        params: Dict[str, Any] = {"page": page, "page_size": page_size}
        if trace_id:
            params["trace_id"] = trace_id
        if event_type:
            params["event_type"] = event_type
        response = await self._client._request("GET", "/events/search", params=params)
        return SearchEventsResponse(**response)

    async def get(self, event_id: str) -> Event:
        response = await self._client._request("GET", f"/events/{event_id}")
        return Event(**response)

    async def ingest(self, events: List[Dict[str, Any]]) -> IngestEventsResponse:
        response = await self._client._request("POST", "/events/ingest", json={"events": events})
        return IngestEventsResponse(**response)


class AsyncCertsAPI:
    """Async Certificates API"""

    def __init__(self, client: AsyncAITraceClient):
        self._client = client

    async def search(self, page: int = 1, page_size: int = 20) -> Dict[str, Any]:
        params = {"page": page, "page_size": page_size}
        return await self._client._request("GET", "/certs/search", params=params)

    async def get(self, cert_id: str) -> Certificate:
        response = await self._client._request("GET", f"/certs/{cert_id}")
        return Certificate(**response)

    async def commit(
        self, trace_id: str, evidence_level: Optional[str] = None
    ) -> CommitCertResponse:
        data: Dict[str, Any] = {"trace_id": trace_id}
        if evidence_level:
            data["evidence_level"] = evidence_level
        response = await self._client._request("POST", "/certs/commit", json=data)
        return CommitCertResponse(**response)

    async def verify(
        self, cert_id: Optional[str] = None, root_hash: Optional[str] = None
    ) -> VerifyResult:
        data: Dict[str, Any] = {}
        if cert_id:
            data["cert_id"] = cert_id
        if root_hash:
            data["root_hash"] = root_hash
        response = await self._client._request("POST", "/certs/verify", json=data)
        return VerifyResult(**response)

    async def generate_proof(
        self, cert_id: str, disclose_events: List[int], disclose_fields: Optional[List[str]] = None
    ) -> MinimalDisclosureProof:
        data = {"disclose_events": disclose_events, "disclose_fields": disclose_fields or []}
        response = await self._client._request("POST", f"/certs/{cert_id}/prove", json=data)
        return MinimalDisclosureProof(**response)


class AsyncChatAPI:
    """Async Chat API"""

    def __init__(self, client: AsyncAITraceClient):
        self._client = client

    async def completions(
        self,
        messages: List[Dict[str, str]],
        model: str = "gpt-3.5-turbo",
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
        data: Dict[str, Any] = {"model": model, "messages": messages}
        if temperature is not None:
            data["temperature"] = temperature
        if max_tokens is not None:
            data["max_tokens"] = max_tokens
        data.update(kwargs)
        return await self._client._request("POST", "/chat/completions", json=data)
