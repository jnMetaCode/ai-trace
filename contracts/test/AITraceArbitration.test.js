const { expect } = require("chai");
const { ethers } = require("hardhat");
const { time } = require("@nomicfoundation/hardhat-network-helpers");

describe("AITraceArbitration", function () {
  let registry;
  let arbitration;
  let owner;
  let plaintiff;
  let defendant;
  let arbitrator1;
  let arbitrator2;
  let arbitrator3;

  const certId = ethers.keccak256(ethers.toUtf8Bytes("dispute-cert"));
  const merkleRoot = ethers.keccak256(ethers.toUtf8Bytes("merkle-root"));

  beforeEach(async function () {
    [owner, plaintiff, defendant, arbitrator1, arbitrator2, arbitrator3] =
      await ethers.getSigners();

    // Deploy Registry
    const AITraceRegistry = await ethers.getContractFactory("AITraceRegistry");
    registry = await AITraceRegistry.deploy();
    await registry.waitForDeployment();

    // Create an attestation for testing
    await registry.createAttestation(
      certId,
      merkleRoot,
      ethers.ZeroHash,
      ethers.ZeroHash,
      ethers.ZeroHash,
      "gpt-4",
      "tenant"
    );

    // Deploy Arbitration
    const AITraceArbitration = await ethers.getContractFactory("AITraceArbitration");
    arbitration = await AITraceArbitration.deploy(await registry.getAddress());
    await arbitration.waitForDeployment();

    // Register additional arbitrators
    await arbitration.registerArbitrator(arbitrator1.address);
    await arbitration.registerArbitrator(arbitrator2.address);
    await arbitration.registerArbitrator(arbitrator3.address);
  });

  describe("Deployment", function () {
    it("Should set the right owner", async function () {
      expect(await arbitration.owner()).to.equal(owner.address);
    });

    it("Should link to registry", async function () {
      expect(await arbitration.registry()).to.equal(await registry.getAddress());
    });

    it("Should register owner as arbitrator", async function () {
      const arb = await arbitration.getArbitrator(owner.address);
      expect(arb.isActive).to.equal(true);
    });

    it("Should have correct arbitrator count", async function () {
      expect(await arbitration.getArbitratorCount()).to.equal(4); // owner + 3
    });
  });

  describe("Dispute Creation", function () {
    const evidenceHash = ethers.keccak256(ethers.toUtf8Bytes("evidence"));
    const minStake = ethers.parseEther("0.01");

    it("Should create dispute with sufficient stake", async function () {
      await arbitration.connect(plaintiff).createDispute(
        certId,
        0, // ContentOwnership
        defendant.address,
        "This is my content",
        evidenceHash,
        { value: minStake }
      );

      const dispute = await arbitration.getDispute(1);
      expect(dispute.plaintiff).to.equal(plaintiff.address);
      expect(dispute.defendant).to.equal(defendant.address);
      expect(dispute.certId).to.equal(certId);
      expect(dispute.status).to.equal(0); // Pending
    });

    it("Should emit DisputeCreated event", async function () {
      await expect(
        arbitration.connect(plaintiff).createDispute(
          certId,
          0,
          defendant.address,
          "Dispute description",
          evidenceHash,
          { value: minStake }
        )
      )
        .to.emit(arbitration, "DisputeCreated")
        .withArgs(1, certId, 0, plaintiff.address, defendant.address, minStake);
    });

    it("Should reject insufficient stake", async function () {
      await expect(
        arbitration.connect(plaintiff).createDispute(
          certId,
          0,
          defendant.address,
          "Dispute",
          evidenceHash,
          { value: ethers.parseEther("0.001") }
        )
      ).to.be.revertedWith("AITraceArbitration: insufficient stake");
    });

    it("Should reject self-dispute", async function () {
      await expect(
        arbitration.connect(plaintiff).createDispute(
          certId,
          0,
          plaintiff.address,
          "Self dispute",
          evidenceHash,
          { value: minStake }
        )
      ).to.be.revertedWith("AITraceArbitration: cannot dispute self");
    });

    it("Should reject non-existent attestation", async function () {
      const fakeCertId = ethers.keccak256(ethers.toUtf8Bytes("fake"));
      await expect(
        arbitration.connect(plaintiff).createDispute(
          fakeCertId,
          0,
          defendant.address,
          "Fake dispute",
          evidenceHash,
          { value: minStake }
        )
      ).to.be.revertedWith("AITraceArbitration: attestation not found");
    });
  });

  describe("Voting Process", function () {
    const minStake = ethers.parseEther("0.01");
    const evidenceHash = ethers.keccak256(ethers.toUtf8Bytes("evidence"));

    beforeEach(async function () {
      // Create dispute
      await arbitration.connect(plaintiff).createDispute(
        certId,
        0,
        defendant.address,
        "Ownership dispute",
        evidenceHash,
        { value: minStake }
      );

      // Start voting
      await arbitration.startVoting(1);
    });

    it("Should start voting", async function () {
      const dispute = await arbitration.getDispute(1);
      expect(dispute.status).to.equal(1); // Voting
    });

    it("Should allow arbitrator to vote", async function () {
      await arbitration.connect(arbitrator1).vote(1, 1, "Plaintiff is right");

      const vote = await arbitration.getVote(1, arbitrator1.address);
      expect(vote.result).to.equal(1); // InFavor
    });

    it("Should emit DisputeVoted event", async function () {
      await expect(arbitration.connect(arbitrator1).vote(1, 1, "Reason"))
        .to.emit(arbitration, "DisputeVoted")
        .withArgs(1, arbitrator1.address, 1);
    });

    it("Should prevent double voting", async function () {
      await arbitration.connect(arbitrator1).vote(1, 1, "First vote");
      await expect(
        arbitration.connect(arbitrator1).vote(1, 2, "Second vote")
      ).to.be.revertedWith("AITraceArbitration: already voted");
    });

    it("Should prevent non-arbitrator voting", async function () {
      await expect(
        arbitration.connect(plaintiff).vote(1, 1, "Not allowed")
      ).to.be.revertedWith("AITraceArbitration: caller is not arbitrator");
    });

    it("Should prevent party from voting", async function () {
      // Register plaintiff as arbitrator (edge case test)
      await arbitration.registerArbitrator(plaintiff.address);
      await expect(
        arbitration.connect(plaintiff).vote(1, 1, "Biased")
      ).to.be.revertedWith("AITraceArbitration: arbitrator is party");
    });
  });

  describe("Dispute Resolution", function () {
    const minStake = ethers.parseEther("0.01");
    const evidenceHash = ethers.keccak256(ethers.toUtf8Bytes("evidence"));

    beforeEach(async function () {
      await arbitration.connect(plaintiff).createDispute(
        certId,
        0,
        defendant.address,
        "Resolution test",
        evidenceHash,
        { value: minStake }
      );
      await arbitration.startVoting(1);
    });

    it("Should resolve in favor of plaintiff", async function () {
      // Vote in favor
      await arbitration.connect(arbitrator1).vote(1, 1, "Support plaintiff");
      await arbitration.connect(arbitrator2).vote(1, 1, "Support plaintiff");
      await arbitration.connect(arbitrator3).vote(1, 2, "Support defendant");

      // Fast forward past voting deadline
      await time.increase(7 * 24 * 60 * 60 + 1);

      const plaintiffBalanceBefore = await ethers.provider.getBalance(plaintiff.address);

      await arbitration.resolveDispute(1);

      const dispute = await arbitration.getDispute(1);
      expect(dispute.status).to.equal(2); // Resolved
      expect(dispute.winner).to.equal(plaintiff.address);

      const plaintiffBalanceAfter = await ethers.provider.getBalance(plaintiff.address);
      expect(plaintiffBalanceAfter - plaintiffBalanceBefore).to.equal(minStake);
    });

    it("Should resolve in favor of defendant", async function () {
      await arbitration.connect(arbitrator1).vote(1, 2, "Support defendant");
      await arbitration.connect(arbitrator2).vote(1, 2, "Support defendant");
      await arbitration.connect(arbitrator3).vote(1, 1, "Support plaintiff");

      await time.increase(7 * 24 * 60 * 60 + 1);

      const defendantBalanceBefore = await ethers.provider.getBalance(defendant.address);

      await arbitration.resolveDispute(1);

      const dispute = await arbitration.getDispute(1);
      expect(dispute.status).to.equal(3); // Rejected
      expect(dispute.winner).to.equal(defendant.address);

      const defendantBalanceAfter = await ethers.provider.getBalance(defendant.address);
      expect(defendantBalanceAfter - defendantBalanceBefore).to.equal(minStake);
    });

    it("Should expire with insufficient votes", async function () {
      // Only one vote
      await arbitration.connect(arbitrator1).vote(1, 1, "Only vote");

      await time.increase(7 * 24 * 60 * 60 + 1);

      await arbitration.resolveDispute(1);

      const dispute = await arbitration.getDispute(1);
      expect(dispute.status).to.equal(4); // Expired
    });

    it("Should emit DisputeResolved event", async function () {
      await arbitration.connect(arbitrator1).vote(1, 1, "Vote 1");
      await arbitration.connect(arbitrator2).vote(1, 1, "Vote 2");
      await arbitration.connect(arbitrator3).vote(1, 1, "Vote 3");

      await time.increase(7 * 24 * 60 * 60 + 1);

      await expect(arbitration.resolveDispute(1))
        .to.emit(arbitration, "DisputeResolved")
        .withArgs(1, 2, plaintiff.address);
    });
  });

  describe("Arbitrator Management", function () {
    it("Should register new arbitrator", async function () {
      const [, , , , , , newArb] = await ethers.getSigners();
      await arbitration.registerArbitrator(newArb.address);

      const arb = await arbitration.getArbitrator(newArb.address);
      expect(arb.isActive).to.equal(true);
      expect(arb.reputation).to.equal(100);
    });

    it("Should remove arbitrator", async function () {
      await arbitration.removeArbitrator(arbitrator1.address);

      const arb = await arbitration.getArbitrator(arbitrator1.address);
      expect(arb.isActive).to.equal(false);
    });

    it("Should prevent duplicate registration", async function () {
      await expect(
        arbitration.registerArbitrator(arbitrator1.address)
      ).to.be.revertedWith("AITraceArbitration: already registered");
    });
  });

  describe("Configuration", function () {
    it("Should update min stake", async function () {
      const newStake = ethers.parseEther("0.1");
      await arbitration.setMinStake(newStake);
      expect(await arbitration.minStake()).to.equal(newStake);
    });

    it("Should update voting period", async function () {
      const newPeriod = 14 * 24 * 60 * 60; // 14 days
      await arbitration.setVotingPeriod(newPeriod);
      expect(await arbitration.votingPeriod()).to.equal(newPeriod);
    });

    it("Should update min arbitrators", async function () {
      await arbitration.setMinArbitrators(5);
      expect(await arbitration.minArbitrators()).to.equal(5);
    });
  });
});
