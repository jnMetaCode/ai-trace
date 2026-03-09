"""
Tests for AI-Trace Client
"""

import pytest
from unittest.mock import Mock, patch

from ai_trace import AITraceClient
from ai_trace.models import SearchEventsResponse, Event, EventType


class TestAITraceClient:
    """Test AITraceClient"""

    def test_init_default(self):
        """Test default initialization"""
        client = AITraceClient()
        assert client.base_url == "http://localhost:8080"
        assert client.tenant_id == "default"
        assert client.api_key is None
        client.close()

    def test_init_custom(self):
        """Test custom initialization"""
        client = AITraceClient(
            base_url="http://custom:9000",
            api_key="test-key",
            tenant_id="tenant1",
            timeout=60.0,
        )
        assert client.base_url == "http://custom:9000"
        assert client.api_key == "test-key"
        assert client.tenant_id == "tenant1"
        client.close()

    def test_get_headers(self):
        """Test headers generation"""
        client = AITraceClient(api_key="test-key", tenant_id="tenant1")
        headers = client._get_headers()

        assert headers["Content-Type"] == "application/json"
        assert headers["X-API-Key"] == "test-key"
        assert headers["X-Tenant-ID"] == "tenant1"
        client.close()

    def test_get_headers_no_api_key(self):
        """Test headers without API key"""
        client = AITraceClient()
        headers = client._get_headers()

        assert "X-API-Key" not in headers
        assert headers["X-Tenant-ID"] == "default"
        client.close()

    def test_context_manager(self):
        """Test context manager"""
        with AITraceClient() as client:
            assert client is not None


class TestEventsAPI:
    """Test Events API"""

    @patch.object(AITraceClient, '_request')
    def test_search(self, mock_request):
        """Test events search"""
        mock_request.return_value = {
            "events": [],
            "page": 1,
            "size": 0
        }

        with AITraceClient() as client:
            result = client.events.search(trace_id="trc_123")

        mock_request.assert_called_once()
        assert isinstance(result, SearchEventsResponse)
        assert result.page == 1

    @patch.object(AITraceClient, '_request')
    def test_search_with_filters(self, mock_request):
        """Test events search with filters"""
        mock_request.return_value = {
            "events": [],
            "page": 1,
            "size": 0
        }

        with AITraceClient() as client:
            client.events.search(
                trace_id="trc_123",
                event_type="INPUT",
                page=2,
                page_size=50
            )

        call_args = mock_request.call_args
        params = call_args[1]["params"]
        assert params["trace_id"] == "trc_123"
        assert params["event_type"] == "INPUT"
        assert params["page"] == 2
        assert params["page_size"] == 50


class TestCertsAPI:
    """Test Certificates API"""

    @patch.object(AITraceClient, '_request')
    def test_commit(self, mock_request):
        """Test certificate commit"""
        mock_request.return_value = {
            "cert_id": "cert_123",
            "trace_id": "trc_123",
            "root_hash": "sha256:abc",
            "event_count": 3,
            "evidence_level": "L1",
            "time_proof": {
                "proof_type": "local",
                "timestamp": "2024-01-01T00:00:00Z"
            },
            "anchor_proof": {
                "anchor_type": "local",
                "anchor_id": "anchor_123",
                "anchor_timestamp": "2024-01-01T00:00:00Z"
            },
            "created_at": "2024-01-01T00:00:00Z"
        }

        with AITraceClient() as client:
            result = client.certs.commit(trace_id="trc_123")

        assert result.cert_id == "cert_123"
        assert result.evidence_level == "L1"

    @patch.object(AITraceClient, '_request')
    def test_verify(self, mock_request):
        """Test certificate verify"""
        mock_request.return_value = {
            "valid": True,
            "checks": {
                "hash_integrity": {"passed": True, "message": "OK"}
            }
        }

        with AITraceClient() as client:
            result = client.certs.verify(cert_id="cert_123")

        assert result.valid is True
        assert result.checks["hash_integrity"].passed is True


class TestModels:
    """Test data models"""

    def test_event_type_enum(self):
        """Test EventType enum"""
        assert EventType.INPUT == "INPUT"
        assert EventType.OUTPUT == "OUTPUT"
        assert EventType.MODEL == "MODEL"

    def test_search_events_response(self):
        """Test SearchEventsResponse model"""
        response = SearchEventsResponse(
            events=[{"event_id": "evt_1"}],
            page=1,
            size=1
        )
        assert response.page == 1
        assert len(response.events) == 1
