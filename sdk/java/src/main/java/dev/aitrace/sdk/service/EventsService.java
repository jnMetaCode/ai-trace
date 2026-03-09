package dev.aitrace.sdk.service;

import com.fasterxml.jackson.annotation.JsonProperty;
import dev.aitrace.sdk.AITraceClient;
import dev.aitrace.sdk.exception.AITraceException;
import dev.aitrace.sdk.model.Event;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Service for event operations.
 */
public class EventsService {

    private final AITraceClient client;

    public EventsService(AITraceClient client) {
        this.client = client;
    }

    /**
     * Ingest a batch of events.
     *
     * @param events the events to ingest
     * @return the ingest response
     */
    public IngestResponse ingest(List<Event> events) throws AITraceException {
        // Input validation
        if (events == null || events.isEmpty()) {
            throw new AITraceException("invalid_request", "at least one event is required", 400);
        }
        for (int i = 0; i < events.size(); i++) {
            Event e = events.get(i);
            if (e.getTraceId() == null || e.getTraceId().trim().isEmpty()) {
                throw new AITraceException("invalid_request", "event[" + i + "]: trace_id is required", 400);
            }
            if (e.getEventType() == null) {
                throw new AITraceException("invalid_request", "event[" + i + "]: event_type is required", 400);
            }
        }

        IngestRequest request = new IngestRequest();
        request.events = events;
        return client.post("/api/v1/events/ingest", request, null, IngestResponse.class);
    }

    /**
     * Search for events.
     *
     * @param request the search request
     * @return the search response
     */
    public SearchResponse search(SearchRequest request) throws AITraceException {
        Map<String, String> params = new HashMap<>();
        if (request.traceId != null) {
            params.put("trace_id", request.traceId);
        }
        if (request.eventType != null) {
            params.put("event_type", request.eventType);
        }
        if (request.startTime != null) {
            params.put("start_time", request.startTime);
        }
        if (request.endTime != null) {
            params.put("end_time", request.endTime);
        }
        if (request.page > 0) {
            params.put("page", String.valueOf(request.page));
        }
        if (request.pageSize > 0) {
            params.put("page_size", String.valueOf(request.pageSize));
        }
        return client.get("/api/v1/events/search", params, SearchResponse.class);
    }

    /**
     * Get a single event by ID.
     *
     * @param eventId the event ID
     * @return the event
     */
    public Event get(String eventId) throws AITraceException {
        if (eventId == null || eventId.trim().isEmpty()) {
            throw new AITraceException("invalid_request", "event_id is required", 400);
        }
        return client.get("/api/v1/events/" + eventId, null, Event.class);
    }

    /**
     * Get all events for a trace.
     *
     * @param traceId the trace ID
     * @return list of events
     */
    public List<Event> getByTrace(String traceId) throws AITraceException {
        if (traceId == null || traceId.trim().isEmpty()) {
            throw new AITraceException("invalid_request", "trace_id is required", 400);
        }
        SearchRequest request = new SearchRequest();
        request.traceId = traceId;
        request.pageSize = 1000;
        SearchResponse response = search(request);
        return response.events;
    }

    // Request/Response classes

    public static class IngestRequest {
        @JsonProperty("events")
        public List<Event> events;
    }

    public static class IngestResponse {
        @JsonProperty("ingested")
        public int ingested;

        @JsonProperty("event_ids")
        public List<String> eventIds;
    }

    public static class SearchRequest {
        public String traceId;
        public String eventType;
        public String startTime;
        public String endTime;
        public int page = 1;
        public int pageSize = 20;

        public static SearchRequest forTrace(String traceId) {
            SearchRequest request = new SearchRequest();
            request.traceId = traceId;
            return request;
        }
    }

    public static class SearchResponse {
        @JsonProperty("events")
        public List<Event> events;

        @JsonProperty("total")
        public int total;

        @JsonProperty("page")
        public int page;

        @JsonProperty("page_size")
        public int pageSize;

        @JsonProperty("total_pages")
        public int totalPages;
    }
}
