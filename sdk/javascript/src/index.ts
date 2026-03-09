/**
 * AI-Trace JavaScript/TypeScript SDK
 *
 * Official SDK for the AI-Trace platform - Enterprise AI decision auditing
 * and tamper-proof attestation.
 *
 * @example
 * ```typescript
 * import { AITraceClient, EvidenceLevel } from '@ai-trace/sdk';
 *
 * const client = new AITraceClient({
 *   apiKey: 'your-api-key',
 *   upstreamApiKey: 'sk-your-openai-key'
 * });
 *
 * const response = await client.chat.create({
 *   model: 'gpt-4',
 *   messages: [{ role: 'user', content: 'Hello!' }]
 * });
 *
 * const cert = await client.certs.commit(response.trace_id, EvidenceLevel.L2);
 * ```
 */

import {
  AITraceClientOptions,
  ChatRequest,
  ChatResponse,
  Certificate,
  Event,
  EventType,
  EventTypeValue,
  EvidenceLevel,
  EvidenceLevelType,
  IngestResponse,
  EventSearchParams,
  EventSearchResponse,
  VerificationResult,
  ProveParams,
  ProofResponse,
  CertSearchParams,
  CertSearchResponse,
  ChatAndCertResult,
  Message,
} from './types';

export * from './types';

/**
 * Custom error class for AI-Trace API errors
 */
export class AITraceError extends Error {
  code: string;
  statusCode: number;

  constructor(message: string, code: string, statusCode: number) {
    super(message);
    this.name = 'AITraceError';
    this.code = code;
    this.statusCode = statusCode;
  }

  isClientError(): boolean {
    return this.statusCode >= 400 && this.statusCode < 500;
  }

  isServerError(): boolean {
    return this.statusCode >= 500;
  }
}

/**
 * Event builder for creating events programmatically
 */
export class EventBuilder {
  private event: Partial<Event> = {};

  traceId(traceId: string): EventBuilder {
    this.event.trace_id = traceId;
    return this;
  }

  eventType(eventType: EventTypeValue | string): EventBuilder {
    this.event.event_type = eventType;
    return this;
  }

  sequence(sequence: number): EventBuilder {
    this.event.sequence = sequence;
    return this;
  }

  timestamp(timestamp: string): EventBuilder {
    this.event.timestamp = timestamp;
    return this;
  }

  payload(payload: Record<string, unknown>): EventBuilder {
    this.event.payload = payload;
    return this;
  }

  addPayload(key: string, value: unknown): EventBuilder {
    if (!this.event.payload) {
      this.event.payload = {};
    }
    this.event.payload[key] = value;
    return this;
  }

  prevEventHash(hash: string): EventBuilder {
    this.event.prev_event_hash = hash;
    return this;
  }

  prevEventHashes(hashes: string[]): EventBuilder {
    this.event.prev_event_hashes = hashes;
    return this;
  }

  build(): Event {
    if (!this.event.trace_id) {
      throw new Error('trace_id is required');
    }
    if (!this.event.event_type) {
      throw new Error('event_type is required');
    }
    if (!this.event.payload) {
      this.event.payload = {};
    }
    return this.event as Event;
  }

  /**
   * Create an input event
   */
  static input(traceId: string, prompt: string, modelId: string): Event {
    return new EventBuilder()
      .traceId(traceId)
      .eventType(EventType.INPUT)
      .addPayload('prompt', prompt)
      .addPayload('model_id', modelId)
      .build();
  }

  /**
   * Create an output event
   */
  static output(traceId: string, content: string, tokens?: number): Event {
    const builder = new EventBuilder()
      .traceId(traceId)
      .eventType(EventType.OUTPUT)
      .addPayload('content', content);
    if (tokens !== undefined) {
      builder.addPayload('tokens', tokens);
    }
    return builder.build();
  }

  /**
   * Create a tool call event
   */
  static toolCall(traceId: string, toolName: string, args: Record<string, unknown>): Event {
    return new EventBuilder()
      .traceId(traceId)
      .eventType(EventType.TOOL_CALL)
      .addPayload('tool_name', toolName)
      .addPayload('arguments', args)
      .build();
  }

  /**
   * Create a tool result event
   */
  static toolResult(traceId: string, toolName: string, result: unknown): Event {
    return new EventBuilder()
      .traceId(traceId)
      .eventType(EventType.TOOL_RESULT)
      .addPayload('tool_name', toolName)
      .addPayload('result', result)
      .build();
  }

  /**
   * Create an error event
   */
  static error(traceId: string, errorCode: string, message: string): Event {
    return new EventBuilder()
      .traceId(traceId)
      .eventType(EventType.ERROR)
      .addPayload('error_code', errorCode)
      .addPayload('message', message)
      .build();
  }
}

/**
 * Message builder helpers
 */
export const MessageHelper = {
  user(content: string): Message {
    return { role: 'user', content };
  },

  assistant(content: string): Message {
    return { role: 'assistant', content };
  },

  system(content: string): Message {
    return { role: 'system', content };
  },
};

interface RequestOptions {
  body?: unknown;
  params?: Record<string, string | number | undefined>;
  headers?: Record<string, string>;
}

/**
 * Chat completions service
 */
export class ChatService {
  private client: AITraceClient;
  private _lastTraceId: string | null = null;

  constructor(client: AITraceClient) {
    this.client = client;
  }

  /**
   * Create a chat completion with AI-Trace attestation
   */
  async create(request: ChatRequest): Promise<ChatResponse> {
    // Input validation
    if (!request.model) {
      throw new AITraceError('model is required', 'invalid_request', 400);
    }
    if (!request.messages || request.messages.length === 0) {
      throw new AITraceError('at least one message is required', 'invalid_request', 400);
    }

    const headers: Record<string, string> = {};
    if (request.traceId) headers['X-Trace-ID'] = request.traceId;
    if (request.sessionId) headers['X-Session-ID'] = request.sessionId;
    if (request.businessId) headers['X-Business-ID'] = request.businessId;

    const { traceId, sessionId, businessId, maxTokens, ...rest } = request;
    const body: Record<string, unknown> = { ...rest };
    if (maxTokens) body.max_tokens = maxTokens;

    const response = await this.client._request<ChatResponse>('POST', '/api/v1/chat/completions', {
      body,
      headers,
    });

    this._lastTraceId = response.trace_id;
    return response;
  }

  /**
   * Create a chat completion and immediately commit a certificate
   */
  async createAndCommit(
    request: ChatRequest,
    evidenceLevel: EvidenceLevelType = EvidenceLevel.L1
  ): Promise<ChatAndCertResult> {
    const chatResponse = await this.create(request);
    const certificate = await this.client.certs.commit(chatResponse.trace_id, evidenceLevel);
    return { chatResponse, certificate };
  }

  /**
   * Get the trace ID from the last chat request
   */
  get lastTraceId(): string | null {
    return this._lastTraceId;
  }
}

/**
 * Events service for managing AI inference events
 */
export class EventsService {
  private client: AITraceClient;

  constructor(client: AITraceClient) {
    this.client = client;
  }

  /**
   * Ingest a batch of events
   */
  async ingest(events: Event[]): Promise<IngestResponse> {
    // Input validation
    if (!events || events.length === 0) {
      throw new AITraceError('at least one event is required', 'invalid_request', 400);
    }
    for (let i = 0; i < events.length; i++) {
      const e = events[i];
      if (!e.trace_id) {
        throw new AITraceError(`event[${i}]: trace_id is required`, 'invalid_request', 400);
      }
      if (!e.event_type) {
        throw new AITraceError(`event[${i}]: event_type is required`, 'invalid_request', 400);
      }
    }

    return this.client._request<IngestResponse>('POST', '/api/v1/events/ingest', {
      body: { events },
    });
  }

  /**
   * Search for events with filters
   */
  async search(params: EventSearchParams = {}): Promise<EventSearchResponse> {
    return this.client._request<EventSearchResponse>('GET', '/api/v1/events/search', {
      params: {
        trace_id: params.traceId,
        event_type: params.eventType,
        start_time: params.startTime,
        end_time: params.endTime,
        page: params.page ?? 1,
        page_size: params.pageSize ?? 20,
      },
    });
  }

  /**
   * Get a single event by ID
   */
  async get(eventId: string): Promise<Event> {
    if (!eventId) {
      throw new AITraceError('event_id is required', 'invalid_request', 400);
    }
    return this.client._request<Event>('GET', `/api/v1/events/${eventId}`);
  }

  /**
   * Get all events for a trace
   */
  async getByTrace(traceId: string): Promise<Event[]> {
    if (!traceId) {
      throw new AITraceError('trace_id is required', 'invalid_request', 400);
    }
    const result = await this.search({ traceId, pageSize: 100 });
    return result.events;
  }
}

/**
 * Certificates service for attestation management
 */
export class CertsService {
  private client: AITraceClient;

  constructor(client: AITraceClient) {
    this.client = client;
  }

  /**
   * Commit a certificate for a trace
   */
  async commit(traceId: string, evidenceLevel: EvidenceLevelType = EvidenceLevel.L1): Promise<Certificate> {
    // Input validation
    if (!traceId) {
      throw new AITraceError('trace_id is required', 'invalid_request', 400);
    }
    if (evidenceLevel !== EvidenceLevel.L1 && evidenceLevel !== EvidenceLevel.L2 && evidenceLevel !== EvidenceLevel.L3) {
      throw new AITraceError('invalid evidence level: must be L1, L2, or L3', 'invalid_request', 400);
    }

    return this.client._request<Certificate>('POST', '/api/v1/certs/commit', {
      body: {
        trace_id: traceId,
        evidence_level: evidenceLevel,
      },
    });
  }

  /**
   * Commit an L1 (basic) certificate
   */
  async commitL1(traceId: string): Promise<Certificate> {
    return this.commit(traceId, EvidenceLevel.L1);
  }

  /**
   * Commit an L2 (WORM storage) certificate
   */
  async commitL2(traceId: string): Promise<Certificate> {
    return this.commit(traceId, EvidenceLevel.L2);
  }

  /**
   * Commit an L3 (blockchain) certificate
   */
  async commitL3(traceId: string): Promise<Certificate> {
    return this.commit(traceId, EvidenceLevel.L3);
  }

  /**
   * Verify a certificate by cert ID
   */
  async verifyByCertId(certId: string): Promise<VerificationResult> {
    if (!certId) {
      throw new AITraceError('cert_id is required', 'invalid_request', 400);
    }
    return this.client._request<VerificationResult>('POST', '/api/v1/certs/verify', {
      body: { cert_id: certId },
    });
  }

  /**
   * Verify a certificate by root hash
   */
  async verifyByRootHash(rootHash: string): Promise<VerificationResult> {
    if (!rootHash) {
      throw new AITraceError('root_hash is required', 'invalid_request', 400);
    }
    return this.client._request<VerificationResult>('POST', '/api/v1/certs/verify', {
      body: { root_hash: rootHash },
    });
  }

  /**
   * Search for certificates
   */
  async search(params: CertSearchParams = {}): Promise<CertSearchResponse> {
    return this.client._request<CertSearchResponse>('GET', '/api/v1/certs/search', {
      params: {
        page: params.page ?? 1,
        page_size: params.pageSize ?? 20,
      },
    });
  }

  /**
   * Get a certificate by ID
   */
  async get(certId: string): Promise<Certificate> {
    if (!certId) {
      throw new AITraceError('cert_id is required', 'invalid_request', 400);
    }
    return this.client._request<Certificate>('GET', `/api/v1/certs/${certId}`);
  }

  /**
   * Generate a minimal disclosure proof
   */
  async prove(certId: string, params: ProveParams = {}): Promise<ProofResponse> {
    if (!certId) {
      throw new AITraceError('cert_id is required', 'invalid_request', 400);
    }
    return this.client._request<ProofResponse>('POST', `/api/v1/certs/${certId}/prove`, {
      body: {
        disclose_events: params.discloseEvents ?? [],
        disclose_fields: params.discloseFields ?? [],
      },
    });
  }

  /**
   * Generate a proof for specific event indices
   */
  async proveWithIndices(certId: string, ...indices: number[]): Promise<ProofResponse> {
    return this.prove(certId, { discloseEvents: indices });
  }
}

/**
 * Main AI-Trace client
 */
export class AITraceClient {
  private apiKey: string;
  private baseUrl: string;
  private upstreamApiKey?: string;
  private upstreamBaseUrl?: string;
  private timeout: number;

  public chat: ChatService;
  public events: EventsService;
  public certs: CertsService;

  constructor(options: AITraceClientOptions) {
    if (!options.apiKey) {
      throw new Error('apiKey is required');
    }

    this.apiKey = options.apiKey;
    this.baseUrl = (options.baseUrl || 'https://api.aitrace.cc').replace(/\/$/, '');
    this.upstreamApiKey = options.upstreamApiKey;
    this.upstreamBaseUrl = options.upstreamBaseUrl;
    this.timeout = options.timeout || 120000;

    this.chat = new ChatService(this);
    this.events = new EventsService(this);
    this.certs = new CertsService(this);
  }

  /**
   * Internal request method
   * @internal
   */
  async _request<T>(
    method: string,
    path: string,
    options: RequestOptions = {}
  ): Promise<T> {
    const url = new URL(path, this.baseUrl);

    if (options.params) {
      Object.entries(options.params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value));
        }
      });
    }

    const headers: Record<string, string> = {
      'X-API-Key': this.apiKey,
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.upstreamApiKey) {
      headers['X-Upstream-API-Key'] = this.upstreamApiKey;
    }
    if (this.upstreamBaseUrl) {
      headers['X-Upstream-Base-URL'] = this.upstreamBaseUrl;
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url.toString(), {
        method,
        headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const error = await response.json().catch(() => ({ code: 'unknown', message: `HTTP ${response.status}` }));
        throw new AITraceError(
          error.message || error.error || `HTTP ${response.status}`,
          error.code || 'unknown',
          response.status
        );
      }

      return response.json() as Promise<T>;
    } catch (error) {
      clearTimeout(timeoutId);
      if (error instanceof AITraceError) {
        throw error;
      }
      if (error instanceof Error && error.name === 'AbortError') {
        throw new AITraceError('Request timeout', 'timeout', 408);
      }
      throw error;
    }
  }
}

// Default export
export default AITraceClient;
