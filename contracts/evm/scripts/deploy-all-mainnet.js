const hre = require("hardhat");

async function main() {
  console.log("========================================");
  console.log("  Deploy Bridge to ALL Mainnets");
  console.log("========================================");
  console.log("");

  console.log("⚠️  ⚠️  ⚠️  WARNING ⚠️  ⚠️  ⚠️");
  console.log("You are about to deploy to MAINNET networks!");
  console.log("This will use REAL money and deploy LIVE contracts!");
  console.log("");
  console.log("Prerequisites:");
  console.log("- Contracts have been audited");
  console.log("- You have sufficient funds on all mainnet chains");
  console.log("- You have verified all validator addresses");
  console.log("- You have tested thoroughly on testnets");
  console.log("");
  console.log("Press Ctrl+C to cancel. Waiting 30 seconds...");
  await new Promise(resolve => setTimeout(resolve, 30000));
  console.log("");

  const networks = [
    { name: "polygon-mainnet", chainId: 137 },
    { name: "bnb-mainnet", chainId: 56 },
    { name: "avalanche-mainnet", chainId: 43114 },
    { name: "ethereum-mainnet", chainId: 1 },
  ];

  const deploymentResults = [];

  for (const network of networks) {
    console.log(`\n${"=".repeat(60)}`);
    console.log(`Deploying to ${network.name} (Chain ID: ${network.chainId})...`);
    console.log("=".repeat(60));

    try {
      // Run deployment for this network
      await hre.run("run", {
        script: "scripts/deploy.js",
        network: network.name,
      });

      deploymentResults.push({
        network: network.name,
        chainId: network.chainId,
        status: "✅ Success",
      });
    } catch (error) {
      console.error(`❌ Failed to deploy to ${network.name}:`, error.message);
      deploymentResults.push({
        network: network.name,
        chainId: network.chainId,
        status: "❌ Failed",
        error: error.message,
      });
    }

    // Wait between deployments
    console.log("\nWaiting 10 seconds before next deployment...");
    await new Promise((resolve) => setTimeout(resolve, 10000));
  }

  // Print summary
  console.log("\n" + "=".repeat(60));
  console.log("  MAINNET DEPLOYMENT SUMMARY");
  console.log("=".repeat(60));
  console.log("");

  deploymentResults.forEach((result) => {
    console.log(`${result.network} (${result.chainId}): ${result.status}`);
    if (result.error) {
      console.log(`  Error: ${result.error}`);
    }
  });

  console.log("");
  console.log("✅ All mainnet deployments completed!");
  console.log("");
  console.log("📋 CRITICAL Next steps:");
  console.log("1. Verify ALL contracts on block explorers");
  console.log("2. Update config/config.mainnet.yaml with all contract addresses");
  console.log("3. Transfer contract ownership to multisig wallets");
  console.log("4. Set up monitoring and alerting");
  console.log("5. Implement emergency pause mechanisms");
  console.log("6. Start with low transaction limits");
  console.log("7. Gradual rollout with increased limits");
  console.log("");
  console.log("⚠️  DO NOT process large transactions until fully tested!");
  console.log("");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
