package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
)

func getToolMakeDir() Tool {
	return Tool{
		Name:        "make_dir",
		Description: "Cria um diretório no sistema de arquivos.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":     map[string]interface{}{"type": "string", "description": "Caminho absoluto do diretório a ser criado"},
				"dir_name": map[string]interface{}{"type": "string", "description": "Nome do diretório a ser criado"},
			},
			"required": []string{"path", "dir_name"},
		},
	}
}

func getToolEditFile() Tool {
	return Tool{
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
	}
}

func getToolReadFile() Tool {
	return Tool{
		Name:        "read_file",
		Description: "Lê conteúdo completo de um arquivo .dart, .json, .yaml. Retorna texto. Use para pegar código antes de editar.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Caminho absoluto do arquivo"},
			},
			"required": []string{"path"},
		},
	}
}

// getToolFindFiles define a tool "find_files" para procurar arquivos e
// diretórios recursivamente dentro de uma raiz permitida.
func getToolFindFiles() Tool {
	return Tool{
		Name:        "find_files",
		Description: "Procura recursivamente por arquivos e diretórios dentro de um caminho, filtrando por nome e tipo. Use para localizar um arquivo/pasta quando não souber o caminho exato.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto da raiz onde buscar (obrigatório).",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Opcional. Parte do nome (substring, sem diferenciar maiúsculas) para filtrar arquivos/diretórios.",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"file", "dir"},
					"description": "Opcional. Filtra por tipo: \"file\" (só arquivos) ou \"dir\" (só diretórios). Vazio retorna todos.",
				},
				"max_depth": map[string]interface{}{
					"type":        "integer",
					"description": "Opcional. Profundidade máxima de diretórios a percorrer (padrão 6).",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Opcional. Limite máximo de resultados (padrão 200).",
				},
			},
			"required": []string{"path"},
		},
	}
}

// executeFindFiles busca arquivos/diretórios a partir de "path" usando filtros
// de nome e tipo. A busca é iterativa (pilha explícita) para evitar recursão
// profunda e ignora diretórios grandes comuns (node_modules, .git, etc.).
func executeFindFiles(args map[string]interface{}) (interface{}, interface{}) {
	root, _ := args["path"].(string)
	if root == "" {
		root = config.RootPath
	}
	if !config.IsAllowed(root) {
		logger.Warn("find_files: caminho não permitido: %s", root)
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed: " + root}
	}

	nameFilter := strings.ToLower(strings.TrimSpace(getStringArg(args, "name")))
	typeFilter := strings.ToLower(strings.TrimSpace(getStringArg(args, "type")))

	maxDepth, limit := 6, 200
	if v, ok := args["max_depth"].(float64); ok && int(v) > 0 {
		maxDepth = int(v)
	}
	if v, ok := args["limit"].(float64); ok && int(v) > 0 {
		limit = int(v)
	}

	// Diretórios ignorados durante a varredura (reduzem ruído e custo).
	skippedDirs := []string{".git", "node_modules", ".cache"}

	logger.Info("find_files: buscando em %s (name=%q, type=%q, max_depth=%d, limit=%d)",
		root, nameFilter, typeFilter, maxDepth, limit)

	var results []map[string]interface{}
	queue := []string{root}
	truncated := false

	for len(queue) > 0 && len(results) < limit {
		cur := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		// Profundidade relativa à raiz.
		rel, _ := filepath.Rel(root, cur)
		if rel == "." {
			rel = ""
		}
		depth := 0
		if rel != "" {
			depth = strings.Count(filepath.ToSlash(rel), "/") + 1
		}
		if depth > maxDepth {
			continue
		}

		entries, err := os.ReadDir(cur)
		if err != nil {
			logger.Warn("find_files: erro ao ler %s: %v", cur, err)
			continue
		}

		for _, e := range entries {
			if len(results) >= limit {
				truncated = true
				break
			}

			name := e.Name()
			isDir := e.IsDir()

			if nameFilter != "" && !strings.Contains(strings.ToLower(name), nameFilter) {
				continue
			}
			if typeFilter == "file" && isDir {
				continue
			}
			if typeFilter == "dir" && !isDir {
				continue
			}

			full := filepath.Join(cur, name)
			results = append(results, map[string]interface{}{
				"name":  name,
				"path":  full,
				"isDir": isDir,
			})

			if isDir && !containsString(skippedDirs, name) {
				queue = append(queue, full)
			}
		}
	}

	summary := fmt.Sprintf("Path: %s | Encontrados: %d", root, len(results))
	if truncated {
		summary += " (atingido o limite de resultados)"
	}
	logger.Info("find_files: %s", summary)

	text := summary + "\n" + toJSON(results)
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"results":   results,
		"total":     len(results),
		"truncated": truncated,
	}, nil
}

// getStringArg retorna o valor de args[k] como string (vazio se ausente/inválido).
func getStringArg(args map[string]interface{}, key string) string {
	if s, ok := args[key].(string); ok {
		return s
	}
	return ""
}

// containsString verifica se a slice contém o valor informado.
func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
