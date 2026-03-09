package dev.aitrace.sdk.model;

/**
 * Event type constants.
 */
public final class EventType {
    public static final String INPUT = "llm.input";
    public static final String OUTPUT = "llm.output";
    public static final String CHUNK = "llm.chunk";
    public static final String TOOL_CALL = "llm.tool_call";
    public static final String TOOL_RESULT = "llm.tool_result";
    public static final String ERROR = "llm.error";

    private EventType() {}
}
