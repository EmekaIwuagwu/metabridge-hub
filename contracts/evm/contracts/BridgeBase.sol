// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/utils/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/ReentrancyGuardUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/token/ERC721/IERC721.sol";

/**
 * @title BridgeBase
 * @notice Base contract for multi-chain bridge supporting EVM, Solana, and NEAR
 * @dev Implements core bridge functionality with multi-sig validation
 */
abstract contract BridgeBase is
    Initializable,
    PausableUpgradeable,
    OwnableUpgradeable,
    ReentrancyGuardUpgradeable
{
    using SafeERC20 for IERC20;

    // ============ State Variables ============

    /// @notice Chain ID of this blockchain
    uint256 public chainId;

    /// @notice Chain name
    string public chainName;

    /// @notice Required number of validator signatures
    uint256 public requiredSignatures;

    /// @notice Mapping of validator addresses
    mapping(address => bool) public validators;

    /// @notice Array of validator addresses
    address[] public validatorList;

    /// @notice Mapping of processed messages (to prevent replay)
    mapping(bytes32 => bool) public processedMessages;

    /// @notice Mapping of locked tokens
    mapping(address => uint256) public lockedTokens;

    /// @notice Nonce for outgoing messages
    uint256 public outgoingNonce;

    /// @notice Transaction limits
    uint256 public maxTransactionAmount;
    uint256 public dailyLimit;
    uint256 public dailySpent;
    uint256 public lastResetTime;

    // ============ Events ============

    event TokenLocked(
        bytes32 indexed messageId,
        address indexed sender,
        address indexed token,
        uint256 amount,
        string destinationChain,
        string destinationAddress,
        uint256 nonce
    );

    event TokenReleased(
        bytes32 indexed messageId,
        address indexed recipient,
        address indexed token,
        uint256 amount,
        string sourceChain,
        bytes32 sourceTxHash
    );

    event NFTLocked(
        bytes32 indexed messageId,
        address indexed sender,
        address indexed nftContract,
        uint256 tokenId,
        string destinationChain,
        string destinationAddress,
        uint256 nonce
    );

    event NFTReleased(
        bytes32 indexed messageId,
        address indexed recipient,
        address indexed nftContract,
        uint256 tokenId,
        string sourceChain,
        bytes32 sourceTxHash
    );

    event ValidatorAdded(address indexed validator);
    event ValidatorRemoved(address indexed validator);
    event RequiredSignaturesChanged(uint256 newRequired);
    event LimitsUpdated(uint256 maxAmount, uint256 dailyLimit);
    event EmergencyPause();
    event EmergencyUnpause();

    // ============ Errors ============

    error InvalidSignature();
    error InsufficientSignatures();
    error MessageAlreadyProcessed();
    error InvalidValidator();
    error AmountExceedsLimit();
    error DailyLimitExceeded();
    error InvalidChainId();
    error ZeroAddress();
    error InvalidAmount();

    // ============ Modifiers ============

    modifier onlyValidator() {
        if (!validators[msg.sender]) revert InvalidValidator();
        _;
    }

    modifier withinLimits(uint256 amount) {
        if (amount > maxTransactionAmount) revert AmountExceedsLimit();

        // Reset daily limit if needed
        if (block.timestamp >= lastResetTime + 1 days) {
            dailySpent = 0;
            lastResetTime = block.timestamp;
        }

        if (dailySpent + amount > dailyLimit) revert DailyLimitExceeded();
        dailySpent += amount;
        _;
    }

    // ============ Initialization ============

    function __BridgeBase_init(
        uint256 _chainId,
        string memory _chainName,
        uint256 _requiredSignatures,
        address[] memory _validators,
        uint256 _maxTransactionAmount,
        uint256 _dailyLimit
    ) internal onlyInitializing {
        __Pausable_init();
        __Ownable_init(msg.sender);
        __ReentrancyGuard_init();

        if (_chainId == 0) revert InvalidChainId();
        if (_requiredSignatures == 0) revert InsufficientSignatures();
        if (_requiredSignatures > _validators.length) revert InsufficientSignatures();

        chainId = _chainId;
        chainName = _chainName;
        requiredSignatures = _requiredSignatures;
        maxTransactionAmount = _maxTransactionAmount;
        dailyLimit = _dailyLimit;
        lastResetTime = block.timestamp;

        for (uint256 i = 0; i < _validators.length; i++) {
            if (_validators[i] == address(0)) revert ZeroAddress();
            validators[_validators[i]] = true;
            validatorList.push(_validators[i]);
        }
    }

    // ============ External Functions ============

    /**
     * @notice Lock ERC20 tokens to bridge to another chain
     * @param token ERC20 token address
     * @param amount Amount to lock
     * @param destinationChain Destination chain name
     * @param destinationAddress Destination address on target chain
     */
    function lockToken(
        address token,
        uint256 amount,
        string calldata destinationChain,
        string calldata destinationAddress
    )
        external
        nonReentrant
        whenNotPaused
        withinLimits(amount)
        returns (bytes32 messageId)
    {
        if (token == address(0)) revert ZeroAddress();
        if (amount == 0) revert InvalidAmount();

        // Transfer tokens from sender to bridge
        IERC20(token).safeTransferFrom(msg.sender, address(this), amount);
        lockedTokens[token] += amount;

        // Generate message ID
        messageId = keccak256(
            abi.encodePacked(
                chainId,
                outgoingNonce,
                msg.sender,
                token,
                amount,
                destinationChain,
                destinationAddress,
                block.timestamp
            )
        );

        emit TokenLocked(
            messageId,
            msg.sender,
            token,
            amount,
            destinationChain,
            destinationAddress,
            outgoingNonce
        );

        outgoingNonce++;
    }

    /**
     * @notice Lock NFT to bridge to another chain
     * @param nftContract ERC721 contract address
     * @param tokenId Token ID to lock
     * @param destinationChain Destination chain name
     * @param destinationAddress Destination address on target chain
     */
    function lockNFT(
        address nftContract,
        uint256 tokenId,
        string calldata destinationChain,
        string calldata destinationAddress
    )
        external
        nonReentrant
        whenNotPaused
        returns (bytes32 messageId)
    {
        if (nftContract == address(0)) revert ZeroAddress();

        // Transfer NFT from sender to bridge
        IERC721(nftContract).transferFrom(msg.sender, address(this), tokenId);

        // Generate message ID
        messageId = keccak256(
            abi.encodePacked(
                chainId,
                outgoingNonce,
                msg.sender,
                nftContract,
                tokenId,
                destinationChain,
                destinationAddress,
                block.timestamp
            )
        );

        emit NFTLocked(
            messageId,
            msg.sender,
            nftContract,
            tokenId,
            destinationChain,
            destinationAddress,
            outgoingNonce
        );

        outgoingNonce++;
    }

    /**
     * @notice Release tokens on this chain (called by relayers with signatures)
     * @param messageId Unique message identifier
     * @param recipient Recipient address
     * @param token Token address
     * @param amount Amount to release
     * @param sourceChain Source chain name
     * @param sourceTxHash Transaction hash on source chain
     * @param signatures Array of validator signatures
     */
    function releaseToken(
        bytes32 messageId,
        address recipient,
        address token,
        uint256 amount,
        string calldata sourceChain,
        bytes32 sourceTxHash,
        bytes[] calldata signatures
    ) external nonReentrant whenNotPaused {
        if (processedMessages[messageId]) revert MessageAlreadyProcessed();
        if (recipient == address(0)) revert ZeroAddress();
        if (amount == 0) revert InvalidAmount();

        // Verify signatures
        _verifySignatures(
            messageId,
            recipient,
            token,
            amount,
            sourceChain,
            sourceTxHash,
            signatures
        );

        // Mark message as processed
        processedMessages[messageId] = true;

        // Release tokens
        lockedTokens[token] -= amount;
        IERC20(token).safeTransfer(recipient, amount);

        emit TokenReleased(
            messageId,
            recipient,
            token,
            amount,
            sourceChain,
            sourceTxHash
        );
    }

    /**
     * @notice Release NFT on this chain (called by relayers with signatures)
     */
    function releaseNFT(
        bytes32 messageId,
        address recipient,
        address nftContract,
        uint256 tokenId,
        string calldata sourceChain,
        bytes32 sourceTxHash,
        bytes[] calldata signatures
    ) external nonReentrant whenNotPaused {
        if (processedMessages[messageId]) revert MessageAlreadyProcessed();
        if (recipient == address(0)) revert ZeroAddress();

        // Verify signatures
        _verifyNFTSignatures(
            messageId,
            recipient,
            nftContract,
            tokenId,
            sourceChain,
            sourceTxHash,
            signatures
        );

        // Mark message as processed
        processedMessages[messageId] = true;

        // Release NFT
        IERC721(nftContract).transferFrom(address(this), recipient, tokenId);

        emit NFTReleased(
            messageId,
            recipient,
            nftContract,
            tokenId,
            sourceChain,
            sourceTxHash
        );
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

        // Remove from array
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
        if (_requiredSignatures == 0) revert InsufficientSignatures();
        if (_requiredSignatures > validatorList.length) revert InsufficientSignatures();

        requiredSignatures = _requiredSignatures;
        emit RequiredSignaturesChanged(_requiredSignatures);
    }

    function updateLimits(
        uint256 _maxTransactionAmount,
        uint256 _dailyLimit
    ) external onlyOwner {
        maxTransactionAmount = _maxTransactionAmount;
        dailyLimit = _dailyLimit;
        emit LimitsUpdated(_maxTransactionAmount, _dailyLimit);
    }

    function emergencyPause() external onlyOwner {
        _pause();
        emit EmergencyPause();
    }

    function emergencyUnpause() external onlyOwner {
        _unpause();
        emit EmergencyUnpause();
    }

    // ============ Internal Functions ============

    function _verifySignatures(
        bytes32 messageId,
        address recipient,
        address token,
        uint256 amount,
        string calldata sourceChain,
        bytes32 sourceTxHash,
        bytes[] calldata signatures
    ) internal view {
        if (signatures.length < requiredSignatures) revert InsufficientSignatures();

        bytes32 hash = keccak256(
            abi.encodePacked(
                messageId,
                recipient,
                token,
                amount,
                sourceChain,
                sourceTxHash,
                chainId
            )
        );

        bytes32 ethSignedHash = _getEthSignedMessageHash(hash);

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

    function _verifyNFTSignatures(
        bytes32 messageId,
        address recipient,
        address nftContract,
        uint256 tokenId,
        string calldata sourceChain,
        bytes32 sourceTxHash,
        bytes[] calldata signatures
    ) internal view {
        // Similar to _verifySignatures but for NFTs
        if (signatures.length < requiredSignatures) revert InsufficientSignatures();

        bytes32 hash = keccak256(
            abi.encodePacked(
                messageId,
                recipient,
                nftContract,
                tokenId,
                sourceChain,
                sourceTxHash,
                chainId
            )
        );

        bytes32 ethSignedHash = _getEthSignedMessageHash(hash);

        address[] memory signers = new address[](signatures.length);
        uint256 validSignatures = 0;

        for (uint256 i = 0; i < signatures.length; i++) {
            address signer = _recoverSigner(ethSignedHash, signatures[i]);

            if (!validators[signer]) continue;

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
        if (sig.length != 65) revert InvalidSignature();

        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }
    }

    // ============ View Functions ============

    function getValidatorCount() external view returns (uint256) {
        return validatorList.length;
    }

    function isValidator(address account) external view returns (bool) {
        return validators[account];
    }

    function getRemainingDailyLimit() external view returns (uint256) {
        if (block.timestamp >= lastResetTime + 1 days) {
            return dailyLimit;
        }
        return dailyLimit - dailySpent;
    }
}
