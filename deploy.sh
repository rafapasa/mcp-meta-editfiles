#!/bin/bash
set -e

PROJECT_DIR="/home/opc/prj/mcp-makeapp"
BINARY_NAME="mcp-makeapp"
TOKEN_FILE="$PROJECT_DIR/.tokens"
CADDYFILE="/home/opc/Caddyfile"
CADDY_BIN="/usr/local/bin/caddy"

pkill -f "$BINARY_NAME" 2>/dev/null || true
sleep 1
mkdir -p "$PROJECT_DIR"

if [ ! -f "$TOKEN_FILE" ]; then
    MCP_TOKEN=$(openssl rand -hex 32)
    echo "MCP_TOKEN=$MCP_TOKEN" > "$TOKEN_FILE"
fi

MCP_TOKEN=$(grep MCP_TOKEN "$TOKEN_FILE" | cut -d= -f2)

cd "$PROJECT_DIR"
go build -o "$BINARY_NAME" .

cat > "$CADDYFILE" <<EOF
{
    email rafael@etoolstec.com.br
}
file.etoolstec.com.br {
    reverse_proxy localhost:8001
}
EOF

# UM servidor só, root em /home/opc/prj já libera tudo (front-openerp tá dentro)
nohup ./"$BINARY_NAME" --http :8001 --root /home/opc/prj --token "$MCP_TOKEN" > /tmp/mcp.log 2>&1 &

sudo pkill caddy 2>/dev/null || true
sleep 1
sudo nohup "$CADDY_BIN" run --config "$CADDYFILE" --adapter caddyfile > /tmp/caddy.log 2>&1 &

sleep 3
echo "=== PRONTO ==="
echo "TOKEN=$MCP_TOKEN"
echo "curl -s https://file.etoolstec.com.br/mcp?token=$MCP_TOKEN -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}'"