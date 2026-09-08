package config

import (
	"os"
	"path/filepath"
	"strings"
)

var (
	AllowedRoots = []string{
		"/home/opc/Documents/meta-ai-files",
		"/home/opc/prj/front-opener",
		"/home/opc/prj/front-openerp",
		"/home/opc/prj",
		"/home/opc",
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

func IsAllowed(p string) bool {
	abs, _ := filepath.Abs(p)
	for _, r := range AllowedRoots {
		if strings.HasPrefix(abs, r) {
			return true
		}
	}
	return false
}
