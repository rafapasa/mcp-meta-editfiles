package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/handlers"
	"mcp-etoolstec-editfiles/logger"
)

func main() {
	var port string
	var readOnly bool

	flag.StringVar(&port, "http", ":8000", "Porta ex :8000 ou :8001")
	flag.StringVar(&port, "port", ":8000", "Alias")
	flag.StringVar(&config.RootPath, "root", "/home/opc/prj", "Root")
	flag.StringVar(&config.BearerToken, "token", "", "Bearer token")
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

	// Inicializar configuração
	config.Init(readOnly)
	logger.Init()

	// Adicionar root customizado aos roots permitidos
	config.AddAllowedRoot(config.RootPath)

	logger.Info("Permitido Ler: %v", config.AllowedRoots)
	logger.Info("Permitido Modificar: %v", config.AllowedRootsModify)
	logger.Info("Root: %s | Token: %v | ReadOnly: %v", config.RootPath, config.BearerToken != "", config.ReadOnly)
	logger.Info("Servidor rodando em http://localhost%s", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", handlers.HandleSSE)
	mux.HandleFunc("/sse/", handlers.HandleSSE)
	mux.HandleFunc("/mcp", handlers.HandleMCP)
	mux.HandleFunc("/mcp/", handlers.HandleMCP)
	mux.HandleFunc("/list", handlers.HandleList)
	mux.HandleFunc("/file", handlers.HandleFile)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte("mcp-makeapp MCP online - SSE at /sse, JSON-RPC at /mcp"))
			return
		}
		handlers.HandleFile(w, r)
	})

	log.Fatal(http.ListenAndServe(port, mux))
}
