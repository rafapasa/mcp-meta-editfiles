#!/bin/bash
set -e

PROJECT_DIR="/home/opc/prj/mcp-meta-editfiles"
BINARY_NAME="mcp-meta-editfiles"
TOKEN_FILE="$PROJECT_DIR/.tokens"
CADDYFILE="/home/opc/Caddyfile"
CADDY_BIN="/usr/local/bin/caddy"
LOG_DIR="/tmp/mcp-logs"
FILE_PORT=8000
MCP_PORT=8001

# Mata tudo
pkill -f "$BINARY_NAME" 2>/dev/null || true
sudo pkill caddy 2>/dev/null || true
sudo fuser -k "$FILE_PORT"/tcp 2>/dev/null || true
sudo fuser -k "$MCP_PORT"/tcp 2>/dev/null || true
sleep 2

# Gera tokens
mkdir -p "$PROJECT_DIR"
if [ ! -f "$TOKEN_FILE" ]; then
    MCP_TOKEN=$(openssl rand -hex 32)
    FILE_TOKEN=$(openssl rand -base64 32 | tr -d '\n=' | tr '+/' '-_')
    echo "MCP_TOKEN=$MCP_TOKEN" > "$TOKEN_FILE"
    echo "FILE_TOKEN=$FILE_TOKEN" >> "$TOKEN_FILE"
fi

MCP_TOKEN=$(grep MCP_TOKEN "$TOKEN_FILE" | cut -d= -f2)
FILE_TOKEN=$(grep FILE_TOKEN "$TOKEN_FILE" | cut -d= -f2)

# Compila
cd "$PROJECT_DIR"
go build -o "$BINARY_NAME" main.go
chmod +x "$BINARY_NAME"

# Gera Caddyfile
cat > "$CADDYFILE" <<EOF
{
    email rafael@etoolstec.com.br
}
file.etoolstec.com.br {
    @mcp {
        path /mcp /mcp/* /sse /sse/*
    }
    handle @mcp {
        @auth {
            header Authorization "Bearer $MCP_TOKEN"
        }
        handle @auth {
            reverse_proxy localhost:$MCP_PORT
        }
        handle {
            respond "Unauthorized - precisa de Bearer" 401
        }
    }
    handle {
        reverse_proxy localhost:$FILE_PORT
    }
}
EOF

# Sobe file server
mkdir -p "$LOG_DIR"
nohup ./"$BINARY_NAME" --http ":$FILE_PORT" --root /home/opc/prj > "$LOG_DIR/file.log" 2>&1 &

# Sobe MCP server
nohup ./"$BINARY_NAME" --http ":$MCP_PORT" --root /home/opc/prj/front-openerp --token "$MCP_TOKEN" --read-only > "$LOG_DIR/mcp.log" 2>&1 &

# Sobe Caddy
sudo nohup "$CADDY_BIN" run --config "$CADDYFILE" --adapter caddyfile > "$LOG_DIR/caddy.log" 2>&1 &

sleep 3

echo ""
echo "=== PRONTO ==="
echo "File: http://localhost:$FILE_PORT"
echo "MCP: http://localhost:$MCP_PORT"
echo "Public: https://file.etoolstec.com.br"
echo ""
echo "MCP_TOKEN=$MCP_TOKEN"
echo "FILE_TOKEN=$FILE_TOKEN"