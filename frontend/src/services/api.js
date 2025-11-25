import axios from 'axios';

// Auto-detect API URL based on current host
const getApiBaseUrl = () => {
  // If environment variable is set, use it
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }

  // Auto-detect based on current hostname
  const protocol = window.location.protocol; // http: or https:
  const hostname = window.location.hostname; // localhost, IP, or domain

  // If running on localhost, use localhost:8080
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    return 'http://localhost:8080/v1';
  }

  // Otherwise, use same host with port 8080
  return `${protocol}//${hostname}:8080/v1`;
};

const API_BASE_URL = getApiBaseUrl();

console.log('API Base URL:', API_BASE_URL);

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
    // Transform frontend format to backend format
    const backendRequest = {
      source_chain: bridgeData.from_chain,
      dest_chain: bridgeData.to_chain,
      token_address: '0x0000000000000000000000000000000000000000', // Native token
      amount: bridgeData.amount,
      recipient: bridgeData.to_address,
      sender: bridgeData.from_address,
    };

    const response = await api.post('/bridge/token', backendRequest);
    return response;
  } catch (error) {
    throw error;
  }
};

export const checkBridgeStatus = async (bridgeId) => {
  try {
    const response = await api.get(`/messages/${bridgeId}/status`);
    return response;
  } catch (error) {
    throw error;
  }
};

export const getUserTransactions = async (address) => {
  try {
    // Use tracking query endpoint to get user's messages
    const response = await api.get(`/track/query?sender=${address}`);
    return response.messages || [];
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
