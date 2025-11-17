# Postman Testing Guide for Metabridge Hub

Complete guide to test the Metabridge API using Postman.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Import Collection](#import-collection)
3. [API Endpoints](#api-endpoints)
4. [Example Requests & Responses](#example-requests--responses)
5. [Testing Workflows](#testing-workflows)
6. [Common Errors](#common-errors)

## Quick Start

### Prerequisites

1. **Postman installed**
   - Download: https://www.postman.com/downloads/
   - Or use web version: https://web.postman.com/

2. **Metabridge running locally**
   ```bash
   ./deploy-testnet.sh
   ```

3. **Verify API is running**
   ```bash
   curl http://localhost:8080/health
   ```

### Import Collection & Environment

**Method 1: Import Files**

1. Open Postman
2. Click "Import" button (top left)
3. Import these files:
   - `postman/Metabridge_API.postman_collection.json`
   - `postman/Metabridge_Local.postman_environment.json`
4. Select "Metabridge Local" environment (top right dropdown)

**Method 2: Import via URL** (after files are committed)

1. Click "Import" → "Link"
2. Paste raw GitHub URLs
3. Click "Continue" → "Import"

## API Endpoints

### Base URL
```
http://localhost:8080
```

### Available Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Health check |
| `/v1/status` | GET | System status |
| `/v1/chains` | GET | List all chains |
| `/v1/chains/:name` | GET | Get chain details |
| `/v1/bridge/token` | POST | Bridge token transfer |
| `/v1/bridge/nft` | POST | Bridge NFT transfer |
| `/v1/bridge/estimate` | POST | Estimate fees |
| `/v1/messages/:id` | GET | Get message details |
| `/v1/messages` | GET | List messages |
| `/v1/messages/:id/signatures` | GET | Get signatures |
| `/v1/stats` | GET | Bridge statistics |

---

## Example Requests & Responses

### 1. Health Check

**Request:**
```http
GET http://localhost:8080/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Status Code:** `200 OK`

---

### 2. List All Chains

**Request:**
```http
GET http://localhost:8080/v1/chains
```

**Expected Response:**
```json
{
  "chains": [
    {
      "name": "polygon-amoy",
      "chain_type": "EVM",
      "chain_id": "80002",
      "enabled": true,
      "bridge_contract": "0x1234567890123456789012345678901234567890",
      "block_time": "2s",
      "confirmation_blocks": 128
    },
    {
      "name": "bnb-testnet",
      "chain_type": "EVM",
      "chain_id": "97",
      "enabled": true,
      "bridge_contract": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
      "block_time": "3s",
      "confirmation_blocks": 15
    },
    {
      "name": "avalanche-fuji",
      "chain_type": "EVM",
      "chain_id": "43113",
      "enabled": true,
      "bridge_contract": "0x9876543210987654321098765432109876543210",
      "block_time": "2s",
      "confirmation_blocks": 10
    },
    {
      "name": "ethereum-sepolia",
      "chain_type": "EVM",
      "chain_id": "11155111",
      "enabled": true,
      "bridge_contract": "0xfedcbafedcbafedcbafedcbafedcbafedcbafed",
      "block_time": "12s",
      "confirmation_blocks": 32
    }
  ],
  "total": 4
}
```

**Status Code:** `200 OK`

---

### 3. Bridge Token Transfer (Main Operation!)

**Request:**
```http
POST http://localhost:8080/v1/bridge/token
Content-Type: application/json

{
  "source_chain": "polygon-amoy",
  "destination_chain": "avalanche-fuji",
  "token_address": "0x0000000000000000000000000000000000001010",
  "amount": "1000000000000000000",
  "sender": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "recipient": "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199"
}
```

**Request Body Fields:**
- `source_chain`: Chain to bridge FROM (polygon-amoy, bnb-testnet, etc.)
- `destination_chain`: Chain to bridge TO
- `token_address`: Token contract address on source chain
- `amount`: Amount in wei (18 decimals)
  - `1000000000000000000` = 1 token
  - `100000000000000000` = 0.1 token
- `sender`: Your wallet address on source chain
- `recipient`: Recipient wallet address on destination chain

**Expected Response (Success):**
```json
{
  "message_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "status": "pending",
  "source_chain": "polygon-amoy",
  "destination_chain": "avalanche-fuji",
  "sender": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "recipient": "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
  "amount": "1000000000000000000",
  "token_address": "0x0000000000000000000000000000000000001010",
  "nonce": 1,
  "created_at": "2024-01-15T10:35:00Z",
  "estimated_completion": "2024-01-15T10:40:00Z"
}
```

**Status Code:** `201 Created`

**Save the `message_id` to track your transfer!**

---

### 4. Get Message Status (Track Your Transfer)

**Request:**
```http
GET http://localhost:8080/v1/messages/0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

**Expected Response (Pending):**
```json
{
  "id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "type": "token_transfer",
  "status": "pending",
  "source_chain": "polygon-amoy",
  "destination_chain": "avalanche-fuji",
  "sender": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "recipient": "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
  "amount": "1000000000000000000",
  "token_address": "0x0000000000000000000000000000000000001010",
  "nonce": 1,
  "validator_signatures": [],
  "source_tx_hash": null,
  "destination_tx_hash": null,
  "created_at": "2024-01-15T10:35:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

**Expected Response (Validated - Getting Signatures):**
```json
{
  "id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "type": "token_transfer",
  "status": "validated",
  "source_chain": "polygon-amoy",
  "destination_chain": "avalanche-fuji",
  "sender": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "recipient": "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
  "amount": "1000000000000000000",
  "token_address": "0x0000000000000000000000000000000000001010",
  "nonce": 1,
  "validator_signatures": [
    {
      "validator_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
      "signature": "0xabcdef1234567890...",
      "timestamp": "2024-01-15T10:36:00Z"
    },
    {
      "validator_address": "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
      "signature": "0x1234567890abcdef...",
      "timestamp": "2024-01-15T10:36:05Z"
    }
  ],
  "source_tx_hash": "0xsource_transaction_hash...",
  "destination_tx_hash": null,
  "created_at": "2024-01-15T10:35:00Z",
  "updated_at": "2024-01-15T10:36:05Z"
}
```

**Expected Response (Completed - Success!):**
```json
{
  "id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "type": "token_transfer",
  "status": "completed",
  "source_chain": "polygon-amoy",
  "destination_chain": "avalanche-fuji",
  "sender": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "recipient": "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
  "amount": "1000000000000000000",
  "token_address": "0x0000000000000000000000000000000000001010",
  "nonce": 1,
  "validator_signatures": [
    {
      "validator_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
      "signature": "0xabcdef1234567890...",
      "timestamp": "2024-01-15T10:36:00Z"
    },
    {
      "validator_address": "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
      "signature": "0x1234567890abcdef...",
      "timestamp": "2024-01-15T10:36:05Z"
    }
  ],
  "source_tx_hash": "0xsource_tx_hash_on_polygon...",
  "destination_tx_hash": "0xdestination_tx_hash_on_avalanche...",
  "created_at": "2024-01-15T10:35:00Z",
  "updated_at": "2024-01-15T10:38:00Z",
  "completed_at": "2024-01-15T10:38:00Z"
}
```

**Status Code:** `200 OK`

**Message Statuses:**
- `pending` - Message created, waiting for processing
- `validated` - Validators have signed the message
- `processing` - Relayer is broadcasting to destination
- `completed` - Transfer successful! ✅
- `failed` - Transfer failed (check error field)

---

### 5. List All Messages

**Request:**
```http
GET http://localhost:8080/v1/messages?limit=10&offset=0
```

**Query Parameters:**
- `status` (optional): Filter by status (pending, completed, failed)
- `limit` (optional): Number of results (default: 10)
- `offset` (optional): Offset for pagination (default: 0)

**Example with Filter:**
```http
GET http://localhost:8080/v1/messages?status=completed&limit=5
```

**Expected Response:**
```json
{
  "messages": [
    {
      "id": "0xmessage1...",
      "status": "completed",
      "source_chain": "polygon-amoy",
      "destination_chain": "avalanche-fuji",
      "amount": "1000000000000000000",
      "created_at": "2024-01-15T10:35:00Z"
    },
    {
      "id": "0xmessage2...",
      "status": "completed",
      "source_chain": "bnb-testnet",
      "destination_chain": "ethereum-sepolia",
      "amount": "500000000000000000",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 2,
  "limit": 10,
  "offset": 0
}
```

**Status Code:** `200 OK`

---

### 6. Get Bridge Statistics

**Request:**
```http
GET http://localhost:8080/v1/stats
```

**Expected Response:**
```json
{
  "total_messages": 150,
  "completed_messages": 142,
  "pending_messages": 5,
  "failed_messages": 3,
  "total_volume": "1500000000000000000000",
  "chains": [
    {
      "name": "polygon-amoy",
      "messages_sent": 45,
      "messages_received": 38,
      "volume": "450000000000000000000"
    },
    {
      "name": "bnb-testnet",
      "messages_sent": 30,
      "messages_received": 35,
      "volume": "350000000000000000000"
    },
    {
      "name": "avalanche-fuji",
      "messages_sent": 40,
      "messages_received": 42,
      "volume": "500000000000000000000"
    }
  ],
  "uptime": "72h30m",
  "last_updated": "2024-01-15T10:40:00Z"
}
```

**Status Code:** `200 OK`

---

## Testing Workflows

### Workflow 1: Complete Transfer Test

1. **Check Health**
   ```
   GET /health
   ```

2. **List Available Chains**
   ```
   GET /v1/chains
   ```

3. **Initiate Transfer**
   ```
   POST /v1/bridge/token
   Body: {source, destination, token, amount, sender, recipient}
   ```
   → Save the `message_id` from response

4. **Monitor Transfer** (repeat every 10-30 seconds)
   ```
   GET /v1/messages/{message_id}
   ```
   → Check `status` field

5. **Verify Completion**
   - Status should be `completed`
   - `destination_tx_hash` should be present
   - Check transaction on destination chain explorer

### Workflow 2: List Recent Transfers

1. **Get All Completed Transfers**
   ```
   GET /v1/messages?status=completed&limit=10
   ```

2. **Get Specific Transfer Details**
   ```
   GET /v1/messages/{message_id}
   ```

3. **Check Signatures**
   ```
   GET /v1/messages/{message_id}/signatures
   ```

---

## Common Errors

### 1. Bridge Not Running

**Error Response:**
```
Could not send request
```

**Solution:**
```bash
# Start the bridge
./deploy-testnet.sh

# Verify it's running
curl http://localhost:8080/health
```

---

### 2. Invalid Chain Name

**Request:**
```json
{
  "source_chain": "invalid-chain",
  "destination_chain": "avalanche-fuji",
  ...
}
```

**Error Response:**
```json
{
  "error": "invalid_chain",
  "message": "Source chain 'invalid-chain' not supported",
  "supported_chains": [
    "polygon-amoy",
    "bnb-testnet",
    "avalanche-fuji",
    "ethereum-sepolia"
  ]
}
```

**Status Code:** `400 Bad Request`

**Solution:** Use correct chain names from `/v1/chains`

---

### 3. Invalid Amount

**Request:**
```json
{
  "amount": "0",
  ...
}
```

**Error Response:**
```json
{
  "error": "invalid_amount",
  "message": "Amount must be greater than zero"
}
```

**Status Code:** `400 Bad Request`

**Solution:** Use amount > 0 in wei (18 decimals)

---

### 4. Message Not Found

**Request:**
```
GET /v1/messages/0xinvalidmessageid
```

**Error Response:**
```json
{
  "error": "not_found",
  "message": "Message with ID '0xinvalidmessageid' not found"
}
```

**Status Code:** `404 Not Found`

**Solution:** Use correct message ID from transfer response

---

### 5. Same Source and Destination

**Request:**
```json
{
  "source_chain": "polygon-amoy",
  "destination_chain": "polygon-amoy",
  ...
}
```

**Error Response:**
```json
{
  "error": "invalid_route",
  "message": "Source and destination chains must be different"
}
```

**Status Code:** `400 Bad Request`

---

## Postman Tips

### 1. Use Environment Variables

Instead of hardcoding values, use variables:

```json
{
  "source_chain": "{{polygon_chain}}",
  "destination_chain": "{{avalanche_chain}}",
  "amount": "{{test_amount}}",
  "sender": "{{test_sender}}",
  "recipient": "{{test_recipient}}"
}
```

### 2. Save Message ID Automatically

Add this to your test script:

```javascript
// In "Tests" tab of "Bridge Token Transfer" request
if (pm.response.code === 201) {
    const response = pm.response.json();
    pm.environment.set("last_message_id", response.message_id);
    console.log("Message ID saved:", response.message_id);
}
```

Then use it:
```
GET /v1/messages/{{last_message_id}}
```

### 3. Poll for Completion

Add to "Tests" tab:

```javascript
const response = pm.response.json();

if (response.status === "completed") {
    console.log("✅ Transfer completed!");
    console.log("Destination TX:", response.destination_tx_hash);
} else if (response.status === "failed") {
    console.log("❌ Transfer failed!");
} else {
    console.log("⏳ Status:", response.status);
    // Wait and retry
    setTimeout(() => {}, 5000);
}
```

### 4. Test Multiple Routes

Create requests for common routes:
- Polygon → Avalanche
- BNB → Ethereum
- Avalanche → Polygon
- Ethereum → BNB

---

## Quick Reference Card

### Amount Conversion (18 decimals)

| Human Amount | Wei Amount (for API) |
|--------------|---------------------|
| 1 token | `1000000000000000000` |
| 0.1 token | `100000000000000000` |
| 0.01 token | `10000000000000000` |
| 0.001 token | `1000000000000000` |

### Chain Names

- Polygon Testnet: `polygon-amoy`
- BNB Testnet: `bnb-testnet`
- Avalanche Testnet: `avalanche-fuji`
- Ethereum Testnet: `ethereum-sepolia`

### Message Status Flow

```
pending → validated → processing → completed ✅
                                 → failed ❌
```

---

## Support

- **Logs**: Check `logs/api.log` for detailed errors
- **Status**: `GET /v1/status` for system health
- **Metrics**: http://localhost:9090 (Prometheus)
- **Dashboards**: http://localhost:3000 (Grafana)

---

**Happy Testing! 🚀**
