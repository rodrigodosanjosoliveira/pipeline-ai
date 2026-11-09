---
delivery_id: DEL-035
task_code: TASK-035
title: "Release v0.11.0: Consolidação da Governança Anti-Overengineering, Debt Ledger, DORA auto-MTTR e 5º artigo técnico"
status: done
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Esta entrega formaliza o lançamento da versão estável **`v0.11.0`** do framework **Pipeline-AI**, consolidando o ciclo de entregas `DEL-032` a `DEL-035`:
1. **Painel de Telemetria Semântica Jev no Dashboard Generative UI (`DEL-032`):** Exibição visual e analítica de `SemanticSummary` na Aba 1 do cockpit HTML e no comando `pipeline-ai metrics summary`.
2. **Fechamento Automático de Métricas DORA e MTTR Inferido (`DEL-033`):** Síntese automática de métricas DORA a partir de eventos `EventGate` do `pipeline-ai run`, registro automático em `pipeline-ai delivery close` (`RecordDeliveryClose`) e cálculo automático de `TimeToRestore` (MTTR) em recuperações de gate.
3. **Governança Anti-Overengineering com Decision Ladder, Debt Ledger e Preset YAGNI (`DEL-034`):** Escada de decisão de 5 degraus nas skills core (`architecture-guardrails`, `dev-implementation`, `code-reviewer`), pacote `internal/debt` (`ParseComment`, `ScanDir`) para inventariar atalhos deliberados (`// shortcut: <limite> | upgrade: <gatilho>`), subcomando CLI `pipeline-ai debt`, tool MCP `pipeline_list_debt`, seção Debt Ledger no Dashboard e preset semântico `pipeline-ai eval -preset yagni`.
4. **Pacote Editorial Completo do 5º Artigo Técnico (`DEL-035`):** Elaboração do artigo *"O Paradoxo da Produtividade em IA: Como Combater o Over-Engineering dos LLMs com Decision Ladders, Debt Ledgers Determinísticos e Sensores YAGNI"* em Markdown (PT-BR), HTML sem tabelas para LinkedIn Articles, Inglês para Dev.to e texto de divulgação para feed do LinkedIn, com estudo de caso real no projeto `gestao-financas`.

# Escopo

- Bump de versão sincronizado para `v0.11.0`:
  - `cmd/pipeline-ai/main.go`: constante `version = "v0.11.0"`.
  - `cmd/pipeline-ai/main_test.go`: asserção de versão `"v0.11.0"` em `TestCLIRouting`.
  - `internal/mcp/server.go`: handshake MCP `serverInfo.version = "v0.11.0"`.
  - `internal/mcp/server_test.go`: asserção de versão MCP `"v0.11.0"` em `TestMCPInitializeAndPing`.
  - `plugin/plugin.json`: campo `"version": "0.11.0"`.
  - `internal/plugin/plugin_test.go`: asserção `"0.11.0"` em `TestOfficialPluginBundleVersion`.
- Publicação do 5º Artigo Técnico (`docs-ai/artigos/`):
  - `docs-ai/artigos/artigo-governanca-anti-overengineering-yagni.md`
  - `docs-ai/artigos/artigo-governanca-anti-overengineering-yagni.html`
  - `docs-ai/artigos/article-anti-overengineering-yagni-go-devto.md`
  - `docs-ai/artigos/post-feed-linkedin-yagni.txt`
- Documentação e Governança:
  - `CHANGELOG.md`: notas de release da versão `0.11.0`.
  - `README.md`: atualização para `v0.11.0`, documentação de `pipeline-ai debt`, `pipeline-ai eval -preset yagni`, tool MCP `pipeline_list_debt` e link para o 5º artigo.
  - `docs-ai/DECISOES.md`: adição da seção 13 (`D-38`).
  - `docs-ai/deliveries/INDEX.md` e `docs-ai/deliveries/DEL-035/report.md`.

**Fora de escopo:**
- Adição de dependências externas em `go.mod` (100% Go 1.24+ standard library mantido).
- Alteração de lógica funcional nos pacotes de domínio além do bump de versão e documentação da release.

# Implementação

1. **Versionamento Sincronizado (`v0.11.0`):**
   - Atualizada a versão da CLI (`cmd/pipeline-ai/main.go`), do servidor MCP (`internal/mcp/server.go`) e do manifesto do plugin oficial (`plugin/plugin.json`) para `v0.11.0` / `0.11.0`, com atualização correspondente nos testes unitários.
2. **Pacote Editorial do 5º Artigo Técnico (`docs-ai/artigos/`):**
   - Redigidos os 4 artefatos de publicação cobrindo o viés aditivo de LLMs, o crédito ao projeto `dietrichgebert/ponytail` e a evolução arquitetural no `pipeline-ai v0.11.0` (Decision Ladder, Debt Ledger determinístico em Go puro e preset semântico YAGNI no Jev), além do estudo de caso real na camada `internal/storage` do projeto `gestao-financas`.
   - O arquivo HTML para LinkedIn Articles foi estruturado exclusivamente com listas semânticas (`<ul>`/`<li>`), sem tags `<table>`, garantindo compatibilidade integral com o editor ProseMirror do LinkedIn.
3. **Documentação de Release (`CHANGELOG.md`, `README.md` e `docs-ai/DECISOES.md`):**
   - Consolidado o histórico da `v0.11.0` no `CHANGELOG.md`, atualizado o `README.md` e registrada a decisão `D-38` em `docs-ai/DECISOES.md`.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Bump de versão para `v0.11.0`
- `cmd/pipeline-ai/main_test.go` — Asserção de versão `v0.11.0` em `TestCLIRouting`
- `internal/mcp/server.go` — Versão do handshake MCP `v0.11.0`
- `internal/mcp/server_test.go` — Asserção de handshake MCP `v0.11.0` em `TestMCPInitializeAndPing`
- `internal/plugin/plugin_test.go` — Asserção de versão `0.11.0` em `TestOfficialPluginBundleVersion`
- `plugin/plugin.json` — Versão do manifesto `0.11.0`
- `CHANGELOG.md` — Notas de release da versão `0.11.0`
- `README.md` — Referência de versão `v0.11.0`, subcomando `debt`, flag `-preset yagni`, MCP `pipeline_list_debt` e link do 5º artigo
- `docs-ai/DECISOES.md` — Seção 13 e decisão `D-38`
- `docs-ai/artigos/artigo-governanca-anti-overengineering-yagni.md` — 5º artigo técnico completo em PT-BR
- `docs-ai/artigos/artigo-governanca-anti-overengineering-yagni.html` — Versão HTML sem tabelas para LinkedIn Articles
- `docs-ai/artigos/article-anti-overengineering-yagni-go-devto.md` — Artigo técnico em Inglês para Dev.to
- `docs-ai/artigos/post-feed-linkedin-yagni.txt` — Post de divulgação para o feed do LinkedIn
- `docs-ai/deliveries/INDEX.md` — Registro da `DEL-035`
- `docs-ai/deliveries/DEL-035/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia, exit code 0)

$ go build ./...
(saída vazia, exit code 0)

$ go vet ./...
(saída vazia, exit code 0)

$ go test -race -count=1 ./cmd/pipeline-ai ./internal/mcp ./internal/plugin ./internal/debt ./internal/sensor/semantic
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	24.410s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	8.820s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.610s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/debt	0.450s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	1.310s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Bump de versão CLI para `v0.11.0` | `TestCLIRouting` em `cmd/pipeline-ai/main_test.go` |
| Bump de versão MCP para `v0.11.0` | `TestMCPInitializeAndPing` em `internal/mcp/server_test.go` |
| Validação de versão `0.11.0` no bundle oficial do plugin | `TestOfficialPluginBundleVersion` em `internal/plugin/plugin_test.go` |
| Validação estrutural do relatório da release | `pipeline-validate -no-git docs-ai/deliveries/DEL-035/report.md` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Formatação do arquivo HTML do 5º artigo sem uso de tags de tabela para compatibilidade total com o LinkedIn Articles.

**Sugestões recusadas:** nenhuma

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
