const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();

  console.log("Deploying contracts with account:", deployer.address);
  console.log("Account balance:", (await ethers.provider.getBalance(deployer.address)).toString());

  // Deploy AITraceRegistry
  console.log("\n1. Deploying AITraceRegistry...");
  const AITraceRegistry = await ethers.getContractFactory("AITraceRegistry");
  const registry = await AITraceRegistry.deploy();
  await registry.waitForDeployment();
  const registryAddress = await registry.getAddress();
  console.log("AITraceRegistry deployed to:", registryAddress);

  // Deploy AITraceArbitration
  console.log("\n2. Deploying AITraceArbitration...");
  const AITraceArbitration = await ethers.getContractFactory("AITraceArbitration");
  const arbitration = await AITraceArbitration.deploy(registryAddress);
  await arbitration.waitForDeployment();
  const arbitrationAddress = await arbitration.getAddress();
  console.log("AITraceArbitration deployed to:", arbitrationAddress);

  // Verify deployment
  console.log("\n3. Verifying deployment...");

  const registryOwner = await registry.owner();
  console.log("Registry owner:", registryOwner);

  const arbitrationOwner = await arbitration.owner();
  console.log("Arbitration owner:", arbitrationOwner);

  const linkedRegistry = await arbitration.registry();
  console.log("Arbitration linked registry:", linkedRegistry);

  // Output deployment info for configuration
  console.log("\n========================================");
  console.log("DEPLOYMENT COMPLETE");
  console.log("========================================");
  console.log("\nAdd to your .env or config:");
  console.log(`REGISTRY_CONTRACT_ADDRESS=${registryAddress}`);
  console.log(`ARBITRATION_CONTRACT_ADDRESS=${arbitrationAddress}`);
  console.log("========================================");

  return {
    registry: registryAddress,
    arbitration: arbitrationAddress,
  };
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
