Alterações e Melhorias Implementadas
    Logs Detalhados:
        Log de todas as operações de edição/criação de arquivos
        Log do tamanho antes/depois da edição
        Log se o arquivo existia ou foi criado
        Log de tentativas de acesso não permitido
    Estrutura Organizada:
        config: Configurações e validação de caminhos
        auth: Autenticação
        handlers: Handlers HTTP (SSE, MCP, File)
        mcp: Lógica MCP (types, tools, sessions)
        logger: Sistema de logs
        utils: Funções utilitárias
    Modo Read-Only:
        Verificação no edit_file para evitar edições quando ativado
    Logs de Operações de Arquivo:
        Formato estruturado JSON no arquivo de log
        Inclui metadados da operação (tamanho, se existia, etc.)
    Arquivo de Log:
        Cria logs diários em ~/.mcp-logs/mcp-YYYY-MM-DD.log
        Formato JSON para fácil processamento
        

make run          # Deploy completo
make stop         # Para tudo
make status       # Status
make test         # Testes
make show-tokens  # Mostra tokens
make help         # Ajuda
make generate-tokens 
        

Instruções de Uso
2. Para fazer deploy completo:
    ./deploy.sh

3. Para parar tudo:
    ./deploy.sh stop

4. Para ver status:
    ./deploy.sh status

5. Para testar:
    ./deploy.sh test

6. Para ver tokens:
    ./deploy.sh tokens

# Compilar
go build -o mcp-server

# Executar com token (recomendado)
./mcp-server -token "seu-token-secreto" -root /home/opc/prj

# Modo read-only
./mcp-server -token "seu-token" -read-only

# Ver logs
tail -f ~/.mcp-logs/mcp-*.log