package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
)

func getToolListFiles() Tool {
	return Tool{
		Name:        "list_files",
		Description: "Lista arquivos e pastas de um diretório. Use para explorar front-openerp. Exemplo: {\"path\": \"/home/opc/prj/front-openerp/lib/presentation/pages/tenants\"}",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Caminho absoluto no servidor"},
			},
			"required": []string{"path"},
		},
	}
}

func GetTools() []Tool {
	return []Tool{
		// Arquivos e Diretórios
		getToolListFiles(),
		getToolReadFile(),
		getToolEditFile(),
		getToolMakeDir(),
		getToolFindFiles(),

		// Go - execução do toolchain (grupo conceitual separado)
		getToolGoFmt(),
		getToolGoImport(),
		getToolGoTest(),
		getToolGoBuild(),

		// Dart/Flutter - execução do toolchain (grupo conceitual separado)
		getToolDartFmt(),
		getToolDartImport(),
		getToolDartTest(),
		getToolDartBuild(),
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
	case "make_dir":
		return executeMakeDir(args)
	case "find_files":
		return executeFindFiles(args)
	case "go_fmt":
		return executeGoFmt(args)
	case "go_import":
		return executeGoImport(args)
	case "go_test":
		return executeGoTest(args)
	case "go_build":
		return executeGoBuild(args)
	case "dart_fmt":
		return executeDartFmt(args)
	case "dart_import":
		return executeDartImport(args)
	case "dart_test":
		return executeDartTest(args)
	case "dart_build":
		return executeDartBuild(args)
	default:
		return nil, map[string]interface{}{"code": -32601, "message": "tool not found: " + name}
	}
}

func executeMakeDir(args map[string]interface{}) (interface{}, interface{}) {
	path, _ := args["path"].(string)
	dirName, _ := args["dir_name"].(string)
	if path == "" || dirName == "" {
		return nil, map[string]interface{}{"code": -32602, "message": "path and dir_name are required"}
	}

	fullPath := filepath.Join(path, dirName)
	if !config.IsAllowedModify(fullPath) {
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed: " + fullPath}
	}

	logger.Info("Criando diretório: %s", fullPath)
	err := os.MkdirAll(fullPath, 0755)
	if err != nil {
		logger.Error("Erro ao criar diretório %s: %v", fullPath, err)
		return nil, map[string]interface{}{"code": -32603, "message": err.Error()}
	}

	logger.Info("Diretório criado com sucesso: %s", fullPath)
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("Diretório criado: %s", fullPath)},
		},
	}, nil
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

	if !config.IsAllowedModify(path) {
		logger.Warn("Tentativa de editar arquivo não permitido: %s", path)
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed"}
	}

	// LOG: Início da operação
	logger.Info("Iniciando edição de arquivo: %s", path)
	logger.Debug("Tamanho do conteúdo: %d bytes", len(content))

	// Verificar se arquivo existe antes de editar
	fileExists := true
	oldSize := 0
	if info, err := os.Stat(path); err == nil {
		oldData, err := os.ReadFile(path)
		if err == nil {
			oldSize = len(oldData)
		}
		logger.Info("Arquivo existente: %s (tamanho: %d bytes, modificado: %s)",
			path, oldSize, info.ModTime().Format("2006-01-02 15:04:05"))
	} else if os.IsNotExist(err) {
		fileExists = false
		logger.Info("Arquivo não existe, será criado: %s", path)
	} else {
		// Erro ao verificar o arquivo
		logger.Error("Erro ao verificar existência do arquivo %s: %v", path, err)
		return nil, map[string]interface{}{"code": -32603, "message": "error checking file: " + err.Error()}
	}

	// Criar diretório se não existir
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("ERRO CRÍTICO: Falha ao criar diretório %s: %v", dir, err)
		logger.FileOperation("edit_file_error", path, map[string]interface{}{
			"error":      err.Error(),
			"error_type": "mkdir_failed",
			"directory":  dir,
		})
		return nil, map[string]interface{}{"code": -32603, "message": "failed to create directory: " + err.Error()}
	}

	// Tentar escrever o arquivo
	logger.Info("Tentando escrever arquivo: %s", path)

	// Criar um arquivo temporário primeiro (para evitar corrupção)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		logger.Error("ERRO CRÍTICO: Falha ao escrever arquivo temporário %s: %v", tmpPath, err)
		logger.FileOperation("edit_file_error", path, map[string]interface{}{
			"error":          err.Error(),
			"error_type":     "write_tmp_failed",
			"temporary_path": tmpPath,
			"content_size":   len(content),
		})

		// Limpar arquivo temporário se existir
		os.Remove(tmpPath)
		return nil, map[string]interface{}{"code": -32603, "message": "failed to write file: " + err.Error()}
	}

	// Renomear temporário para o arquivo final (atômico)
	if err := os.Rename(tmpPath, path); err != nil {
		logger.Error("ERRO CRÍTICO: Falha ao renomear arquivo %s -> %s: %v", tmpPath, path, err)
		logger.FileOperation("edit_file_error", path, map[string]interface{}{
			"error":          err.Error(),
			"error_type":     "rename_failed",
			"temporary_path": tmpPath,
		})

		// Limpar arquivo temporário
		os.Remove(tmpPath)
		return nil, map[string]interface{}{"code": -32603, "message": "failed to save file: " + err.Error()}
	}

	// Verificar se o arquivo foi realmente escrito
	if info, err := os.Stat(path); err == nil {
		if info.Size() != int64(len(content)) {
			logger.Error("ERRO: Tamanho do arquivo não corresponde! Esperado: %d, Obtido: %d",
				len(content), info.Size())
			logger.FileOperation("edit_file_size_mismatch", path, map[string]interface{}{
				"expected_size": len(content),
				"actual_size":   info.Size(),
			})
			// Não falha, mas registra o aviso
		} else {
			logger.Info("Arquivo escrito com sucesso: %s (tamanho: %d bytes)", path, info.Size())
		}
	} else {
		logger.Error("ERRO: Não foi possível verificar o arquivo após escrita: %v", err)
	}

	// Log detalhado da operação
	metadata := map[string]interface{}{
		"action":      "edit_file",
		"path":        path,
		"file_exists": fileExists,
		"new_size":    len(content),
		"old_size":    oldSize,
		"size_diff":   len(content) - oldSize,
		"success":     true,
		"timestamp":   time.Now().Format(time.RFC3339Nano),
	}
	logger.FileOperation("edit_file_success", path, metadata)

	if fileExists {
		logger.Info("Arquivo atualizado com sucesso: %s (antigo: %d bytes, novo: %d bytes, diff: %d bytes)",
			path, oldSize, len(content), len(content)-oldSize)
	} else {
		logger.Info("Arquivo criado com sucesso: %s (%d bytes)", path, len(content))
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
