package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
)

func GetTools() []Tool {
	return []Tool{
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
	}
}

func ExecuteTool(name string, args map[string]interface{}) (interface{}, interface{}) {
	switch name {
	case "list_files":
		return executeListFiles(args)
	case "read_file":
		return executeReadFile(args)
	case "edit_file":
		return executeEditFile(args)
	default:
		return nil, map[string]interface{}{"code": -32601, "message": "tool not found: " + name}
	}
}

func executeListFiles(args map[string]interface{}) (interface{}, interface{}) {
	path, _ := args["path"].(string)
	if path == "" {
		path = config.RootPath
	}
	if !config.IsAllowed(path) {
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed: " + path}
	}

	logger.Info("Listando diretório: %s", path)
	entries, err := os.ReadDir(path)
	if err != nil {
		logger.Error("Erro ao listar %s: %v", path, err)
		return nil, map[string]interface{}{"code": -32603, "message": err.Error()}
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

	logger.Info("Encontrados %d itens em %s", len(files), path)
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("Path: %s | Total: %d", path, len(files)) + "\n" + toJSON(files)},
		},
	}, nil
}

func executeReadFile(args map[string]interface{}) (interface{}, interface{}) {
	path, _ := args["path"].(string)
	if !config.IsAllowed(path) {
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed"}
	}

	logger.Info("Lendo arquivo: %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Erro ao ler %s: %v", path, err)
		return nil, map[string]interface{}{"code": -32603, "message": err.Error()}
	}

	logger.Info("Arquivo lido: %s (%d bytes)", path, len(data))
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(data)},
		},
	}, nil
}

func executeEditFile(args map[string]interface{}) (interface{}, interface{}) {
	if config.ReadOnly {
		logger.Warn("Tentativa de edição em modo read-only: %v", args)
		return nil, map[string]interface{}{"code": -32603, "message": "server is in read-only mode"}
	}

	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if !config.IsAllowed(path) {
		logger.Warn("Tentativa de editar arquivo não permitido: %s", path)
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed"}
	}

	// Verificar se arquivo existe antes de editar
	fileExists := true
	oldContent := ""
	if _, err := os.Stat(path); err == nil {
		oldData, err := os.ReadFile(path)
		if err == nil {
			oldContent = string(oldData)
		}
	} else {
		fileExists = false
	}

	// Criar diretório se não existir
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("Erro ao criar diretório %s: %v", dir, err)
		return nil, map[string]interface{}{"code": -32603, "message": err.Error()}
	}

	// Escrever arquivo
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		logger.Error("Erro ao escrever %s: %v", path, err)
		return nil, map[string]interface{}{"code": -32603, "message": err.Error()}
	}

	// Log detalhado da operação
	metadata := map[string]interface{}{
		"action":      "edit_file",
		"path":        path,
		"file_exists": fileExists,
		"new_size":    len(content),
		"old_size":    len(oldContent),
	}
	logger.FileOperation("edit_file", path, metadata)

	if fileExists {
		logger.Info("Arquivo atualizado: %s (old: %d bytes, new: %d bytes)", path, len(oldContent), len(content))
	} else {
		logger.Info("Arquivo criado: %s (%d bytes)", path, len(content))
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("Arquivo salvo: %s (%d bytes)", path, len(content))},
		},
	}, nil
}

func toJSON(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}
