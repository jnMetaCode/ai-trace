package dev.aitrace.sdk.exception;

/**
 * Exception thrown by AI-Trace SDK operations.
 */
public class AITraceException extends RuntimeException {

    private final String code;
    private final int statusCode;

    public AITraceException(String message) {
        super(message);
        this.code = null;
        this.statusCode = 0;
    }

    public AITraceException(String message, Throwable cause) {
        super(message, cause);
        this.code = null;
        this.statusCode = 0;
    }

    public AITraceException(String code, String message, int statusCode) {
        super(message);
        this.code = code;
        this.statusCode = statusCode;
    }

    public AITraceException(String code, String message, int statusCode, Throwable cause) {
        super(message, cause);
        this.code = code;
        this.statusCode = statusCode;
    }

    /**
     * Get the error code.
     */
    public String getCode() {
        return code;
    }

    /**
     * Get the HTTP status code.
     */
    public int getStatusCode() {
        return statusCode;
    }

    /**
     * Check if this is a client error (4xx).
     */
    public boolean isClientError() {
        return statusCode >= 400 && statusCode < 500;
    }

    /**
     * Check if this is a server error (5xx).
     */
    public boolean isServerError() {
        return statusCode >= 500;
    }

    @Override
    public String toString() {
        if (code != null) {
            return "AITraceException{code='" + code + "', message='" + getMessage() +
                   "', statusCode=" + statusCode + "}";
        }
        return "AITraceException{message='" + getMessage() + "'}";
    }
}
