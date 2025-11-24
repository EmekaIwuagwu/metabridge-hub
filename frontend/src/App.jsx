import { useState } from 'react'
import { Toaster } from 'react-hot-toast'
import Header from './components/Header'
import BridgeForm from './components/BridgeForm'
import TransactionHistory from './components/TransactionHistory'
import { WalletProvider } from './context/WalletContext'

function App() {
  return (
    <WalletProvider>
      <div className="min-h-screen">
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 5000,
            style: {
              background: 'rgba(17, 24, 39, 0.95)',
              color: '#fff',
              border: '1px rgba(255, 255, 255, 0.1)',
              backdropFilter: 'blur(10px)',
            },
            success: {
              iconTheme: {
                primary: '#10b981',
                secondary: '#fff',
              },
            },
            error: {
              iconTheme: {
                primary: '#ef4444',
                secondary: '#fff',
              },
            },
          }}
        />

        <Header />

        <main className="container mx-auto px-4 py-8 max-w-7xl">
          <div className="text-center mb-12">
            <h1 className="text-5xl font-bold mb-4 bg-gradient-to-r from-blue-400 via-cyan-400 to-blue-500 bg-clip-text text-transparent">
              Articium Bridge
            </h1>
            <p className="text-xl text-gray-300">
              Seamlessly transfer tokens across multiple blockchain testnets
            </p>
          </div>

          <div className="grid lg:grid-cols-3 gap-8">
            <div className="lg:col-span-2">
              <BridgeForm />
            </div>

            <div className="lg:col-span-1">
              <TransactionHistory />
            </div>
          </div>
        </main>

        <footer className="text-center py-8 text-gray-400">
          <p>Powered by Articium • Cross-Chain Bridge Protocol</p>
        </footer>
      </div>
    </WalletProvider>
  )
}

export default App
