package dev.aitrace.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Instant;

/**
 * Represents an attestation certificate.
 */
public class Certificate {

    @JsonProperty("cert_id")
    private String certId;

    @JsonProperty("trace_id")
    private String traceId;

    @JsonProperty("root_hash")
    private String rootHash;

    @JsonProperty("event_count")
    private Integer eventCount;

    @JsonProperty("evidence_level")
    private String evidenceLevel;

    @JsonProperty("created_at")
    private Instant createdAt;

    @JsonProperty("time_proof")
    private TimeProof timeProof;

    @JsonProperty("anchor_proof")
    private AnchorProof anchorProof;

    // Getters and Setters
    public String getCertId() {
        return certId;
    }

    public void setCertId(String certId) {
        this.certId = certId;
    }

    public String getTraceId() {
        return traceId;
    }

    public void setTraceId(String traceId) {
        this.traceId = traceId;
    }

    public String getRootHash() {
        return rootHash;
    }

    public void setRootHash(String rootHash) {
        this.rootHash = rootHash;
    }

    public Integer getEventCount() {
        return eventCount;
    }

    public void setEventCount(Integer eventCount) {
        this.eventCount = eventCount;
    }

    public String getEvidenceLevel() {
        return evidenceLevel;
    }

    public void setEvidenceLevel(String evidenceLevel) {
        this.evidenceLevel = evidenceLevel;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(Instant createdAt) {
        this.createdAt = createdAt;
    }

    public TimeProof getTimeProof() {
        return timeProof;
    }

    public void setTimeProof(TimeProof timeProof) {
        this.timeProof = timeProof;
    }

    public AnchorProof getAnchorProof() {
        return anchorProof;
    }

    public void setAnchorProof(AnchorProof anchorProof) {
        this.anchorProof = anchorProof;
    }

    @Override
    public String toString() {
        return "Certificate{certId='" + certId + "', traceId='" + traceId +
               "', rootHash='" + rootHash + "', evidenceLevel='" + evidenceLevel + "'}";
    }

    /**
     * Represents a timestamp proof.
     */
    public static class TimeProof {
        @JsonProperty("timestamp")
        private Instant timestamp;

        @JsonProperty("timestamp_id")
        private String timestampId;

        @JsonProperty("tsa_name")
        private String tsaName;

        @JsonProperty("tsa_hash")
        private String tsaHash;

        public Instant getTimestamp() {
            return timestamp;
        }

        public void setTimestamp(Instant timestamp) {
            this.timestamp = timestamp;
        }

        public String getTimestampId() {
            return timestampId;
        }

        public void setTimestampId(String timestampId) {
            this.timestampId = timestampId;
        }

        public String getTsaName() {
            return tsaName;
        }

        public void setTsaName(String tsaName) {
            this.tsaName = tsaName;
        }

        public String getTsaHash() {
            return tsaHash;
        }

        public void setTsaHash(String tsaHash) {
            this.tsaHash = tsaHash;
        }
    }

    /**
     * Represents a blockchain anchor proof.
     */
    public static class AnchorProof {
        @JsonProperty("network")
        private String network;

        @JsonProperty("transaction_id")
        private String transactionId;

        @JsonProperty("block_number")
        private Long blockNumber;

        @JsonProperty("block_hash")
        private String blockHash;

        @JsonProperty("timestamp")
        private Long timestamp;

        public String getNetwork() {
            return network;
        }

        public void setNetwork(String network) {
            this.network = network;
        }

        public String getTransactionId() {
            return transactionId;
        }

        public void setTransactionId(String transactionId) {
            this.transactionId = transactionId;
        }

        public Long getBlockNumber() {
            return blockNumber;
        }

        public void setBlockNumber(Long blockNumber) {
            this.blockNumber = blockNumber;
        }

        public String getBlockHash() {
            return blockHash;
        }

        public void setBlockHash(String blockHash) {
            this.blockHash = blockHash;
        }

        public Long getTimestamp() {
            return timestamp;
        }

        public void setTimestamp(Long timestamp) {
            this.timestamp = timestamp;
        }
    }
}
