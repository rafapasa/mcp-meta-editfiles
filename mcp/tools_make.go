package mcp

import (
	"fmt"
	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
	"os/exec"
	"strings"
	"time"
)

func getToolWireGen() Tool {
	return Tool{
		Name:        "wire_gen",
		Description: "Gera código para injeção de dependências no projeto Go.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

func getToolMOcksGen() Tool {
	return Tool{
		Name:        "mocks_gen",
		Description: "Gera código para testes unitarios no projeto Go.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

func executeWireGen() (interface{}, interface{}) {
	return runMakeCommand("wire_gen", "wire", config.RootPath)
}

func executeMocksGen() (interface{}, interface{}) {
	return runMakeCommand("mocks_gen", "mockgen", config.RootPath)
}

func runMakeCommand(toolName, wireOptions string, pathProject string) (interface{}, interface{}) {
	if config.IsAllowedModify(pathProject) {
		logger.Warn("[%s] Caminho não permitido: %s", toolName, pathProject)
		return nil, map[string]interface{}{"code": -32602, "message": "path not allowed: " + pathProject}
	}

	if config.IsAllowed(pathProject) && config.ReadOnly {
		logger.Warn("[%s] Bloqueado em modo read-only (comando altera arquivos do projeto)", toolName)
		return nil, map[string]interface{}{"code": -32603, "message": "server is in read-only mode"}
	}

	cmd := "make " + wireOptions

	logger.Info("[%s] Executando: %s (dir=%s)", toolName, cmd, pathProject)

	var out strings.Builder
	cmdOut := exec.Command(cmd)
	cmdOut.Dir = pathProject
	cmdOut.Stdout = &out
	cmdOut.Stderr = &out

	start := time.Now()
	runErr := cmdOut.Run()
	elapsed := time.Since(start).Round(time.Millisecond)

	exitCode := 0
	success := runErr == nil
	if cmdOut.ProcessState != nil {
		exitCode = cmdOut.ProcessState.ExitCode()
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
		cmd, pathProject, elapsed, exitCode, output)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"success":  success,
		"exitCode": exitCode,
		"dir":      pathProject,
	}, nil
}
