package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"mcp-etoolstec-editfiles/auth"
	"mcp-etoolstec-editfiles/logger"
	"mcp-etoolstec-editfiles/mcp"
)

func HandleMCP(w http.ResponseWriter, r *http.Request) {
	if !auth.CheckAuth(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized - precisa de Bearer"})
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if len(body) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "mcp online", "transport": "sse"})
		return
	}

	var req mcp.JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error("Erro ao decodificar JSON-RPC: %v", err)
		http.Error(w, "invalid jsonrpc: "+err.Error(), 400)
		return
	}

	logger.Debug("MCP Request: %s (id: %v)", req.Method, req.ID)

	var result interface{}
	var rpcErr interface{}

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "mcp-meta-editfiles",
				"version": "1.0.0",
			},
		}
		logger.Info("Cliente inicializado")

	case "tools/list":
		result = map[string]interface{}{
			"tools": mcp.GetTools(),
		}
		logger.Debug("Lista de tools solicitada")

	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("Erro ao decodificar params do tool call: %v", err)
			rpcErr = map[string]interface{}{"code": -32602, "message": "invalid params"}
			break
		}

		logger.Info("Tool chamada: %s com args: %+v", params.Name, params.Arguments)
		result, rpcErr = mcp.ExecuteTool(params.Name, params.Arguments)
		if rpcErr == nil {
			logger.Info("Tool executada com sucesso: %s", params.Name)
		} else {
			logger.Error("Erro ao executar tool %s: %v", params.Name, rpcErr)
		}

	default:
		logger.Warn("Método não encontrado: %s", req.Method)
		rpcErr = map[string]interface{}{"code": -32601, "message": "method not found: " + req.Method}
	}

	resp := mcp.JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  result,
		Error:   rpcErr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// Push to SSE session if sessionId provided
	sessID := r.URL.Query().Get("sessionId")
	if sessID != "" {
		mcp.SessionsMu.Lock()
		if sess, ok := mcp.Sessions[sessID]; ok {
			data, _ := json.Marshal(resp)
			select {
			case sess.Channel <- string(data):
				logger.Debug("Resposta enviada para sessão SSE: %s", sessID)
			default:
				logger.Warn("Canal SSE cheio para sessão: %s", sessID)
			}
		}
		mcp.SessionsMu.Unlock()
	}
}
