package config

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	AllowedRoots = []string{
		"/home/opc/prj/mcp-server-openerp",
		"/home/opc/prj/front-openerp",
		"/home/opc/prj",
		"/home/opc",
	}
	AllowedRootsModify = []string{
		"/home/opc/prj/mcp-server-openerp",
		"/home/opc/prj/front-openerp",
	}
	BearerToken string
	RootPath    string
	ReadOnly    bool
)

func Init(readOnly bool) {
	ReadOnly = readOnly
	// Carregar variáveis de ambiente se não foram passadas via flag
	if BearerToken == "" {
		BearerToken = os.Getenv("MCP_BEARER_TOKEN")
	}
	if RootPath == "" {
		RootPath = os.Getenv("MCP_ROOT_PATH")
	}
}

func AddAllowedRoot(path string) {
	if path == "" {
		return
	}
	abs, _ := filepath.Abs(path)
	for _, r := range AllowedRoots {
		if r == abs {
			return
		}
	}
	AllowedRoots = append(AllowedRoots, abs)
}

// withinRoot informa se "abs" está dentro da raiz "root" (incluindo a própria
// raiz). A comparação respeita a fronteira de diretório, evitando falsos
// positivos por prefixo — ex.: "/home/opc/prj/front-openerp-FAKE" NÃO deve ser
// considerado dentro de "/home/opc/prj/front-openerp".
func withinRoot(abs, root string) bool {
	if root == "" {
		return false
	}
	root = filepath.Clean(root)
	if abs == root {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix += string(os.PathSeparator)
	}
	return strings.HasPrefix(abs, prefix)
}

// IsAllowed indica se o caminho pode ser LIDO (lista AllowedRoots).
func IsAllowed(p string) bool {
	abs, _ := filepath.Abs(p)
	for _, r := range AllowedRoots {
		if withinRoot(abs, r) {
			return true
		}
	}
	return false
}

// IsAllowedModify indica se o caminho pode ser MODIFICADO (lista AllowedRootsModify).
func IsAllowedModify(p string) bool {
	abs, _ := filepath.Abs(p)
	for _, r := range AllowedRootsModify {
		if withinRoot(abs, r) {
			return true
		}
	}
	return false
}
