package dev.aitrace.sdk.service;

import com.fasterxml.jackson.annotation.JsonProperty;
import dev.aitrace.sdk.AITraceClient;
import dev.aitrace.sdk.exception.AITraceException;
import dev.aitrace.sdk.model.Certificate;
import dev.aitrace.sdk.model.Event;
import dev.aitrace.sdk.model.EvidenceLevel;
import dev.aitrace.sdk.model.VerificationResult;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Service for certificate operations.
 */
public class CertsService {

    private final AITraceClient client;

    public CertsService(AITraceClient client) {
        this.client = client;
    }

    /**
     * Commit a certificate for a trace.
     *
     * @param traceId the trace ID
     * @param evidenceLevel the evidence level (L1, L2, or L3)
     * @return the committed certificate
     */
    public Certificate commit(String traceId, String evidenceLevel) throws AITraceException {
        // Input validation
        if (traceId == null || traceId.trim().isEmpty()) {
            throw new AITraceException("invalid_request", "trace_id is required", 400);
        }
        if (!EvidenceLevel.isValid(evidenceLevel)) {
            throw new AITraceException("invalid_request", "invalid evidence level: must be L1, L2, or L3", 400);
        }

        CommitRequest request = new CommitRequest();
        request.traceId = traceId;
        request.evidenceLevel = evidenceLevel;

        return client.post("/api/v1/certs/commit", request, null, Certificate.class);
    }

    /**
     * Commit a certificate with L1 (basic) evidence level.
     */
    public Certificate commitL1(String traceId) throws AITraceException {
        return commit(traceId, EvidenceLevel.L1);
    }

    /**
     * Commit a certificate with L2 (WORM storage) evidence level.
     */
    public Certificate commitL2(String traceId) throws AITraceException {
        return commit(traceId, EvidenceLevel.L2);
    }

    /**
     * Commit a certificate with L3 (blockchain anchor) evidence level.
     */
    public Certificate commitL3(String traceId) throws AITraceException {
        return commit(traceId, EvidenceLevel.L3);
    }

    /**
     * Verify a certificate's integrity.
     *
     * @param certId the certificate ID (optional if rootHash provided)
     * @param rootHash the root hash (optional if certId provided)
     * @return the verification result
     */
    public VerificationResult verify(String certId, String rootHash) throws AITraceException {
        // Input validation
        boolean hasCertId = certId != null && !certId.trim().isEmpty();
        boolean hasRootHash = rootHash != null && !rootHash.trim().isEmpty();
        if (!hasCertId && !hasRootHash) {
            throw new AITraceException("invalid_request", "either cert_id or root_hash is required", 400);
        }

        VerifyRequest request = new VerifyRequest();
        request.certId = certId;
        request.rootHash = rootHash;

        return client.post("/api/v1/certs/verify", request, null, VerificationResult.class);
    }

    /**
     * Verify a certificate by its ID.
     */
    public VerificationResult verifyByCertId(String certId) throws AITraceException {
        return verify(certId, null);
    }

    /**
     * Verify a certificate by its root hash.
     */
    public VerificationResult verifyByRootHash(String rootHash) throws AITraceException {
        return verify(null, rootHash);
    }

    /**
     * Search for certificates.
     *
     * @param page the page number
     * @param pageSize the page size
     * @return the search response
     */
    public SearchResponse search(int page, int pageSize) throws AITraceException {
        Map<String, String> params = new HashMap<>();
        params.put("page", String.valueOf(page));
        params.put("page_size", String.valueOf(pageSize));

        return client.get("/api/v1/certs/search", params, SearchResponse.class);
    }

    /**
     * Get a certificate by ID.
     */
    public Certificate get(String certId) throws AITraceException {
        if (certId == null || certId.trim().isEmpty()) {
            throw new AITraceException("invalid_request", "cert_id is required", 400);
        }
        return client.get("/api/v1/certs/" + certId, null, Certificate.class);
    }

    /**
     * Generate a minimal disclosure proof.
     *
     * @param certId the certificate ID
     * @param request the proof request
     * @return the proof response
     */
    public ProofResponse prove(String certId, ProveRequest request) throws AITraceException {
        if (certId == null || certId.trim().isEmpty()) {
            throw new AITraceException("invalid_request", "cert_id is required", 400);
        }
        return client.post("/api/v1/certs/" + certId + "/prove", request, null, ProofResponse.class);
    }

    /**
     * Generate a minimal disclosure proof with specific event indices.
     */
    public ProofResponse proveWithIndices(String certId, int... indices) throws AITraceException {
        ProveRequest request = new ProveRequest();
        request.discloseEvents = new int[indices.length];
        System.arraycopy(indices, 0, request.discloseEvents, 0, indices.length);
        return prove(certId, request);
    }

    /**
     * Get a certificate with its events.
     */
    public CertificateWithEvents getWithEvents(String certId) throws AITraceException {
        Certificate cert = get(certId);
        List<Event> events = client.events().getByTrace(cert.getTraceId());
        return new CertificateWithEvents(cert, events);
    }

    // Request/Response classes

    public static class CommitRequest {
        @JsonProperty("trace_id")
        public String traceId;

        @JsonProperty("evidence_level")
        public String evidenceLevel;
    }

    public static class VerifyRequest {
        @JsonProperty("cert_id")
        public String certId;

        @JsonProperty("root_hash")
        public String rootHash;
    }

    public static class ProveRequest {
        @JsonProperty("disclose_events")
        public int[] discloseEvents;

        @JsonProperty("disclose_fields")
        public String[] discloseFields;
    }

    public static class ProofResponse {
        @JsonProperty("cert_id")
        public String certId;

        @JsonProperty("root_hash")
        public String rootHash;

        @JsonProperty("disclosed_events")
        public List<Event> disclosedEvents;

        @JsonProperty("merkle_proofs")
        public List<MerkleProof> merkleProofs;

        @JsonProperty("metadata")
        public Map<String, Object> metadata;
    }

    public static class MerkleProof {
        @JsonProperty("event_index")
        public int eventIndex;

        @JsonProperty("siblings")
        public List<String> siblings;

        @JsonProperty("direction")
        public List<Integer> direction;
    }

    public static class SearchResponse {
        @JsonProperty("certificates")
        public List<Certificate> certificates;

        @JsonProperty("total")
        public int total;

        @JsonProperty("page")
        public int page;

        @JsonProperty("page_size")
        public int pageSize;

        @JsonProperty("total_pages")
        public int totalPages;
    }

    public static class CertificateWithEvents {
        private final Certificate certificate;
        private final List<Event> events;

        public CertificateWithEvents(Certificate certificate, List<Event> events) {
            this.certificate = certificate;
            this.events = events;
        }

        public Certificate getCertificate() {
            return certificate;
        }

        public List<Event> getEvents() {
            return events;
        }
    }
}
