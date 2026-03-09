package dev.aitrace.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Map;

/**
 * Represents a verification result.
 */
public class VerificationResult {

    @JsonProperty("valid")
    private Boolean valid;

    @JsonProperty("checks")
    private Map<String, Object> checks;

    @JsonProperty("certificate")
    private Certificate certificate;

    public Boolean isValid() {
        return valid;
    }

    public void setValid(Boolean valid) {
        this.valid = valid;
    }

    public Map<String, Object> getChecks() {
        return checks;
    }

    public void setChecks(Map<String, Object> checks) {
        this.checks = checks;
    }

    public Certificate getCertificate() {
        return certificate;
    }

    public void setCertificate(Certificate certificate) {
        this.certificate = certificate;
    }

    @Override
    public String toString() {
        return "VerificationResult{valid=" + valid + ", checks=" + checks + "}";
    }
}
