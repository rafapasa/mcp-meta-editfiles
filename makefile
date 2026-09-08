# Makefile para mcp-etoolstec-editfiles
.PHONY: all build clean run run-file run-mcp run-caddy stop status test help

# Variáveis
PROJECT_DIR := /home/opc/prj/mcp-etoolstec-editfiles
BINARY_NAME := mcp-etoolstec-editfiles
CADDYFILE := /home/opc/Caddyfile
CADDY_BIN := /usr/local/bin/caddy
LOG_DIR := /tmp/mcp-logs

# Tokens (serão gerados se não existirem)
TOKEN_FILE := $(PROJECT_DIR)/.tokens
TOKEN ?= $(shell cat $(TOKEN_FILE) 2>/dev/null | grep MCP_TOKEN | cut -d= -f2 || echo "")
FILE_TOKEN ?= $(shell cat $(TOKEN_FILE) 2>/dev/null | grep FILE_TOKEN | cut -d= -f2 || echo "")

# Portas
FILE_PORT := 8000
MCP_PORT := 8001

# Cores para output (opcional - pode remover se não funcionar)
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m

all: generate-tokens build

generate-tokens:
	@echo "$(BLUE)=== GERANDO TOKENS ===$(NC)"
	@mkdir -p $(PROJECT_DIR)
	@if [ ! -f $(TOKEN_FILE) ] || [ -z "$$(cat $(TOKEN_FILE) 2>/dev/null)" ]; then \
		MCP_TOKEN=$$(openssl rand -hex 32); \
		FILE_TOKEN=$$(openssl rand -base64 32 | tr -d '\n=' | tr '+/' '-_'); \
		echo "MCP_TOKEN=$$MCP_TOKEN" > $(TOKEN_FILE); \
		echo "FILE_TOKEN=$$FILE_TOKEN" >> $(TOKEN_FILE); \
		echo "$(GREEN)✓ Tokens gerados e salvos em $(TOKEN_FILE)$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Tokens já existem em $(TOKEN_FILE)$(NC)"; \
	fi
	@echo "$(GREEN)MCP_TOKEN: $(shell grep MCP_TOKEN $(TOKEN_FILE) | cut -d= -f2)$(NC)"
	@echo "$(GREEN)FILE_TOKEN: $(shell grep FILE_TOKEN $(TOKEN_FILE) | cut -d= -f2)$(NC)"

show-tokens:
	@if [ -f $(TOKEN_FILE) ]; then \
		echo "$(BLUE)=== TOKENS ATUAIS ===$(NC)"; \
		cat $(TOKEN_FILE); \
	else \
		echo "$(RED)⚠ Arquivo de tokens não encontrado. Execute 'make generate-tokens'$(NC)"; \
	fi

build:
	@echo "$(BLUE)=== COMPILANDO ===$(NC)"
	@cd $(PROJECT_DIR) && \
		go build -o $(BINARY_NAME) . && \
		chmod +x $(BINARY_NAME)
	@echo "$(GREEN)✓ Build concluído: $(PROJECT_DIR)/$(BINARY_NAME)$(NC)"
	@ls -lh $(PROJECT_DIR)/$(BINARY_NAME)

clean:
	@echo "$(BLUE)=== LIMPANDO ===$(NC)"
	@-pkill -f $(BINARY_NAME) 2>/dev/null || true
	@-sudo pkill caddy 2>/dev/null || true
	@-sudo fuser -k $(FILE_PORT)/tcp 2>/dev/null || true
	@-sudo fuser -k $(MCP_PORT)/tcp 2>/dev/null || true
	@-rm -f $(PROJECT_DIR)/$(BINARY_NAME)
	@-rm -f $(LOG_DIR)/*.log
	@echo "$(GREEN)✓ Limpeza concluída$(NC)"

stop: clean
	@echo "$(GREEN)✓ Todos os serviços parados$(NC)"

caddyfile:
	@echo "$(BLUE)=== GERANDO CADDYFILE ===$(NC)"
	@TOKEN=$$(grep MCP_TOKEN $(TOKEN_FILE) | cut -d= -f2); \
	cat > $(CADDYFILE) <<EOF
{
    email rafael@etoolstec.com.br
}
file.etoolstec.com.br {
    @mcp {
        path /mcp /mcp/* /sse /sse/*
    }
    handle @mcp {
        @auth {
            header Authorization "Bearer $$TOKEN"
        }
        handle @auth {
            reverse_proxy localhost:$(MCP_PORT)
        }
        handle {
            respond "Unauthorized - precisa de Bearer" 401
        }
    }
    handle {
        reverse_proxy localhost:$(FILE_PORT)
    }
}
EOF
	@echo "$(GREEN)✓ Caddyfile gerado em $(CADDYFILE)$(NC)"
	@cat $(CADDYFILE)

run-file:
	@echo "$(BLUE)=== SUBINDO FILE SERVER (porta $(FILE_PORT)) ===$(NC)"
	@mkdir -p $(LOG_DIR)
	@TOKEN=$$(grep FILE_TOKEN $(TOKEN_FILE) | cut -d= -f2); \
	cd $(PROJECT_DIR) && \
	nohup ./$(BINARY_NAME) --http :$(FILE_PORT) --root /home/opc/prj > $(LOG_DIR)/file.log 2>&1 &
	@sleep 2
	@echo "$(GREEN)✓ File server rodando em http://localhost:$(FILE_PORT)$(NC)"
	@echo "$(YELLOW)Log: $(LOG_DIR)/file.log$(NC)"
	@tail -3 $(LOG_DIR)/file.log 2>/dev/null || echo "Aguardando logs..."

run-mcp:
	@echo "$(BLUE)=== SUBINDO MCP SERVER (porta $(MCP_PORT)) ===$(NC)"
	@mkdir -p $(LOG_DIR)
	@TOKEN=$$(grep MCP_TOKEN $(TOKEN_FILE) | cut -d= -f2); \
	cd $(PROJECT_DIR) && \
	nohup ./$(BINARY_NAME) --http :$(MCP_PORT) --root /home/opc/prj/front-openerp --token $$TOKEN --read-only > $(LOG_DIR)/mcp.log 2>&1 &
	@sleep 2
	@echo "$(GREEN)✓ MCP server rodando em http://localhost:$(MCP_PORT)$(NC)"
	@echo "$(YELLOW)Log: $(LOG_DIR)/mcp.log$(NC)"
	@tail -3 $(LOG_DIR)/mcp.log 2>/dev/null || echo "Aguardando logs..."

run-caddy: caddyfile
	@echo "$(BLUE)=== SUBINDO CADDY ===$(NC)"
	@mkdir -p $(LOG_DIR)
	@sudo nohup $(CADDY_BIN) run --config $(CADDYFILE) --adapter caddyfile > $(LOG_DIR)/caddy.log 2>&1 &
	@sleep 3
	@echo "$(GREEN)✓ Caddy rodando$(NC)"
	@echo "$(YELLOW)Log: $(LOG_DIR)/caddy.log$(NC)"
	@tail -5 $(LOG_DIR)/caddy.log 2>/dev/null || echo "Aguardando logs..."

run: clean generate-tokens build run-file run-mcp run-caddy
	@echo ""
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)✓ TODOS OS SERVIÇOS INICIADOS!$(NC)"
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(YELLOW)File Server: http://localhost:$(FILE_PORT)$(NC)"
	@echo "$(YELLOW)MCP Server: http://localhost:$(MCP_PORT)$(NC)"
	@echo "$(YELLOW)Public: https://file.etoolstec.com.br$(NC)"
	@echo ""
	@echo "$(BLUE)TOKENS:$(NC)"
	@cat $(TOKEN_FILE)
	@echo ""
	@echo "$(BLUE)LOGS:$(NC)"
	@echo "  File: $(LOG_DIR)/file.log"
	@echo "  MCP:  $(LOG_DIR)/mcp.log"
	@echo "  Caddy: $(LOG_DIR)/caddy.log"

status:
	@echo "$(BLUE)=== STATUS DOS SERVIÇOS ===$(NC)"
	@echo ""
	@echo "$(YELLOW)Processos:$(NC)"
	@ps aux | grep -E "$(BINARY_NAME)|caddy" | grep -v grep || echo "Nenhum processo encontrado"
	@echo ""
	@echo "$(YELLOW)Portas:$(NC)"
	@sudo lsof -i :$(FILE_PORT) 2>/dev/null || echo "Porta $(FILE_PORT) livre"
	@sudo lsof -i :$(MCP_PORT) 2>/dev/null || echo "Porta $(MCP_PORT) livre"
	@echo ""
	@echo "$(YELLOW)Testes rápidos:$(NC)"
	@-curl -s "http://localhost:$(FILE_PORT)/list?path=/home/opc/prj/front-openerp" -w "\nHTTP: %{http_code}\n" 2>/dev/null | head -3 || echo "File server offline"
	@-curl -s "http://localhost:$(MCP_PORT)/mcp" -w "\nHTTP: %{http_code}\n" 2>/dev/null || echo "MCP server offline"
	@echo ""
	@echo "$(YELLOW)Tokens:$(NC)"
	@cat $(TOKEN_FILE) 2>/dev/null || echo "Arquivo de tokens não encontrado"

test:
	@echo "$(BLUE)=== TESTANDO ENDPOINTS ===$(NC)"
	@echo ""
	@FILE_TOKEN=$$(grep FILE_TOKEN $(TOKEN_FILE) | cut -d= -f2); \
	MCP_TOKEN=$$(grep MCP_TOKEN $(TOKEN_FILE) | cut -d= -f2); \
	echo "$(YELLOW)1. File Server Local:$(NC)"; \
	curl -s "http://localhost:$(FILE_PORT)/list?path=/home/opc/prj/front-openerp&token=$$FILE_TOKEN" | head -c 200; \
	echo "\n"; \
	echo "$(YELLOW)2. MCP Server Local:$(NC)"; \
	curl -s "http://localhost:$(MCP_PORT)/mcp"; \
	echo "\n"; \
	echo "$(YELLOW)3. MCP Server Local (com auth):$(NC)"; \
	curl -s -H "Authorization: Bearer $$MCP_TOKEN" "http://localhost:$(MCP_PORT)/mcp"; \
	echo "\n"; \
	echo "$(YELLOW)4. Public File Server:$(NC)"; \
	curl -s "https://file.etoolstec.com.br/list?path=/home/opc/prj/front-openerp&token=$$FILE_TOKEN" | head -c 200; \
	echo "\n"; \
	echo "$(YELLOW)5. Public MCP (com auth):$(NC)"; \
	curl -s -H "Authorization: Bearer $$MCP_TOKEN" https://file.etoolstec.com.br/mcp; \
	echo "\n"; \
	echo "$(YELLOW)6. Public MCP (sem auth - deve dar 401):$(NC)"; \
	curl -s https://file.etoolstec.com.br/mcp -w "\nHTTP: %{http_code}\n"; \
	echo ""

help:
	@echo "$(BLUE)=== COMANDOS DISPONÍVEIS ===$(NC)"
	@echo ""
	@echo "$(GREEN)make all$(NC)           - Gera tokens e compila"
	@echo "$(GREEN)make generate-tokens$(NC) - Gera novos tokens automaticamente"
	@echo "$(GREEN)make show-tokens$(NC)   - Mostra os tokens atuais"
	@echo "$(GREEN)make build$(NC)         - Compila o binário"
	@echo "$(GREEN)make run$(NC)           - Inicia todos os serviços (limpa, compila, gera tokens, inicia)"
	@echo "$(GREEN)make stop$(NC)          - Para todos os serviços"
	@echo "$(GREEN)make status$(NC)        - Mostra status dos serviços"
	@echo "$(GREEN)make test$(NC)          - Testa todos os endpoints"
	@echo "$(GREEN)make clean$(NC)         - Limpa binários e logs"
	@echo "$(GREEN)make help$(NC)          - Mostra esta ajuda"
	@echo ""
	@echo "$(BLUE)Comandos individuais:$(NC)"
	@echo "$(GREEN)make run-file$(NC)      - Inicia apenas o file server"
	@echo "$(GREEN)make run-mcp$(NC)       - Inicia apenas o MCP server"
	@echo "$(GREEN)make run-caddy$(NC)     - Inicia apenas o Caddy"
	@echo "$(GREEN)make caddyfile$(NC)     - Gera o Caddyfile"