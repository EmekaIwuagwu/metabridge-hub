# Testnet Faucets Guide

Complete guide to getting free testnet tokens for all supported chains.

## Quick Reference Table

| Chain | Network | Token | Faucet Link | Amount | Wait Time |
|-------|---------|-------|-------------|--------|-----------|
| Polygon | Amoy | MATIC | [faucet.polygon.technology](https://faucet.polygon.technology/) | 0.5-1 MATIC | 1-2 min |
| BNB | Testnet | tBNB | [testnet.bnbchain.org](https://testnet.bnbchain.org/faucet-smart) | 0.1-0.5 tBNB | 1-2 min |
| Avalanche | Fuji | AVAX | [core.app/faucet](https://core.app/tools/testnet-faucet/) | 1-2 AVAX | 1-2 min |
| Ethereum | Sepolia | SepoliaETH | [sepoliafaucet.com](https://sepoliafaucet.com/) | 0.5-1 ETH | 1-5 min |

## Detailed Instructions

### 1. Polygon Amoy Testnet (MATIC)

#### Primary Faucet: Polygon Official

**URL**: https://faucet.polygon.technology/

**Steps**:
1. Visit the faucet
2. Select "POL (Amoy)" from network dropdown
3. Choose "POL Token" or "MATIC Token"
4. Paste your wallet address
5. Complete CAPTCHA
6. Click "Submit"
7. Wait 1-2 minutes

**Amount**: 0.5-1 MATIC per request
**Frequency**: Once per 24 hours
**Requirements**: None

#### Alternative Faucets:

**Alchemy Faucet**
- URL: https://www.alchemy.com/faucets/polygon-amoy
- Requirements: Alchemy account
- Amount: 0.5 MATIC
- Frequency: Once per day

**QuickNode Faucet**
- URL: https://faucet.quicknode.com/polygon/amoy
- Requirements: Email verification
- Amount: 0.1 MATIC
- Frequency: Once per day

### 2. BNB Smart Chain Testnet (tBNB)

#### Primary Faucet: BNB Chain Official

**URL**: https://testnet.bnbchain.org/faucet-smart

**Steps**:
1. Visit the faucet
2. Paste your BNB address (0x...)
3. Complete CAPTCHA
4. Click "Give me BNB"
5. Wait 1-2 minutes

**Amount**: 0.1-0.5 tBNB per request
**Frequency**: Once per 24 hours
**Requirements**: None

#### Getting Extra tBNB:

If you need more tBNB:

1. **Testnet BNB Faucet (Discord)**
   - Join BNB Chain Discord: https://discord.gg/bnbchain
   - Go to #testnet-faucet channel
   - Type: `!faucet YOUR_ADDRESS`
   - Amount: 1 tBNB

2. **Ankr Faucet**
   - URL: https://www.ankr.com/faucet/
   - Select "BNB Smart Chain Testnet"
   - Amount: 0.05 tBNB

### 3. Avalanche Fuji Testnet (AVAX)

#### Primary Faucet: Core Wallet

**URL**: https://core.app/tools/testnet-faucet/

**Steps**:
1. Visit Core faucet
2. Select "Fuji (C-Chain)" from dropdown
3. Paste your C-Chain address (0x...)
4. Complete CAPTCHA
5. Click "Request 2 AVAX"
6. Wait 1-2 minutes

**Amount**: 2 AVAX per request
**Frequency**: Once per 24 hours
**Requirements**: None

#### Alternative Faucets:

**Avalanche Official Faucet** (Requires Social)
- URL: https://faucet.avax.network/
- Requirements: Twitter or GitHub account
- Steps:
  1. Connect Twitter or GitHub
  2. Select "Fuji (C-Chain)"
  3. Paste address
  4. Complete verification
  5. Request AVAX
- Amount: 10 AVAX
- Frequency: Once per 24 hours

**Chainlink Faucet**
- URL: https://faucets.chain.link/fuji
- Requirements: None
- Amount: 1 AVAX
- Frequency: Once per day

### 4. Ethereum Sepolia Testnet (SepoliaETH)

#### Primary Faucets:

**Alchemy Sepolia Faucet**
- URL: https://www.alchemy.com/faucets/ethereum-sepolia
- Requirements: Alchemy account
- Amount: 0.5 ETH per day
- Steps:
  1. Create Alchemy account (free)
  2. Verify email
  3. Paste address
  4. Click "Send Me ETH"

**Sepolia PoW Faucet**
- URL: https://sepolia-faucet.pk910.de/
- Requirements: None (mining-based)
- Amount: Variable (depends on mining time)
- Steps:
  1. Visit faucet
  2. Paste address
  3. Start mining in browser
  4. Wait for rewards
  5. Claim ETH

**QuickNode Faucet**
- URL: https://faucet.quicknode.com/ethereum/sepolia
- Requirements: Twitter account
- Amount: 0.1 ETH
- Steps:
  1. Connect Twitter
  2. Tweet verification
  3. Paste address
  4. Claim

**Infura Faucet**
- URL: https://www.infura.io/faucet/sepolia
- Requirements: Infura account
- Amount: 0.5 ETH
- Frequency: Once per 24 hours

## Multi-Chain Faucet Aggregators

### Chainlink Faucets
- **URL**: https://faucets.chain.link/
- **Chains**: Sepolia, Fuji, BNB Testnet
- **Benefits**: One interface for multiple chains

### Alchemy Faucets
- **URL**: https://www.alchemy.com/faucets
- **Chains**: Sepolia, Polygon Amoy
- **Benefits**: Higher amounts, reliable

## Tips & Best Practices

### 1. Plan Ahead
- Request tokens **before** you need them
- Faucets have rate limits (usually 24 hours)
- Some have verification delays

### 2. Save Addresses
Keep a list of addresses you've funded:
```
Polygon Amoy: 0x...
BNB Testnet: 0x...
Avalanche Fuji: 0x...
Ethereum Sepolia: 0x...
```

### 3. Check Balances

```bash
# Using cast (Foundry)
cast balance 0xYOUR_ADDRESS --rpc-url https://polygon-amoy.g.alchemy.com/v2/demo

# Using Hardhat
npx hardhat run scripts/check-balance.js --network polygon-amoy
```

### 4. Troubleshooting

**"Address already used"**
- Wait 24 hours
- Try alternative faucets
- Use different wallet address

**"CAPTCHA failed"**
- Try different browser
- Disable ad blockers
- Use incognito mode

**"Faucet dry"**
- Try at different time
- Use alternative faucets
- Join Discord for help

### 5. Conservation

Testnet tokens are free but limited:
- Only request what you need
- Don't spam faucets
- Report dry faucets to community

## Required Amounts for Articium

### Minimum per Chain (for testing)
- **Deployment**: 0.1-0.2 native tokens
- **Testing**: 0.05-0.1 native tokens
- **Total**: ~0.2-0.3 per chain

### Recommended per Chain
- **Deployment + Testing**: 0.5-1 native tokens
- **Buffer**: 0.5 native tokens
- **Total**: ~1-1.5 per chain

### Total for All Testnets
- **Minimum**: ~0.8-1.2 tokens across all chains
- **Recommended**: ~4-6 tokens total
- **Cost**: $0 (all free from faucets!)

## Deployment Costs Estimate

| Chain | Deployment | Testing | Total Needed |
|-------|-----------|---------|--------------|
| Polygon Amoy | ~0.15 MATIC | ~0.05 MATIC | 0.2 MATIC |
| BNB Testnet | ~0.015 tBNB | ~0.01 tBNB | 0.03 tBNB |
| Avalanche Fuji | ~0.15 AVAX | ~0.05 AVAX | 0.2 AVAX |
| Ethereum Sepolia | ~0.08 ETH | ~0.02 ETH | 0.1 ETH |

**All costs covered by free faucets!**

## Emergency: Out of Tokens

If you run out mid-deployment:

1. **Check rate limits**: Wait for 24h reset
2. **Try all alternatives**: Use all faucet options
3. **Ask community**:
   - Polygon Discord: https://discord.gg/polygon
   - BNB Discord: https://discord.gg/bnbchain
   - Avalanche Discord: https://discord.gg/avax
   - Ethereum Discord: https://discord.gg/ethereum

4. **Bridge from other testnet** (if supported)
5. **Ask team members** to share

## Verification

After getting tokens, verify you have enough:

```bash
# Check all balances at once
./scripts/check-all-balances.sh
```

Expected output:
```
✅ Polygon Amoy: 1.5 MATIC (sufficient)
✅ BNB Testnet: 0.5 tBNB (sufficient)
✅ Avalanche Fuji: 2.0 AVAX (sufficient)
✅ Ethereum Sepolia: 0.8 ETH (sufficient)

All chains funded! Ready to deploy.
```

## Social Verification Tips

Some faucets require social verification:

### Twitter Verification
1. Follow required accounts
2. Like/retweet announcement
3. Paste tweet URL
4. Wait for verification (1-5 minutes)

### GitHub Verification
1. Connect GitHub account
2. Must have account > 30 days old
3. Public profile required
4. Some activity preferred

### Discord Verification
1. Join server
2. Complete verification
3. Use faucet bot commands
4. Wait for delivery

## Mainnet Tokens

⚠️ **Mainnet requires REAL money!**

Never deploy to mainnet without:
1. Security audit completed
2. Sufficient real tokens
3. Testing on all testnets
4. Team approval

---

**Pro Tip**: Bookmark all faucet URLs and set calendar reminder for daily claims during testing phase!
