package dev.aitrace.sdk;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import dev.aitrace.sdk.exception.AITraceException;
import dev.aitrace.sdk.service.CertsService;
import dev.aitrace.sdk.service.ChatService;
import dev.aitrace.sdk.service.EventsService;
import okhttp3.*;

import java.io.IOException;
import java.time.Duration;
import java.util.Map;

/**
 * Main client for the AI-Trace platform.
 * <p>
 * Example usage:
 * <pre>{@code
 * AITraceClient client = AITraceClient.builder()
 *     .apiKey("your-api-key")
 *     .upstreamApiKey("sk-your-openai-key")
 *     .build();
 *
 * ChatResponse response = client.chat().create(ChatRequest.builder()
 *     .model("gpt-4")
 *     .addUserMessage("Hello!")
 *     .build());
 *
 * Certificate cert = client.certs().commit(response.getTraceId(), EvidenceLevel.L2);
 * }</pre>
 */
public class AITraceClient {

    private static final String DEFAULT_BASE_URL = "https://api.ai-trace.dev";
    private static final Duration DEFAULT_TIMEOUT = Duration.ofSeconds(120);
    private static final MediaType JSON = MediaType.parse("application/json; charset=utf-8");

    private final String apiKey;
    private final String baseUrl;
    private final String upstreamApiKey;
    private final String upstreamBaseUrl;
    private final OkHttpClient httpClient;
    private final ObjectMapper objectMapper;

    // Services
    private final ChatService chatService;
    private final EventsService eventsService;
    private final CertsService certsService;

    private AITraceClient(Builder builder) {
        this.apiKey = builder.apiKey;
        String url = builder.baseUrl != null ? builder.baseUrl : DEFAULT_BASE_URL;
        // Strip trailing slash to avoid double slashes when concatenating paths
        this.baseUrl = url.endsWith("/") ? url.substring(0, url.length() - 1) : url;
        this.upstreamApiKey = builder.upstreamApiKey;
        this.upstreamBaseUrl = builder.upstreamBaseUrl;

        Duration timeout = builder.timeout != null ? builder.timeout : DEFAULT_TIMEOUT;
        this.httpClient = builder.httpClient != null ? builder.httpClient : new OkHttpClient.Builder()
                .connectTimeout(timeout)
                .readTimeout(timeout)
                .writeTimeout(timeout)
                .build();

        this.objectMapper = new ObjectMapper()
                .registerModule(new JavaTimeModule())
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

        // Initialize services
        this.chatService = new ChatService(this);
        this.eventsService = new EventsService(this);
        this.certsService = new CertsService(this);
    }

    /**
     * Create a new builder.
     */
    public static Builder builder() {
        return new Builder();
    }

    /**
     * Create a client with just an API key.
     */
    public static AITraceClient create(String apiKey) {
        return builder().apiKey(apiKey).build();
    }

    /**
     * Get the chat service.
     */
    public ChatService chat() {
        return chatService;
    }

    /**
     * Get the events service.
     */
    public EventsService events() {
        return eventsService;
    }

    /**
     * Get the certs service.
     */
    public CertsService certs() {
        return certsService;
    }

    /**
     * Execute a GET request.
     */
    public <T> T get(String path, Map<String, String> params, Class<T> responseType) throws AITraceException {
        HttpUrl.Builder urlBuilder = HttpUrl.parse(baseUrl + path).newBuilder();
        if (params != null) {
            params.forEach(urlBuilder::addQueryParameter);
        }

        Request request = new Request.Builder()
                .url(urlBuilder.build())
                .addHeader("X-API-Key", apiKey)
                .get()
                .build();

        return executeRequest(request, responseType);
    }

    /**
     * Execute a POST request.
     */
    public <T> T post(String path, Object body, Map<String, String> headers, Class<T> responseType) throws AITraceException {
        try {
            String jsonBody = objectMapper.writeValueAsString(body);
            RequestBody requestBody = RequestBody.create(jsonBody, JSON);

            Request.Builder requestBuilder = new Request.Builder()
                    .url(baseUrl + path)
                    .addHeader("X-API-Key", apiKey)
                    .addHeader("Content-Type", "application/json")
                    .post(requestBody);

            // Add upstream headers
            if (upstreamApiKey != null) {
                requestBuilder.addHeader("X-Upstream-API-Key", upstreamApiKey);
            }
            if (upstreamBaseUrl != null) {
                requestBuilder.addHeader("X-Upstream-Base-URL", upstreamBaseUrl);
            }

            // Add custom headers
            if (headers != null) {
                headers.forEach(requestBuilder::addHeader);
            }

            return executeRequest(requestBuilder.build(), responseType);
        } catch (IOException e) {
            throw new AITraceException("Failed to serialize request body", e);
        }
    }

    private <T> T executeRequest(Request request, Class<T> responseType) throws AITraceException {
        try (Response response = httpClient.newCall(request).execute()) {
            String responseBody = response.body() != null ? response.body().string() : "";

            if (!response.isSuccessful()) {
                try {
                    ErrorResponse error = objectMapper.readValue(responseBody, ErrorResponse.class);
                    throw new AITraceException(error.code, error.message, response.code());
                } catch (IOException e) {
                    throw new AITraceException("HTTP_" + response.code(), responseBody, response.code());
                }
            }

            if (responseType == Void.class) {
                return null;
            }

            return objectMapper.readValue(responseBody, responseType);
        } catch (IOException e) {
            throw new AITraceException("Request failed", e);
        }
    }

    /**
     * Get the object mapper for serialization.
     */
    public ObjectMapper getObjectMapper() {
        return objectMapper;
    }

    /**
     * Builder for AITraceClient.
     */
    public static class Builder {
        private String apiKey;
        private String baseUrl;
        private String upstreamApiKey;
        private String upstreamBaseUrl;
        private Duration timeout;
        private OkHttpClient httpClient;

        /**
         * Set the AI-Trace API key.
         */
        public Builder apiKey(String apiKey) {
            this.apiKey = apiKey;
            return this;
        }

        /**
         * Set the base URL.
         */
        public Builder baseUrl(String baseUrl) {
            this.baseUrl = baseUrl;
            return this;
        }

        /**
         * Set the upstream API key (e.g., OpenAI API key).
         * This key is passed through to the upstream provider and never stored.
         */
        public Builder upstreamApiKey(String upstreamApiKey) {
            this.upstreamApiKey = upstreamApiKey;
            return this;
        }

        /**
         * Set a custom upstream base URL (e.g., for proxy).
         */
        public Builder upstreamBaseUrl(String upstreamBaseUrl) {
            this.upstreamBaseUrl = upstreamBaseUrl;
            return this;
        }

        /**
         * Set the request timeout.
         */
        public Builder timeout(Duration timeout) {
            this.timeout = timeout;
            return this;
        }

        /**
         * Set a custom HTTP client.
         */
        public Builder httpClient(OkHttpClient httpClient) {
            this.httpClient = httpClient;
            return this;
        }

        /**
         * Build the client.
         */
        public AITraceClient build() {
            if (apiKey == null || apiKey.isEmpty()) {
                throw new IllegalArgumentException("API key is required");
            }
            return new AITraceClient(this);
        }
    }

    /**
     * Internal error response structure.
     */
    private static class ErrorResponse {
        public String code;
        public String message;
    }
}
