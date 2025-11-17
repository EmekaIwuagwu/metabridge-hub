package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// Chain handlers

func (s *Server) handleListChains(w http.ResponseWriter, r *http.Request) {
	chains := make([]map[string]interface{}, 0)

	for name, client := range s.clients {
		info := client.GetChainInfo()
		chains = append(chains, map[string]interface{}{
			"name":        info.Name,
			"type":        info.Type,
			"chain_id":    info.ChainID,
			"network_id":  info.NetworkID,
			"environment": info.Environment,
			"healthy":     client.IsHealthy(r.Context()),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chains": chains,
		"total":  len(chains),
	})
}

func (s *Server) handleChainStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainName := vars["chain"]

	client, exists := s.clients[chainName]
	if !exists {
		respondError(w, http.StatusNotFound, "chain not found", nil)
		return
	}

	info := client.GetChainInfo()
	blockNumber, err := client.GetLatestBlockNumber(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get block number", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"name":         info.Name,
		"type":         info.Type,
		"chain_id":     info.ChainID,
		"network_id":   info.NetworkID,
		"environment":  info.Environment,
		"healthy":      client.IsHealthy(r.Context()),
		"block_number": blockNumber,
		"block_time":   client.GetBlockTime().String(),
		"confirmations": client.GetConfirmationBlocks(),
	})
}

func (s *Server) handleAllChainsStatus(w http.ResponseWriter, r *http.Request) {
	status := make(map[string]interface{})

	for name, client := range s.clients {
		info := client.GetChainInfo()
		blockNumber, _ := client.GetLatestBlockNumber(r.Context())

		status[name] = map[string]interface{}{
			"healthy":      client.IsHealthy(r.Context()),
			"block_number": blockNumber,
			"chain_type":   info.Type,
		}
	}

	respondJSON(w, http.StatusOK, status)
}

// Bridge handlers

type BridgeTokenRequest struct {
	SourceChain      string `json:"source_chain"`
	DestinationChain string `json:"dest_chain"`
	TokenAddress     string `json:"token_address"`
	Amount           string `json:"amount"`
	Recipient        string `json:"recipient"`
	Sender           string `json:"sender,omitempty"`
}

func (s *Server) handleBridgeToken(w http.ResponseWriter, r *http.Request) {
	var req BridgeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.SourceChain == "" || req.DestinationChain == "" {
		respondError(w, http.StatusBadRequest, "source_chain and dest_chain are required", nil)
		return
	}

	if req.TokenAddress == "" || req.Amount == "" || req.Recipient == "" {
		respondError(w, http.StatusBadRequest, "token_address, amount, and recipient are required", nil)
		return
	}

	// Check if chains exist
	if _, exists := s.clients[req.SourceChain]; !exists {
		respondError(w, http.StatusBadRequest, "invalid source chain", nil)
		return
	}
	if _, exists := s.clients[req.DestinationChain]; !exists {
		respondError(w, http.StatusBadRequest, "invalid destination chain", nil)
		return
	}

	// TODO: Create and queue bridge message
	// This would be implemented by the relayer service

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "pending",
		"message": "Bridge request received and will be processed",
		"request": req,
	})
}

type BridgeNFTRequest struct {
	SourceChain      string `json:"source_chain"`
	DestinationChain string `json:"dest_chain"`
	NFTContract      string `json:"nft_contract"`
	TokenID          string `json:"token_id"`
	Recipient        string `json:"recipient"`
	Sender           string `json:"sender,omitempty"`
}

func (s *Server) handleBridgeNFT(w http.ResponseWriter, r *http.Request) {
	var req BridgeNFTRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.SourceChain == "" || req.DestinationChain == "" {
		respondError(w, http.StatusBadRequest, "source_chain and dest_chain are required", nil)
		return
	}

	if req.NFTContract == "" || req.TokenID == "" || req.Recipient == "" {
		respondError(w, http.StatusBadRequest, "nft_contract, token_id, and recipient are required", nil)
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "pending",
		"message": "NFT bridge request received and will be processed",
		"request": req,
	})
}

// Message handlers

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := 50
	offset := 0

	// TODO: Query messages from database

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": []interface{}{},
		"total":    0,
		"limit":    limit,
		"offset":   offset,
	})
}

func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	// TODO: Query message from database

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":     messageID,
		"status": "not_found",
	})
}

func (s *Server) handleMessageStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	// TODO: Query message status from database

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message_id": messageID,
		"status":     "unknown",
	})
}

// Statistics handlers

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Get bridge statistics from database

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total_messages":      0,
		"pending_messages":    0,
		"completed_messages":  0,
		"failed_messages":     0,
		"total_volume_usd":    "0",
		"supported_chains":    len(s.clients),
	})
}

func (s *Server) handleChainStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainName := vars["chain"]

	if _, exists := s.clients[chainName]; !exists {
		respondError(w, http.StatusNotFound, "chain not found", nil)
		return
	}

	// TODO: Get chain-specific statistics

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chain":               chainName,
		"total_messages":      0,
		"completed_messages":  0,
		"failed_messages":     0,
	})
}

// Transaction handlers

func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txHash := vars["hash"]

	// TODO: Query transaction from database

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tx_hash": txHash,
		"status":  "not_found",
	})
}
