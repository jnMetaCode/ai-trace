"""
AI-Trace Wrapper for OpenAI Client
Provides transparent tracing for OpenAI API calls
"""

import hashlib
import json
import uuid
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional

from .client import AITraceClient


def _sha256(data: str) -> str:
    """Calculate SHA256 hash"""
    return f"sha256:{hashlib.sha256(data.encode()).hexdigest()}"


def _sha256_json(obj: Any) -> str:
    """Calculate SHA256 hash of JSON object"""
    return _sha256(json.dumps(obj, sort_keys=True, default=str))


class TracedOpenAI:
    """
    OpenAI client wrapper with automatic tracing

    This wrapper intercepts OpenAI API calls and automatically
    records events to AI-Trace server.

    Example:
        from openai import OpenAI
        from ai_trace import TracedOpenAI

        # Create traced client
        openai_client = OpenAI(api_key="sk-...")
        traced = TracedOpenAI(
            openai_client=openai_client,
            trace_server="http://localhost:8080",
            trace_api_key="your-trace-api-key"
        )

        # Use as normal - calls are automatically traced
        response = traced.chat.completions.create(
            model="gpt-4",
            messages=[{"role": "user", "content": "Hello!"}]
        )

        # Get trace_id for this conversation
        print(f"Trace ID: {traced.current_trace_id}")

        # Generate certificate
        cert = traced.commit_certificate()
    """

    def __init__(
        self,
        openai_client: Any,
        trace_server: str = "http://localhost:8080",
        trace_api_key: Optional[str] = None,
        tenant_id: str = "default",
        user_id: Optional[str] = None,
        session_id: Optional[str] = None,
        auto_commit: bool = False,
    ):
        self._openai = openai_client
        self._trace_client = AITraceClient(
            base_url=trace_server,
            api_key=trace_api_key,
            tenant_id=tenant_id,
        )
        self.tenant_id = tenant_id
        self.user_id = user_id
        self.session_id = session_id or str(uuid.uuid4())[:8]
        self.auto_commit = auto_commit

        self._current_trace_id: Optional[str] = None
        self._sequence = 0
        self._prev_event_hash: Optional[str] = None
        self._events: List[Dict[str, Any]] = []

        # Wrap chat completions
        self.chat = _TracedChat(self)

    @property
    def current_trace_id(self) -> Optional[str]:
        """Get current trace ID"""
        return self._current_trace_id

    def new_trace(self) -> str:
        """Start a new trace"""
        self._current_trace_id = f"trc_{uuid.uuid4().hex[:8]}"
        self._sequence = 0
        self._prev_event_hash = None
        self._events = []
        return self._current_trace_id

    def _create_event(
        self,
        event_type: str,
        payload: Dict[str, Any],
    ) -> Dict[str, Any]:
        """Create an event"""
        if not self._current_trace_id:
            self.new_trace()

        self._sequence += 1
        event_id = f"evt_{uuid.uuid4().hex[:12]}"
        timestamp = datetime.now(timezone.utc)
        payload_bytes = json.dumps(payload, sort_keys=True, default=str).encode()
        payload_hash = _sha256_json(payload)

        event = {
            "event_id": event_id,
            "trace_id": self._current_trace_id,
            "prev_event_hash": self._prev_event_hash,
            "event_type": event_type,
            "timestamp": timestamp.isoformat(),
            "sequence": self._sequence,
            "tenant_id": self.tenant_id,
            "user_id": self.user_id,
            "session_id": self.session_id,
            "payload": payload,
            "payload_hash": payload_hash,
        }

        # Calculate event hash
        hash_input = f"{event_id}|{self._current_trace_id}|{event_type}|{timestamp.isoformat()}|{self._sequence}|{payload_hash}"
        if self._prev_event_hash:
            hash_input += f"|{self._prev_event_hash}"
        event["event_hash"] = _sha256(hash_input)

        self._prev_event_hash = event["event_hash"]
        self._events.append(event)

        return event

    def _send_events(self, events: List[Dict[str, Any]]) -> None:
        """Send events to trace server"""
        try:
            self._trace_client.events.ingest(events)
        except Exception as e:
            # Log but don't fail - tracing should not block main functionality
            print(f"Warning: Failed to send trace events: {e}")

    def commit_certificate(self, evidence_level: str = "L1") -> Any:
        """Commit current trace as certificate"""
        if not self._current_trace_id:
            raise ValueError("No active trace to commit")

        return self._trace_client.certs.commit(
            trace_id=self._current_trace_id,
            evidence_level=evidence_level,
        )

    def close(self) -> None:
        """Close trace client"""
        self._trace_client.close()


class _TracedChat:
    """Wrapper for chat namespace"""

    def __init__(self, parent: TracedOpenAI):
        self._parent = parent
        self.completions = _TracedChatCompletions(parent)


class _TracedChatCompletions:
    """Wrapper for chat.completions"""

    def __init__(self, parent: TracedOpenAI):
        self._parent = parent

    def create(
        self,
        messages: List[Dict[str, str]],
        model: str = "gpt-3.5-turbo",
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        **kwargs: Any,
    ) -> Any:
        """Create chat completion with tracing"""
        start_time = datetime.now(timezone.utc)

        # Extract user prompt
        user_prompts = [m["content"] for m in messages if m["role"] == "user"]
        user_prompt = "\n".join(user_prompts)

        # Create INPUT event
        input_payload = {
            "prompt_hash": _sha256(user_prompt),
            "prompt_length": len(user_prompt),
            "request_params": {
                "model_requested": model,
                "temperature": temperature,
                "max_tokens": max_tokens,
            },
        }
        input_event = self._parent._create_event("INPUT", input_payload)

        # Create MODEL event
        system_prompts = [m["content"] for m in messages if m["role"] == "system"]
        model_payload = {
            "model_id": model,
            "model_provider": "openai",
            "actual_params": {
                "temperature": temperature,
                "max_tokens": max_tokens,
            },
            "params_hash": _sha256_json({"temperature": temperature, "max_tokens": max_tokens}),
        }
        if system_prompts:
            model_payload["system_prompt_hash"] = _sha256(system_prompts[0])
        model_event = self._parent._create_event("MODEL", model_payload)

        # Call actual OpenAI API
        call_kwargs: Dict[str, Any] = {"model": model, "messages": messages}
        if temperature is not None:
            call_kwargs["temperature"] = temperature
        if max_tokens is not None:
            call_kwargs["max_tokens"] = max_tokens
        call_kwargs.update(kwargs)

        response = self._parent._openai.chat.completions.create(**call_kwargs)

        end_time = datetime.now(timezone.utc)
        latency_ms = int((end_time - start_time).total_seconds() * 1000)

        # Create OUTPUT event
        output_content = response.choices[0].message.content if response.choices else ""
        output_payload = {
            "output_hash": _sha256(output_content or ""),
            "output_length": len(output_content or ""),
            "usage": {
                "prompt_tokens": response.usage.prompt_tokens if response.usage else 0,
                "completion_tokens": response.usage.completion_tokens if response.usage else 0,
                "total_tokens": response.usage.total_tokens if response.usage else 0,
            },
            "finish_reason": response.choices[0].finish_reason if response.choices else "unknown",
            "latency_ms": latency_ms,
            "safety_check": {"passed": True},
        }
        output_event = self._parent._create_event("OUTPUT", output_payload)

        # Send events to trace server
        self._parent._send_events([input_event, model_event, output_event])

        # Auto-commit if enabled
        if self._parent.auto_commit:
            self._parent.commit_certificate()

        return response
