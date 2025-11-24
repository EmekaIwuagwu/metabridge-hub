export const TESTNET_CHAINS = [
  {
    id: 'polygon-amoy',
    name: 'Polygon Amoy',
    symbol: 'MATIC',
    chainId: 80002,
    rpcUrl: 'https://rpc-amoy.polygon.technology',
    explorerUrl: 'https://amoy.polygonscan.com',
    icon: '🔷',
    color: 'from-purple-500 to-violet-600'
  },
  {
    id: 'bnb-testnet',
    name: 'BNB Testnet',
    symbol: 'tBNB',
    chainId: 97,
    rpcUrl: 'https://data-seed-prebsc-1-s1.binance.org:8545',
    explorerUrl: 'https://testnet.bscscan.com',
    icon: '🟡',
    color: 'from-yellow-500 to-orange-600'
  },
  {
    id: 'avalanche-fuji',
    name: 'Avalanche Fuji',
    symbol: 'AVAX',
    chainId: 43113,
    rpcUrl: 'https://api.avax-test.network/ext/bc/C/rpc',
    explorerUrl: 'https://testnet.snowtrace.io',
    icon: '🔺',
    color: 'from-red-500 to-rose-600'
  },
  {
    id: 'ethereum-sepolia',
    name: 'Ethereum Sepolia',
    symbol: 'SepoliaETH',
    chainId: 11155111,
    rpcUrl: 'https://ethereum-sepolia-rpc.publicnode.com',
    explorerUrl: 'https://sepolia.etherscan.io',
    icon: '⟠',
    color: 'from-blue-500 to-indigo-600'
  },
  {
    id: 'arbitrum-sepolia',
    name: 'Arbitrum Sepolia',
    symbol: 'ETH',
    chainId: 421614,
    rpcUrl: 'https://sepolia-rollup.arbitrum.io/rpc',
    explorerUrl: 'https://sepolia.arbiscan.io',
    icon: '🔵',
    color: 'from-cyan-500 to-blue-600'
  },
  {
    id: 'optimism-sepolia',
    name: 'Optimism Sepolia',
    symbol: 'ETH',
    chainId: 11155420,
    rpcUrl: 'https://sepolia.optimism.io',
    explorerUrl: 'https://sepolia-optimism.etherscan.io',
    icon: '🔴',
    color: 'from-red-500 to-pink-600'
  },
  {
    id: 'fantom-testnet',
    name: 'Fantom Testnet',
    symbol: 'FTM',
    chainId: 4002,
    rpcUrl: 'https://rpc.testnet.fantom.network',
    explorerUrl: 'https://testnet.ftmscan.com',
    icon: '👻',
    color: 'from-blue-400 to-cyan-500'
  }
];

export const getChainById = (chainId) => {
  return TESTNET_CHAINS.find(chain => chain.id === chainId);
};

export const getChainByChainId = (chainId) => {
  return TESTNET_CHAINS.find(chain => chain.chainId === chainId);
};
