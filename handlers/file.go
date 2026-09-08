package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"mcp-etoolstec-editfiles/auth"
	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
)

func HandleList(w http.ResponseWriter, r *http.Request) {
	if !auth.CheckAuth(r) {
		http.Error(w, "Unauthorized -?token= invalido", 401)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		path = config.RootPath
	}
	if !config.IsAllowed(path) {
		http.Error(w, "path not allowed", 403)
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"files": files})
}

func HandleFile(w http.ResponseWriter, r *http.Request) {
	if !auth.CheckAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	path := r.URL.Query().Get("path")
	if !config.IsAllowed(path) {
		logger.Warn("Tentativa de acesso a arquivo não permitido: %s", path)
		http.Error(w, "not allowed", 403)
		return
	}
	http.ServeFile(w, r, path)
}
