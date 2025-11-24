import { useState, useEffect } from 'react';
import { Clock, ExternalLink, Loader2, CheckCircle, XCircle } from 'lucide-react';
import { useWallet } from '../context/WalletContext';
import { getUserTransactions } from '../services/api';
import { getChainById } from '../config/chains';

const TransactionHistory = () => {
  const { account } = useWallet();
  const [transactions, setTransactions] = useState([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (account) {
      fetchTransactions();
    }
  }, [account]);

  const fetchTransactions = async () => {
    setIsLoading(true);
    try {
      const txs = await getUserTransactions(account);
      setTransactions(txs);
    } catch (error) {
      console.error('Failed to fetch transactions:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-400" />;
      case 'pending':
      case 'processing':
        return <Loader2 className="w-5 h-5 text-yellow-400 animate-spin" />;
      default:
        return <Clock className="w-5 h-5 text-gray-400" />;
    }
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="glass-card p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">Recent Transactions</h2>
        {account && (
          <button
            onClick={fetchTransactions}
            className="p-2 hover:bg-white/10 rounded-lg transition-colors"
            title="Refresh"
          >
            <Loader2 className="w-5 h-5" />
          </button>
        )}
      </div>

      {!account ? (
        <div className="text-center py-12">
          <Clock className="w-12 h-12 mx-auto mb-4 text-gray-500" />
          <p className="text-gray-400">Connect your wallet to view transactions</p>
        </div>
      ) : isLoading ? (
        <div className="text-center py-12">
          <Loader2 className="w-12 h-12 mx-auto mb-4 text-blue-400 animate-spin" />
          <p className="text-gray-400">Loading transactions...</p>
        </div>
      ) : transactions.length === 0 ? (
        <div className="text-center py-12">
          <Clock className="w-12 h-12 mx-auto mb-4 text-gray-500" />
          <p className="text-gray-400">No transactions yet</p>
          <p className="text-sm text-gray-500 mt-2">Your bridge transfers will appear here</p>
        </div>
      ) : (
        <div className="space-y-3">
          {transactions.map((tx) => {
            const fromChain = getChainById(tx.from_chain);
            const toChain = getChainById(tx.to_chain);

            return (
              <div
                key={tx.id}
                className="p-4 bg-white/5 hover:bg-white/10 border border-white/10 rounded-xl transition-all"
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    {fromChain && <span className="text-xl">{fromChain.icon}</span>}
                    <span className="text-gray-400">→</span>
                    {toChain && <span className="text-xl">{toChain.icon}</span>}
                  </div>
                  {getStatusIcon(tx.status)}
                </div>

                <div className="space-y-1">
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-400">Amount:</span>
                    <span className="font-semibold">{tx.amount} {fromChain?.symbol}</span>
                  </div>

                  <div className="flex justify-between text-sm">
                    <span className="text-gray-400">From:</span>
                    <span className="font-medium">{fromChain?.name}</span>
                  </div>

                  <div className="flex justify-between text-sm">
                    <span className="text-gray-400">To:</span>
                    <span className="font-medium">{toChain?.name}</span>
                  </div>

                  <div className="flex justify-between text-sm">
                    <span className="text-gray-400">Time:</span>
                    <span className="text-gray-500">{formatDate(tx.created_at)}</span>
                  </div>
                </div>

                {tx.tx_hash && fromChain && (
                  <a
                    href={`${fromChain.explorerUrl}/tx/${tx.tx_hash}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center justify-center space-x-2 mt-3 text-xs text-blue-400 hover:text-blue-300"
                  >
                    <span>View Transaction</span>
                    <ExternalLink className="w-3 h-3" />
                  </a>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default TransactionHistory;
