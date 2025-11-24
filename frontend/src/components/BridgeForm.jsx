import { useState, useEffect } from 'react';
import { ArrowDownUp, Send, Loader2 } from 'lucide-react';
import { parseEther, formatEther } from 'ethers';
import toast from 'react-hot-toast';
import { useWallet } from '../context/WalletContext';
import { TESTNET_CHAINS, getChainById } from '../config/chains';
import ChainSelector from './ChainSelector';
import { initiateBridge, checkBridgeStatus } from '../services/api';

const BridgeForm = () => {
  const { account, provider, chainId, switchNetwork } = useWallet();
  const [fromChain, setFromChain] = useState(TESTNET_CHAINS[0]);
  const [toChain, setToChain] = useState(TESTNET_CHAINS[1]);
  const [amount, setAmount] = useState('');
  const [destinationAddress, setDestinationAddress] = useState('');
  const [balance, setBalance] = useState('0');
  const [isLoading, setIsLoading] = useState(false);
  const [txHash, setTxHash] = useState('');
  const [bridgeStatus, setBridgeStatus] = useState(null);

  // Debug: Log loaded chains
  console.log('BridgeForm loaded with chains:', {
    totalChains: TESTNET_CHAINS.length,
    chainList: TESTNET_CHAINS.map(c => ({ id: c.id, name: c.name }))
  });

  useEffect(() => {
    if (account && provider) {
      fetchBalance();
    }
  }, [account, provider, chainId]);

  const fetchBalance = async () => {
    try {
      const balance = await provider.getBalance(account);
      setBalance(formatEther(balance));
    } catch (error) {
      console.error('Failed to fetch balance:', error);
    }
  };

  const handleSwapChains = () => {
    const temp = fromChain;
    setFromChain(toChain);
    setToChain(temp);
  };

  const handleBridge = async () => {
    if (!account) {
      toast.error('Please connect your wallet first');
      return;
    }

    if (!amount || parseFloat(amount) <= 0) {
      toast.error('Please enter a valid amount');
      return;
    }

    if (parseFloat(amount) > parseFloat(balance)) {
      toast.error('Insufficient balance');
      return;
    }

    if (fromChain.id === toChain.id) {
      toast.error('Source and destination chains cannot be the same');
      return;
    }

    // Check if we're on the correct network
    if (chainId !== fromChain.chainId) {
      toast.error(`Please switch to ${fromChain.name} network`);
      const switched = await switchNetwork(fromChain.chainId);
      if (!switched) return;
    }

    setIsLoading(true);
    try {
      // Send transaction on source chain
      toast.loading('Preparing transaction...', { id: 'bridge-tx' });

      const signer = await provider.getSigner();
      const tx = await signer.sendTransaction({
        to: account, // In a real bridge, this would be the bridge contract
        value: parseEther(amount),
      });

      toast.loading('Transaction sent, waiting for confirmation...', { id: 'bridge-tx' });
      const receipt = await tx.wait();

      setTxHash(receipt.hash);
      toast.success('Transaction confirmed!', { id: 'bridge-tx' });

      // Call backend to initiate bridge
      toast.loading('Initiating bridge transfer...', { id: 'bridge-api' });

      const bridgeData = await initiateBridge({
        from_chain: fromChain.id,
        to_chain: toChain.id,
        from_address: account,
        to_address: destinationAddress || account,
        amount: amount,
        tx_hash: receipt.hash,
      });

      toast.success('Bridge initiated successfully!', { id: 'bridge-api' });

      // Start polling for status
      pollBridgeStatus(bridgeData.bridge_id);

      // Reset form
      setAmount('');
      fetchBalance();
    } catch (error) {
      console.error('Bridge failed:', error);
      toast.error(error.message || 'Bridge transfer failed', { id: 'bridge-tx' });
      toast.dismiss('bridge-api');
    } finally {
      setIsLoading(false);
    }
  };

  const pollBridgeStatus = async (bridgeId) => {
    const interval = setInterval(async () => {
      try {
        const status = await checkBridgeStatus(bridgeId);
        setBridgeStatus(status);

        if (status.status === 'completed') {
          clearInterval(interval);
          toast.success('Bridge transfer completed!');
        } else if (status.status === 'failed') {
          clearInterval(interval);
          toast.error('Bridge transfer failed');
        }
      } catch (error) {
        console.error('Failed to check bridge status:', error);
      }
    }, 5000);

    // Stop polling after 10 minutes
    setTimeout(() => clearInterval(interval), 600000);
  };

  const setMaxAmount = () => {
    // Leave some for gas fees
    const maxAmount = Math.max(0, parseFloat(balance) - 0.01);
    setAmount(maxAmount.toString());
  };

  return (
    <div className="glass-card p-6 md:p-8">
      <h2 className="text-2xl font-bold mb-6">Bridge Transfer</h2>

      {/* From Chain */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-300 mb-2">From</label>
        <ChainSelector
          chains={TESTNET_CHAINS}
          selectedChain={fromChain}
          onSelect={setFromChain}
          disabled={[toChain]}
        />
      </div>

      {/* Swap Button */}
      <div className="flex justify-center my-4">
        <button
          onClick={handleSwapChains}
          className="p-3 rounded-full bg-white/10 hover:bg-white/20 transition-all transform hover:rotate-180 duration-300"
        >
          <ArrowDownUp className="w-5 h-5" />
        </button>
      </div>

      {/* To Chain */}
      <div className="mb-6">
        <label className="block text-sm font-medium text-gray-300 mb-2">To</label>
        <ChainSelector
          chains={TESTNET_CHAINS}
          selectedChain={toChain}
          onSelect={setToChain}
          disabled={[fromChain]}
        />
      </div>

      {/* Destination Address */}
      <div className="mb-6">
        <div className="flex justify-between items-center mb-2">
          <label className="block text-sm font-medium text-gray-300">Destination Address (Optional)</label>
          {account && (
            <button
              onClick={() => setDestinationAddress(account)}
              className="text-xs text-blue-400 hover:text-blue-300"
            >
              Use My Address
            </button>
          )}
        </div>
        <input
          type="text"
          value={destinationAddress}
          onChange={(e) => setDestinationAddress(e.target.value)}
          placeholder={account || "Enter destination address or leave empty for your wallet"}
          className="input-field"
        />
        <p className="text-xs text-gray-500 mt-1">
          Leave empty to send to your own address on the destination chain
        </p>
      </div>

      {/* Amount Input */}
      <div className="mb-6">
        <div className="flex justify-between items-center mb-2">
          <label className="block text-sm font-medium text-gray-300">Amount</label>
          {account && (
            <span className="text-sm text-gray-400">
              Balance: {parseFloat(balance).toFixed(4)} {fromChain.symbol}
            </span>
          )}
        </div>
        <div className="relative">
          <input
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="0.0"
            className="input-field pr-20"
            step="0.0001"
            min="0"
          />
          <button
            onClick={setMaxAmount}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-semibold text-blue-400 hover:text-blue-300"
            disabled={!account}
          >
            MAX
          </button>
        </div>
      </div>

      {/* Bridge Button */}
      <button
        onClick={handleBridge}
        disabled={!account || isLoading || !amount}
        className="btn-primary w-full flex items-center justify-center space-x-2"
      >
        {isLoading ? (
          <>
            <Loader2 className="w-5 h-5 animate-spin" />
            <span>Processing...</span>
          </>
        ) : (
          <>
            <Send className="w-5 h-5" />
            <span>Bridge Tokens</span>
          </>
        )}
      </button>

      {/* Transaction Info */}
      {txHash && (
        <div className="mt-6 p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
          <p className="text-sm text-green-400 font-semibold mb-2">Transaction Submitted</p>
          <a
            href={`${fromChain.explorerUrl}/tx/${txHash}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-blue-400 hover:text-blue-300 break-all"
          >
            {txHash}
          </a>
        </div>
      )}

      {/* Bridge Status */}
      {bridgeStatus && (
        <div className="mt-4 p-4 bg-blue-500/10 border border-blue-500/30 rounded-xl">
          <p className="text-sm font-semibold mb-2">Bridge Status</p>
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-300">Status:</span>
            <span className={`text-sm font-semibold ${
              bridgeStatus.status === 'completed' ? 'text-green-400' :
              bridgeStatus.status === 'failed' ? 'text-red-400' :
              'text-yellow-400'
            }`}>
              {bridgeStatus.status.toUpperCase()}
            </span>
          </div>
        </div>
      )}

      {/* Info Box */}
      <div className="mt-6 p-4 bg-blue-500/10 border border-blue-500/20 rounded-xl">
        <p className="text-xs text-gray-400">
          ⚠️ <strong>Testnet Bridge:</strong> This is a testnet bridge for testing purposes only.
          Transfers may take a few minutes to complete.
        </p>
      </div>
    </div>
  );
};

export default BridgeForm;
