"""
AI-Trace data models
"""

from datetime import datetime
from enum import Enum
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class EventType(str, Enum):
    """Event types"""

    INPUT = "INPUT"
    MODEL = "MODEL"
    RETRIEVAL = "RETRIEVAL"
    TOOL_CALL = "TOOL_CALL"
    OUTPUT = "OUTPUT"
    POST_EDIT = "POST_EDIT"


class EvidenceLevel(str, Enum):
    """Evidence level"""

    L1 = "L1"  # Local signature
    L2 = "L2"  # WORM + TSA
    L3 = "L3"  # Blockchain


class EventContext(BaseModel):
    """Event context"""

    business_id: Optional[str] = None
    business_type: Optional[str] = None
    department: Optional[str] = None
    client_ip: Optional[str] = None
    client_type: Optional[str] = None


class Event(BaseModel):
    """Event model"""

    event_id: str
    trace_id: str
    parent_event_id: Optional[str] = None
    prev_event_hash: Optional[str] = None
    event_type: EventType
    timestamp: datetime
    sequence: int
    tenant_id: str
    user_id: Optional[str] = None
    session_id: Optional[str] = None
    context: Optional[EventContext] = None
    payload: Dict[str, Any]
    payload_hash: str
    event_hash: str


class TimeProof(BaseModel):
    """Time proof"""

    proof_type: str
    timestamp: datetime
    tsa_authority: Optional[str] = None
    tsa_token: Optional[str] = None
    signature: Optional[str] = None


class BlockchainProof(BaseModel):
    """Blockchain proof"""

    chain_id: str
    tx_hash: str
    block_height: int
    contract_address: Optional[str] = None


class AnchorProof(BaseModel):
    """Anchor proof"""

    anchor_type: str
    anchor_id: str
    storage_provider: Optional[str] = None
    object_key: Optional[str] = None
    anchor_timestamp: datetime
    blockchain: Optional[BlockchainProof] = None


class CertMetadata(BaseModel):
    """Certificate metadata"""

    tenant_id: str
    created_at: datetime
    created_by: str
    evidence_level: EvidenceLevel


class Certificate(BaseModel):
    """Certificate model"""

    cert_id: str
    cert_version: str
    schema_version: str
    trace_id: str
    event_hashes: List[str]
    root_hash: str
    time_proof: Optional[TimeProof] = None
    anchor_proof: Optional[AnchorProof] = None
    metadata: CertMetadata


class VerifyCheck(BaseModel):
    """Verification check result"""

    passed: bool
    message: Optional[str] = None


class VerifyResult(BaseModel):
    """Verification result"""

    valid: bool
    checks: Dict[str, VerifyCheck]
    certificate: Optional[Certificate] = None


class ProofNode(BaseModel):
    """Merkle proof node"""

    hash: str
    position: str  # "left" or "right"


class EventMerkleProof(BaseModel):
    """Event Merkle proof"""

    event_index: int
    event_hash: str
    proof_path: List[ProofNode]


class DisclosedEvent(BaseModel):
    """Disclosed event in proof"""

    event_index: int
    event_type: str
    event_hash: str
    disclosed_fields: Dict[str, Any]


class VerificationInstructions(BaseModel):
    """Verification instructions"""

    verifier_url: str
    verify_command: str


class MinimalDisclosureProof(BaseModel):
    """Minimal disclosure proof"""

    schema_version: str
    cert_id: str
    root_hash: str
    disclosed_events: List[DisclosedEvent]
    merkle_proofs: List[EventMerkleProof]
    time_proof: Optional[TimeProof] = None
    anchor_proof: Optional[AnchorProof] = None
    verification_instructions: Optional[VerificationInstructions] = None


# Request/Response models


class IngestEventRequest(BaseModel):
    """Ingest event request"""

    events: List[Dict[str, Any]]


class IngestEventResult(BaseModel):
    """Single event ingest result"""

    event_id: str
    event_hash: Optional[str] = None
    error: Optional[str] = None


class IngestEventsResponse(BaseModel):
    """Ingest events response"""

    success: bool
    results: List[IngestEventResult]


class SearchEventsResponse(BaseModel):
    """Search events response"""

    events: List[Dict[str, Any]]
    page: int
    size: int


class CommitCertRequest(BaseModel):
    """Commit certificate request"""

    trace_id: str
    evidence_level: Optional[str] = None


class CommitCertResponse(BaseModel):
    """Commit certificate response"""

    cert_id: str
    trace_id: str
    root_hash: str
    event_count: int
    evidence_level: str
    time_proof: TimeProof
    anchor_proof: AnchorProof
    created_at: datetime


class GenerateProofRequest(BaseModel):
    """Generate proof request"""

    disclose_events: List[int]
    disclose_fields: List[str] = Field(default_factory=list)
