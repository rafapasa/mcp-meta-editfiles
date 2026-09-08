#!/bin/bash
set -e

TOKEN="48e7bf9b57eb78c1648d5ab4b7010b8763628560e2076f3b45f1cf3eca98327c"
FILE_TOKEN="3HkXIqlut1To8pIYMyltA4FZre0_2fASryp24NCjysgGSpwE"
PRJ_DIR="/home/opc/prj/mcp-meta-editfiles"
CADDYFILE="/home/opc/Caddyfile"
CADDY_BIN="/usr/local/bin/caddy"

echo "=== MATANDO TUDO ==="
pkill -f mcp-meta-editfiles || true
sudo pkill caddy || true
sudo fuser -k 8000/tcp || true
sudo fuser -k 8001/tcp || true
sleep 2

echo "=== COMPILANDO ==="
cd $PRJ_DIR
go build -o mcp-meta-editfiles main.go
chmod +x mcp-meta-editfiles
ls -lh mcp-meta-editfiles

echo "=== CRIANDO Caddyfile ==="
cat > $CADDYFILE <<EOF
{
    email rafael@etoolstec.com.br
}
file.etoolstec.com.br {
    @mcp {
        path /mcp /mcp/* /sse /sse/*
    }
    handle @mcp {
        @auth {
            header Authorization "Bearer $TOKEN"
        }
        handle @auth {
            reverse_proxy localhost:8001
        }
        handle {
            respond "Unauthorized - precisa de Bearer" 401
        }
    }
    handle {
        reverse_proxy localhost:8000
    }
}
EOF
cat $CADDYFILE

echo "=== SUBINDO FILE SERVER 8000 ==="
nohup ./mcp-meta-editfiles --http :8000 --root /home/opc/prj > /tmp/file.log 2>&1 &
sleep 1
cat /tmp/file.log

echo "=== SUBINDO MCP SERVER 8001 ==="
nohup ./mcp-meta-editfiles --http :8001 --root /home/opc/prj/front-openerp --token $TOKEN --read-only > /tmp/mcp.log 2>&1 &
sleep 1
cat /tmp/mcp.log

echo "=== SUBINDO CADDY ==="
nohup sudo $CADDY_BIN run --config $CADDYFILE --adapter caddyfile > /tmp/caddy.log 2>&1 &
sleep 3
cat /tmp/caddy.log | tail -50

echo "=== STATUS ==="
ps aux | grep mcp-meta | grep -v grep
sudo lsof -i :8000 || true
sudo lsof -i :8001 || true

echo ""
echo "=== TESTES LOCAIS ==="
curl -s "http://localhost:8000/list?path=/home/opc/prj/front-openerp&token=$FILE_TOKEN" | head -c 200
echo ""
curl -s http://localhost:8001/mcp
echo ""

echo ""
echo "=== TESTES PUBLICOS ==="
curl -s "https://file.etoolstec.com.br/list?path=/home/opc/prj/front-openerp&token=$FILE_TOKEN" | head -c 200
echo ""
curl -s -H "Authorization: Bearer $TOKEN" https://file.etoolstec.com.br/mcp
echo ""
curl -s https://file.etoolstec.com.br/mcp -w " HTTP:%{http_code}"
echo ""

echo "DONE"

