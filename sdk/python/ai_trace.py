"""
AI-Trace Python SDK

Usage:
    from ai_trace import AITraceClient

    client = AITraceClient(
        api_key="your-api-key",
        base_url="https://api.ai-trace.dev"
    )

    # Use as OpenAI drop-in replacement
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "Hello!"}]
    )

    # Get trace certificate
    cert = client.certs.commit(trace_id=response.trace_id, evidence_level="L2")
"""

import hashlib
import json
import time
from dataclasses import dataclass
from typing import Any, Dict, List, Optional
from urllib.parse import urljoin

import requests


@dataclass
class Certificate:
    """存证证书"""
    cert_id: str
    trace_id: str
    root_hash: str
    event_count: int
    evidence_level: str
    created_at: str
    time_proof: Optional[Dict] = None
    anchor_proof: Optional[Dict] = None


@dataclass
class VerificationResult:
    """验证结果"""
    valid: bool
    checks: Dict[str, Any]
    certificate: Optional[Certificate] = None


class AITraceClient:
    """AI-Trace 客户端"""

    def __init__(
        self,
        api_key: str,
        base_url: str = "https://api.ai-trace.dev",
        upstream_api_key: Optional[str] = None,
        upstream_base_url: Optional[str] = None,
        timeout: int = 120
    ):
        """
        初始化客户端

        Args:
            api_key: AI-Trace 平台 API Key
            base_url: AI-Trace 服务地址
            upstream_api_key: 上游 API Key (OpenAI/Claude)，透传不存储
            upstream_base_url: 自定义上游代理 URL
            timeout: 请求超时时间
        """
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.upstream_api_key = upstream_api_key
        self.upstream_base_url = upstream_base_url
        self.timeout = timeout
        self.session = requests.Session()

        # 子模块
        self.chat = ChatCompletions(self)
        self.events = Events(self)
        self.certs = Certificates(self)

    def _request(
        self,
        method: str,
        path: str,
        json_data: Optional[Dict] = None,
        params: Optional[Dict] = None,
        extra_headers: Optional[Dict] = None
    ) -> Dict:
        """发送请求"""
        url = urljoin(self.base_url, path)

        headers = {
            "X-API-Key": self.api_key,
            "Content-Type": "application/json"
        }

        # 添加上游配置
        if self.upstream_api_key:
            headers["X-Upstream-API-Key"] = self.upstream_api_key
        if self.upstream_base_url:
            headers["X-Upstream-Base-URL"] = self.upstream_base_url

        if extra_headers:
            headers.update(extra_headers)

        response = self.session.request(
            method=method,
            url=url,
            json=json_data,
            params=params,
            headers=headers,
            timeout=self.timeout
        )

        response.raise_for_status()
        return response.json()


class ChatCompletions:
    """聊天完成接口（OpenAI 兼容）"""

    def __init__(self, client: AITraceClient):
        self.client = client
        self._last_trace_id = None

    def create(
        self,
        model: str,
        messages: List[Dict[str, str]],
        temperature: float = 1.0,
        max_tokens: Optional[int] = None,
        trace_id: Optional[str] = None,
        session_id: Optional[str] = None,
        business_id: Optional[str] = None,
        **kwargs
    ) -> Dict:
        """
        创建聊天完成

        Args:
            model: 模型名称 (gpt-4, claude-3-opus, etc.)
            messages: 消息列表
            temperature: 温度参数
            max_tokens: 最大 token 数
            trace_id: 追踪 ID（可选，自动生成）
            session_id: 会话 ID
            business_id: 业务 ID

        Returns:
            OpenAI 兼容的响应
        """
        headers = {}
        if trace_id:
            headers["X-Trace-ID"] = trace_id
        if session_id:
            headers["X-Session-ID"] = session_id
        if business_id:
            headers["X-Business-ID"] = business_id

        data = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            **kwargs
        }
        if max_tokens:
            data["max_tokens"] = max_tokens

        response = self.client._request(
            "POST",
            "/api/v1/chat/completions",
            json_data=data,
            extra_headers=headers
        )

        # 保存 trace_id 用于后续存证
        self._last_trace_id = headers.get("X-Trace-ID")

        return response

    @property
    def last_trace_id(self) -> Optional[str]:
        """获取最后一次请求的 trace_id"""
        return self._last_trace_id


class Events:
    """事件接口"""

    def __init__(self, client: AITraceClient):
        self.client = client

    def ingest(self, events: List[Dict]) -> Dict:
        """批量写入事件"""
        return self.client._request(
            "POST",
            "/api/v1/events/ingest",
            json_data={"events": events}
        )

    def search(
        self,
        trace_id: Optional[str] = None,
        event_type: Optional[str] = None,
        start_time: Optional[str] = None,
        end_time: Optional[str] = None,
        page: int = 1,
        page_size: int = 20
    ) -> Dict:
        """搜索事件"""
        params = {"page": page, "page_size": page_size}
        if trace_id:
            params["trace_id"] = trace_id
        if event_type:
            params["event_type"] = event_type
        if start_time:
            params["start_time"] = start_time
        if end_time:
            params["end_time"] = end_time

        return self.client._request("GET", "/api/v1/events/search", params=params)

    def get(self, event_id: str) -> Dict:
        """获取事件详情"""
        return self.client._request("GET", f"/api/v1/events/{event_id}")


class Certificates:
    """存证接口"""

    def __init__(self, client: AITraceClient):
        self.client = client

    def commit(self, trace_id: str, evidence_level: str = "L1") -> Certificate:
        """
        生成存证证书

        Args:
            trace_id: 追踪 ID
            evidence_level: 存证级别 (L1/L2/L3)

        Returns:
            存证证书
        """
        response = self.client._request(
            "POST",
            "/api/v1/certs/commit",
            json_data={
                "trace_id": trace_id,
                "evidence_level": evidence_level
            }
        )

        return Certificate(
            cert_id=response["cert_id"],
            trace_id=response["trace_id"],
            root_hash=response["root_hash"],
            event_count=response["event_count"],
            evidence_level=response["evidence_level"],
            created_at=response["created_at"],
            time_proof=response.get("time_proof"),
            anchor_proof=response.get("anchor_proof")
        )

    def verify(
        self,
        cert_id: Optional[str] = None,
        root_hash: Optional[str] = None
    ) -> VerificationResult:
        """
        验证存证证书

        Args:
            cert_id: 证书 ID
            root_hash: 根哈希

        Returns:
            验证结果
        """
        data = {}
        if cert_id:
            data["cert_id"] = cert_id
        if root_hash:
            data["root_hash"] = root_hash

        response = self.client._request(
            "POST",
            "/api/v1/certs/verify",
            json_data=data
        )

        return VerificationResult(
            valid=response["valid"],
            checks=response["checks"],
            certificate=Certificate(**response["certificate"]) if response.get("certificate") else None
        )

    def search(self, page: int = 1, page_size: int = 20) -> Dict:
        """搜索存证"""
        return self.client._request(
            "GET",
            "/api/v1/certs/search",
            params={"page": page, "page_size": page_size}
        )

    def get(self, cert_id: str) -> Certificate:
        """获取存证详情"""
        response = self.client._request("GET", f"/api/v1/certs/{cert_id}")
        return Certificate(**response)

    def prove(
        self,
        cert_id: str,
        disclose_events: List[int],
        disclose_fields: Optional[List[str]] = None
    ) -> Dict:
        """
        生成最小披露证明

        Args:
            cert_id: 证书 ID
            disclose_events: 要披露的事件索引
            disclose_fields: 要披露的字段

        Returns:
            最小披露证明
        """
        return self.client._request(
            "POST",
            f"/api/v1/certs/{cert_id}/prove",
            json_data={
                "disclose_events": disclose_events,
                "disclose_fields": disclose_fields or []
            }
        )


# 便捷函数
def create_client(
    api_key: str,
    base_url: str = "https://api.ai-trace.dev",
    upstream_api_key: Optional[str] = None
) -> AITraceClient:
    """创建 AI-Trace 客户端"""
    return AITraceClient(
        api_key=api_key,
        base_url=base_url,
        upstream_api_key=upstream_api_key
    )


def hash_content(content: str) -> str:
    """计算内容哈希"""
    return hashlib.sha256(content.encode()).hexdigest()


# 使用示例
if __name__ == "__main__":
    # 创建客户端
    client = create_client(
        api_key="your-ai-trace-api-key",
        base_url="http://localhost:8080",
        upstream_api_key="sk-your-openai-key"  # 透传，不存储
    )

    # 调用 AI
    print("Calling AI...")
    response = client.chat.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "What is 2+2?"}],
        trace_id="demo_trace_001"
    )
    print(f"Response: {response['choices'][0]['message']['content']}")

    # 生成存证
    print("\nCommitting certificate...")
    cert = client.certs.commit(
        trace_id="demo_trace_001",
        evidence_level="L2"  # WORM 存储
    )
    print(f"Certificate ID: {cert.cert_id}")
    print(f"Root Hash: {cert.root_hash}")

    # 验证存证
    print("\nVerifying certificate...")
    result = client.certs.verify(cert_id=cert.cert_id)
    print(f"Valid: {result.valid}")
    print(f"Checks: {json.dumps(result.checks, indent=2)}")
