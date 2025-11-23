# NEAR Bridge Contract

Production-ready NEAR smart contract for the Articium cross-chain bridge protocol.

## Features

- **Fungible Token Locking/Unlocking**: Lock NEP-141 tokens for cross-chain transfers
- **NEAR Token Support**: Native NEAR token locking and unlocking
- **Multi-Signature Validation**: Configurable validator set with required signature threshold
- **Replay Protection**: Message ID tracking to prevent double-spending
- **Admin Controls**: Pause/unpause, validator management, configuration updates
- **Event Emission**: Comprehensive event logging for indexers

## Architecture

### Core Functions

1. **Lock Operations**:
   - `lock_ft`: Lock fungible tokens (NEP-141)
   - `lock_near`: Lock native NEAR tokens

2. **Unlock Operations**:
   - `unlock_ft`: Unlock fungible tokens with validator signatures
   - `unlock_near`: Unlock native NEAR with validator signatures

3. **Admin Operations**:
   - `add_validator`: Add new validator public key
   - `remove_validator`: Remove validator
   - `update_required_signatures`: Change signature threshold
   - `pause`/`unpause`: Emergency pause controls
   - `transfer_ownership`: Transfer contract ownership

### State Structure

```rust
pub struct BridgeContract {
    pub owner: AccountId,
    pub validators: UnorderedSet<PublicKey>,
    pub required_signatures: u8,
    pub is_paused: bool,
    pub total_locked: UnorderedMap<AccountId, Balance>,
    pub total_unlocked: UnorderedMap<AccountId, Balance>,
    pub processed_messages: UnorderedSet<MessageId>,
    pub lock_records: UnorderedMap<String, LockRecord>,
    pub message_count: u64,
}
```

## Building

### Prerequisites

- Rust 1.70+
- `wasm32-unknown-unknown` target
- NEAR CLI (optional, for deployment)

### Build Commands

```bash
# Build the contract
./build.sh

# Or manually:
cargo build --target wasm32-unknown-unknown --release

# Run tests
cargo test
```

## Deployment

### Testnet Deployment

```bash
# Set your testnet account
NEAR_ACCOUNT="your-account.testnet"

# Deploy contract
near deploy --accountId $NEAR_ACCOUNT \
    --wasmFile ./res/near_bridge.wasm

# Initialize contract
near call $NEAR_ACCOUNT new \
    '{
        "owner": "'$NEAR_ACCOUNT'",
        "validators": [
            "ed25519:2xyzabc...",
            "ed25519:3xyzdef...",
            "ed25519:4xyzghi..."
        ],
        "required_signatures": 2
    }' \
    --accountId $NEAR_ACCOUNT
```

### Mainnet Deployment

```bash
# Use a dedicated mainnet account
NEAR_ACCOUNT="bridge.your-project.near"

# Deploy to mainnet
near deploy --accountId $NEAR_ACCOUNT \
    --wasmFile ./res/near_bridge.wasm \
    --networkId mainnet

# Initialize with mainnet validators
near call $NEAR_ACCOUNT new \
    '{
        "owner": "'$NEAR_ACCOUNT'",
        "validators": [
            "ed25519:validator1...",
            "ed25519:validator2...",
            "ed25519:validator3...",
            "ed25519:validator4...",
            "ed25519:validator5..."
        ],
        "required_signatures": 3
    }' \
    --accountId $NEAR_ACCOUNT \
    --networkId mainnet
```

## Usage Examples

### Lock NEAR Tokens

```bash
near call bridge.testnet lock_near \
    '{
        "destination_chain": "ethereum",
        "destination_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"
    }' \
    --accountId user.testnet \
    --deposit 10
```

### Lock Fungible Tokens

```bash
# First, register with the token contract
near call token.testnet storage_deposit \
    '{"account_id": "bridge.testnet"}' \
    --accountId user.testnet \
    --deposit 0.00125

# Lock tokens
near call token.testnet ft_transfer_call \
    '{
        "receiver_id": "bridge.testnet",
        "amount": "1000000000",
        "msg": "{\"destination_chain\":\"ethereum\",\"destination_address\":\"0x742...\"}"
    }' \
    --accountId user.testnet \
    --depositYocto 1 \
    --gas 100000000000000
```

### Unlock Tokens

```bash
near call bridge.testnet unlock_ft \
    '{
        "message_id": [1,2,3,...,32],
        "source_chain": "ethereum",
        "sender_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
        "recipient": "user.testnet",
        "token_contract": "token.testnet",
        "amount": "1000000000",
        "signatures": [
            {
                "public_key": "ed25519:2xyz...",
                "signature": [...]
            },
            {
                "public_key": "ed25519:3xyz...",
                "signature": [...]
            }
        ]
    }' \
    --accountId relayer.testnet \
    --gas 200000000000000
```

### View Functions

```bash
# Get bridge configuration
near view bridge.testnet get_config

# Check if message processed
near view bridge.testnet is_message_processed \
    '{"message_id": [1,2,3,...,32]}'

# Get lock record
near view bridge.testnet get_lock_record \
    '{"message_id": "abc123..."}'

# Get total locked for a token
near view bridge.testnet get_total_locked \
    '{"token_contract": "token.testnet"}'
```

### Admin Operations

```bash
# Add validator
near call bridge.testnet add_validator \
    '{"validator": "ed25519:newvalidator..."}' \
    --accountId owner.testnet

# Update required signatures
near call bridge.testnet update_required_signatures \
    '{"required_signatures": 3}' \
    --accountId owner.testnet

# Pause bridge
near call bridge.testnet pause '{}' \
    --accountId owner.testnet

# Unpause bridge
near call bridge.testnet unpause '{}' \
    --accountId owner.testnet
```

## Security Considerations

1. **Validator Management**: Only add trusted validator public keys
2. **Signature Threshold**: Set appropriate threshold based on environment:
   - Testnet: 2-of-3 minimum
   - Mainnet: 3-of-5 recommended
3. **Emergency Pause**: Owner can pause contract in case of issues
4. **Message Replay**: Contract prevents replay attacks via message ID tracking
5. **Access Control**: Admin functions restricted to contract owner

## Testing

```bash
# Run unit tests
cargo test

# Run integration tests with workspaces
cargo test --features near-workspaces
```

## Events

The contract emits standardized events:

### TokenLocked Event
```json
{
  "standard": "articium",
  "version": "1.0.0",
  "event": "token_locked",
  "data": {
    "message_id": "...",
    "sender": "user.testnet",
    "token_contract": "token.testnet",
    "amount": "1000000000",
    "destination_chain": "ethereum",
    "destination_address": "0x742...",
    "nonce": 123,
    "timestamp": 1234567890
  }
}
```

### TokenUnlocked Event
```json
{
  "standard": "articium",
  "version": "1.0.0",
  "event": "token_unlocked",
  "data": {
    "message_id": "...",
    "source_chain": "ethereum",
    "sender_address": "0x742...",
    "recipient": "user.testnet",
    "token_contract": "token.testnet",
    "amount": "1000000000",
    "timestamp": 1234567890
  }
}
```

## Gas Costs

Approximate gas costs:
- `lock_near`: ~5 TGas
- `lock_ft`: ~10 TGas
- `unlock_ft`: ~20 TGas (depends on signature count)
- `unlock_near`: ~15 TGas (depends on signature count)

## License

MIT
