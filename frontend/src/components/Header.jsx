import { Wallet, LogOut } from 'lucide-react';
import { useWallet } from '../context/WalletContext';
import { getChainByChainId } from '../config/chains';

const Header = () => {
  const { account, chainId, connectWallet, disconnectWallet, isConnecting } = useWallet();

  const currentChain = chainId ? getChainByChainId(chainId) : null;

  const formatAddress = (address) => {
    if (!address) return '';
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  return (
    <header className="border-b border-white/10 backdrop-blur-lg bg-white/5">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="text-3xl">🌉</div>
            <div>
              <h1 className="text-xl font-bold">Articium Bridge</h1>
              <p className="text-xs text-gray-400">Cross-Chain Token Bridge</p>
            </div>
          </div>

          <div className="flex items-center space-x-4">
            {currentChain && account && (
              <div className="hidden md:flex items-center space-x-2 px-4 py-2 rounded-lg bg-white/5 border border-white/10">
                <span className="text-2xl">{currentChain.icon}</span>
                <span className="text-sm font-medium">{currentChain.name}</span>
              </div>
            )}

            {account ? (
              <div className="flex items-center space-x-2">
                <div className="px-4 py-2 rounded-lg bg-gradient-to-r from-green-500/20 to-emerald-500/20 border border-green-500/30">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
                    <span className="font-mono text-sm">{formatAddress(account)}</span>
                  </div>
                </div>
                <button
                  onClick={disconnectWallet}
                  className="p-2 hover:bg-white/10 rounded-lg transition-colors"
                  title="Disconnect Wallet"
                >
                  <LogOut className="w-5 h-5" />
                </button>
              </div>
            ) : (
              <button
                onClick={connectWallet}
                disabled={isConnecting}
                className="btn-primary flex items-center space-x-2"
              >
                <Wallet className="w-5 h-5" />
                <span>{isConnecting ? 'Connecting...' : 'Connect Wallet'}</span>
              </button>
            )}
          </div>
        </div>
      </div>
    </header>
  );
};

export default Header;
