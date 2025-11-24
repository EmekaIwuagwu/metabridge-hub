import { createContext, useContext, useState, useEffect } from 'react';
import { BrowserProvider } from 'ethers';
import toast from 'react-hot-toast';
import { getChainByChainId } from '../config/chains';

const WalletContext = createContext();

export const useWallet = () => {
  const context = useContext(WalletContext);
  if (!context) {
    throw new Error('useWallet must be used within a WalletProvider');
  }
  return context;
};

export const WalletProvider = ({ children }) => {
  const [account, setAccount] = useState(null);
  const [provider, setProvider] = useState(null);
  const [chainId, setChainId] = useState(null);
  const [isConnecting, setIsConnecting] = useState(false);

  useEffect(() => {
    if (window.ethereum) {
      // Check if already connected
      window.ethereum.request({ method: 'eth_accounts' })
        .then(accounts => {
          if (accounts.length > 0) {
            setAccount(accounts[0]);
            const provider = new BrowserProvider(window.ethereum);
            setProvider(provider);

            window.ethereum.request({ method: 'eth_chainId' })
              .then(chainId => setChainId(parseInt(chainId, 16)));
          }
        });

      // Listen for account changes
      window.ethereum.on('accountsChanged', (accounts) => {
        if (accounts.length > 0) {
          setAccount(accounts[0]);
          toast.success('Account changed');
        } else {
          setAccount(null);
          setProvider(null);
          toast.info('Wallet disconnected');
        }
      });

      // Listen for chain changes
      window.ethereum.on('chainChanged', (chainId) => {
        setChainId(parseInt(chainId, 16));
        window.location.reload(); // Recommended by MetaMask
      });
    }

    return () => {
      if (window.ethereum) {
        window.ethereum.removeAllListeners('accountsChanged');
        window.ethereum.removeAllListeners('chainChanged');
      }
    };
  }, []);

  const connectWallet = async () => {
    if (!window.ethereum) {
      toast.error('MetaMask is not installed. Please install it to use this app.');
      return;
    }

    setIsConnecting(true);
    try {
      const accounts = await window.ethereum.request({
        method: 'eth_requestAccounts'
      });

      const provider = new BrowserProvider(window.ethereum);
      setProvider(provider);
      setAccount(accounts[0]);

      const network = await provider.getNetwork();
      setChainId(Number(network.chainId));

      toast.success('Wallet connected successfully!');
    } catch (error) {
      console.error('Failed to connect wallet:', error);
      toast.error(error.message || 'Failed to connect wallet');
    } finally {
      setIsConnecting(false);
    }
  };

  const disconnectWallet = () => {
    setAccount(null);
    setProvider(null);
    setChainId(null);
    toast.info('Wallet disconnected');
  };

  const switchNetwork = async (targetChainId) => {
    if (!window.ethereum) {
      toast.error('MetaMask is not installed');
      return false;
    }

    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: `0x${targetChainId.toString(16)}` }],
      });
      return true;
    } catch (error) {
      // This error code indicates that the chain has not been added to MetaMask
      if (error.code === 4902) {
        const chain = getChainByChainId(targetChainId);
        if (!chain) {
          toast.error('Chain configuration not found');
          return false;
        }

        try {
          await window.ethereum.request({
            method: 'wallet_addEthereumChain',
            params: [{
              chainId: `0x${targetChainId.toString(16)}`,
              chainName: chain.name,
              nativeCurrency: {
                name: chain.symbol,
                symbol: chain.symbol,
                decimals: 18,
              },
              rpcUrls: [chain.rpcUrl],
              blockExplorerUrls: [chain.explorerUrl],
            }],
          });
          return true;
        } catch (addError) {
          console.error('Failed to add network:', addError);
          toast.error('Failed to add network to MetaMask');
          return false;
        }
      }
      console.error('Failed to switch network:', error);
      toast.error('Failed to switch network');
      return false;
    }
  };

  const value = {
    account,
    provider,
    chainId,
    isConnecting,
    connectWallet,
    disconnectWallet,
    switchNetwork,
  };

  return <WalletContext.Provider value={value}>{children}</WalletContext.Provider>;
};
