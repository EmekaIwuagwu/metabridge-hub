const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

async function main() {
  console.log("========================================");
  console.log("  Articium Bridge Contract Deployment");
  console.log("========================================");
  console.log("");

  const [deployer] = await hre.ethers.getSigners();
  const network = hre.network.name;
  const chainId = hre.network.config.chainId;

  console.log("Network:", network);
  console.log("Chain ID:", chainId);
  console.log("Deployer address:", deployer.address);

  // Check deployer balance
  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("Deployer balance:", hre.ethers.formatEther(balance), "ETH");
  console.log("");

  if (balance === 0n) {
    console.error("❌ Error: Deployer has no funds. Please fund the account first.");
    console.log("");
    console.log("Get testnet tokens from faucets:");
    console.log("- Polygon Amoy: https://faucet.polygon.technology/");
    console.log("- BNB Testnet: https://testnet.bnbchain.org/faucet-smart");
    console.log("- Avalanche Fuji: https://core.app/tools/testnet-faucet/");
    console.log("- Ethereum Sepolia: https://sepoliafaucet.com/");
    process.exit(1);
  }

  // Default validator addresses for testnet (2-of-3 multisig)
  // IMPORTANT: Replace these with your actual validator addresses!
  const validatorAddresses = [
    "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
    "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
    "0xdD2FD4581271e230360230F9337D5c0430Bf44C0",
  ];

  const requiredSignatures = 2; // 2-of-3 for testnet

  // For mainnet, use 3-of-5
  if (network.includes("mainnet")) {
    console.log("⚠️  WARNING: Deploying to MAINNET!");
    console.log("⚠️  Make sure you have reviewed the contract and validators!");
    console.log("");
    console.log("Waiting 10 seconds... Press Ctrl+C to cancel.");
    await new Promise(resolve => setTimeout(resolve, 10000));
  }

  console.log("Deployment parameters:");
  console.log("- Validators:", validatorAddresses);
  console.log("- Required signatures:", requiredSignatures);
  console.log("");

  // Transaction limits
  const maxTransactionAmount = hre.ethers.parseEther("10000"); // 10,000 tokens for testnet
  const dailyLimit = hre.ethers.parseEther("100000"); // 100,000 tokens for testnet

  // Deploy PolygonBridge contract (upgradeable proxy pattern)
  console.log("📦 Deploying PolygonBridge contract...");

  const PolygonBridge = await hre.ethers.getContractFactory("PolygonBridge");
  const bridge = await PolygonBridge.deploy();

  await bridge.waitForDeployment();
  const bridgeAddress = await bridge.getAddress();

  console.log("✅ PolygonBridge deployed to:", bridgeAddress);
  console.log("");

  // Initialize the bridge
  console.log("🔧 Initializing bridge contract...");
  const initTx = await bridge.initialize(
    requiredSignatures,
    validatorAddresses,
    maxTransactionAmount,
    dailyLimit
  );
  await initTx.wait();

  console.log("✅ Bridge initialized successfully");
  console.log("");

  // Wait for a few block confirmations
  console.log("⏳ Waiting for block confirmations...");
  const deploymentTx = bridge.deploymentTransaction();
  if (deploymentTx) {
    await deploymentTx.wait(5); // Wait for 5 confirmations
    console.log("✅ Transaction confirmed!");
  }
  console.log("");

  // Save deployment info
  const deploymentInfo = {
    network: network,
    chainId: chainId,
    contractAddress: bridgeAddress,
    deployer: deployer.address,
    validators: validatorAddresses,
    requiredSignatures: requiredSignatures,
    deploymentTx: deploymentTx ? deploymentTx.hash : null,
    timestamp: new Date().toISOString(),
    blockNumber: deploymentTx ? deploymentTx.blockNumber : null,
  };

  const deploymentsDir = path.join(__dirname, "../deployments");
  if (!fs.existsSync(deploymentsDir)) {
    fs.mkdirSync(deploymentsDir, { recursive: true });
  }

  const filename = `${network}_${chainId}.json`;
  const filepath = path.join(deploymentsDir, filename);

  fs.writeFileSync(filepath, JSON.stringify(deploymentInfo, null, 2));
  console.log("💾 Deployment info saved to:", filepath);
  console.log("");

  // Verify contract on block explorer
  if (network !== "hardhat" && network !== "localhost") {
    console.log("📝 Contract verification info:");
    console.log("To verify on block explorer, run:");
    console.log("");
    console.log(`npx hardhat verify --network ${network} ${bridgeAddress}`);
    console.log("");
    console.log("Note: PolygonBridge is deployed with no constructor arguments.");
    console.log("Initialization is done via the initialize() function separately.");
    console.log("");

    // Try automatic verification
    try {
      console.log("🔍 Attempting automatic verification...");
      await hre.run("verify:verify", {
        address: bridgeAddress,
        constructorArguments: [], // No constructor arguments for upgradeable pattern
      });
      console.log("✅ Contract verified successfully!");
    } catch (error) {
      console.log("⚠️  Automatic verification failed. You can verify manually later.");
      console.log("Error:", error.message);
    }
  }

  console.log("");
  console.log("========================================");
  console.log("  Deployment Summary");
  console.log("========================================");
  console.log("");
  console.log("Network:          ", network);
  console.log("Chain ID:         ", chainId);
  console.log("Contract Address: ", bridgeAddress);
  console.log("Deployer:         ", deployer.address);
  console.log("");
  console.log("✅ Deployment completed successfully!");
  console.log("");
  console.log("📋 Next steps:");
  console.log("1. Update config/config.testnet.yaml with the contract address");
  console.log("2. Verify the contract on the block explorer");
  console.log("3. Fund the contract with tokens for testing");
  console.log("4. Test lock/unlock operations");
  console.log("");
  console.log("Environment variable to set:");

  const envVarName = `${network.toUpperCase().replace(/-/g, '_')}_BRIDGE_CONTRACT`;
  console.log(`export ${envVarName}="${bridgeAddress}"`);
  console.log("");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
