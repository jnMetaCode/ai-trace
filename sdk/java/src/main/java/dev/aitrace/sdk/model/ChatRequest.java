package dev.aitrace.sdk.model;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.ArrayList;
import java.util.List;

/**
 * Represents a chat completion request.
 */
@JsonInclude(JsonInclude.Include.NON_NULL)
public class ChatRequest {

    @JsonProperty("model")
    private String model;

    @JsonProperty("messages")
    private List<Message> messages = new ArrayList<>();

    @JsonProperty("temperature")
    private Double temperature;

    @JsonProperty("max_tokens")
    private Integer maxTokens;

    @JsonProperty("top_p")
    private Double topP;

    @JsonProperty("n")
    private Integer n;

    @JsonProperty("stream")
    private Boolean stream;

    @JsonProperty("stop")
    private List<String> stop;

    // AI-Trace specific fields (sent via headers)
    @JsonIgnore
    private String traceId;

    @JsonIgnore
    private String sessionId;

    @JsonIgnore
    private String businessId;

    public ChatRequest() {}

    public static Builder builder() {
        return new Builder();
    }

    // Builder pattern
    public static class Builder {
        private final ChatRequest request = new ChatRequest();

        public Builder model(String model) {
            request.model = model;
            return this;
        }

        public Builder messages(List<Message> messages) {
            request.messages = messages;
            return this;
        }

        public Builder addMessage(Message message) {
            request.messages.add(message);
            return this;
        }

        public Builder addUserMessage(String content) {
            request.messages.add(Message.user(content));
            return this;
        }

        public Builder addSystemMessage(String content) {
            request.messages.add(Message.system(content));
            return this;
        }

        public Builder temperature(Double temperature) {
            request.temperature = temperature;
            return this;
        }

        public Builder maxTokens(Integer maxTokens) {
            request.maxTokens = maxTokens;
            return this;
        }

        public Builder topP(Double topP) {
            request.topP = topP;
            return this;
        }

        public Builder n(Integer n) {
            request.n = n;
            return this;
        }

        public Builder stream(Boolean stream) {
            request.stream = stream;
            return this;
        }

        public Builder stop(List<String> stop) {
            request.stop = stop;
            return this;
        }

        public Builder traceId(String traceId) {
            request.traceId = traceId;
            return this;
        }

        public Builder sessionId(String sessionId) {
            request.sessionId = sessionId;
            return this;
        }

        public Builder businessId(String businessId) {
            request.businessId = businessId;
            return this;
        }

        public ChatRequest build() {
            return request;
        }
    }

    // Getters and Setters
    public String getModel() {
        return model;
    }

    public void setModel(String model) {
        this.model = model;
    }

    public List<Message> getMessages() {
        return messages;
    }

    public void setMessages(List<Message> messages) {
        this.messages = messages;
    }

    public Double getTemperature() {
        return temperature;
    }

    public void setTemperature(Double temperature) {
        this.temperature = temperature;
    }

    public Integer getMaxTokens() {
        return maxTokens;
    }

    public void setMaxTokens(Integer maxTokens) {
        this.maxTokens = maxTokens;
    }

    public Double getTopP() {
        return topP;
    }

    public void setTopP(Double topP) {
        this.topP = topP;
    }

    public Integer getN() {
        return n;
    }

    public void setN(Integer n) {
        this.n = n;
    }

    public Boolean getStream() {
        return stream;
    }

    public void setStream(Boolean stream) {
        this.stream = stream;
    }

    public List<String> getStop() {
        return stop;
    }

    public void setStop(List<String> stop) {
        this.stop = stop;
    }

    public String getTraceId() {
        return traceId;
    }

    public void setTraceId(String traceId) {
        this.traceId = traceId;
    }

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getBusinessId() {
        return businessId;
    }

    public void setBusinessId(String businessId) {
        this.businessId = businessId;
    }
}
