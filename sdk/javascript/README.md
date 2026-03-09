# AI-Trace JavaScript/TypeScript SDK

Official JavaScript/TypeScript SDK for the AI-Trace platform - Enterprise AI decision auditing and tamper-proof attestation.

## Installation

```bash
npm install @ai-trace/sdk
```

Or using yarn:

```bash
yarn add @ai-trace/sdk
```

Or using pnpm:

```bash
pnpm add @ai-trace/sdk
```

## Quick Start

```typescript
import { AITraceClient, EvidenceLevel } from '@ai-trace/sdk';

// Create client
const client = new AITraceClient({
  apiKey: 'your-api-key',
  upstreamApiKey: 'sk-your-openai-key',  // Pass-through, never stored
});

// Create chat completion with attestation
const response = await client.chat.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: 'What is 2+2?' }],
});

console.log('Response:', response.choices[0].message.content);
console.log('Trace ID:', response.trace_id);

// Commit certificate
const cert = await client.certs.commit(response.trace_id, EvidenceLevel.L2);
console.log('Certificate ID:', cert.cert_id);
console.log('Root Hash:', cert.root_hash);

// Verify certificate
const result = await client.certs.verifyByCertId(cert.cert_id);
console.log('Valid:', result.valid);
```

## Features

### Chat Completions (OpenAI Compatible)

```typescript
const response = await client.chat.create({
  model: 'gpt-4',
  messages: [
    { role: 'system', content: 'You are a helpful assistant.' },
    { role: 'user', content: 'Hello!' },
  ],
  temperature: 0.7,
  maxTokens: 100,
  traceId: 'custom-trace-id',      // Optional
  sessionId: 'session-123',         // Optional
  businessId: 'business-456',       // Optional
});
```

### Create and Commit in One Call

```typescript
const { chatResponse, certificate } = await client.chat.createAndCommit(
  {
    model: 'gpt-4',
    messages: [{ role: 'user', content: 'Hello!' }],
  },
  EvidenceLevel.L2
);
```

### Event Management

```typescript
import { EventBuilder } from '@ai-trace/sdk';

// Create events programmatically
const events = [
  EventBuilder.input('trace-123', 'User prompt', 'gpt-4'),
  EventBuilder.output('trace-123', 'AI response', 50),
];

// Ingest events
const resp = await client.events.ingest(events);

// Search events
const searchResp = await client.events.search({
  traceId: 'trace-123',
});

// Get all events for a trace
const traceEvents = await client.events.getByTrace('trace-123');
```

### Event Builder Pattern

```typescript
import { EventBuilder, EventType } from '@ai-trace/sdk';

const event = new EventBuilder()
  .traceId('trace-123')
  .eventType(EventType.INPUT)
  .sequence(1)
  .addPayload('prompt', 'Hello')
  .addPayload('model_id', 'gpt-4')
  .build();
```

### Certificate Management

```typescript
// Commit with different evidence levels
const cert = await client.certs.commit(traceId, EvidenceLevel.L2);

// Convenience methods
const l1Cert = await client.certs.commitL1(traceId);  // Basic
const l2Cert = await client.certs.commitL2(traceId);  // WORM storage
const l3Cert = await client.certs.commitL3(traceId);  // Blockchain anchor

// Verify certificate
const result = await client.certs.verifyByCertId('cert-123');
if (result.valid) {
  console.log('Certificate is valid!');
  console.log('Checks:', result.checks);
}

// Generate minimal disclosure proof
const proof = await client.certs.prove('cert-123', {
  discloseEvents: [0, 2, 4],
  discloseFields: ['prompt', 'response'],
});
```

### Message Helpers

```typescript
import { MessageHelper } from '@ai-trace/sdk';

const messages = [
  MessageHelper.system('You are a helpful assistant.'),
  MessageHelper.user('Hello!'),
  MessageHelper.assistant('Hi there!'),
];
```

## Client Configuration

```typescript
const client = new AITraceClient({
  apiKey: 'api-key',                              // Required
  baseUrl: 'https://custom.example.com',          // Optional
  upstreamApiKey: 'sk-upstream-key',              // Optional
  upstreamBaseUrl: 'https://upstream.example.com', // Optional
  timeout: 30000,                                 // Optional (ms)
});
```

## Error Handling

```typescript
import { AITraceError } from '@ai-trace/sdk';

try {
  const response = await client.chat.create(request);
} catch (error) {
  if (error instanceof AITraceError) {
    console.log('Error code:', error.code);
    console.log('Message:', error.message);
    console.log('Status:', error.statusCode);

    if (error.isClientError()) {
      // Handle 4xx errors
    } else if (error.isServerError()) {
      // Handle 5xx errors
    }
  }
}
```

## Event Types

```typescript
import { EventType } from '@ai-trace/sdk';

EventType.INPUT       // 'llm.input'
EventType.OUTPUT      // 'llm.output'
EventType.CHUNK       // 'llm.chunk'
EventType.TOOL_CALL   // 'llm.tool_call'
EventType.TOOL_RESULT // 'llm.tool_result'
EventType.ERROR       // 'llm.error'
```

## Evidence Levels

| Level | Description |
|-------|-------------|
| L1 | Basic attestation with Merkle tree and timestamp |
| L2 | WORM (Write Once Read Many) storage for legal compliance |
| L3 | Blockchain anchor for maximum tamper-proof guarantee |

## TypeScript Support

This SDK is written in TypeScript and includes full type definitions. All types are exported:

```typescript
import type {
  AITraceClientOptions,
  ChatRequest,
  ChatResponse,
  Certificate,
  Event,
  VerificationResult,
  // ... and more
} from '@ai-trace/sdk';
```

## Browser Support

This SDK uses the Fetch API and works in modern browsers. For Node.js < 18, you may need a fetch polyfill.

## Building from Source

```bash
npm install
npm run build
```

## Running Tests

```bash
npm test
```

## License

Apache-2.0
