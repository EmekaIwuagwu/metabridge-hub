use near_sdk::serde::{Deserialize, Serialize};
use near_sdk::{log, AccountId, Balance};

/// Event emitted when tokens are locked
#[derive(Serialize, Deserialize, Debug)]
#[serde(crate = "near_sdk::serde")]
pub struct TokenLockedEvent {
    pub message_id: String,
    pub sender: AccountId,
    pub token_contract: AccountId,
    pub amount: Balance,
    pub destination_chain: String,
    pub destination_address: String,
    pub nonce: u64,
    pub timestamp: u64,
}

/// Event emitted when tokens are unlocked
#[derive(Serialize, Deserialize, Debug)]
#[serde(crate = "near_sdk::serde")]
pub struct TokenUnlockedEvent {
    pub message_id: String,
    pub source_chain: String,
    pub sender_address: String,
    pub recipient: AccountId,
    pub token_contract: AccountId,
    pub amount: Balance,
    pub timestamp: u64,
}

/// Emit a token locked event
pub fn emit_token_locked_event(event: &TokenLockedEvent) {
    let event_json = near_sdk::serde_json::to_string(event)
        .unwrap_or_else(|_| "{}".to_string());

    log!(
        "EVENT_JSON:{{\"standard\":\"articium\",\"version\":\"1.0.0\",\"event\":\"token_locked\",\"data\":{}}}",
        event_json
    );
}

/// Emit a token unlocked event
pub fn emit_token_unlocked_event(event: &TokenUnlockedEvent) {
    let event_json = near_sdk::serde_json::to_string(event)
        .unwrap_or_else(|_| "{}".to_string());

    log!(
        "EVENT_JSON:{{\"standard\":\"articium\",\"version\":\"1.0.0\",\"event\":\"token_unlocked\",\"data\":{}}}",
        event_json
    );
}
