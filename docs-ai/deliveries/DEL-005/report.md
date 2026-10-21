---
delivery_id: DEL-005
task_code: TASK-005
title: Expansão MCP Fase 2: Resources pipeline:// e Tools de Scaffold e Sync
type: feat
status: merged
require_security: false
require_migration: false
require_performance: false
date: 2026-10-21
---

# Contexto

Após a introdução do servidor MCP nativo na entrega DEL-004, o servidor contava com suporte às ferramentas básicas `pipeline_doctor` e `pipeline_validate`. No entanto, assistentes de IA ainda precisavam consumir tokens adicionais lendo arquivos estáticos de governança via `read_file`, e o processo de inicialização de novas entregas (`DEL-NNN`) permanecia manual, sujeito a divergências no frontmatter e no `INDEX.md`.

Esta entrega implementa a **Fase 2 da arquitetura MCP**: adiciona a capacidade de **MCP Resources** (`pipeline://...`) para carregamento de contexto sob demanda e duas novas **MCP Tools** para governança automatizada (`pipeline_scaffold_delivery` e `pipeline_sync_skills`).

# Escopo

- Adição da capability `resources` no handshake `initialize` do MCP.
- Implementação dos handlers `resources/list` e `resources/read` para:
  - `pipeline://invariants`: regras inegociáveis de governança e arquitetura.
  - `pipeline://active-delivery`: tabela e status da entrega corrente no `INDEX.md`.
  - `pipeline://catalog`: catálogo de skills e módulos habilitados em `.agents/skills`.
- Implementação de novas MCP Tools:
  - `pipeline_scaffold_delivery`: calcula automaticamente o próximo `DEL-NNN`, gera `report.md` com seções obrigatórias e insere linha no topo do `INDEX.md`.
  - `pipeline_sync_skills`: sincronização de skills a partir dos templates da CLI direto via protocolo MCP.
- Suíte completa de testes unitários para Resources e novas Tools em `internal/mcp/server_test.go`.
- Atualização do `README.md`.

**Fora de escopo:**
- Transporte SSE / HTTP.
- Modificações na lógica de compilação ou gates de validação.

# Implementação

Desenvolvido estritamente em Go puro stdlib sem dependências externas:
- `internal/mcp/protocol.go`:
  - Adicionadas structs tipadas `Resource`, `ResourceContent` e `ReadResourceResult`.
- `internal/mcp/scaffold.go`:
  - Função `ScaffoldDelivery` que avalia regex de `DEL-NNN` no `INDEX.md`, incrementa o contador sequencial, gera a pasta da delivery e o `report.md` com o frontmatter YAML completo e 6 seções obrigatórias, e insere a nova linha atômica no `INDEX.md`.
- `internal/mcp/server.go`:
  - Anúncio de `resources` no handshake de inicialização.
  - Handlers `handleResourcesList` e `handleResourcesRead`.
  - Expansão de `handleToolsList` e `handleToolsCall` para suportar `pipeline_scaffold_delivery` e `pipeline_sync_skills`.
- `README.md`:
  - Documentação atualizada dos Resources e Tools disponíveis via MCP.

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.090s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	(cached)
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Listagem e leitura de Resources MCP | `TestMCPResources` em `internal/mcp/server_test.go` |
| Scaffolding automático de delivery | `TestMCPScaffoldDelivery` em `internal/mcp/server_test.go` |
| Sincronização de skills via MCP | `TestMCPSyncSkills` em `internal/mcp/server_test.go` |
| Handshake com capability de resources | `TestMCPInitializeAndPing` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** Nenhum.

**Sugestões acatadas:**
- Encapsular a lógica de detecção de ID e escrita de relatório em `internal/mcp/scaffold.go` para facilitar reaproveitamento futuro em outros comandos da CLI.
- Adição de fallback na leitura de invariantes: tenta `docs-ai/01-INVARIANTES-GLOBAIS.md`, depois `.agents/rules/invariants.md` e por fim `AGENTS.md`.

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
