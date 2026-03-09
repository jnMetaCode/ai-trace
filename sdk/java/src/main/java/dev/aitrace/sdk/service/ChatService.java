package dev.aitrace.sdk.service;

import dev.aitrace.sdk.AITraceClient;
import dev.aitrace.sdk.exception.AITraceException;
import dev.aitrace.sdk.model.Certificate;
import dev.aitrace.sdk.model.ChatRequest;
import dev.aitrace.sdk.model.ChatResponse;

import java.util.HashMap;
import java.util.Map;
import java.util.function.Consumer;

/**
 * Service for chat completion operations.
 */
public class ChatService {

    private final AITraceClient client;
    private String lastTraceId;

    public ChatService(AITraceClient client) {
        this.client = client;
    }

    /**
     * Create a chat completion with AI-Trace attestation.
     *
     * @param request the chat request
     * @return the chat response
     * @throws AITraceException if the request fails
     */
    public ChatResponse create(ChatRequest request) throws AITraceException {
        // Input validation
        if (request == null) {
            throw new AITraceException("invalid_request", "request is required", 400);
        }
        if (request.getModel() == null || request.getModel().trim().isEmpty()) {
            throw new AITraceException("invalid_request", "model is required", 400);
        }
        if (request.getMessages() == null || request.getMessages().isEmpty()) {
            throw new AITraceException("invalid_request", "at least one message is required", 400);
        }

        Map<String, String> headers = new HashMap<>();

        if (request.getTraceId() != null) {
            headers.put("X-Trace-ID", request.getTraceId());
        }
        if (request.getSessionId() != null) {
            headers.put("X-Session-ID", request.getSessionId());
        }
        if (request.getBusinessId() != null) {
            headers.put("X-Business-ID", request.getBusinessId());
        }

        ChatResponse response = client.post("/api/v1/chat/completions", request, headers, ChatResponse.class);

        // Store trace ID for later use
        if (response.getTraceId() != null) {
            lastTraceId = response.getTraceId();
        } else if (request.getTraceId() != null) {
            lastTraceId = request.getTraceId();
        }

        return response;
    }

    /**
     * Create a chat completion and execute a callback with the trace ID.
     *
     * @param request the chat request
     * @param callback callback to execute with the trace ID
     * @return the chat response
     */
    public ChatResponse createWithCallback(ChatRequest request, Consumer<String> callback) throws AITraceException {
        ChatResponse response = create(request);
        if (callback != null && lastTraceId != null) {
            callback.accept(lastTraceId);
        }
        return response;
    }

    /**
     * Create a chat completion and immediately commit a certificate.
     *
     * @param request the chat request
     * @param evidenceLevel the evidence level (L1, L2, or L3)
     * @return a result containing both the chat response and certificate
     */
    public ChatAndCertResult createAndCommit(ChatRequest request, String evidenceLevel) throws AITraceException {
        ChatResponse response = create(request);

        if (lastTraceId == null) {
            throw new AITraceException("No trace ID available for commitment");
        }

        Certificate certificate = client.certs().commit(lastTraceId, evidenceLevel);
        return new ChatAndCertResult(response, certificate);
    }

    /**
     * Get the trace ID from the last request.
     */
    public String getLastTraceId() {
        return lastTraceId;
    }

    /**
     * Result containing both chat response and certificate.
     */
    public static class ChatAndCertResult {
        private final ChatResponse chatResponse;
        private final Certificate certificate;

        public ChatAndCertResult(ChatResponse chatResponse, Certificate certificate) {
            this.chatResponse = chatResponse;
            this.certificate = certificate;
        }

        public ChatResponse getChatResponse() {
            return chatResponse;
        }

        public Certificate getCertificate() {
            return certificate;
        }
    }
}
