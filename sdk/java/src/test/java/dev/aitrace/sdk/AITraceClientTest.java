package dev.aitrace.sdk;

import com.fasterxml.jackson.databind.ObjectMapper;
import dev.aitrace.sdk.exception.AITraceException;
import dev.aitrace.sdk.model.*;
import dev.aitrace.sdk.service.EventsService;
import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.MockWebServer;
import okhttp3.mockwebserver.RecordedRequest;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class AITraceClientTest {

    private MockWebServer mockWebServer;
    private AITraceClient client;
    private ObjectMapper objectMapper;

    @BeforeEach
    void setUp() throws IOException {
        mockWebServer = new MockWebServer();
        mockWebServer.start();

        client = AITraceClient.builder()
                .apiKey("test-api-key")
                .baseUrl(mockWebServer.url("/").toString())
                .build();

        objectMapper = client.getObjectMapper();
    }

    @AfterEach
    void tearDown() throws IOException {
        mockWebServer.shutdown();
    }

    @Test
    void testClientBuilder() {
        AITraceClient client = AITraceClient.builder()
                .apiKey("test-key")
                .baseUrl("https://custom.example.com")
                .upstreamApiKey("sk-upstream")
                .build();

        assertNotNull(client);
        assertNotNull(client.chat());
        assertNotNull(client.events());
        assertNotNull(client.certs());
    }

    @Test
    void testClientBuilderRequiresApiKey() {
        assertThrows(IllegalArgumentException.class, () -> {
            AITraceClient.builder().build();
        });
    }

    @Test
    void testChatCreate() throws Exception {
        // Prepare mock response
        ChatResponse mockResponse = new ChatResponse();
        mockResponse.setId("chatcmpl-123");
        mockResponse.setModel("gpt-4");
        mockResponse.setTraceId("trace-123");

        ChatResponse.Choice choice = new ChatResponse.Choice();
        choice.setIndex(0);
        Message message = new Message("assistant", "Hello!");
        choice.setMessage(message);
        choice.setFinishReason("stop");
        mockResponse.setChoices(Arrays.asList(choice));

        mockWebServer.enqueue(new MockResponse()
                .setBody(objectMapper.writeValueAsString(mockResponse))
                .addHeader("Content-Type", "application/json"));

        // Make request
        ChatRequest request = ChatRequest.builder()
                .model("gpt-4")
                .addUserMessage("Hello!")
                .temperature(0.7)
                .build();

        ChatResponse response = client.chat().create(request);

        // Verify response
        assertEquals("chatcmpl-123", response.getId());
        assertEquals("trace-123", response.getTraceId());
        assertEquals(1, response.getChoices().size());
        assertEquals("Hello!", response.getContent());

        // Verify request
        RecordedRequest recorded = mockWebServer.takeRequest();
        assertEquals("POST", recorded.getMethod());
        assertEquals("/api/v1/chat/completions", recorded.getPath());
        assertEquals("test-api-key", recorded.getHeader("X-API-Key"));
    }

    @Test
    void testEventsIngest() throws Exception {
        // Prepare mock response
        EventsService.IngestResponse mockResponse = new EventsService.IngestResponse();
        mockResponse.ingested = 2;
        mockResponse.eventIds = Arrays.asList("event-1", "event-2");

        mockWebServer.enqueue(new MockResponse()
                .setBody(objectMapper.writeValueAsString(mockResponse))
                .addHeader("Content-Type", "application/json"));

        // Make request
        List<Event> events = Arrays.asList(
                Event.input("trace-123", "Hello", "gpt-4"),
                Event.output("trace-123", "World", 10)
        );

        EventsService.IngestResponse response = client.events().ingest(events);

        // Verify
        assertEquals(2, response.ingested);
        assertEquals(2, response.eventIds.size());
    }

    @Test
    void testCertsCommit() throws Exception {
        // Prepare mock response
        Certificate mockCert = new Certificate();
        mockCert.setCertId("cert-123");
        mockCert.setTraceId("trace-123");
        mockCert.setRootHash("abc123");
        mockCert.setEventCount(5);
        mockCert.setEvidenceLevel(EvidenceLevel.L2);

        mockWebServer.enqueue(new MockResponse()
                .setBody(objectMapper.writeValueAsString(mockCert))
                .addHeader("Content-Type", "application/json"));

        // Make request
        Certificate cert = client.certs().commit("trace-123", EvidenceLevel.L2);

        // Verify
        assertEquals("cert-123", cert.getCertId());
        assertEquals(EvidenceLevel.L2, cert.getEvidenceLevel());
    }

    @Test
    void testCertsVerify() throws Exception {
        // Prepare mock response
        VerificationResult mockResult = new VerificationResult();
        mockResult.setValid(true);

        mockWebServer.enqueue(new MockResponse()
                .setBody(objectMapper.writeValueAsString(mockResult))
                .addHeader("Content-Type", "application/json"));

        // Make request
        VerificationResult result = client.certs().verifyByCertId("cert-123");

        // Verify
        assertTrue(result.isValid());
    }

    @Test
    void testApiError() throws Exception {
        // Prepare error response
        mockWebServer.enqueue(new MockResponse()
                .setResponseCode(400)
                .setBody("{\"code\":\"invalid_request\",\"message\":\"Invalid trace ID\"}")
                .addHeader("Content-Type", "application/json"));

        // Make request and expect exception
        AITraceException exception = assertThrows(AITraceException.class, () -> {
            client.certs().commit("", EvidenceLevel.L1);
        });

        assertEquals("invalid_request", exception.getCode());
        assertEquals(400, exception.getStatusCode());
        assertTrue(exception.isClientError());
    }

    @Test
    void testEventBuilder() {
        Event event = Event.builder()
                .traceId("trace-123")
                .eventType(EventType.INPUT)
                .sequence(1)
                .addPayload("prompt", "Hello")
                .addPayload("model_id", "gpt-4")
                .build();

        assertEquals("trace-123", event.getTraceId());
        assertEquals(EventType.INPUT, event.getEventType());
        assertEquals(1, event.getSequence());
        assertEquals("Hello", event.getPayload().get("prompt"));
    }

    @Test
    void testEventHelpers() {
        Event input = Event.input("trace-123", "Hello", "gpt-4");
        assertEquals(EventType.INPUT, input.getEventType());
        assertEquals("Hello", input.getPayload().get("prompt"));

        Event output = Event.output("trace-123", "World", 10);
        assertEquals(EventType.OUTPUT, output.getEventType());
        assertEquals("World", output.getPayload().get("content"));

        Event error = Event.error("trace-123", "ERR001", "Something went wrong");
        assertEquals(EventType.ERROR, error.getEventType());
        assertEquals("ERR001", error.getPayload().get("error_code"));
    }

    @Test
    void testEvidenceLevelValidation() {
        assertTrue(EvidenceLevel.isValid(EvidenceLevel.L1));
        assertTrue(EvidenceLevel.isValid(EvidenceLevel.L2));
        assertTrue(EvidenceLevel.isValid(EvidenceLevel.L3));
        assertFalse(EvidenceLevel.isValid("L4"));
        assertFalse(EvidenceLevel.isValid("invalid"));
    }

    @Test
    void testMessageHelpers() {
        Message user = Message.user("Hello");
        assertEquals("user", user.getRole());
        assertEquals("Hello", user.getContent());

        Message assistant = Message.assistant("Hi there!");
        assertEquals("assistant", assistant.getRole());

        Message system = Message.system("You are helpful.");
        assertEquals("system", system.getRole());
    }

    @Test
    void testChatRequestBuilder() {
        ChatRequest request = ChatRequest.builder()
                .model("gpt-4")
                .addSystemMessage("You are a helpful assistant.")
                .addUserMessage("Hello!")
                .temperature(0.7)
                .maxTokens(100)
                .traceId("custom-trace")
                .sessionId("session-123")
                .build();

        assertEquals("gpt-4", request.getModel());
        assertEquals(2, request.getMessages().size());
        assertEquals(0.7, request.getTemperature());
        assertEquals(100, request.getMaxTokens());
        assertEquals("custom-trace", request.getTraceId());
        assertEquals("session-123", request.getSessionId());
    }
}
