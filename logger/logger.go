package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	logFile *os.File
)

type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

func Init() {
	// Criar diretório de logs
	logDir := filepath.Join(os.Getenv("HOME"), ".mcp-logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Warning: Could not create log dir: %v", err)
		return
	}

	// Abrir arquivo de log
	logPath := filepath.Join(logDir, fmt.Sprintf("mcp-%s.log", time.Now().Format("2006-01-02")))
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not open log file: %v", err)
	}
}

func log(level, msg string, args ...interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     level,
		Message:   fmt.Sprintf(msg, args...),
	}

	// Log para console
	fmt.Printf("[%s] %s", level, entry.Message)

	// Log para arquivo
	if logFile != nil {
		data, _ := json.Marshal(entry)
		logFile.Write(append(data, '\n'))
		logFile.Sync()
	}
}

func Info(msg string, args ...interface{}) {
	log("INFO", msg, args...)
}

func Warn(msg string, args ...interface{}) {
	log("WARN", msg, args...)
}

func Error(msg string, args ...interface{}) {
	log("ERROR", msg, args...)
}

func Debug(msg string, args ...interface{}) {
	log("DEBUG", msg, args...)
}

func FileOperation(action, path string, metadata interface{}) {
	log("FILE_OP", "Action: %s | Path: %s | Metadata: %+v", action, path, metadata)
}

func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

func FileError(action, path string, err error, metadata interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     "FILE_ERROR",
		Message:   fmt.Sprintf("ERRO em %s | Path: %s | Erro: %v", action, path, err),
		Metadata:  metadata,
	}

	// Log para console em vermelho
	fmt.Printf("\033[31m[FILE_ERROR] %s\033[0m", entry.Message)

	// Log para arquivo
	if logFile != nil {
		data, _ := json.Marshal(entry)
		logFile.Write(append(data, '\n'))
		logFile.Sync()
	}
}

func FileWarning(action, path string, warning string, metadata interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     "FILE_WARNING",
		Message:   fmt.Sprintf("AVISO em %s | Path: %s | %s", action, path, warning),
		Metadata:  metadata,
	}

	// Log para console em amarelo
	fmt.Printf("\033[33m[FILE_WARNING] %s\033[0m", entry.Message)

	// Log para arquivo
	if logFile != nil {
		data, _ := json.Marshal(entry)
		logFile.Write(append(data, '\n'))
		logFile.Sync()
	}
}
