---
delivery_id: DEL-004
task_code: TASK-004
title: Servidor MCP stdio nativo no comando pipeline-ai mcp
type: feat
status: draft
require_security: false
require_migration: false
require_performance: false
date: 2026-10-21
---

# Contexto

O ecossistema `pipeline-ai` utilizava primordialmente invocação via linha de comando (`run_command`) e hooks de ciclo de vida (`PreToolUse`) para aplicação de governança e validação de entregas. Para elevar a integração com agentes de IA e IDEs como o Antigravity e Claude Desktop a um nível de primeira classe, era necessária a implementação do **Model Context Protocol (MCP)**.

Esta entrega introduz o subcomando `pipeline-ai mcp`, um servidor nativo sobre transporte `stdio` compatível com JSON-RPC 2.0 (versão do protocolo "2024-11-05"), expondo ferramentas determinísticas de governança (`pipeline_doctor` e `pipeline_validate`) sem necessitar de subprocessos de terminal ou bibliotecas de terceiros.

# Escopo

- Implementação do pacote `internal/mcp` em Go puro stdlib:
  - Protocolo JSON-RPC 2.0 (`Request`, `Response`, `RPCError`, `Tool`, `ToolResult`).
  - Handshake e ciclo de vida: `initialize`, `notifications/initialized`, `ping`.
  - Descoberta de ferramentas: `tools/list`.
  - Execução de ferramentas: `tools/call`.
- Exposição das ferramentas:
  - `pipeline_doctor`: diagnóstico de ambiente e configuração.
  - `pipeline_validate`: validação de relatórios de entrega e integridade do Git diff.
- Integração da CLI em `cmd/pipeline-ai/main.go` via subcomando `mcp`.
- Suíte completa de testes unitários para o protocolo MCP e integração de chamadas.
- Documentação no `README.md` com instruções de configuração para IDEs.

**Fora de escopo:**
- Transporte SSE / HTTP (foco inicial no transporte stdio padrão).
- Resources dinâmicos e Prompts MCP (planejados para a Fase 2).
- Dependências externas no `go.mod`.

# Implementação

Construído 100% sobre a biblioteca padrão do Go (`encoding/json`, `bufio`, `io`, `fmt`):
- `internal/mcp/protocol.go`:
  - Estruturas tipadas segundo a especificação MCP e JSON-RPC 2.0.
  - Constantes de erro padrão (`CodeMethodNotFound`, `CodeInvalidParams`, `CodeParseError`).
- `internal/mcp/server.go`:
  - `Server` com método `ServeStream` lendo mensagens delimitadas por quebra de linha.
  - `handleToolsList` registrando esquemas JSON Schema das ferramentas.
  - `handleToolsCall` integrando diretamente com `doctor.Run` e `validate.Run`, encapsulando a saída em `ToolResult` com flag `IsError`.
  - Garantia de isolamento: logs e diagnósticos não poluem o canal de dados `stdout`.
- `cmd/pipeline-ai/main.go`:
  - Roteamento do comando `mcp` com suporte a `-dir` para workspace alvo.
- `README.md`:
  - Nova seção descrevendo a configuração do servidor MCP no `mcpServers`.

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia)

$ go build ./...
(saída vazia)

$ go vet ./...
(saída vazia)

$ go test -race ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.165s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.070s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	(cached)
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Handshake initialize e ping | `TestMCPInitializeAndPing` em `internal/mcp/server_test.go` |
| Listagem de ferramentas tools/list | `TestMCPToolsList` em `internal/mcp/server_test.go` |
| Execução da ferramenta pipeline_doctor | `TestMCPToolsCallDoctor` em `internal/mcp/server_test.go` |
| Execução da ferramenta pipeline_validate | `TestMCPToolsCallValidate` em `internal/mcp/server_test.go` |
| Subcomando CLI pipeline-ai mcp | `TestCLIMCP` em `cmd/pipeline-ai/main_test.go` |

# Code Review

**Bloqueios:** Nenhum.

**Sugestões acatadas:**
- Isolamento estrito de canais: qualquer erro ou log de diagnóstico é direcionado a `stderr`, assegurando que `stdout` transporte apenas frames JSON válidos.
- Suporte a `runWithStdin` na CLI para testes automatizados determinísticos sem bloqueio de I/O.

**Sugestões recusadas:** Nenhuma.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
