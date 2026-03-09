"""Tests for AI-Trace client."""

import pytest
from unittest.mock import MagicMock, patch

from ai_trace import AITrace, EvidenceLevel
from ai_trace.exceptions import AuthenticationError, NotFoundError, ValidationError


class TestAITraceClient:
    """Test cases for AITrace client."""

    def test_client_initialization(self):
        """Test client can be initialized with default values."""
        client = AITrace()
        assert client.server_url == "http://localhost:8006"
        assert client.tenant_id == "default"
        assert client.api_key is None
        client.close()

    def test_client_with_custom_values(self):
        """Test client initialization with custom values."""
        client = AITrace(
            server_url="http://custom:9000",
            api_key="test-key",
            tenant_id="my-tenant",
            timeout=60.0,
        )
        assert client.server_url == "http://custom:9000"
        assert client.api_key == "test-key"
        assert client.tenant_id == "my-tenant"
        assert client.timeout == 60.0
        client.close()

    def test_client_context_manager(self):
        """Test client can be used as context manager."""
        with AITrace() as client:
            assert client is not None

    def test_get_headers_without_api_key(self):
        """Test headers without API key."""
        client = AITrace()
        headers = client._get_headers()
        assert "Content-Type" in headers
        assert "X-Tenant-ID" in headers
        assert "X-API-Key" not in headers
        client.close()

    def test_get_headers_with_api_key(self):
        """Test headers with API key."""
        client = AITrace(api_key="test-key")
        headers = client._get_headers()
        assert headers["X-API-Key"] == "test-key"
        client.close()


class TestTracesAPI:
    """Test cases for Traces API."""

    @patch("ai_trace.client.AITrace._post")
    def test_create_trace(self, mock_post):
        """Test creating a trace."""
        mock_post.return_value = {
            "trace_id": "test-trace-123",
            "tenant_id": "default",
            "created_at": "2024-01-01T00:00:00Z",
            "event_count": 0,
            "status": "active",
        }

        client = AITrace()
        trace = client.traces.create(name="Test Trace")

        assert trace.id == "test-trace-123"
        mock_post.assert_called_once()
        client.close()

    @patch("ai_trace.client.AITrace._get")
    def test_get_trace(self, mock_get):
        """Test getting a trace."""
        mock_get.return_value = {
            "trace_id": "test-trace-123",
            "tenant_id": "default",
            "name": "Test Trace",
            "created_at": "2024-01-01T00:00:00Z",
            "event_count": 5,
            "status": "active",
        }

        client = AITrace()
        trace = client.traces.get("test-trace-123")

        assert trace.id == "test-trace-123"
        assert trace.name == "Test Trace"
        assert trace.event_count == 5
        client.close()


class TestEventsAPI:
    """Test cases for Events API."""

    @patch("ai_trace.client.AITrace._post")
    def test_add_event(self, mock_post):
        """Test adding an event."""
        mock_post.return_value = {
            "event_id": "event-456",
            "trace_id": "trace-123",
            "event_type": "input",
            "sequence": 0,
            "timestamp": "2024-01-01T00:00:00Z",
            "hash": "abc123",
        }

        client = AITrace()
        event = client.events.add(
            trace_id="trace-123",
            event_type="input",
            payload={"prompt": "Hello"},
        )

        assert event.id == "event-456"
        assert event.event_type == "input"
        client.close()


class TestCertsAPI:
    """Test cases for Certs API."""

    @patch("ai_trace.client.AITrace._post")
    def test_commit_certificate(self, mock_post):
        """Test committing a certificate."""
        mock_post.return_value = {
            "cert_id": "cert-789",
            "trace_id": "trace-123",
            "tenant_id": "default",
            "evidence_level": "L2",
            "root_hash": "abc123",
            "event_count": 5,
            "signature": "sig123",
            "created_at": "2024-01-01T00:00:00Z",
        }

        client = AITrace()
        cert = client.certs.commit(
            trace_id="trace-123",
            evidence_level=EvidenceLevel.L2,
        )

        assert cert.id == "cert-789"
        assert cert.evidence_level == EvidenceLevel.L2
        client.close()

    @patch("ai_trace.client.AITrace._post")
    def test_verify_certificate(self, mock_post):
        """Test verifying a certificate."""
        mock_post.return_value = {
            "valid": True,
            "cert_id": "cert-789",
            "evidence_level": "L2",
            "hash_valid": True,
            "signature_valid": True,
            "timestamp_valid": True,
            "verified_at": "2024-01-01T00:00:00Z",
        }

        client = AITrace()
        result = client.certs.verify("cert-789")

        assert result.valid is True
        assert result.hash_valid is True
        assert result.signature_valid is True
        client.close()


class TestExceptionHandling:
    """Test exception handling."""

    @patch("ai_trace.client.httpx.Client.get")
    def test_authentication_error(self, mock_get):
        """Test authentication error handling."""
        mock_response = MagicMock()
        mock_response.status_code = 401
        mock_response.json.return_value = {"error": "Invalid API key"}
        mock_get.return_value = mock_response

        client = AITrace()
        with pytest.raises(AuthenticationError):
            client.traces.get("test-trace")
        client.close()

    @patch("ai_trace.client.httpx.Client.get")
    def test_not_found_error(self, mock_get):
        """Test not found error handling."""
        mock_response = MagicMock()
        mock_response.status_code = 404
        mock_response.json.return_value = {"error": "Trace not found"}
        mock_get.return_value = mock_response

        client = AITrace()
        with pytest.raises(NotFoundError):
            client.traces.get("nonexistent")
        client.close()

    @patch("ai_trace.client.httpx.Client.post")
    def test_validation_error(self, mock_post):
        """Test validation error handling."""
        mock_response = MagicMock()
        mock_response.status_code = 400
        mock_response.json.return_value = {"error": "Invalid input"}
        mock_post.return_value = mock_response

        client = AITrace()
        with pytest.raises(ValidationError):
            client.traces.create(name="")
        client.close()
