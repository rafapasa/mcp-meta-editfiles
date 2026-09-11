package mcp

import (
	"context"
	"fmt"
	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
	"os/exec"
	"strings"
	"time"
)

// defaultCommandTimeout limita a duração de comandos make para evitar
// travar o servidor MCP caso um alvo fique preso (ex.: watcher, servidor).
var defaultCommandTimeout = 10 * time.Minute

func getToolWireGen() Tool {
	return Tool{
		Name:        "wire_gen",
		Description: "Gera código para injeção de dependências no projeto Go (executa `make wire`).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto onde o comando deve ser executado.",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Argumentos adicionais repassados ao make (ex.: [\"./...\"]).",
				},
			},
			"required": []string{"path"},
		},
	}
}

func getToolMocksGen() Tool {
	return Tool{
		Name:        "mocks_gen",
		Description: "Gera código para testes unitários no projeto Go (executa `make mockgen`).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto onde o comando deve ser executado.",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Argumentos adicionais repassados ao make (ex.: [\"./...\"]).",
				},
			},
			"required": []string{"path"},
		},
	}
}

func getToolMake() Tool {
	return Tool{
		Name:        "make",
		Description: "Executa um alvo do Makefile do projeto. Os itens de `packages` são passados como alvos/argumentos para o make.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto onde o comando deve ser executado.",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Alvos/argumentos do make (ex.: [\"build\"], [\"test\", \"./...\"]).",
				},
			},
			"required": []string{"path"},
		},
	}
}

func executeWireGen(args map[string]interface{}) (interface{}, interface{}) {
	return runMakeCommand("wire_gen", "wire", args, false, true)
}

func executeMocksGen(args map[string]interface{}) (interface{}, interface{}) {
	return runMakeCommand("mocks_gen", "mockgen", args, false, true)
}

func executeMake(args map[string]interface{}) (interface{}, interface{}) {
	// Não injeta subcomando fixo: os alvos vêm de `packages`.
	return runMakeCommand("make", "", args, false, true)
}

func runMakeCommand(toolName, makeSub string, args map[string]interface{}, packagesRequired, needsWrite bool) (interface{}, interface{}) {
	dir, _ := args["path"].(string)
	if dir == "" {
		if raw, exists := args["path"]; exists && raw != nil {
			logger.Warn("[%s] 'path' inválido (%T); usando RootPath (%s)", toolName, raw, config.RootPath)
		}
		dir = config.RootPath
	}

	// Comandos que escrevem no projeto (make wire, make mockgen) exigem
	// permissão de MODIFICAÇÃO (AllowedRootsModify); comandos somente-leitura
	// usam a lista de LEITURA (AllowedRoots).
	canRun := config.IsAllowed(dir)
	if needsWrite {
		canRun = config.IsAllowedModify(dir)
	}
	if !canRun {
		logger.Warn("[%s] Caminho não permitido: %s", toolName, dir)
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed: " + dir}
	}

	if needsWrite && config.ReadOnly {
		logger.Warn("[%s] Bloqueado em modo read-only (comando altera arquivos do projeto)", toolName)
		return nil, map[string]interface{}{"code": -32603, "message": "server is in read-only mode"}
	}

	// Coleta de pacotes/argumentos (aceita array ou string única).
	packages := collectStringArgs(args, "packages")

	if packagesRequired && len(packages) == 0 {
		return nil, map[string]interface{}{
			"code":    -32602,
			"message": "packages é obrigatório para " + toolName + " (ex.: [\"github.com/google/uuid\"] ou [\"./...\"])",
		}
	}

	// Monta o comando: make [makeSub] [packages...], ignorando strings vazias.
	raw := make([]string, 0, 2+len(packages))
	raw = append(raw, "make")
	if makeSub != "" {
		raw = append(raw, makeSub)
	}
	raw = append(raw, packages...)

	full := make([]string, 0, len(raw))
	for _, s := range raw {
		if s != "" {
			full = append(full, s)
		}
	}

	logger.Info("[%s] Executando: %s (dir=%s)", toolName, strings.Join(full, " "), dir)

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	var out strings.Builder
	cmd := exec.CommandContext(ctx, full[0], full[1:]...)
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

	// Timeout: o contexto expirou e o processo foi morto pelo CommandContext.
	timedOut := ctx.Err() == context.DeadlineExceeded
	if timedOut {
		success = false
		if exitCode == 0 {
			exitCode = -1
		}
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		if runErr != nil {
			output = runErr.Error()
		} else {
			output = "(sem saída)"
		}
	}
	if timedOut {
		output = fmt.Sprintf("comando excedeu o timeout de %s\n%s", defaultCommandTimeout, output)
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
