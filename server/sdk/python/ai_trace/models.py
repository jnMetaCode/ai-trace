"""
Data models for AI-Trace SDK.
"""

from datetime import datetime
from enum import Enum
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class EvidenceLevel(str, Enum):
    """Evidence level for certificates."""

    L1 = "L1"  # Local signature
    L2 = "L2"  # WORM storage + TSA
    L3 = "L3"  # Blockchain anchor


class ChainType(str, Enum):
    """Blockchain type for L3 anchoring."""

    ETHEREUM = "ethereum"
    POLYGON = "polygon"
    ARBITRUM = "arbitrum"


class Trace(BaseModel):
    """Represents an AI decision trace."""

    id: str = Field(..., alias="trace_id", description="Unique trace identifier")
    tenant_id: str = Field(default="default", description="Tenant identifier")
    name: Optional[str] = Field(None, description="Human-readable trace name")
    user_id: Optional[str] = Field(None, description="User who initiated the trace")
    session_id: Optional[str] = Field(None, description="Session identifier")
    created_at: datetime = Field(..., description="Trace creation timestamp")
    event_count: int = Field(default=0, description="Number of events in trace")
    status: str = Field(default="active", description="Trace status")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="Additional metadata")

    class Config:
        populate_by_name = True


class Event(BaseModel):
    """Represents an event within a trace."""

    id: str = Field(..., alias="event_id", description="Unique event identifier")
    trace_id: str = Field(..., description="Parent trace identifier")
    event_type: str = Field(..., description="Event type (input, output, custom)")
    sequence: int = Field(..., description="Event sequence number")
    timestamp: datetime = Field(..., description="Event timestamp")
    hash: str = Field(..., description="Event content hash")
    payload: Dict[str, Any] = Field(default_factory=dict, description="Event payload data")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="Event metadata")

    class Config:
        populate_by_name = True


class MerkleProof(BaseModel):
    """Merkle proof for an event."""

    leaf_hash: str = Field(..., description="Hash of the leaf node")
    siblings: List[str] = Field(..., description="Sibling hashes for proof")
    path: List[int] = Field(..., description="Path indices (0=left, 1=right)")
    root_hash: str = Field(..., description="Root hash to verify against")


class Certificate(BaseModel):
    """Represents an attestation certificate."""

    id: str = Field(..., alias="cert_id", description="Unique certificate identifier")
    trace_id: str = Field(..., description="Associated trace identifier")
    tenant_id: str = Field(default="default", description="Tenant identifier")
    evidence_level: EvidenceLevel = Field(..., description="Evidence level")
    root_hash: str = Field(..., description="Merkle root hash")
    event_count: int = Field(..., description="Number of events in certificate")
    signature: str = Field(..., description="Digital signature")
    created_at: datetime = Field(..., description="Certificate creation timestamp")
    expires_at: Optional[datetime] = Field(None, description="Certificate expiration")

    # L2 specific fields
    tsa_timestamp: Optional[str] = Field(None, description="TSA timestamp token")
    worm_location: Optional[str] = Field(None, description="WORM storage location")

    # L3 specific fields
    chain_type: Optional[ChainType] = Field(None, description="Blockchain type")
    tx_hash: Optional[str] = Field(None, description="Blockchain transaction hash")
    block_number: Optional[int] = Field(None, description="Block number")
    anchor_timestamp: Optional[datetime] = Field(None, description="Anchor timestamp")

    metadata: Dict[str, Any] = Field(default_factory=dict, description="Additional metadata")

    class Config:
        populate_by_name = True


class VerificationResult(BaseModel):
    """Result of certificate verification."""

    valid: bool = Field(..., description="Overall verification result")
    cert_id: str = Field(..., description="Certificate identifier")
    evidence_level: EvidenceLevel = Field(..., description="Evidence level")

    # Individual checks
    hash_valid: bool = Field(..., description="Merkle hash integrity check")
    signature_valid: bool = Field(..., description="Signature verification")
    timestamp_valid: bool = Field(..., description="Timestamp verification")
    anchor_verified: Optional[bool] = Field(None, description="Blockchain anchor verification")

    # Details
    verified_at: datetime = Field(..., description="Verification timestamp")
    details: Dict[str, Any] = Field(default_factory=dict, description="Verification details")
    errors: List[str] = Field(default_factory=list, description="Verification errors if any")


class Proof(BaseModel):
    """Minimal disclosure proof for specific events."""

    proof_id: str = Field(..., description="Unique proof identifier")
    cert_id: str = Field(..., description="Associated certificate identifier")
    root_hash: str = Field(..., description="Merkle root hash")
    created_at: datetime = Field(..., description="Proof creation timestamp")

    # Disclosed data
    event_indices: List[int] = Field(..., description="Indices of disclosed events")
    disclosed_events: List[Dict[str, Any]] = Field(..., description="Disclosed event data")
    merkle_proofs: List[MerkleProof] = Field(..., description="Merkle proofs for events")

    # Verification info
    verifiable: bool = Field(default=True, description="Whether proof can be verified")


class PaginatedResponse(BaseModel):
    """Paginated response wrapper."""

    items: List[Any] = Field(..., description="Response items")
    total: int = Field(..., description="Total number of items")
    limit: int = Field(..., description="Items per page")
    offset: int = Field(..., description="Current offset")
    has_more: bool = Field(..., description="Whether more items exist")


class TraceListResponse(PaginatedResponse):
    """Paginated trace list response."""

    items: List[Trace]


class EventListResponse(PaginatedResponse):
    """Paginated event list response."""

    items: List[Event]


class CertificateListResponse(PaginatedResponse):
    """Paginated certificate list response."""

    items: List[Certificate]
