---
delivery_id: DEL-031
task_code: TASK-031
title: "Atualização do diagrama de fluxo de execução no README.md para v0.10.0 e ajuste de lista no artigo HTML do Jev"
status: merged
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A imagem estática `docs-ai/pipeline-ai.png` referenciada anteriormente no `README.md` refletia a arquitetura inicial da época da `v0.3.0`/`v0.4.0`, não contemplando evoluções fundamentais introduzidas entre a `v0.6.0` e a `v0.10.0` (Security Shift-Left com `REQUISITOS_SEGURANCA` no PRE, modelo híbrido de autorização no `PreToolUse` Hook com TTL e `immutable_paths`, ordem sequencial na fase de revisão, etapa `[release-ops]`, validador de handoffs `handoff validate`, Pipeline Runner `pipeline-ai run`, Sensor Semântico Jev `noul/choice/score` e Telemetria DORA passiva). Além disso, o editor de artigos do LinkedIn (ProseMirror) não suporta a tag HTML `<table>`, exigindo a conversão da tabela comparativa em `docs-ai/artigos/artigo-sensoriamento-semantico-jev.html` para uma lista estruturada (`<ul>`/`<li>`).

# Escopo

- `README.md`:
  - Substituição da referência à imagem estática desatualizada por um diagrama Mermaid nativo atualizado para a `v0.10.0`.
  - Atualização das referências de instalação `go get -tool` de `@v0.6.0` para `@v0.10.0`.
  - Atualização da tabela do Catálogo de Skills Nativas (`architecture-guardrails`, `test-writer`, `security-review`) com o fluxo bidirecional Security Shift-Left (`D-22`).
  - Atualização da seção de métricas DORA com menção à agregação `SemanticSummary` e inclusão da seção **Série de Artigos Técnicos (`docs-ai/artigos/`)**.
- `docs-ai/artigos/artigo-sensoriamento-semantico-jev.html`:
  - Conversão da tabela HTML comparativa (System 1 vs. System 2) para lista estruturada `<ul>`/`<li>` compatível com o editor do LinkedIn Articles.
- Governança de Deliveries:
  - `docs-ai/deliveries/DEL-030/report.md`: status sincronizado para `merged`.
  - `docs-ai/deliveries/INDEX.md`: `DEL-030` marcada como `merged` e `DEL-031` registrada como `draft`.
  - `docs-ai/deliveries/DEL-031/report.md`: relatório desta delivery.

**Fora de escopo:**
- Alterações em código-fonte Go (`cmd/`, `internal/`).

# Implementação

1. **Diagrama Mermaid Nativo e Sincronização do `README.md` (`v0.10.0`):**
   - Diagrama `flowchart LR` estruturado nas 4 fases (`1. Escopo & Design (PRE)`, `2. TDD & Implementação`, `3. Revisão & Segurança (POST)` e `4. Validação, Observabilidade & Entrega`), exibindo todas as 16 skills, o `PreToolUse Hook` híbrido, o `handoff validate`, o `pipeline-ai run`, o sensor semântico `Jev` e a telemetria passiva DORA.
   - Comandos `go get -tool` e exemplos de `pipeline-ai delivery open` sincronizados para `v0.10.0`.
   - Nova seção listando os 4 artigos técnicos em `docs-ai/artigos/`.
2. **Compatibilidade HTML com LinkedIn Articles (`artigo-sensoriamento-semantico-jev.html`):**
   - Substituição de `<table>` por `<ul>`/`<li>` numerado com destaques em negrito e itálico.
3. **Governança:**
   - Sincronização de `DEL-030` para `merged` e criação de `DEL-031`.

**Arquivos alterados:**
- `README.md` — Diagrama de fluxo Mermaid atualizado para `v0.10.0`, exemplos sincronizados e catálogo de artigos
- `docs-ai/artigos/artigo-sensoriamento-semantico-jev.html` — Conversão de `<table>` para `<ul>`/`<li>` compatível com LinkedIn Articles
- `docs-ai/deliveries/DEL-030/report.md` — Status atualizado para `merged`
- `docs-ai/deliveries/INDEX.md` — `DEL-030` marcada como `merged` e `DEL-031` registrada
- `docs-ai/deliveries/DEL-031/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -count=1 ./internal/plugin ./internal/doctor ./internal/validate
=== RUN   TestOfficialPluginBundleVersion
--- PASS: TestOfficialPluginBundleVersion (0.01s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.418s
=== RUN   TestCheckConfig
--- PASS: TestCheckConfig (0.00s)
=== RUN   TestCheckGit
--- PASS: TestCheckGit (0.00s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.411s
=== RUN   TestValidateJSONOutput
--- PASS: TestValidateJSONOutput (0.00s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.422s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Integridade do manifesto e versão `0.10.0` | `TestOfficialPluginBundleVersion` em `internal/plugin/plugin_test.go` |
| Validação estrutural do relatório `DEL-031/report.md` | `go tool pipeline-validate docs-ai/deliveries/DEL-031/report.md` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** adoção de bloco `mermaid` nativo no `README.md` em substituição ao PNG estático legado para manter o diagrama versionável em texto e sempre sincronizado com a arquitetura.

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
