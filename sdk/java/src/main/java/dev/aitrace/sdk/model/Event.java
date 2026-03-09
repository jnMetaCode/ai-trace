package dev.aitrace.sdk.model;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Represents an AI-Trace event.
 */
@JsonInclude(JsonInclude.Include.NON_NULL)
public class Event {

    @JsonProperty("event_id")
    private String eventId;

    @JsonProperty("trace_id")
    private String traceId;

    @JsonProperty("event_type")
    private String eventType;

    @JsonProperty("timestamp")
    private Instant timestamp;

    @JsonProperty("sequence")
    private Integer sequence;

    @JsonProperty("payload")
    private Map<String, Object> payload;

    @JsonProperty("event_hash")
    private String eventHash;

    @JsonProperty("prev_event_hash")
    private String prevEventHash;

    @JsonProperty("payload_hash")
    private String payloadHash;

    public Event() {
        this.eventId = UUID.randomUUID().toString();
        this.timestamp = Instant.now();
        this.payload = new HashMap<>();
    }

    public static Builder builder() {
        return new Builder();
    }

    /**
     * Create an input event.
     */
    public static Event input(String traceId, String prompt, String modelId) {
        return builder()
                .traceId(traceId)
                .eventType(EventType.INPUT)
                .addPayload("prompt", prompt)
                .addPayload("model_id", modelId)
                .build();
    }

    /**
     * Create an output event.
     */
    public static Event output(String traceId, String content, int tokensUsed) {
        return builder()
                .traceId(traceId)
                .eventType(EventType.OUTPUT)
                .addPayload("content", content)
                .addPayload("tokens_used", tokensUsed)
                .build();
    }

    /**
     * Create a tool call event.
     */
    public static Event toolCall(String traceId, String toolName, Map<String, Object> arguments) {
        return builder()
                .traceId(traceId)
                .eventType(EventType.TOOL_CALL)
                .addPayload("tool_name", toolName)
                .addPayload("arguments", arguments)
                .build();
    }

    /**
     * Create an error event.
     */
    public static Event error(String traceId, String errorCode, String errorMessage) {
        return builder()
                .traceId(traceId)
                .eventType(EventType.ERROR)
                .addPayload("error_code", errorCode)
                .addPayload("error_message", errorMessage)
                .build();
    }

    public static class Builder {
        private final Event event = new Event();

        public Builder eventId(String eventId) {
            event.eventId = eventId;
            return this;
        }

        public Builder traceId(String traceId) {
            event.traceId = traceId;
            return this;
        }

        public Builder eventType(String eventType) {
            event.eventType = eventType;
            return this;
        }

        public Builder timestamp(Instant timestamp) {
            event.timestamp = timestamp;
            return this;
        }

        public Builder sequence(Integer sequence) {
            event.sequence = sequence;
            return this;
        }

        public Builder payload(Map<String, Object> payload) {
            event.payload = payload;
            return this;
        }

        public Builder addPayload(String key, Object value) {
            event.payload.put(key, value);
            return this;
        }

        public Builder prevEventHash(String prevEventHash) {
            event.prevEventHash = prevEventHash;
            return this;
        }

        public Event build() {
            return event;
        }
    }

    // Getters and Setters
    public String getEventId() {
        return eventId;
    }

    public void setEventId(String eventId) {
        this.eventId = eventId;
    }

    public String getTraceId() {
        return traceId;
    }

    public void setTraceId(String traceId) {
        this.traceId = traceId;
    }

    public String getEventType() {
        return eventType;
    }

    public void setEventType(String eventType) {
        this.eventType = eventType;
    }

    public Instant getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Instant timestamp) {
        this.timestamp = timestamp;
    }

    public Integer getSequence() {
        return sequence;
    }

    public void setSequence(Integer sequence) {
        this.sequence = sequence;
    }

    public Map<String, Object> getPayload() {
        return payload;
    }

    public void setPayload(Map<String, Object> payload) {
        this.payload = payload;
    }

    public String getEventHash() {
        return eventHash;
    }

    public void setEventHash(String eventHash) {
        this.eventHash = eventHash;
    }

    public String getPrevEventHash() {
        return prevEventHash;
    }

    public void setPrevEventHash(String prevEventHash) {
        this.prevEventHash = prevEventHash;
    }

    public String getPayloadHash() {
        return payloadHash;
    }

    public void setPayloadHash(String payloadHash) {
        this.payloadHash = payloadHash;
    }

    @Override
    public String toString() {
        return "Event{eventId='" + eventId + "', traceId='" + traceId +
               "', eventType='" + eventType + "'}";
    }
}
