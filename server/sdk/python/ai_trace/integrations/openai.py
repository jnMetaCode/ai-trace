"""
OpenAI integration for automatic tracing.

This module provides a drop-in replacement for the OpenAI client
that automatically traces all API calls.

Example:
    from ai_trace.integrations import TracedOpenAI

    client = TracedOpenAI(
        openai_api_key="sk-...",
        ai_trace_url="http://localhost:8006",
        ai_trace_key="your-key",
    )

    # All calls are automatically traced
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "Hello!"}]
    )

    # Access the trace
    print(f"Trace ID: {client.current_trace_id}")
"""

from datetime import datetime
from typing import Any, Dict, Iterator, List, Optional, Union

try:
    from openai import OpenAI
    from openai.types.chat import ChatCompletion, ChatCompletionChunk

    HAS_OPENAI = True
except ImportError:
    HAS_OPENAI = False
    OpenAI = object  # type: ignore

from ai_trace.client import AITrace
from ai_trace.models import EvidenceLevel


class TracedChatCompletions:
    """Traced wrapper for OpenAI chat completions."""

    def __init__(self, tracer: "TracedOpenAI") -> None:
        self._tracer = tracer

    def create(
        self,
        messages: List[Dict[str, Any]],
        model: str = "gpt-4",
        stream: bool = False,
        **kwargs: Any,
    ) -> Union["ChatCompletion", Iterator["ChatCompletionChunk"]]:
        """Create a chat completion with automatic tracing.

        Args:
            messages: Chat messages
            model: Model to use
            stream: Whether to stream the response
            **kwargs: Additional arguments passed to OpenAI

        Returns:
            Chat completion or stream of chunks
        """
        # Start a new trace if not in an existing one
        if not self._tracer._current_trace_id:
            trace = self._tracer._ai_trace.traces.create(
                name=f"OpenAI Chat - {model}",
                metadata={"model": model, "provider": "openai"},
            )
            self._tracer._current_trace_id = trace.id

        # Log input event
        self._tracer._ai_trace.events.add(
            trace_id=self._tracer._current_trace_id,
            event_type="input",
            payload={
                "messages": messages,
                "model": model,
                "parameters": kwargs,
            },
        )

        # Make the actual API call
        start_time = datetime.utcnow()

        if stream:
            return self._create_stream(messages, model, start_time, **kwargs)
        else:
            return self._create_sync(messages, model, start_time, **kwargs)

    def _create_sync(
        self,
        messages: List[Dict[str, Any]],
        model: str,
        start_time: datetime,
        **kwargs: Any,
    ) -> "ChatCompletion":
        """Create a synchronous chat completion."""
        response = self._tracer._openai.chat.completions.create(
            messages=messages,
            model=model,
            stream=False,
            **kwargs,
        )

        end_time = datetime.utcnow()
        duration_ms = (end_time - start_time).total_seconds() * 1000

        # Log output event
        self._tracer._ai_trace.events.add(
            trace_id=self._tracer._current_trace_id,
            event_type="output",
            payload={
                "response": response.model_dump(),
                "duration_ms": duration_ms,
                "usage": {
                    "prompt_tokens": response.usage.prompt_tokens if response.usage else 0,
                    "completion_tokens": response.usage.completion_tokens if response.usage else 0,
                    "total_tokens": response.usage.total_tokens if response.usage else 0,
                },
            },
        )

        return response

    def _create_stream(
        self,
        messages: List[Dict[str, Any]],
        model: str,
        start_time: datetime,
        **kwargs: Any,
    ) -> Iterator["ChatCompletionChunk"]:
        """Create a streaming chat completion."""
        stream = self._tracer._openai.chat.completions.create(
            messages=messages,
            model=model,
            stream=True,
            **kwargs,
        )

        collected_content = []
        for chunk in stream:
            if chunk.choices and chunk.choices[0].delta.content:
                collected_content.append(chunk.choices[0].delta.content)
            yield chunk

        end_time = datetime.utcnow()
        duration_ms = (end_time - start_time).total_seconds() * 1000

        # Log output event after stream completes
        self._tracer._ai_trace.events.add(
            trace_id=self._tracer._current_trace_id,
            event_type="output",
            payload={
                "response_content": "".join(collected_content),
                "duration_ms": duration_ms,
                "streamed": True,
            },
        )


class TracedChat:
    """Traced wrapper for OpenAI chat."""

    def __init__(self, tracer: "TracedOpenAI") -> None:
        self.completions = TracedChatCompletions(tracer)


class TracedOpenAI:
    """OpenAI client wrapper with automatic tracing.

    This provides a drop-in replacement for the OpenAI client
    that automatically traces all API calls to AI-Trace.

    Example:
        client = TracedOpenAI(
            openai_api_key="sk-...",
            ai_trace_url="http://localhost:8006",
            ai_trace_key="your-key",
        )

        response = client.chat.completions.create(
            model="gpt-4",
            messages=[{"role": "user", "content": "Hello!"}]
        )

        # Commit the trace to a certificate
        cert = client.commit_trace(evidence_level="L2")
    """

    def __init__(
        self,
        openai_api_key: str,
        ai_trace_url: str = "http://localhost:8006",
        ai_trace_key: Optional[str] = None,
        tenant_id: str = "default",
        auto_commit: bool = False,
        evidence_level: Union[EvidenceLevel, str] = EvidenceLevel.L1,
        **openai_kwargs: Any,
    ) -> None:
        """Initialize the traced OpenAI client.

        Args:
            openai_api_key: OpenAI API key
            ai_trace_url: AI-Trace server URL
            ai_trace_key: AI-Trace API key
            tenant_id: Tenant identifier
            auto_commit: Whether to auto-commit traces
            evidence_level: Default evidence level for commits
            **openai_kwargs: Additional arguments for OpenAI client
        """
        if not HAS_OPENAI:
            raise ImportError(
                "OpenAI package not installed. Install with: pip install ai-trace[openai]"
            )

        self._openai = OpenAI(api_key=openai_api_key, **openai_kwargs)
        self._ai_trace = AITrace(
            server_url=ai_trace_url,
            api_key=ai_trace_key,
            tenant_id=tenant_id,
        )

        self._current_trace_id: Optional[str] = None
        self._auto_commit = auto_commit
        self._evidence_level = evidence_level

        # Set up traced API endpoints
        self.chat = TracedChat(self)

    @property
    def current_trace_id(self) -> Optional[str]:
        """Get the current trace ID."""
        return self._current_trace_id

    def start_trace(self, name: Optional[str] = None, **metadata: Any) -> str:
        """Start a new trace.

        Args:
            name: Trace name
            **metadata: Additional metadata

        Returns:
            Trace ID
        """
        trace = self._ai_trace.traces.create(name=name, metadata=metadata)
        self._current_trace_id = trace.id
        return trace.id

    def end_trace(self) -> Optional[str]:
        """End the current trace.

        Returns:
            Trace ID that was ended
        """
        trace_id = self._current_trace_id
        self._current_trace_id = None
        return trace_id

    def commit_trace(
        self,
        trace_id: Optional[str] = None,
        evidence_level: Optional[Union[EvidenceLevel, str]] = None,
    ) -> Any:
        """Commit the current or specified trace to a certificate.

        Args:
            trace_id: Trace ID to commit (defaults to current)
            evidence_level: Evidence level (defaults to instance default)

        Returns:
            Certificate object
        """
        tid = trace_id or self._current_trace_id
        if not tid:
            raise ValueError("No trace to commit")

        level = evidence_level or self._evidence_level
        return self._ai_trace.certs.commit(trace_id=tid, evidence_level=level)

    def __enter__(self) -> "TracedOpenAI":
        return self

    def __exit__(self, *args: Any) -> None:
        if self._auto_commit and self._current_trace_id:
            self.commit_trace()
        self._ai_trace.close()
