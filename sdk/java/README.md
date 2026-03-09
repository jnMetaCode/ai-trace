# AI-Trace Java SDK

Official Java SDK for the AI-Trace platform - Enterprise AI decision auditing and tamper-proof attestation.

## Requirements

- Java 11 or higher
- Maven or Gradle

## Installation

### Maven

```xml
<dependency>
    <groupId>dev.aitrace</groupId>
    <artifactId>aitrace-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'dev.aitrace:aitrace-sdk:1.0.0'
```

## Quick Start

```java
import dev.aitrace.sdk.AITraceClient;
import dev.aitrace.sdk.model.*;

public class Example {
    public static void main(String[] args) {
        // Create client
        AITraceClient client = AITraceClient.builder()
            .apiKey("your-api-key")
            .upstreamApiKey("sk-your-openai-key")  // Pass-through, never stored
            .build();

        // Create chat completion with attestation
        ChatResponse response = client.chat().create(ChatRequest.builder()
            .model("gpt-4")
            .addUserMessage("What is 2+2?")
            .build());

        System.out.println("Response: " + response.getContent());
        System.out.println("Trace ID: " + response.getTraceId());

        // Commit certificate
        Certificate cert = client.certs().commit(response.getTraceId(), EvidenceLevel.L2);
        System.out.println("Certificate ID: " + cert.getCertId());
        System.out.println("Root Hash: " + cert.getRootHash());

        // Verify certificate
        VerificationResult result = client.certs().verifyByCertId(cert.getCertId());
        System.out.println("Valid: " + result.isValid());
    }
}
```

## Features

### Chat Completions (OpenAI Compatible)

```java
ChatResponse response = client.chat().create(ChatRequest.builder()
    .model("gpt-4")
    .addSystemMessage("You are a helpful assistant.")
    .addUserMessage("Hello!")
    .temperature(0.7)
    .maxTokens(100)
    .traceId("custom-trace-id")      // Optional
    .sessionId("session-123")         // Optional
    .businessId("business-456")       // Optional
    .build());
```

### Event Management

```java
// Ingest custom events
List<Event> events = Arrays.asList(
    Event.input("trace-123", "User prompt", "gpt-4"),
    Event.output("trace-123", "AI response", 50)
);
EventsService.IngestResponse resp = client.events().ingest(events);

// Search events
EventsService.SearchResponse searchResp = client.events().search(
    EventsService.SearchRequest.forTrace("trace-123")
);

// Get all events for a trace
List<Event> traceEvents = client.events().getByTrace("trace-123");
```

### Certificate Management

```java
// Commit with different evidence levels
Certificate cert = client.certs().commit(traceId, EvidenceLevel.L2);

// Convenience methods
Certificate l1Cert = client.certs().commitL1(traceId);  // Basic
Certificate l2Cert = client.certs().commitL2(traceId);  // WORM storage
Certificate l3Cert = client.certs().commitL3(traceId);  // Blockchain anchor

// Verify certificate
VerificationResult result = client.certs().verifyByCertId("cert-123");
if (result.isValid()) {
    System.out.println("Certificate is valid!");
}

// Generate minimal disclosure proof
CertsService.ProofResponse proof = client.certs().proveWithIndices("cert-123", 0, 2);
```

### Convenience Methods

```java
// Create chat and immediately commit certificate
ChatService.ChatAndCertResult result = client.chat()
    .createAndCommit(request, EvidenceLevel.L2);

ChatResponse response = result.getChatResponse();
Certificate certificate = result.getCertificate();

// Build events programmatically
Event event = Event.builder()
    .traceId("trace-123")
    .eventType(EventType.INPUT)
    .sequence(1)
    .addPayload("prompt", "Hello")
    .addPayload("model_id", "gpt-4")
    .build();
```

## Client Configuration

```java
AITraceClient client = AITraceClient.builder()
    .apiKey("api-key")
    .baseUrl("https://custom.example.com")
    .upstreamApiKey("sk-upstream-key")
    .upstreamBaseUrl("https://upstream.example.com")
    .timeout(Duration.ofSeconds(30))
    .httpClient(customOkHttpClient)
    .build();
```

## Error Handling

```java
try {
    ChatResponse response = client.chat().create(request);
} catch (AITraceException e) {
    System.out.println("Error code: " + e.getCode());
    System.out.println("Message: " + e.getMessage());
    System.out.println("Status: " + e.getStatusCode());

    if (e.isClientError()) {
        // Handle 4xx errors
    } else if (e.isServerError()) {
        // Handle 5xx errors
    }
}
```

## Event Types

```java
EventType.INPUT       // "llm.input"
EventType.OUTPUT      // "llm.output"
EventType.CHUNK       // "llm.chunk"
EventType.TOOL_CALL   // "llm.tool_call"
EventType.TOOL_RESULT // "llm.tool_result"
EventType.ERROR       // "llm.error"
```

## Evidence Levels

| Level | Description |
|-------|-------------|
| L1 | Basic attestation with Merkle tree and timestamp |
| L2 | WORM (Write Once Read Many) storage for legal compliance |
| L3 | Blockchain anchor for maximum tamper-proof guarantee |

## Building from Source

```bash
mvn clean install
```

## Running Tests

```bash
mvn test
```

## License

Apache-2.0
