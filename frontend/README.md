# Articium Bridge Frontend

Beautiful React frontend for the Articium cross-chain token bridge. Transfer tokens seamlessly between multiple blockchain testnets.

## Features

- 🌉 **Cross-Chain Transfers** - Bridge tokens between 7+ testnets
- 🎨 **Beautiful UI** - Modern, responsive design with Tailwind CSS
- 🦊 **MetaMask Integration** - Seamless wallet connection
- 📊 **Transaction History** - Track all your bridge transfers
- ⚡ **Real-time Updates** - Live transaction status updates
- 🔒 **Secure** - Non-custodial, you control your private keys

## Supported Testnets

- Polygon Amoy
- BNB Smart Chain Testnet
- Avalanche Fuji
- Ethereum Sepolia
- Arbitrum Sepolia
- Optimism Sepolia
- Fantom Testnet

## Prerequisites

- Node.js 18+ and npm
- MetaMask browser extension
- Articium backend API running (default: http://localhost:8080)

## Installation

```bash
# Install dependencies
cd frontend
npm install

# Create environment file
cp .env.example .env

# Start development server
npm run dev
```

The frontend will be available at `http://localhost:3000`

## Configuration

Edit `.env` file to configure the API endpoint:

```env
VITE_API_URL=http://localhost:8080/api/v1
```

## Building for Production

```bash
# Build optimized production bundle
npm run build

# Preview production build
npm run preview
```

The build output will be in the `dist/` directory.

## Usage

### 1. Connect Wallet

Click "Connect Wallet" button in the header to connect your MetaMask wallet.

### 2. Select Chains

- Choose source chain (where your tokens are)
- Choose destination chain (where you want to send tokens)
- Use the swap button to quickly reverse the chains

### 3. Enter Amount

- Enter the amount of tokens to bridge
- Click "MAX" to use your full balance (minus gas fees)
- Balance is displayed for your convenience

### 4. Bridge Tokens

- Click "Bridge Tokens" to initiate the transfer
- Confirm the transaction in MetaMask
- Wait for confirmation and bridge processing

### 5. Track Transfer

- View transaction status in the bridge form
- See all your transfers in the "Recent Transactions" sidebar
- Click on transactions to view on block explorer

## Project Structure

```
frontend/
├── public/                 # Static assets
├── src/
│   ├── components/        # React components
│   │   ├── Header.jsx     # App header with wallet connection
│   │   ├── BridgeForm.jsx # Main bridge transfer form
│   │   ├── ChainSelector.jsx # Chain selection dropdown
│   │   └── TransactionHistory.jsx # Transaction list
│   ├── config/
│   │   └── chains.js      # Supported chains configuration
│   ├── context/
│   │   └── WalletContext.jsx # Wallet state management
│   ├── services/
│   │   └── api.js         # Backend API integration
│   ├── App.jsx            # Main app component
│   ├── main.jsx           # App entry point
│   └── index.css          # Global styles
├── index.html             # HTML template
├── package.json           # Dependencies
├── tailwind.config.js     # Tailwind CSS configuration
└── vite.config.js         # Vite configuration
```

## API Integration

The frontend communicates with the Articium backend API:

- `POST /api/v1/bridge/initiate` - Initiate bridge transfer
- `GET /api/v1/bridge/status/:id` - Check bridge status
- `GET /api/v1/bridge/transactions/:address` - Get user transactions
- `GET /api/v1/chains` - Get supported chains
- `POST /api/v1/bridge/estimate-fee` - Estimate bridge fees

## Troubleshooting

### MetaMask not detected

- Ensure MetaMask extension is installed and enabled
- Refresh the page after installing MetaMask
- Check browser console for errors

### Wrong network

- The app will prompt you to switch networks
- Confirm the network switch in MetaMask
- If the network is not in MetaMask, it will be added automatically

### Transaction failed

- Check your wallet balance (including gas fees)
- Ensure you're on the correct network
- Check backend API is running and accessible
- View error details in browser console

### Balance not showing

- Ensure wallet is connected
- Try refreshing the page
- Check network connection
- Verify RPC endpoint is accessible

## Development

```bash
# Start dev server with hot reload
npm run dev

# Run linting
npm run lint

# Build for production
npm run build
```

## Technologies

- **React 18** - UI framework
- **Vite** - Build tool and dev server
- **Tailwind CSS** - Utility-first CSS framework
- **ethers.js** - Ethereum library
- **Lucide React** - Icon library
- **React Hot Toast** - Toast notifications
- **Axios** - HTTP client

## License

MIT

## Support

For issues and questions, please open an issue on GitHub or contact the development team.
