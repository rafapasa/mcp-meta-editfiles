package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mcp-etoolstec-editfiles/config"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupTestConfig ajusta a config global para permitir o diretório de teste.
// Guarda os valores originais para restaurar ao final.
func setupTestConfig(t *testing.T, dir string) {
	t.Helper()

	prevRoot := config.RootPath
	prevReadOnly := config.ReadOnly

	config.RootPath = dir
	config.ReadOnly = false
	config.AddAllowedRoot(dir)
	config.AddAllowedRootModify(dir)
	t.Cleanup(func() {
		config.RootPath = prevRoot
		config.ReadOnly = prevReadOnly
	})
}

// createMakefile cria um Makefile dentro de dir com os alvos dados.
// Cada alvo imprime "ran <target>" no stdout. O primeiro alvo é definido
// como .DEFAULT_GOAL para que `make` sem argumentos não falhe.
func createMakefile(t *testing.T, dir string, targets ...string) {
	t.Helper()

	var sb strings.Builder
	if len(targets) > 0 {
		sb.WriteString(".DEFAULT_GOAL := " + targets[0] + "\n")
	}
	sb.WriteString(".PHONY: " + strings.Join(targets, " ") + "\n\n")
	for _, tgt := range targets {
		sb.WriteString(tgt + ":\n\t@echo \"ran " + tgt + "\"\n\n")
	}

	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("falha ao criar Makefile: %v", err)
	}
}

// extractText devolve o texto do primeiro content retornado pela tool.
func extractText(t *testing.T, res interface{}) string {
	t.Helper()
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("resultado não é map: %T", res)
	}
	content, ok := m["content"].([]map[string]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("content inválido: %#v", m["content"])
	}
	text, _ := content[0]["text"].(string)
	return text
}

// ---------------------------------------------------------------------------
// Schemas das tools
// ---------------------------------------------------------------------------

func TestGetToolWireGen_Schema(t *testing.T) {
	tool := getToolWireGen()

	if tool.Name != "wire_gen" {
		t.Errorf("Name = %q, esperado %q", tool.Name, "wire_gen")
	}

	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("InputSchema.properties não é map[string]interface{}")
	}
	if _, ok := props["path"]; !ok {
		t.Errorf("schema deveria conter 'path'")
	}
	if _, ok := props["packages"]; !ok {
		t.Errorf("schema deveria conter 'packages'")
	}

	required, ok := tool.InputSchema["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "path" {
		t.Errorf("required = %v, esperado [\"path\"]", required)
	}
}

func TestGetToolMocksGen_Schema(t *testing.T) {
	tool := getToolMocksGen()
	if tool.Name != "mocks_gen" {
		t.Errorf("Name = %q, esperado %q", tool.Name, "mocks_gen")
	}
}

func TestGetToolMake_Schema(t *testing.T) {
	tool := getToolMake()
	if tool.Name != "make" {
		t.Errorf("Name = %q, esperado %q", tool.Name, "make")
	}
	// Schema deve ser coerente com os demais: path obrigatório, packages opcional.
	required, ok := tool.InputSchema["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "path" {
		t.Errorf("required = %v, esperado [\"path\"]", required)
	}
}

// ---------------------------------------------------------------------------
// wire_gen / mocks_gen — caminho válido
// ---------------------------------------------------------------------------

func TestExecuteWireGen_SucessoCriaSaida(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	res, errIface := executeWireGen(map[string]interface{}{"path": dir})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Errorf("success = %v, esperado true", m["success"])
	}
	if m["exitCode"] != 0 {
		t.Errorf("exitCode = %v, esperado 0", m["exitCode"])
	}

	text := extractText(t, res)
	if !strings.Contains(text, "ran wire") {
		t.Errorf("saída não contém 'ran wire': %q", text)
	}
}

func TestExecuteMocksGen_Sucesso(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "mockgen")

	res, errIface := executeMocksGen(map[string]interface{}{"path": dir})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Errorf("success = %v, esperado true", m["success"])
	}
	text := extractText(t, res)
	if !strings.Contains(text, "ran mockgen") {
		t.Errorf("saída não contém 'ran mockgen': %q", text)
	}
}

// ---------------------------------------------------------------------------
// make — comportamento novo (sem subcomando fixo)
// ---------------------------------------------------------------------------

func TestExecuteMake_SemPackagesUsaDefaultGoal(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "build")

	res, errIface := executeMake(map[string]interface{}{"path": dir})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Errorf("success = %v, esperado true (exit=%v)", m["success"], m["exitCode"])
	}

	text := extractText(t, res)
	// Deve executar `make` sem "mockgen" hardcoded.
	if strings.Contains(text, "make mockgen") {
		t.Errorf("comando não deveria conter 'make mockgen': %q", text)
	}
	if !strings.Contains(text, "ran build") {
		t.Errorf("saída não contém 'ran build': %q", text)
	}
}

func TestExecuteMake_ComPackagesComoAlvo(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "build", "test")

	res, errIface := executeMake(map[string]interface{}{
		"path":     dir,
		"packages": []interface{}{"test"},
	})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	text := extractText(t, res)
	if !strings.Contains(text, "ran test") {
		t.Errorf("saída não contém 'ran test': %q", text)
	}
	if strings.Contains(text, "ran build") {
		t.Errorf("não deveria ter executado 'build': %q", text)
	}
}

// ---------------------------------------------------------------------------
// Comportamento com path vazio / default
// ---------------------------------------------------------------------------

func TestRunMakeCommand_PathVazioUsaRootPath(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	res, errIface := executeWireGen(map[string]interface{}{})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	m := res.(map[string]interface{})
	if m["dir"] != dir {
		t.Errorf("dir = %v, esperado %v", m["dir"], dir)
	}
}

func TestRunMakeCommand_PathTipoInvalido(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	// path é número → cai no RootPath com warn, não deve dar panic.
	res, errIface := executeWireGen(map[string]interface{}{"path": 12345})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	m := res.(map[string]interface{})
	if m["dir"] != dir {
		t.Errorf("dir = %v, esperado %v (fallback RootPath)", m["dir"], dir)
	}
}

// ---------------------------------------------------------------------------
// Caminho não permitido
// ---------------------------------------------------------------------------

func TestExecuteWireGen_PathNaoPermitido(t *testing.T) {
	allowed := t.TempDir()
	forbidden := t.TempDir()
	setupTestConfig(t, allowed)

	_, errIface := executeWireGen(map[string]interface{}{"path": forbidden})
	if errIface == nil {
		t.Fatal("esperava erro de path não permitido, veio nil")
	}

	errMap, ok := errIface.(map[string]interface{})
	if !ok {
		t.Fatalf("erro não é map: %T", errIface)
	}
	msg, _ := errMap["message"].(string)
	if !strings.Contains(msg, "path not allowed") {
		t.Errorf("mensagem = %q, esperada conter 'path not allowed'", msg)
	}
}

// ---------------------------------------------------------------------------
// Modo read-only
// ---------------------------------------------------------------------------

func TestExecuteWireGen_BloqueadoEmReadOnly(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	config.ReadOnly = true
	t.Cleanup(func() { config.ReadOnly = false })

	_, errIface := executeWireGen(map[string]interface{}{"path": dir})
	if errIface == nil {
		t.Fatal("esperava erro de read-only, veio nil")
	}
	errMap := errIface.(map[string]interface{})
	if msg, _ := errMap["message"].(string); !strings.Contains(msg, "read-only") {
		t.Errorf("mensagem = %q, esperada conter 'read-only'", msg)
	}
}

func TestExecuteMake_BloqueadoEmReadOnly(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "build")

	config.ReadOnly = true
	t.Cleanup(func() { config.ReadOnly = false })

	_, errIface := executeMake(map[string]interface{}{"path": dir})
	if errIface == nil {
		t.Fatal("esperava erro de read-only, veio nil")
	}
	errMap := errIface.(map[string]interface{})
	if msg, _ := errMap["message"].(string); !strings.Contains(msg, "read-only") {
		t.Errorf("mensagem = %q, esperada conter 'read-only'", msg)
	}
}

// ---------------------------------------------------------------------------
// Falha do make (alvo inexistente)
// ---------------------------------------------------------------------------

func TestExecuteWireGen_MakeFalha(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	// Makefile sem alvo "wire"
	createMakefile(t, dir, "outro")

	res, errIface := executeWireGen(map[string]interface{}{"path": dir})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	m := res.(map[string]interface{})
	if m["success"] != false {
		t.Errorf("success = %v, esperado false", m["success"])
	}
	if m["exitCode"] == 0 {
		t.Errorf("exitCode = %v, esperado != 0", m["exitCode"])
	}
}

// ---------------------------------------------------------------------------
// packages: array e string única; filtragem de strings vazias
// ---------------------------------------------------------------------------

func TestRunMakeCommand_PackagesArray(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	res, errIface := executeWireGen(map[string]interface{}{
		"path":     dir,
		"packages": []interface{}{"./...", "pkg/a"},
	})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	text := extractText(t, res)
	// O comando foi montado com os argumentos extras, mesmo que make falhe.
	if !strings.Contains(text, "./...") || !strings.Contains(text, "pkg/a") {
		t.Errorf("comando não incluiu pacotes: %q", text)
	}
}

func TestRunMakeCommand_PackagesStringUnica(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	res, errIface := executeWireGen(map[string]interface{}{
		"path":     dir,
		"packages": "./...",
	})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	text := extractText(t, res)
	if !strings.Contains(text, "./...") {
		t.Errorf("comando não incluiu pacote: %q", text)
	}
}

// Verifica que strings vazias em `packages` são filtradas — evita
// `make wire "" foo` que quebraria o make.
func TestRunMakeCommand_FiltraStringsVazias(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	res, errIface := executeWireGen(map[string]interface{}{
		"path":     dir,
		"packages": []interface{}{"", "  ", "extra"},
	})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	text := extractText(t, res)
	// O comando esperado é "make wire extra" (sem strings vazias/brancas).
	if strings.Contains(text, `""`) {
		t.Errorf("comando não deveria conter string vazia: %q", text)
	}
	if !strings.Contains(text, "make wire extra") {
		t.Errorf("comando esperado 'make wire extra': %q", text)
	}
}

// ---------------------------------------------------------------------------
// executeMake não deve injetar subcomando fixo
// ---------------------------------------------------------------------------

func TestExecuteMake_NaoInjetaMockgen(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "build")

	res, errIface := executeMake(map[string]interface{}{
		"path":     dir,
		"packages": []interface{}{"build"},
	})
	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}
	text := extractText(t, res)
	if strings.Contains(text, "make mockgen") {
		t.Errorf("executeMake ainda injeta 'mockgen': %q", text)
	}
	if !strings.Contains(text, "make build") {
		t.Errorf("comando esperado conter 'make build': %q", text)
	}
}

// ---------------------------------------------------------------------------
// Sanidade: exitCode e dir retornados
// ---------------------------------------------------------------------------

func TestRunMakeCommand_RetornaDirEExitCode(t *testing.T) {
	dir := t.TempDir()
	setupTestConfig(t, dir)
	createMakefile(t, dir, "wire")

	res, _ := executeWireGen(map[string]interface{}{"path": dir})
	m := res.(map[string]interface{})
	if m["dir"] != dir {
		t.Errorf("dir = %v, esperado %v", m["dir"], dir)
	}
	if _, ok := m["exitCode"].(int); !ok {
		t.Errorf("exitCode não é int: %T", m["exitCode"])
	}
}

// ---------------------------------------------------------------------------
// Timeout
// ---------------------------------------------------------------------------

// TestRunMakeCommand_TimeoutRespeitado sobrescreve defaultCommandTimeout
// (variável de pacote) para não travar o teste por 10 minutos.
//
// Requer que `defaultCommandTimeout` seja uma var (não const) no mcp.go.
func TestRunMakeCommand_TimeoutRespeitado(t *testing.T) {
	if testing.Short() {
		t.Skip("pula teste de timeout em -short")
	}

	dir := t.TempDir()
	setupTestConfig(t, dir)

	// Makefile com alvo que dorme mais que o timeout do teste.
	makefile := ".PHONY: slow\nslow:\n\t@sleep 3 && echo done\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0o644); err != nil {
		t.Fatalf("falha ao criar Makefile: %v", err)
	}

	prev := defaultCommandTimeout
	defaultCommandTimeout = 500 * time.Millisecond
	t.Cleanup(func() { defaultCommandTimeout = prev })

	start := time.Now()
	res, errIface := executeMake(map[string]interface{}{
		"path":     dir,
		"packages": []interface{}{"slow"},
	})
	elapsed := time.Since(start)

	if errIface != nil {
		t.Fatalf("erro inesperado: %v", errIface)
	}

	// Não deve ter esperado os 3s do sleep.
	if elapsed > 2*time.Second {
		t.Errorf("comando não respeitou timeout: levou %s", elapsed)
	}

	m := res.(map[string]interface{})
	if m["success"] != false {
		t.Errorf("success = %v, esperado false por timeout", m["success"])
	}
	text := extractText(t, res)
	if !strings.Contains(text, "timeout") {
		t.Errorf("saída não menciona timeout: %q", text)
	}
}

// ---------------------------------------------------------------------------
// Nota: config global pode não sobreviver a testes paralelos;
// mantenha os testes deste arquivo rodando em série.
// ---------------------------------------------------------------------------

var _ = context.Background // mantém import "context" caso timeout seja removido
