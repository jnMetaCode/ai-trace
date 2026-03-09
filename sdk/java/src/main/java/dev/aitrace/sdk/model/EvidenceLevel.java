package dev.aitrace.sdk.model;

/**
 * Evidence level constants.
 */
public final class EvidenceLevel {
    /**
     * L1: Basic attestation with Merkle tree and timestamp.
     */
    public static final String L1 = "L1";

    /**
     * L2: WORM (Write Once Read Many) storage for legal compliance.
     */
    public static final String L2 = "L2";

    /**
     * L3: Blockchain anchor for maximum tamper-proof guarantee.
     */
    public static final String L3 = "L3";

    private EvidenceLevel() {}

    /**
     * Check if the evidence level is valid.
     */
    public static boolean isValid(String level) {
        return L1.equals(level) || L2.equals(level) || L3.equals(level);
    }
}
