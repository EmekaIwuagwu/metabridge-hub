import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000,
});

// Add request interceptor for auth (if needed)
api.interceptors.request.use(
  (config) => {
    // Add any auth tokens here if needed
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Add response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error) => {
    const message = error.response?.data?.message || error.message || 'An error occurred';
    console.error('API Error:', message);
    throw new Error(message);
  }
);

// Bridge API endpoints
export const initiateBridge = async (bridgeData) => {
  try {
    const response = await api.post('/bridge/initiate', bridgeData);
    return response;
  } catch (error) {
    throw error;
  }
};

export const checkBridgeStatus = async (bridgeId) => {
  try {
    const response = await api.get(`/bridge/status/${bridgeId}`);
    return response;
  } catch (error) {
    throw error;
  }
};

export const getUserTransactions = async (address) => {
  try {
    const response = await api.get(`/bridge/transactions/${address}`);
    return response.transactions || [];
  } catch (error) {
    console.error('Failed to fetch transactions:', error);
    return [];
  }
};

export const getSupportedChains = async () => {
  try {
    const response = await api.get('/chains');
    return response.chains || [];
  } catch (error) {
    throw error;
  }
};

export const getChainStatus = async (chainId) => {
  try {
    const response = await api.get(`/chains/${chainId}/status`);
    return response;
  } catch (error) {
    throw error;
  }
};

export const estimateBridgeFee = async (fromChain, toChain, amount) => {
  try {
    const response = await api.post('/bridge/estimate-fee', {
      from_chain: fromChain,
      to_chain: toChain,
      amount: amount,
    });
    return response;
  } catch (error) {
    throw error;
  }
};

export default api;
