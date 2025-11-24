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
  const [walletType, setWalletType] = useState(null); // 'metamask' or 'phantom'
  const [solanaAccount, setSolanaAccount] = useState(null);

  // Auto-connect MetaMask if previously connected
  useEffect(() => {
    if (window.ethereum) {
      window.ethereum.request({ method: 'eth_accounts' })
        .then(accounts => {
          if (accounts.length > 0) {
            setAccount(accounts[0]);
            const provider = new BrowserProvider(window.ethereum);
            setProvider(provider);
            setWalletType('metamask');

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
          setWalletType(null);
          toast.info('MetaMask disconnected');
        }
      });

      // Listen for chain changes
      window.ethereum.on('chainChanged', (chainId) => {
        setChainId(parseInt(chainId, 16));
        window.location.reload();
      });
    }

    // Auto-connect Phantom if previously connected
    if (window.solana && window.solana.isPhantom) {
      window.solana.connect({ onlyIfTrusted: true })
        .then(({ publicKey }) => {
          setSolanaAccount(publicKey.toString());
          setAccount(publicKey.toString());
          setWalletType('phantom');
        })
        .catch(() => {
          // User hasn't connected before
        });

      // Listen for Phantom account changes
      window.solana.on('accountChanged', (publicKey) => {
        if (publicKey) {
          setSolanaAccount(publicKey.toString());
          setAccount(publicKey.toString());
          toast.success('Phantom account changed');
        } else {
          setSolanaAccount(null);
          setAccount(null);
          setWalletType(null);
          toast.info('Phantom disconnected');
        }
      });
    }

    return () => {
      if (window.ethereum) {
        window.ethereum.removeAllListeners('accountsChanged');
        window.ethereum.removeAllListeners('chainChanged');
      }
      if (window.solana) {
        window.solana.removeAllListeners('accountChanged');
      }
    };
  }, []);

  const connectMetaMask = async () => {
    if (!window.ethereum) {
      toast.error('MetaMask is not installed. Please install it to use EVM chains.');
      window.open('https://metamask.io/download/', '_blank');
      return false;
    }

    setIsConnecting(true);
    try {
      const accounts = await window.ethereum.request({
        method: 'eth_requestAccounts'
      });

      const provider = new BrowserProvider(window.ethereum);
      setProvider(provider);
      setAccount(accounts[0]);
      setWalletType('metamask');

      const network = await provider.getNetwork();
      setChainId(Number(network.chainId));

      toast.success('MetaMask connected successfully!');
      return true;
    } catch (error) {
      console.error('Failed to connect MetaMask:', error);
      toast.error(error.message || 'Failed to connect MetaMask');
      return false;
    } finally {
      setIsConnecting(false);
    }
  };

  const connectPhantom = async () => {
    if (!window.solana || !window.solana.isPhantom) {
      toast.error('Phantom wallet is not installed. Please install it to use Solana.');
      window.open('https://phantom.app/', '_blank');
      return false;
    }

    setIsConnecting(true);
    try {
      const response = await window.solana.connect();
      const publicKey = response.publicKey.toString();

      setSolanaAccount(publicKey);
      setAccount(publicKey);
      setWalletType('phantom');
      setChainId(null); // Solana doesn't use chainId

      toast.success('Phantom wallet connected successfully!');
      return true;
    } catch (error) {
      console.error('Failed to connect Phantom:', error);
      toast.error(error.message || 'Failed to connect Phantom');
      return false;
    } finally {
      setIsConnecting(false);
    }
  };

  const connectWallet = async (type = 'auto') => {
    if (type === 'metamask') {
      return await connectMetaMask();
    } else if (type === 'phantom') {
      return await connectPhantom();
    } else {
      // Auto-detect: prefer MetaMask if available
      if (window.ethereum) {
        return await connectMetaMask();
      } else if (window.solana) {
        return await connectPhantom();
      } else {
        toast.error('No wallet detected. Please install MetaMask or Phantom.');
        return false;
      }
    }
  };

  const disconnectWallet = async () => {
    if (walletType === 'phantom' && window.solana) {
      await window.solana.disconnect();
      setSolanaAccount(null);
    }

    setAccount(null);
    setProvider(null);
    setChainId(null);
    setWalletType(null);
    toast.info('Wallet disconnected');
  };

  const switchNetwork = async (targetChainId) => {
    if (!window.ethereum || walletType !== 'metamask') {
      toast.error('Please connect MetaMask to switch networks');
      return false;
    }

    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: `0x${targetChainId.toString(16)}` }],
      });
      return true;
    } catch (error) {
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
    walletType,
    solanaAccount,
    connectWallet,
    connectMetaMask,
    connectPhantom,
    disconnectWallet,
    switchNetwork,
  };

  return <WalletContext.Provider value={value}>{children}</WalletContext.Provider>;
};
