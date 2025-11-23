#!/usr/bin/env node

/**
 * Cross-Chain Transfer Example
 *
 * This script demonstrates how to perform a cross-chain token transfer
 * from one chain to another using the Articium Engine.
 *
 * Example: Transfer tokens from Polygon to Avalanche
 */

const axios = require('axios');

// Configuration
const API_URL = process.env.BRIDGE_API_URL || 'http://localhost:8080';

// Example: Polygon Amoy -> Avalanche Fuji
const config = {
  sourceChain: 'polygon-amoy',
  destinationChain: 'avalanche-fuji',
  tokenAddress: '0x0000000000000000000000000000000000001010', // Example: WMATIC
  amount: '1000000000000000000', // 1 token (18 decimals)
  sender: '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0',
  recipient: '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199',
};

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  blue: '\x1b[34m',
  yellow: '\x1b[33m',
  red: '\x1b[31m',
};

function log(message, color = colors.reset) {
  console.log(`${color}${message}${colors.reset}`);
}

async function checkHealth() {
  try {
    log('\n🏥 Checking bridge health...', colors.blue);
    const response = await axios.get(`${API_URL}/health`);
    log(`✅ Bridge is ${response.data.status}`, colors.green);
    return true;
  } catch (error) {
    log(`❌ Bridge health check failed: ${error.message}`, colors.red);
    return false;
  }
}

async function getChains() {
  try {
    log('\n🔗 Fetching supported chains...', colors.blue);
    const response = await axios.get(`${API_URL}/v1/chains`);

    log(`\n📋 Supported chains:`, colors.green);
    response.data.chains.forEach(chain => {
      log(`  - ${chain.name} (${chain.chain_type}) ${chain.enabled ? '✅' : '❌'}`, colors.yellow);
    });

    return response.data.chains;
  } catch (error) {
    log(`❌ Failed to fetch chains: ${error.message}`, colors.red);
    return [];
  }
}

async function initiateBridgeTransfer() {
  try {
    log('\n🌉 Initiating cross-chain transfer...', colors.blue);
    log(`\nTransfer Details:`, colors.yellow);
    log(`  From: ${config.sourceChain}`);
    log(`  To: ${config.destinationChain}`);
    log(`  Token: ${config.tokenAddress}`);
    log(`  Amount: ${config.amount}`);
    log(`  Sender: ${config.sender}`);
    log(`  Recipient: ${config.recipient}`);

    const payload = {
      source_chain: config.sourceChain,
      destination_chain: config.destinationChain,
      token_address: config.tokenAddress,
      amount: config.amount,
      sender: config.sender,
      recipient: config.recipient,
    };

    const response = await axios.post(`${API_URL}/v1/bridge/token`, payload, {
      headers: {
        'Content-Type': 'application/json',
      },
    });

    log(`\n✅ Transfer initiated successfully!`, colors.green);
    log(`\nMessage Details:`, colors.yellow);
    log(`  Message ID: ${response.data.message_id}`);
    log(`  Status: ${response.data.status}`);
    log(`  Nonce: ${response.data.nonce}`);
    log(`  Created: ${response.data.created_at}`);

    return response.data.message_id;
  } catch (error) {
    log(`❌ Transfer failed: ${error.message}`, colors.red);
    if (error.response) {
      log(`   Error: ${JSON.stringify(error.response.data, null, 2)}`, colors.red);
    }
    return null;
  }
}

async function checkMessageStatus(messageId) {
  try {
    log(`\n🔍 Checking message status...`, colors.blue);
    const response = await axios.get(`${API_URL}/v1/messages/${messageId}`);

    log(`\nMessage Status:`, colors.yellow);
    log(`  ID: ${response.data.id}`);
    log(`  Status: ${response.data.status}`);
    log(`  Source: ${response.data.source_chain}`);
    log(`  Destination: ${response.data.destination_chain}`);
    log(`  Signatures: ${response.data.validator_signatures ? response.data.validator_signatures.length : 0}`);

    if (response.data.destination_tx_hash) {
      log(`  Destination TX: ${response.data.destination_tx_hash}`, colors.green);
    }

    return response.data;
  } catch (error) {
    log(`❌ Failed to check status: ${error.message}`, colors.red);
    return null;
  }
}

async function monitorTransfer(messageId, maxAttempts = 60) {
  log(`\n⏳ Monitoring transfer progress...`, colors.blue);
  log(`   (This may take several minutes)`);

  for (let i = 0; i < maxAttempts; i++) {
    await new Promise(resolve => setTimeout(resolve, 5000)); // Wait 5 seconds

    const status = await checkMessageStatus(messageId);

    if (!status) continue;

    if (status.status === 'completed') {
      log(`\n🎉 Transfer completed successfully!`, colors.green);
      log(`   Destination TX: ${status.destination_tx_hash}`);
      return true;
    } else if (status.status === 'failed') {
      log(`\n❌ Transfer failed!`, colors.red);
      return false;
    } else {
      log(`   Status: ${status.status} (${i + 1}/${maxAttempts})`, colors.yellow);
    }
  }

  log(`\n⚠️  Monitoring timeout. Check status manually.`, colors.yellow);
  return false;
}

async function getTokenInfo(chain, tokenAddress) {
  try {
    const response = await axios.get(`${API_URL}/v1/tokens/${chain}/${tokenAddress}`);
    return response.data;
  } catch (error) {
    return null;
  }
}

async function estimateFees() {
  try {
    log('\n💰 Estimating fees...', colors.blue);

    const response = await axios.post(`${API_URL}/v1/bridge/estimate`, {
      source_chain: config.sourceChain,
      destination_chain: config.destinationChain,
      amount: config.amount,
    });

    log(`\nFee Estimate:`, colors.yellow);
    log(`  Source Gas: ${response.data.source_gas_fee}`);
    log(`  Destination Gas: ${response.data.destination_gas_fee}`);
    log(`  Bridge Fee: ${response.data.bridge_fee}`);
    log(`  Total: ${response.data.total_fee}`);

    return response.data;
  } catch (error) {
    log(`⚠️  Fee estimation not available`, colors.yellow);
    return null;
  }
}

// Main execution
async function main() {
  console.log('');
  console.log('========================================');
  console.log('  Articium Cross-Chain Transfer');
  console.log('========================================');

  // Step 1: Check health
  const healthy = await checkHealth();
  if (!healthy) {
    log('\n❌ Bridge is not healthy. Exiting.', colors.red);
    process.exit(1);
  }

  // Step 2: Get supported chains
  const chains = await getChains();
  if (chains.length === 0) {
    log('\n❌ No chains available. Exiting.', colors.red);
    process.exit(1);
  }

  // Step 3: Estimate fees (optional)
  await estimateFees();

  // Step 4: Initiate transfer
  const messageId = await initiateBridgeTransfer();
  if (!messageId) {
    log('\n❌ Failed to initiate transfer. Exiting.', colors.red);
    process.exit(1);
  }

  // Step 5: Monitor transfer
  const success = await monitorTransfer(messageId);

  console.log('');
  console.log('========================================');
  if (success) {
    log('  ✅ Transfer Complete!', colors.green);
  } else {
    log('  ⚠️  Transfer Status Unknown', colors.yellow);
    log(`  Check manually: ${API_URL}/v1/messages/${messageId}`);
  }
  console.log('========================================');
  console.log('');
}

// Run if called directly
if (require.main === module) {
  main().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  checkHealth,
  getChains,
  initiateBridgeTransfer,
  checkMessageStatus,
  monitorTransfer,
};
