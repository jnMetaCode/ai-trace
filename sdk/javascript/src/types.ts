/**
 * AI-Trace SDK Type Definitions
 */

// Evidence Levels
export const EvidenceLevel = {
  L1: 'L1',
  L2: 'L2',
  L3: 'L3',
} as const;

export type EvidenceLevelType = typeof EvidenceLevel[keyof typeof EvidenceLevel];

// Event Types
export const EventType = {
  INPUT: 'llm.input',
  OUTPUT: 'llm.output',
  CHUNK: 'llm.chunk',
  TOOL_CALL: 'llm.tool_call',
  TOOL_RESULT: 'llm.tool_result',
  ERROR: 'llm.error',
} as const;

export type EventTypeValue = typeof EventType[keyof typeof EventType];

// Client Options
export interface AITraceClientOptions {
  apiKey: string;
  baseUrl?: string;
  upstreamApiKey?: string;
  upstreamBaseUrl?: string;
  timeout?: number;
}

// Message
export interface Message {
  role: 'system' | 'user' | 'assistant' | 'function' | 'tool';
  content: string;
  name?: string;
  function_call?: {
    name: string;
    arguments: string;
  };
  tool_calls?: ToolCall[];
}

export interface ToolCall {
  id: string;
  type: 'function';
  function: {
    name: string;
    arguments: string;
  };
}

// Chat Request
export interface ChatRequest {
  model: string;
  messages: Message[];
  temperature?: number;
  maxTokens?: number;
  topP?: number;
  n?: number;
  stream?: boolean;
  stop?: string | string[];
  traceId?: string;
  sessionId?: string;
  businessId?: string;
  [key: string]: unknown;
}

// Chat Response
export interface ChatResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: ChatChoice[];
  usage?: Usage;
  trace_id: string;
}

export interface ChatChoice {
  index: number;
  message: Message;
  finish_reason: string;
}

export interface Usage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

// Event
export interface Event {
  event_id?: string;
  trace_id: string;
  event_type: EventTypeValue | string;
  timestamp?: string;
  sequence?: number;
  payload: Record<string, unknown>;
  prev_event_hash?: string;
  prev_event_hashes?: string[];
  event_hash?: string;
  payload_hash?: string;
}

// Event Ingest Response
export interface IngestResponse {
  ingested: number;
  event_ids: string[];
}

// Event Search
export interface EventSearchParams {
  traceId?: string;
  eventType?: string;
  startTime?: string;
  endTime?: string;
  page?: number;
  pageSize?: number;
}

export interface EventSearchResponse {
  events: Event[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Certificate
export interface Certificate {
  cert_id: string;
  trace_id: string;
  root_hash: string;
  event_count: number;
  evidence_level: EvidenceLevelType;
  created_at: string;
  time_proof?: TimeProof;
  anchor_proof?: AnchorProof;
}

export interface TimeProof {
  timestamp: string;
  timestamp_id: string;
  tsa_name: string;
  tsa_hash: string;
}

export interface AnchorProof {
  tx_hash: string;
  block_number: number;
  chain_id: number;
  contract_address: string;
}

// Verification
export interface VerificationResult {
  valid: boolean;
  checks: VerificationChecks;
  certificate?: Certificate;
}

export interface VerificationChecks {
  merkle_root: boolean;
  timestamp: boolean;
  event_hashes: boolean;
  causal_chain: boolean;
}

// Proof
export interface ProveParams {
  discloseEvents?: number[];
  discloseFields?: string[];
}

export interface ProofResponse {
  cert_id: string;
  root_hash: string;
  disclosed_events: Event[];
  merkle_proofs: MerkleProof[];
  metadata: Record<string, unknown>;
}

export interface MerkleProof {
  event_index: number;
  siblings: string[];
  direction: number[];
}

// Certificate Search
export interface CertSearchParams {
  page?: number;
  pageSize?: number;
}

export interface CertSearchResponse {
  certificates: Certificate[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Chat and Cert Result
export interface ChatAndCertResult {
  chatResponse: ChatResponse;
  certificate: Certificate;
}

// API Error
export interface APIError {
  code: string;
  message: string;
}
