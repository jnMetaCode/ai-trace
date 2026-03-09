import { describe, it, expect, beforeAll, afterAll, beforeEach } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import {
  AITraceClient,
  AITraceError,
  EventBuilder,
  MessageHelper,
  EvidenceLevel,
  EventType,
  ChatResponse,
  Certificate,
  VerificationResult,
  Event,
  IngestResponse,
} from '../index';

// Mock server setup
const mockServer = setupServer();

describe('AITraceClient', () => {
  const baseUrl = 'https://test.api.aitrace.cc';

  beforeAll(() => {
    mockServer.listen({ onUnhandledRequest: 'error' });
  });

  afterAll(() => {
    mockServer.close();
  });

  beforeEach(() => {
    mockServer.resetHandlers();
  });

  describe('constructor', () => {
    it('should create client with required options', () => {
      const client = new AITraceClient({
        apiKey: 'test-api-key',
        baseUrl,
      });

      expect(client).toBeInstanceOf(AITraceClient);
      expect(client.chat).toBeDefined();
      expect(client.events).toBeDefined();
      expect(client.certs).toBeDefined();
    });

    it('should throw error without apiKey', () => {
      expect(() => {
        new AITraceClient({ apiKey: '' });
      }).toThrow('apiKey is required');
    });

    it('should use default baseUrl', () => {
      const client = new AITraceClient({ apiKey: 'test-key' });
      expect(client).toBeInstanceOf(AITraceClient);
    });
  });

  describe('ChatService', () => {
    it('should create chat completion', async () => {
      const mockResponse: ChatResponse = {
        id: 'chatcmpl-123',
        object: 'chat.completion',
        created: 1704067200,
        model: 'gpt-4',
        choices: [
          {
            index: 0,
            message: { role: 'assistant', content: 'Hello!' },
            finish_reason: 'stop',
          },
        ],
        usage: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 },
        trace_id: 'trace-123',
      };

      mockServer.use(
        http.post(`${baseUrl}/api/v1/chat/completions`, () => {
          return HttpResponse.json(mockResponse);
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const response = await client.chat.create({
        model: 'gpt-4',
        messages: [{ role: 'user', content: 'Hi!' }],
      });

      expect(response.id).toBe('chatcmpl-123');
      expect(response.trace_id).toBe('trace-123');
      expect(response.choices[0].message.content).toBe('Hello!');
      expect(client.chat.lastTraceId).toBe('trace-123');
    });

    it('should create chat completion and commit certificate', async () => {
      const mockChatResponse: ChatResponse = {
        id: 'chatcmpl-123',
        object: 'chat.completion',
        created: 1704067200,
        model: 'gpt-4',
        choices: [{ index: 0, message: { role: 'assistant', content: 'Hello!' }, finish_reason: 'stop' }],
        trace_id: 'trace-123',
      };

      const mockCert: Certificate = {
        cert_id: 'cert-123',
        trace_id: 'trace-123',
        root_hash: 'abc123',
        event_count: 2,
        evidence_level: 'L2',
        created_at: '2024-01-01T00:00:00Z',
      };

      mockServer.use(
        http.post(`${baseUrl}/api/v1/chat/completions`, () => {
          return HttpResponse.json(mockChatResponse);
        }),
        http.post(`${baseUrl}/api/v1/certs/commit`, () => {
          return HttpResponse.json(mockCert);
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const result = await client.chat.createAndCommit(
        { model: 'gpt-4', messages: [{ role: 'user', content: 'Hi!' }] },
        EvidenceLevel.L2
      );

      expect(result.chatResponse.trace_id).toBe('trace-123');
      expect(result.certificate.cert_id).toBe('cert-123');
    });
  });

  describe('EventsService', () => {
    it('should ingest events', async () => {
      const mockResponse: IngestResponse = {
        ingested: 2,
        event_ids: ['event-1', 'event-2'],
      };

      mockServer.use(
        http.post(`${baseUrl}/api/v1/events/ingest`, () => {
          return HttpResponse.json(mockResponse);
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const events = [
        EventBuilder.input('trace-123', 'Hello', 'gpt-4'),
        EventBuilder.output('trace-123', 'Hi there!', 10),
      ];

      const response = await client.events.ingest(events);

      expect(response.ingested).toBe(2);
      expect(response.event_ids).toHaveLength(2);
    });

    it('should search events', async () => {
      mockServer.use(
        http.get(`${baseUrl}/api/v1/events/search`, ({ request }) => {
          const url = new URL(request.url);
          expect(url.searchParams.get('trace_id')).toBe('trace-123');
          return HttpResponse.json({
            events: [],
            total: 0,
            page: 1,
            page_size: 20,
            total_pages: 0,
          });
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const response = await client.events.search({ traceId: 'trace-123' });

      expect(response.events).toEqual([]);
    });

    it('should get event by ID', async () => {
      const mockEvent: Event = {
        event_id: 'event-123',
        trace_id: 'trace-123',
        event_type: 'llm.input',
        payload: { prompt: 'Hello' },
      };

      mockServer.use(
        http.get(`${baseUrl}/api/v1/events/event-123`, () => {
          return HttpResponse.json(mockEvent);
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const event = await client.events.get('event-123');

      expect(event.event_id).toBe('event-123');
    });
  });

  describe('CertsService', () => {
    it('should commit certificate', async () => {
      const mockCert: Certificate = {
        cert_id: 'cert-123',
        trace_id: 'trace-123',
        root_hash: 'abc123',
        event_count: 5,
        evidence_level: 'L2',
        created_at: '2024-01-01T00:00:00Z',
      };

      mockServer.use(
        http.post(`${baseUrl}/api/v1/certs/commit`, () => {
          return HttpResponse.json(mockCert);
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const cert = await client.certs.commit('trace-123', EvidenceLevel.L2);

      expect(cert.cert_id).toBe('cert-123');
      expect(cert.evidence_level).toBe('L2');
    });

    it('should verify certificate by ID', async () => {
      const mockResult: VerificationResult = {
        valid: true,
        checks: {
          merkle_root: true,
          timestamp: true,
          event_hashes: true,
          causal_chain: true,
        },
      };

      mockServer.use(
        http.post(`${baseUrl}/api/v1/certs/verify`, () => {
          return HttpResponse.json(mockResult);
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const result = await client.certs.verifyByCertId('cert-123');

      expect(result.valid).toBe(true);
      expect(result.checks.merkle_root).toBe(true);
    });

    it('should generate proof', async () => {
      mockServer.use(
        http.post(`${baseUrl}/api/v1/certs/cert-123/prove`, () => {
          return HttpResponse.json({
            cert_id: 'cert-123',
            root_hash: 'abc123',
            disclosed_events: [],
            merkle_proofs: [],
            metadata: {},
          });
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });
      const proof = await client.certs.prove('cert-123', { discloseEvents: [0, 2] });

      expect(proof.cert_id).toBe('cert-123');
    });

    it('should use convenience methods for evidence levels', async () => {
      const createMockHandler = (expectedLevel: string) =>
        http.post(`${baseUrl}/api/v1/certs/commit`, async ({ request }) => {
          const body = await request.json() as { evidence_level: string };
          expect(body.evidence_level).toBe(expectedLevel);
          return HttpResponse.json({
            cert_id: 'cert-123',
            trace_id: 'trace-123',
            root_hash: 'abc',
            event_count: 1,
            evidence_level: expectedLevel,
            created_at: '2024-01-01T00:00:00Z',
          });
        });

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });

      mockServer.use(createMockHandler('L1'));
      await client.certs.commitL1('trace-123');

      mockServer.use(createMockHandler('L2'));
      await client.certs.commitL2('trace-123');

      mockServer.use(createMockHandler('L3'));
      await client.certs.commitL3('trace-123');
    });
  });

  describe('Error Handling', () => {
    it('should throw AITraceError on API error', async () => {
      mockServer.use(
        http.post(`${baseUrl}/api/v1/certs/commit`, () => {
          return HttpResponse.json(
            { code: 'invalid_request', message: 'Invalid trace ID' },
            { status: 400 }
          );
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });

      await expect(client.certs.commit('', EvidenceLevel.L1)).rejects.toThrow(AITraceError);

      try {
        await client.certs.commit('', EvidenceLevel.L1);
      } catch (error) {
        expect(error).toBeInstanceOf(AITraceError);
        const aiError = error as AITraceError;
        expect(aiError.code).toBe('invalid_request');
        expect(aiError.statusCode).toBe(400);
        expect(aiError.isClientError()).toBe(true);
        expect(aiError.isServerError()).toBe(false);
      }
    });

    it('should handle server errors', async () => {
      mockServer.use(
        http.get(`${baseUrl}/api/v1/events/test`, () => {
          return HttpResponse.json(
            { code: 'internal_error', message: 'Server error' },
            { status: 500 }
          );
        })
      );

      const client = new AITraceClient({ apiKey: 'test-key', baseUrl });

      try {
        await client.events.get('test');
      } catch (error) {
        expect(error).toBeInstanceOf(AITraceError);
        const aiError = error as AITraceError;
        expect(aiError.isServerError()).toBe(true);
      }
    });
  });
});

describe('EventBuilder', () => {
  it('should build event with builder pattern', () => {
    const event = new EventBuilder()
      .traceId('trace-123')
      .eventType(EventType.INPUT)
      .sequence(1)
      .addPayload('prompt', 'Hello')
      .addPayload('model_id', 'gpt-4')
      .build();

    expect(event.trace_id).toBe('trace-123');
    expect(event.event_type).toBe('llm.input');
    expect(event.sequence).toBe(1);
    expect(event.payload.prompt).toBe('Hello');
    expect(event.payload.model_id).toBe('gpt-4');
  });

  it('should throw error without required fields', () => {
    expect(() => new EventBuilder().build()).toThrow('trace_id is required');
    expect(() => new EventBuilder().traceId('t').build()).toThrow('event_type is required');
  });

  it('should create input event', () => {
    const event = EventBuilder.input('trace-123', 'Hello', 'gpt-4');
    expect(event.event_type).toBe('llm.input');
    expect(event.payload.prompt).toBe('Hello');
    expect(event.payload.model_id).toBe('gpt-4');
  });

  it('should create output event', () => {
    const event = EventBuilder.output('trace-123', 'Hi there!', 10);
    expect(event.event_type).toBe('llm.output');
    expect(event.payload.content).toBe('Hi there!');
    expect(event.payload.tokens).toBe(10);
  });

  it('should create tool call event', () => {
    const event = EventBuilder.toolCall('trace-123', 'search', { query: 'test' });
    expect(event.event_type).toBe('llm.tool_call');
    expect(event.payload.tool_name).toBe('search');
    expect(event.payload.arguments).toEqual({ query: 'test' });
  });

  it('should create tool result event', () => {
    const event = EventBuilder.toolResult('trace-123', 'search', { results: [] });
    expect(event.event_type).toBe('llm.tool_result');
    expect(event.payload.result).toEqual({ results: [] });
  });

  it('should create error event', () => {
    const event = EventBuilder.error('trace-123', 'ERR001', 'Something went wrong');
    expect(event.event_type).toBe('llm.error');
    expect(event.payload.error_code).toBe('ERR001');
    expect(event.payload.message).toBe('Something went wrong');
  });
});

describe('MessageHelper', () => {
  it('should create user message', () => {
    const msg = MessageHelper.user('Hello');
    expect(msg.role).toBe('user');
    expect(msg.content).toBe('Hello');
  });

  it('should create assistant message', () => {
    const msg = MessageHelper.assistant('Hi there!');
    expect(msg.role).toBe('assistant');
    expect(msg.content).toBe('Hi there!');
  });

  it('should create system message', () => {
    const msg = MessageHelper.system('You are helpful.');
    expect(msg.role).toBe('system');
    expect(msg.content).toBe('You are helpful.');
  });
});

describe('Constants', () => {
  it('should have correct evidence levels', () => {
    expect(EvidenceLevel.L1).toBe('L1');
    expect(EvidenceLevel.L2).toBe('L2');
    expect(EvidenceLevel.L3).toBe('L3');
  });

  it('should have correct event types', () => {
    expect(EventType.INPUT).toBe('llm.input');
    expect(EventType.OUTPUT).toBe('llm.output');
    expect(EventType.CHUNK).toBe('llm.chunk');
    expect(EventType.TOOL_CALL).toBe('llm.tool_call');
    expect(EventType.TOOL_RESULT).toBe('llm.tool_result');
    expect(EventType.ERROR).toBe('llm.error');
  });
});
