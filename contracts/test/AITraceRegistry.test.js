const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("AITraceRegistry", function () {
  let registry;
  let owner;
  let addr1;
  let addr2;

  beforeEach(async function () {
    [owner, addr1, addr2] = await ethers.getSigners();
    const AITraceRegistry = await ethers.getContractFactory("AITraceRegistry");
    registry = await AITraceRegistry.deploy();
    await registry.waitForDeployment();
  });

  describe("Deployment", function () {
    it("Should set the right owner", async function () {
      expect(await registry.owner()).to.equal(owner.address);
    });

    it("Should authorize owner as submitter", async function () {
      expect(await registry.authorizedSubmitters(owner.address)).to.equal(true);
    });

    it("Should start with zero attestations", async function () {
      expect(await registry.totalAttestations()).to.equal(0);
    });
  });

  describe("Attestation Creation", function () {
    const certId = ethers.keccak256(ethers.toUtf8Bytes("cert-001"));
    const merkleRoot = ethers.keccak256(ethers.toUtf8Bytes("merkle-root"));
    const fingerprintHash = ethers.keccak256(ethers.toUtf8Bytes("fingerprint"));
    const inputHash = ethers.keccak256(ethers.toUtf8Bytes("input"));
    const outputHash = ethers.keccak256(ethers.toUtf8Bytes("output"));
    const modelId = "gpt-4";
    const tenantId = "tenant-001";

    it("Should create attestation successfully", async function () {
      await registry.createAttestation(
        certId,
        merkleRoot,
        fingerprintHash,
        inputHash,
        outputHash,
        modelId,
        tenantId
      );

      const attestation = await registry.getAttestation(certId);
      expect(attestation.certId).to.equal(certId);
      expect(attestation.merkleRoot).to.equal(merkleRoot);
      expect(attestation.fingerprintHash).to.equal(fingerprintHash);
      expect(attestation.isValid).to.equal(true);
      expect(attestation.modelId).to.equal(modelId);
    });

    it("Should emit AttestationCreated event", async function () {
      await expect(
        registry.createAttestation(
          certId,
          merkleRoot,
          fingerprintHash,
          inputHash,
          outputHash,
          modelId,
          tenantId
        )
      )
        .to.emit(registry, "AttestationCreated")
        .withArgs(
          certId,
          merkleRoot,
          fingerprintHash,
          owner.address,
          modelId,
          tenantId,
          await getBlockTimestamp()
        );
    });

    it("Should increment total attestations", async function () {
      await registry.createAttestation(
        certId,
        merkleRoot,
        fingerprintHash,
        inputHash,
        outputHash,
        modelId,
        tenantId
      );
      expect(await registry.totalAttestations()).to.equal(1);
    });

    it("Should reject duplicate certId", async function () {
      await registry.createAttestation(
        certId,
        merkleRoot,
        fingerprintHash,
        inputHash,
        outputHash,
        modelId,
        tenantId
      );

      await expect(
        registry.createAttestation(
          certId,
          merkleRoot,
          fingerprintHash,
          inputHash,
          outputHash,
          modelId,
          tenantId
        )
      ).to.be.revertedWith("AITraceRegistry: attestation exists");
    });

    it("Should reject unauthorized submitter", async function () {
      await expect(
        registry.connect(addr1).createAttestation(
          certId,
          merkleRoot,
          fingerprintHash,
          inputHash,
          outputHash,
          modelId,
          tenantId
        )
      ).to.be.revertedWith("AITraceRegistry: caller is not authorized");
    });
  });

  describe("Simple Anchor Function", function () {
    const certHash = ethers.keccak256(ethers.toUtf8Bytes("simple-cert"));
    const rootHash = ethers.keccak256(ethers.toUtf8Bytes("root-hash"));
    const timestamp = Math.floor(Date.now() / 1000);

    it("Should anchor successfully", async function () {
      await registry.anchor(certHash, rootHash, timestamp);

      const attestation = await registry.getAttestation(certHash);
      expect(attestation.merkleRoot).to.equal(rootHash);
      expect(attestation.isValid).to.equal(true);
    });
  });

  describe("Batch Attestations", function () {
    it("Should create batch attestations", async function () {
      const certIds = [
        ethers.keccak256(ethers.toUtf8Bytes("batch-1")),
        ethers.keccak256(ethers.toUtf8Bytes("batch-2")),
        ethers.keccak256(ethers.toUtf8Bytes("batch-3")),
      ];
      const merkleRoots = [
        ethers.keccak256(ethers.toUtf8Bytes("root-1")),
        ethers.keccak256(ethers.toUtf8Bytes("root-2")),
        ethers.keccak256(ethers.toUtf8Bytes("root-3")),
      ];
      const fingerprintHashes = [
        ethers.keccak256(ethers.toUtf8Bytes("fp-1")),
        ethers.keccak256(ethers.toUtf8Bytes("fp-2")),
        ethers.keccak256(ethers.toUtf8Bytes("fp-3")),
      ];

      await registry.batchCreateAttestations(certIds, merkleRoots, fingerprintHashes);

      expect(await registry.totalAttestations()).to.equal(3);

      for (let i = 0; i < certIds.length; i++) {
        const attestation = await registry.getAttestation(certIds[i]);
        expect(attestation.isValid).to.equal(true);
        expect(attestation.merkleRoot).to.equal(merkleRoots[i]);
      }
    });
  });

  describe("Verification", function () {
    const certId = ethers.keccak256(ethers.toUtf8Bytes("verify-cert"));
    const merkleRoot = ethers.keccak256(ethers.toUtf8Bytes("verify-root"));
    const fingerprintHash = ethers.keccak256(ethers.toUtf8Bytes("verify-fp"));

    beforeEach(async function () {
      await registry.createAttestation(
        certId,
        merkleRoot,
        fingerprintHash,
        ethers.ZeroHash,
        ethers.ZeroHash,
        "gpt-4",
        "tenant"
      );
    });

    it("Should verify valid attestation", async function () {
      expect(await registry.verifyAttestation(certId, merkleRoot)).to.equal(true);
    });

    it("Should reject invalid merkle root", async function () {
      const wrongRoot = ethers.keccak256(ethers.toUtf8Bytes("wrong"));
      expect(await registry.verifyAttestation(certId, wrongRoot)).to.equal(false);
    });

    it("Should verify fingerprint", async function () {
      expect(await registry.verifyFingerprint(certId, fingerprintHash)).to.equal(true);
    });

    it("Should check attestation exists", async function () {
      expect(await registry.attestationExists(certId)).to.equal(true);
      expect(await registry.attestationExists(ethers.ZeroHash)).to.equal(false);
    });
  });

  describe("Revocation", function () {
    const certId = ethers.keccak256(ethers.toUtf8Bytes("revoke-cert"));
    const merkleRoot = ethers.keccak256(ethers.toUtf8Bytes("revoke-root"));

    beforeEach(async function () {
      await registry.createAttestation(
        certId,
        merkleRoot,
        ethers.ZeroHash,
        ethers.ZeroHash,
        ethers.ZeroHash,
        "gpt-4",
        "tenant"
      );
    });

    it("Should allow submitter to revoke", async function () {
      await registry.revokeAttestation(certId);
      expect(await registry.attestationExists(certId)).to.equal(false);
    });

    it("Should emit AttestationRevoked event", async function () {
      await expect(registry.revokeAttestation(certId))
        .to.emit(registry, "AttestationRevoked")
        .withArgs(certId, owner.address, await getBlockTimestamp());
    });

    it("Should reject unauthorized revocation", async function () {
      await expect(
        registry.connect(addr1).revokeAttestation(certId)
      ).to.be.revertedWith("AITraceRegistry: not authorized to revoke");
    });
  });

  describe("Authorization", function () {
    it("Should authorize new submitter", async function () {
      await registry.setAuthorizedSubmitter(addr1.address, true);
      expect(await registry.authorizedSubmitters(addr1.address)).to.equal(true);
    });

    it("Should revoke authorization", async function () {
      await registry.setAuthorizedSubmitter(addr1.address, true);
      await registry.setAuthorizedSubmitter(addr1.address, false);
      expect(await registry.authorizedSubmitters(addr1.address)).to.equal(false);
    });

    it("Should emit SubmitterAuthorizationChanged event", async function () {
      await expect(registry.setAuthorizedSubmitter(addr1.address, true))
        .to.emit(registry, "SubmitterAuthorizationChanged")
        .withArgs(addr1.address, true);
    });
  });

  describe("Tenant Management", function () {
    const tenantId = "my-tenant";

    it("Should register tenant", async function () {
      await registry.registerTenant(tenantId);
      const tenant = await registry.getTenant(tenantId);
      expect(tenant.owner).to.equal(owner.address);
      expect(tenant.isActive).to.equal(true);
    });

    it("Should reject duplicate tenant", async function () {
      await registry.registerTenant(tenantId);
      await expect(registry.registerTenant(tenantId)).to.be.revertedWith(
        "AITraceRegistry: tenant exists"
      );
    });
  });
});

async function getBlockTimestamp() {
  const block = await ethers.provider.getBlock("latest");
  return block.timestamp;
}
