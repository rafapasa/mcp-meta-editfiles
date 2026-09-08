package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var allowedRoots = []string{
	"/home/opc/Documents/meta-ai-files",
	"/home/opc/prj/front-opener",
	"/home/opc/prj/front-openerp",
	"/home/opc/prj",
	"/home/opc",
}

var bearerToken string
var rootPath string

type Session struct {
	ID      string
	Channel chan string
}

var sessions = make(map[string]*Session)
var sessionsMu sync.Mutex

// ---- File helpers ----
func isAllowed(p string) bool {
	abs, _ := filepath.Abs(p)
	for _, r := range allowedRoots {
		if strings.HasPrefix(abs, r) {
			return true
		}
	}
	return false
}

// ---- MCP types ----
type JSONRPCRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func checkAuth(r *http.Request) bool {
	if bearerToken == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	return auth == "Bearer "+bearerToken
}

// SSE handler
func handleSSE(w http.ResponseWriter, r *http.Request) {
	if !checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sessionID := uuid.New().String()
	ch := make(chan string, 100)
	sess := &Session{ID: sessionID, Channel: ch}

	sessionsMu.Lock()
	sessions[sessionID] = sess
	sessionsMu.Unlock()

	// Send endpoint event
	endpoint := fmt.Sprintf("/mcp?sessionId=%s", sessionID)
	// Full URL for client
	scheme := "https"
	if r.TLS == nil && r.Host == "localhost:8001" {
		scheme = "http"
	}
	// For Meta AI, we return relative but also absolute
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint)
	// Also send absolute URL for easier debugging
	absURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, endpoint)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", absURL)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", 500)
		return
	}
	flusher.Flush()

	// Keep connection open and stream messages
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			sessionsMu.Lock()
			delete(sessions, sessionID)
			sessionsMu.Unlock()
			return
		case <-time.After(15 * time.Second):
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func handleMCP(w http.ResponseWriter, r *http.Request) {
	if !checkAuth(r) {
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

	// Support both /mcp and /mcp?sessionId=
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

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid jsonrpc: "+err.Error(), 400)
		return
	}

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
	case "tools/list":
		result = map[string]interface{}{
			"tools": []Tool{
				{
					Name:        "list_files",
					Description: "Lista arquivos e pastas de um diretório. Use para explorar front-openerp. Exemplo: {\"path\": \"/home/opc/prj/front-openerp/lib/presentation/pages/tenants\"}",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{"type": "string", "description": "Caminho absoluto no servidor"},
						},
						"required": []string{"path"},
					},
				},
				{
					Name:        "read_file",
					Description: "Lê conteúdo completo de um arquivo .dart, .json, .yaml. Retorna texto. Use para pegar código antes de editar.",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{"type": "string", "description": "Caminho absoluto do arquivo"},
						},
						"required": []string{"path"},
					},
				},
				{
					Name:        "edit_file",
					Description: "Edita/cria arquivo com conteúdo novo. Usado para refatorar tenants, auth, etc. Sobrescreve arquivo.",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":    map[string]interface{}{"type": "string"},
							"content": map[string]interface{}{"type": "string", "description": "Conteúdo completo novo do arquivo"},
						},
						"required": []string{"path", "content"},
					},
				},
			},
		}
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		json.Unmarshal(req.Params, &params)

		switch params.Name {
		case "list_files":
			path, _ := params.Arguments["path"].(string)
			if path == "" {
				path = rootPath
			}
			if !isAllowed(path) {
				rpcErr = map[string]interface{}{"code": -32602, "message": "path not allowed: " + path}
				break
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				rpcErr = map[string]interface{}{"code": -32603, "message": err.Error()}
				break
			}
			var files []map[string]interface{}
			for _, e := range entries {
				info, _ := e.Info()
				sz := int64(0)
				if info != nil {
					sz = info.Size()
				}
				files = append(files, map[string]interface{}{
					"name":  e.Name(),
					"path":  filepath.Join(path, e.Name()),
					"isDir": e.IsDir(),
					"size":  sz,
				})
			}
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("Path: %s | Total: %d", path, len(files)) + "\n" + toJSON(files)},
				},
			}

		case "read_file":
			path, _ := params.Arguments["path"].(string)
			if !isAllowed(path) {
				rpcErr = map[string]interface{}{"code": -32602, "message": "path not allowed"}
				break
			}
			data, err := os.ReadFile(path)
			if err != nil {
				rpcErr = map[string]interface{}{"code": -32603, "message": err.Error()}
				break
			}
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": string(data)},
				},
			}

		case "edit_file":
			path, _ := params.Arguments["path"].(string)
			content, _ := params.Arguments["content"].(string)
			if !isAllowed(path) {
				rpcErr = map[string]interface{}{"code": -32602, "message": "path not allowed"}
				break
			}
			// garante diretório existe
			os.MkdirAll(filepath.Dir(path), 0755)
			err := os.WriteFile(path, []byte(content), 0644)
			if err != nil {
				rpcErr = map[string]interface{}{"code": -32603, "message": err.Error()}
				break
			}
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("Arquivo salvo: %s (%d bytes)", path, len(content))},
				},
			}
		default:
			rpcErr = map[string]interface{}{"code": -32601, "message": "tool not found: " + params.Name}
		}

	default:
		rpcErr = map[string]interface{}{"code": -32601, "message": "method not found: " + req.Method}
	}

	resp := JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  result,
		Error:   rpcErr,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// Also push to SSE session if sessionId provided
	sessID := r.URL.Query().Get("sessionId")
	if sessID != "" {
		sessionsMu.Lock()
		if sess, ok := sessions[sessID]; ok {
			data, _ := json.Marshal(resp)
			select {
			case sess.Channel <- string(data):
			default:
			}
		}
		sessionsMu.Unlock()
	}
}

func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// Legacy file server handlers for compat
func handleList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = rootPath
	}
	if !isAllowed(path) {
		http.Error(w, "path not allowed", 403)
		return
	}
	entries, _ := os.ReadDir(path)
	var files []map[string]interface{}
	for _, e := range entries {
		info, _ := e.Info()
		sz := int64(0)
		if info != nil {
			sz = info.Size()
		}
		files = append(files, map[string]interface{}{"name": e.Name(), "path": filepath.Join(path, e.Name()), "isDir": e.IsDir(), "size": sz})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"files": files})
}

func handleFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !isAllowed(path) {
		http.Error(w, "not allowed", 403)
		return
	}
	http.ServeFile(w, r, path)
}

func main() {
	var port string
	var readOnly bool

	flag.StringVar(&port, "http", ":8000", "Porta ex :8000 ou :8001")
	flag.StringVar(&port, "port", ":8000", "Alias")
	flag.StringVar(&rootPath, "root", "/home/opc/prj", "Root")
	flag.StringVar(&bearerToken, "token", "", "Bearer token")
	flag.BoolVar(&readOnly, "read-only", false, "read only")
	flag.Parse()

	if flag.NArg() > 0 {
		arg := flag.Arg(0)
		if strings.HasPrefix(arg, ":") {
			port = arg
		}
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	// add custom root to allowed
	if rootPath != "" {
		found := false
		for _, r := range allowedRoots {
			if r == rootPath {
				found = true
				break
			}
		}
		if !found {
			allowedRoots = append(allowedRoots, rootPath)
		}
	}

	fmt.Printf("Permitido: %v\n", allowedRoots)
	fmt.Printf("Root: %s | Token: %v | ReadOnly: %v\n", rootPath, bearerToken != "", readOnly)
	fmt.Printf("Servidor rodando em http://localhost%s\n", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", handleSSE)
	mux.HandleFunc("/sse/", handleSSE)
	mux.HandleFunc("/mcp", handleMCP)
	mux.HandleFunc("/mcp/", handleMCP)
	mux.HandleFunc("/list", handleList)
	mux.HandleFunc("/file", handleFile)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte("mcp-meta-editfiles MCP online - SSE at /sse, JSON-RPC at /mcp"))
			return
		}
		handleFile(w, r)
	})

	log.Fatal(http.ListenAndServe(port, mux))
}

