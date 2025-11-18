// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/utils/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/ReentrancyGuardUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/**
 * @title BatchSettler
 * @notice Settles multiple cross-chain messages in a single transaction
 * @dev Uses Merkle trees for efficient batch verification
 */
contract BatchSettler is
    Initializable,
    PausableUpgradeable,
    OwnableUpgradeable,
    ReentrancyGuardUpgradeable
{
    using SafeERC20 for IERC20;

    // ============ State Variables ============

    /// @notice Chain ID of this blockchain
    uint256 public chainId;

    /// @notice Required number of validator signatures
    uint256 public requiredSignatures;

    /// @notice Mapping of validator addresses
    mapping(address => bool) public validators;

    /// @notice Array of validator addresses
    address[] public validatorList;

    /// @notice Mapping of processed batch roots
    mapping(bytes32 => bool) public processedBatches;

    /// @notice Mapping of processed message hashes (within batches)
    mapping(bytes32 => bool) public processedMessages;

    /// @notice Batch nonce for unique identification
    uint256 public batchNonce;

    /// @notice Total batches processed
    uint256 public totalBatchesProcessed;

    /// @notice Total messages processed via batching
    uint256 public totalMessagesProcessed;

    // ============ Structs ============

    struct BatchHeader {
        bytes32 merkleRoot;
        uint256 messageCount;
        uint256 totalValue;
        string sourceChain;
        uint256 timestamp;
    }

    struct MessageData {
        bytes32 messageId;
        address recipient;
        address token;
        uint256 amount;
        bytes32 sourceTxHash;
    }

    // ============ Events ============

    event BatchSubmitted(
        bytes32 indexed batchRoot,
        uint256 indexed batchNonce,
        uint256 messageCount,
        uint256 totalValue,
        string sourceChain
    );

    event BatchSettled(
        bytes32 indexed batchRoot,
        uint256 messagesSettled,
        uint256 gasSaved
    );

    event MessageSettledInBatch(
        bytes32 indexed batchRoot,
        bytes32 indexed messageId,
        address indexed recipient,
        address token,
        uint256 amount
    );

    event ValidatorAdded(address indexed validator);
    event ValidatorRemoved(address indexed validator);

    // ============ Errors ============

    error BatchAlreadyProcessed();
    error MessageAlreadyProcessed();
    error InvalidMerkleProof();
    error InsufficientSignatures();
    error InvalidValidator();
    error ZeroAddress();
    error InvalidAmount();
    error InvalidBatchSize();

    // ============ Initialization ============

    function initialize(
        uint256 _chainId,
        uint256 _requiredSignatures,
        address[] memory _validators
    ) external initializer {
        __Pausable_init();
        __Ownable_init(msg.sender);
        __ReentrancyGuard_init();

        require(_chainId != 0, "Invalid chain ID");
        require(_requiredSignatures > 0, "Invalid signature count");
        require(_requiredSignatures <= _validators.length, "Too many required signatures");

        chainId = _chainId;
        requiredSignatures = _requiredSignatures;

        for (uint256 i = 0; i < _validators.length; i++) {
            require(_validators[i] != address(0), "Invalid validator");
            validators[_validators[i]] = true;
            validatorList.push(_validators[i]);
        }
    }

    // ============ Core Functions ============

    /**
     * @notice Settle multiple messages in a single batch
     * @param header Batch header containing metadata
     * @param messages Array of message data
     * @param proofs Array of Merkle proofs for each message
     * @param signatures Validator signatures for the batch
     */
    function settleBatch(
        BatchHeader calldata header,
        MessageData[] calldata messages,
        bytes32[][] calldata proofs,
        bytes[] calldata signatures
    ) external nonReentrant whenNotPaused {
        // Validate batch hasn't been processed
        if (processedBatches[header.merkleRoot]) revert BatchAlreadyProcessed();

        // Validate batch size
        if (messages.length == 0) revert InvalidBatchSize();
        if (messages.length != proofs.length) revert InvalidBatchSize();

        // Verify validator signatures
        _verifyBatchSignatures(header, signatures);

        // Mark batch as processed
        processedBatches[header.merkleRoot] = true;

        uint256 settledCount = 0;
        uint256 startGas = gasleft();

        // Process each message in the batch
        for (uint256 i = 0; i < messages.length; i++) {
            if (_settleMessage(header.merkleRoot, messages[i], proofs[i])) {
                settledCount++;
            }
        }

        // Calculate gas saved
        uint256 gasUsed = startGas - gasleft();
        uint256 estimatedIndividualGas = messages.length * 100000; // Rough estimate
        uint256 gasSaved = estimatedIndividualGas > gasUsed
            ? estimatedIndividualGas - gasUsed
            : 0;

        // Update global counters
        batchNonce++;
        totalBatchesProcessed++;
        totalMessagesProcessed += settledCount;

        emit BatchSettled(header.merkleRoot, settledCount, gasSaved);
        emit BatchSubmitted(
            header.merkleRoot,
            batchNonce,
            messages.length,
            header.totalValue,
            header.sourceChain
        );
    }

    /**
     * @notice Settle a single message within a batch
     * @param batchRoot Merkle root of the batch
     * @param message Message data
     * @param proof Merkle proof for the message
     * @return success Whether the message was successfully settled
     */
    function _settleMessage(
        bytes32 batchRoot,
        MessageData calldata message,
        bytes32[] calldata proof
    ) internal returns (bool success) {
        // Check if message already processed
        if (processedMessages[message.messageId]) {
            return false; // Skip already processed messages
        }

        // Verify Merkle proof
        bytes32 leaf = _hashMessage(message);
        if (!_verifyMerkleProof(proof, batchRoot, leaf)) {
            revert InvalidMerkleProof();
        }

        // Validate message data
        if (message.recipient == address(0)) revert ZeroAddress();
        if (message.amount == 0) revert InvalidAmount();

        // Mark message as processed
        processedMessages[message.messageId] = true;

        // Transfer tokens
        IERC20(message.token).safeTransfer(message.recipient, message.amount);

        emit MessageSettledInBatch(
            batchRoot,
            message.messageId,
            message.recipient,
            message.token,
            message.amount
        );

        return true;
    }

    // ============ Verification Functions ============

    /**
     * @notice Verify validator signatures for a batch
     */
    function _verifyBatchSignatures(
        BatchHeader calldata header,
        bytes[] calldata signatures
    ) internal view {
        if (signatures.length < requiredSignatures) revert InsufficientSignatures();

        bytes32 headerHash = _hashBatchHeader(header);
        bytes32 ethSignedHash = _getEthSignedMessageHash(headerHash);

        address[] memory signers = new address[](signatures.length);
        uint256 validSignatures = 0;

        for (uint256 i = 0; i < signatures.length; i++) {
            address signer = _recoverSigner(ethSignedHash, signatures[i]);

            if (!validators[signer]) continue;

            // Check for duplicate signers
            bool isDuplicate = false;
            for (uint256 j = 0; j < validSignatures; j++) {
                if (signers[j] == signer) {
                    isDuplicate = true;
                    break;
                }
            }

            if (!isDuplicate) {
                signers[validSignatures] = signer;
                validSignatures++;
            }
        }

        if (validSignatures < requiredSignatures) revert InsufficientSignatures();
    }

    /**
     * @notice Verify Merkle proof for a message
     */
    function _verifyMerkleProof(
        bytes32[] calldata proof,
        bytes32 root,
        bytes32 leaf
    ) internal pure returns (bool) {
        bytes32 computedHash = leaf;

        for (uint256 i = 0; i < proof.length; i++) {
            bytes32 proofElement = proof[i];

            if (computedHash <= proofElement) {
                computedHash = keccak256(abi.encodePacked(computedHash, proofElement));
            } else {
                computedHash = keccak256(abi.encodePacked(proofElement, computedHash));
            }
        }

        return computedHash == root;
    }

    // ============ Hashing Functions ============

    function _hashBatchHeader(BatchHeader calldata header) internal view returns (bytes32) {
        return keccak256(
            abi.encodePacked(
                header.merkleRoot,
                header.messageCount,
                header.totalValue,
                header.sourceChain,
                header.timestamp,
                chainId
            )
        );
    }

    function _hashMessage(MessageData calldata message) internal pure returns (bytes32) {
        return keccak256(
            abi.encodePacked(
                message.messageId,
                message.recipient,
                message.token,
                message.amount,
                message.sourceTxHash
            )
        );
    }

    function _getEthSignedMessageHash(bytes32 messageHash) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash));
    }

    function _recoverSigner(
        bytes32 ethSignedMessageHash,
        bytes memory signature
    ) internal pure returns (address) {
        (bytes32 r, bytes32 s, uint8 v) = _splitSignature(signature);
        address signer = ecrecover(ethSignedMessageHash, v, r, s);

        // SECURITY: ecrecover returns address(0) on error
        // We must reject this to prevent signature malleability attacks
        require(signer != address(0), "Invalid signature");

        return signer;
    }

    function _splitSignature(bytes memory sig)
        internal
        pure
        returns (bytes32 r, bytes32 s, uint8 v)
    {
        require(sig.length == 65, "Invalid signature length");

        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }
    }

    // ============ Admin Functions ============

    function addValidator(address validator) external onlyOwner {
        if (validator == address(0)) revert ZeroAddress();
        if (validators[validator]) return;

        validators[validator] = true;
        validatorList.push(validator);

        emit ValidatorAdded(validator);
    }

    function removeValidator(address validator) external onlyOwner {
        if (!validators[validator]) return;

        validators[validator] = false;

        for (uint256 i = 0; i < validatorList.length; i++) {
            if (validatorList[i] == validator) {
                validatorList[i] = validatorList[validatorList.length - 1];
                validatorList.pop();
                break;
            }
        }

        emit ValidatorRemoved(validator);
    }

    function setRequiredSignatures(uint256 _requiredSignatures) external onlyOwner {
        require(_requiredSignatures > 0, "Invalid signature count");
        require(_requiredSignatures <= validatorList.length, "Too many required");

        requiredSignatures = _requiredSignatures;
    }

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }

    // ============ View Functions ============

    function getValidatorCount() external view returns (uint256) {
        return validatorList.length;
    }

    function isValidator(address account) external view returns (bool) {
        return validators[account];
    }

    function isBatchProcessed(bytes32 batchRoot) external view returns (bool) {
        return processedBatches[batchRoot];
    }

    function isMessageProcessed(bytes32 messageId) external view returns (bool) {
        return processedMessages[messageId];
    }

    function getBatchStats() external view returns (
        uint256 totalBatches,
        uint256 totalMessages,
        uint256 currentNonce
    ) {
        return (totalBatchesProcessed, totalMessagesProcessed, batchNonce);
    }
}
