package mcp

import (
	"sync"
)

var (
	Sessions   = make(map[string]*Session)
	SessionsMu sync.Mutex
)
