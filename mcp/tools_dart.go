package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mcp-etoolstec-editfiles/config"
	"mcp-etoolstec-editfiles/logger"
)

// ---------------------------------------------------------------------------
// Grupo conceitual separado: ferramentas de execução do toolchain Dart/Flutter.
// Cada tool executa um subcomando do SDK detectado (dart ou flutter) no
// diretório de trabalho informado, respeitando as raízes permitidas/config.
//
// A detecção do SDK é automática: procura o pubspec.yaml a partir do diretório
// informado (subindo até a raiz do sistema). Se o pubspec declara
// "sdk: flutter" o SDK usado é o Flutter; caso contrário, o Dart puro. O
// parâmetro opcional "sdk" permite forçar um dos dois ("auto" é o padrão).
// ---------------------------------------------------------------------------

// dartFlutterSDKPattern detecta projetos Flutter no pubspec.yaml (linha
// "  sdk: flutter" dentro da seção "flutter:").
var dartFlutterSDKPattern = regexp.MustCompile(`(?m)^\s*sdk:\s*flutter\s*$`)

// dartPubspecFiles são os nomes de manifesto reconhecidos em ordem de consulta.
var dartPubspecFiles = []string{"pubspec.yaml", "pubspec.yml"}

// dartToolSDKSchema é o schema do parâmetro opcional "sdk".
func dartToolSDKSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"enum":        []string{"auto", "dart", "flutter"},
		"description": "Opcional. SDK a usar: \"auto\" (padrão) detecta pelo pubspec.yaml; \"dart\" força Dart puro; \"flutter\" força Flutter.",
	}
}

// getToolDartFmt define a tool "dart_fmt" => `dart format`.
// O comando de formatação é sempre o binário "dart", mesmo em projetos Flutter
// (não existe "flutter format").
func getToolDartFmt() Tool {
	return Tool{
		Name:        "dart_fmt",
		Description: "Executa 'dart format' para formatar o código Dart/Flutter do projeto informado (reescreve os arquivos .dart). Em projetos Flutter o comando também é 'dart format'.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Dart/Flutter (diretório que contém o pubspec.yaml).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Opcional. Alvos a formatar (arquivos ou diretórios, ex.: [\"lib\", \"test\"]). Vazio formata o diretório do projeto (\".\").",
				},
				"sdk": dartToolSDKSchema(),
			},
			"required": []string{"path"},
		},
	}
}

// getToolDartImport define a ferramenta "dart_import" => `dart pub add` /
// `flutter pub add` (importa/adiciona dependências e atualiza pubspec).
func getToolDartImport() Tool {
	return Tool{
		Name:        "dart_import",
		Description: "Executa 'pub add' ('dart pub add' ou 'flutter pub add', conforme o SDK do projeto) para importar/adicionar dependências ao projeto Dart/Flutter (atualiza pubspec.yaml e pubspec.lock). Ex.: packages [\"http\", \"provider@^6.0.0\"].",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Dart/Flutter (diretório que contém o pubspec.yaml).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Obrigatório. Uma ou mais dependências a importar (ex.: [\"http\"], [\"provider\", \"intl@^0.19.0\"]).",
				},
				"sdk": dartToolSDKSchema(),
			},
			"required": []string{"path", "packages"},
		},
	}
}

// getToolDartTest define a ferramenta "dart_test" => `dart test` / `flutter test`.
func getToolDartTest() Tool {
	return Tool{
		Name:        "dart_test",
		Description: "Executa 'dart test' ou 'flutter test' (conforme o SDK do projeto) para rodar os testes do projeto Dart/Flutter informado.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Dart/Flutter (diretório que contém o pubspec.yaml).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Opcional. Caminhos de teste a executar (ex.: [\"test/unit\"]). Vazio roda todos os testes.",
				},
				"sdk": dartToolSDKSchema(),
			},
			"required": []string{"path"},
		},
	}
}

// getToolDartBuild define a ferramenta "dart_build" => `dart compile` /
// `flutter build`.
func getToolDartBuild() Tool {
	return Tool{
		Name:        "dart_build",
		Description: "Executa 'dart compile' ou 'flutter build' (conforme o SDK do projeto) para compilar/gerar o build do projeto Dart/Flutter informado.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Caminho absoluto do projeto Dart/Flutter (diretório que contém o pubspec.yaml).",
				},
				"packages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Opcional. Alvo do build. Ex.: Flutter [\"apk\", \"--release\"], [\"web\"]; Dart [\"exe\", \"bin/main.dart\"], [\"js\", \"bin/main.dart\"].",
				},
				"sdk": dartToolSDKSchema(),
			},
			"required": []string{"path"},
		},
	}
}

// executeDartFmt executa `dart format` (sempre o binário dart, mesmo em projeto Flutter).
func executeDartFmt(args map[string]interface{}) (interface{}, interface{}) {
	return runDartCommand("dart_fmt", "format", "", true, args, false, true)
}

// executeDartImport executa `dart pub add`/`flutter pub add`.
func executeDartImport(args map[string]interface{}) (interface{}, interface{}) {
	return runDartCommand("dart_import", "pub add", "pub add", false, args, true, true)
}

// executeDartTest executa `dart test`/`flutter test`.
func executeDartTest(args map[string]interface{}) (interface{}, interface{}) {
	return runDartCommand("dart_test", "test", "test", false, args, false, false)
}

// executeDartBuild executa `dart compile`/`flutter build`.
func executeDartBuild(args map[string]interface{}) (interface{}, interface{}) {
	return runDartCommand("dart_build", "compile", "build", false, args, false, false)
}


// runDartCommand executa um subcomando do SDK (dart ou flutter) dentro do
// diretório informado e devolve a saída combinada (stdout+stderr) com código
// de saída e duração.
//
// dartSub:        subcomando usado com o SDK Dart puro (ex.: "format", "test").
// flutterSub:     subcomando usado com o SDK Flutter (ex.: "test", "build").
//
//	Vazio usa o mesmo subcomando do Dart.
//
// forceDart:      true quando o subcomando é sempre do binário dart
//
//	(dart_fmt => não existe "flutter format").
//
// packagesRequired: exige ao menos um item em "packages".
// needsWrite:     true quando o comando altera arquivos do projeto
//
//	(dart_fmt reescreve fontes; dart_import altera pubspec). Nesses
//	casos o comando é bloqueado em modo read-only e exige raiz de
//	MODIFICAÇÃO (AllowedRootsModify). Testes/build usam leitura.
func runDartCommand(toolName, dartSub, flutterSub string, forceDart bool, args map[string]interface{}, packagesRequired, needsWrite bool) (interface{}, interface{}) {
	dir, _ := args["path"].(string)
	if dir == "" {
		dir = config.RootPath
	}

	// Comandos que escrevem no projeto (dart_fmt, dart_import) exigem permissão
	// de MODIFICAÇÃO (AllowedRootsModify); comandos somente-leitura (dart_test,
	// dart_build) usam a lista de LEITURA (AllowedRoots).
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

	// Resolve o SDK do projeto (auto por pubspec.yaml ou override "sdk").
	exe, sdkDetected := detectDartExecutable(dir, getStringArg(args, "sdk"))
	sdkUsed := sdkDetected
	if forceDart && sdkDetected == "flutter" {
		// Formatação é sempre com o binário dart; informa o SDK efetivamente usado.
		sdkUsed = "dart"
	}

	sub := dartSub
	if !forceDart && sdkDetected == "flutter" && strings.TrimSpace(flutterSub) != "" {
		sub = flutterSub
	}

	packages := collectStringArgs(args, "packages")
	if dartSub == "format" && len(packages) == 0 {
		packages = []string{"."}
	}
	if packagesRequired && len(packages) == 0 {
		return nil, map[string]interface{}{
			"code":    -32602,
			"message": "packages é obrigatório para " + toolName + " (ex.: [\"http\"] )",
		}
	}

	// Monta o comando: <sdk> <sub...> <packages...>
	head := append([]string{exe}, strings.Fields(sub)...)
	full := append(head, packages...)
	logger.Info("[%s] Executando: %s (dir=%s, sdk=%s)", toolName, strings.Join(full, " "), dir, sdkUsed)

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

	text := fmt.Sprintf("Comando: %s\nDiretório: %s\nSDK: %s\nDuração: %s | Exit: %d\n\n%s",
		strings.Join(full, " "), dir, sdkUsed, elapsed, exitCode, output)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"success":  success,
		"exitCode": exitCode,
		"dir":      dir,
		"sdk":      sdkUsed,
	}, nil
}

// detectDartExecutable resolve o executável do SDK para o diretório informado,
// retornando o binário e o nome do SDK ("dart" ou "flutter").
func detectDartExecutable(dir, sdk string) (string, string) {
	if strings.EqualFold(sdk, "flutter") {
		return dartResolveExecutable("flutter"), "flutter"
	}
	if strings.EqualFold(sdk, "dart") {
		return dartResolveExecutable("dart"), "dart"
	}

	// auto: procura pubspec.yaml a partir do diretório informado para cima.
	cur, err := filepath.Abs(dir)
	if err != nil {
		cur = dir
	}
	for {
		for _, name := range dartPubspecFiles {
			if data, err := os.ReadFile(filepath.Join(cur, name)); err == nil {
				if dartFlutterSDKPattern.Match(data) {
					return dartResolveExecutable("flutter"), "flutter"
				}
				return dartResolveExecutable("dart"), "dart"
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return dartResolveExecutable("dart"), "dart"
}

// dartResolveExecutable resolve o binário do SDK. Prioriza o PATH (preservando
// wrappers como o /home/opc/bin/flutter, que roda sob xvfb-run) e cai para
// caminhos conhecidos caso o binário não esteja no PATH.
func dartResolveExecutable(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	switch name {
	case "dart":
		return filepath.Join(home, "flutter", "bin", "dart")
	case "flutter":
		return filepath.Join(home, "bin", "flutter")
	}
	return name
}

// collectStringArgs extrai uma lista de strings de um argumento que aceita
// array ou string única (ex.: "packages").
func collectStringArgs(args map[string]interface{}, key string) []string {
	var out []string
	if raw, ok := args[key].([]interface{}); ok {
		for _, p := range raw {
			if s, ok := p.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	} else if s, ok := args[key].(string); ok && strings.TrimSpace(s) != "" {
		out = append(out, strings.TrimSpace(s))
	}
	return out
}
