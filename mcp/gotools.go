package mcp

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
)

// ---------------------------------------------------------------------------
// Grupo conceitual separado: ferramentas de execução do toolchain Go.
// Cada tool executa um subcomando do binário "go" (fmt, get, test, build) no
// diretório de trabalho informado, respeitando as raízes permitidas/config.
// ---------------------------------------------------------------------------

// getToolGoFmt define a tool "go_fmt" => `go fmt`.
func getToolGoFmt() Tool {
	return Tool{
		Name:        "go_fmt",
		Description: "Executa 'go fmt' para formatar o código Go do projeto informado (reescreve os arquivos .go).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Go (diretório que contém o go.mod).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Opcional. Pacotes a formatar (ex.: [\"./...\"]). Vazio formata o pacote do diretório.",
				},
			},
			"required": []string{"path"},
		},
	}
}

// getToolGoImport define a ferramenta "go_import" => `go get`.
// No fluxo Go, "import" corresponde ao comando 'go get': importa/adiciona
// dependências e atualiza go.mod e go.sum.
func getToolGoImport() Tool {
	return Tool{
		Name:        "go_import",
		Description: "Executa 'go get' para importar/adicionar dependências ao módulo Go (atualiza go.mod e go.sum). Ex.: packages [\"github.com/google/uuid\", \"github.com/foo/bar@v1.2.3\"].",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Go (diretório que contém o go.mod).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Obrigatório. Uma ou mais dependências a importar (ex.: \"github.com/google/uuid\" ou \"pkg@v1.2.3\").",
				},
			},
			"required": []string{"path", "packages"},
		},
	}
}

// getToolGoTest define a ferramenta "go_test" => `go test`.
func getToolGoTest() Tool {
	return Tool{
		Name:        "go_test",
		Description: "Executa 'go test' para rodar os testes do projeto Go informado.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Go (diretório que contém o go.mod).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Opcional. Pacotes a testar (ex.: [\"./...\"]). Vazio testa o pacote do diretório.",
				},
			},
			"required": []string{"path"},
		},
	}
}

// getToolGoBuild define a ferramenta "go_build" => `go build`.
func getToolGoBuild() Tool {
	return Tool{
		Name:        "go_build",
		Description: "Executa 'go build' para compilar o projeto Go informado.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Go (diretório que contém o go.mod).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Opcional. Pacotes a compilar (ex.: [\"./...\"]). vazio compila o pacote do diretório.",
				},
			},
			"required": []string{"path"},
		},
	}
}

// executeGoFmt executa `go fmt`.
func executeGoFmt(args map[string]interface{}) (interface{}, interface{}) {
	return runGoCommand("go_fmt", "fmt", args, false, true)
}

// executeGoImport executa `go get` (importa/baixa dependências).
func executeGoImport(args map[string]interface{}) (interface{}, interface{}) {
	return runGoCommand("go_import", "get", args, true, true)
}

// executeGoTest executa `go test`.
func executeGoTest(args map[string]interface{}) (interface{}, interface{}) {
	return runGoCommand("go_test", "test", args, false, false)
}

// executeGoBuild executa `go build`.
func executeGoBuild(args map[string]interface{}) (interface{}, interface{}) {
	return runGoCommand("go_build", "build", args, false, false)
}

// runGoCommand executa um subcomando do "go" dentro do diretório informado e
// devolve a saída combinada (stdout+stderr) com código de saída e duração.
//
//	packagesRequired: exige ao menos um item em "packages".
//	needsWrite:       true quando o comando altera arquivos do projeto
//	                  (go fmt reescreve fontes; go get altera go.mod/go.sum).
//	                  Nesses casos o comando é bloqueado em modo read-only.
func runGoCommand(toolName, goSub string, args map[string]interface{}, packagesRequired, needsWrite bool) (interface{}, interface{}) {
	dir, _ := args["path"].(string)
	if dir == "" {
		dir = config.RootPath
	}

	if !config.IsAllowed(dir) {
		logger.Warn("[%s] Caminho não permitido: %s", toolName, dir)
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed: " + dir}
	}

	if needsWrite && config.ReadOnly {
		logger.Warn("[%s] Bloqueado em modo read-only (comando altera arquivos do projeto)", toolName)
		return nil, map[string]interface{}{"code": -32603, "message": "server is in read-only mode"}
	}

	// Coleta de pacotes/argumentos (aceita array ou string única).
	var packages []string
	if raw, ok := args["packages"].([]interface{}); ok {
		for _, p := range raw {
			if s, ok := p.(string); ok && strings.TrimSpace(s) != "" {
				packages = append(packages, strings.TrimSpace(s))
			}
		}
	} else if s, ok := args["packages"].(string); ok && strings.TrimSpace(s) != "" {
		packages = append(packages, strings.TrimSpace(s))
	}

	if packagesRequired && len(packages) == 0 {
		return nil, map[string]interface{}{
			"code":    -32602,
			"message": "packages é obrigatório para " + toolName + " (ex.: [\"github.com/google/uuid\"] ou [\"./...\"])",
		}
	}

	// Monta o comando: go <sub> <packages...>
	full := append([]string{"go", goSub}, packages...)
	logger.Info("[%s] Executando: %s (dir=%s)", toolName, strings.Join(full, " "), dir)

	var out strings.Builder
	cmd := exec.Command(full[0], full[1:]...)
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &out

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start).Round(time.Millisecond)

	exitCode := 0
	success := runErr == nil
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		if runErr != nil {
			output = runErr.Error()
		} else {
			output = "(sem saída)"
		}
	}

	if runErr != nil {
		logger.Error("[%s] Falhou (exit=%d): %s", toolName, exitCode, output)
	} else {
		logger.Info("[%s] Concluído com sucesso em %s", toolName, elapsed)
	}

	text := fmt.Sprintf("Comando: %s\nDiretório: %s\nDuração: %s | Exit: %d\n\n%s",
		strings.Join(full, " "), dir, elapsed, exitCode, output)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"success":  success,
		"exitCode": exitCode,
		"dir":      dir,
	}, nil
}
