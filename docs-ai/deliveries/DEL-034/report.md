---
delivery_id: DEL-034
task_code: TASK-034
title: "Governança Anti-Overengineering: Decision Ladder, Debt Ledger (internal/debt) e preset semântico YAGNI"
status: done
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Agentes de IA frequentemente introduzem complexidade acidental (over-engineering), como interfaces de implementação única desnecessárias, abstrações especulativas e comentários de pendência vagos sem critério objetivo de evolução. Inspirado nas práticas da skill `ponytail` (Decision Ladder, Minimalism Discipline e Debt Tracking estruturado), o `pipeline-ai` passa a incorporar mecanismos determinísticos e semânticos nativos em Go puro para prevenir over-engineering e rastrear atalhos deliberados com gatilhos explícitos de upgrade.

# Escopo

- `internal/debt`:
  - Criação do pacote `internal/debt` (`debt.go` e `debt_test.go`) com `ShortcutItem`, `Summary`, `ParseComment(line string)` e `ScanDir(rootDir string)`.
  - Suporte aos formatos `// shortcut: <limite> | upgrade: <gatilho>` e `// shortcut: <limite>, <gatilho>`, ignorando comentários comuns, arquivos `*_test.go` e diretórios ocultos, `vendor` e `node_modules`.
- `internal/sensor/semantic`:
  - Adição de `ResolvePreset(name string)` em `presets.go` para resolver o preset `yagni` (e aliases `minimalism` e `ponytail`) para o modo `noul` com pergunta calibrada contra over-engineering.
- `cmd/pipeline-ai`:
  - Novo subcomando `pipeline-ai debt [-dir <path>] [-json]` para listar o inventário de atalhos deliberados em formato tabular/texto ou JSON.
  - Nova flag `-preset <name>` no subcomando `pipeline-ai eval`.
- `internal/mcp`:
  - Nova ferramenta MCP `pipeline_list_debt` (`dir` opcional) que retorna o JSON do `debt.Summary`.
  - Suporte ao argumento opcional `preset` na ferramenta `pipeline_eval_semantic`.
- `internal/dashboard`:
  - Inclusão do campo `Debt *debt.Summary` em `DashboardData`, populado automaticamente em `GenerateData` e renderizado como seção "Atalhos Deliberados (Debt Ledger)" no HTML e Markdown.
- Skills e Templates:
  - Atualização de `.agents/skills/`, `internal/scaffold/templates/agents/skills/` e `plugin/skills/` para `architecture-guardrails` (Decision Ladder), `dev-implementation` (minimalismo estrito e marcador `// shortcut:`) e `code-reviewer` (auditoria anti-overengineering e validação de atalhos estruturados).

**Fora de escopo:**
- Adição de dependências externas em `go.mod` (100% Go 1.24+ standard library).
- Alteração de invariantes globais existentes (D-01 a D-37).

# Implementação

1. **Rastreador Determinístico de Atalhos Deliberados (`internal/debt/debt.go`):**
   - `ParseComment` localiza comentários `//`, verifica o prefixo case-insensitive `shortcut:` e extrai o par `(limit, trigger)` separado por `|` ou `,`, removendo o rótulo opcional `upgrade:`.
   - `ScanDir` percorre recursivamente o diretório raiz via `filepath.WalkDir`, ignorando diretórios iniciados por `.` (exceto a própria raiz), `vendor`, `node_modules` e arquivos que não sejam `.go` ou que terminem em `_test.go`, ordenando o resultado por `File` e `Line`.
2. **Preset Semântico YAGNI (`internal/sensor/semantic/presets.go`):**
   - `ResolvePreset` mapeia `"yagni"`, `"minimalism"` e `"ponytail"` para o modo binário `"noul"` com pergunta calibrada de verificação de minimalismo e ausência de abstrações especulativas.
3. **Integração na CLI (`cmd/pipeline-ai/main.go`), MCP (`internal/mcp/server.go`) e Dashboard (`internal/dashboard/dashboard.go`):**
   - Adicionado `runDebt` na CLI com suporte a `-dir` e `-json`, além da flag `-preset` em `runEval`.
   - Registrada a tool `pipeline_list_debt` e o parâmetro `preset` em `pipeline_eval_semantic` no servidor MCP.
   - Integrado `debt.ScanDir` em `dashboard.GenerateData`, exibindo o card e tabela de "Atalhos Deliberados (Debt Ledger)" em `RenderHTML` e `RenderMarkdown`.
4. **Skills Nativas Sincronizadas:**
   - Incorporados a Escada de Decisão (`1. YAGNI -> 2. Reuso no repo -> 3. Stdlib -> 4. Dependência já instalada -> 5. Código novo mínimo`), o contrato de anotação `// shortcut: <limite> | upgrade: <gatilho>` e o checklist de revisão em `.agents/skills/`, `internal/scaffold/templates/agents/skills/` e `plugin/skills/`.

**Arquivos alterados:**
- `internal/debt/debt.go` — Parser `ParseComment` e scanner `ScanDir` para atalhos deliberados
- `internal/debt/debt_test.go` — Testes unitários table-driven (`TestParseComment`, `TestScanDir`) e fuzzing (`FuzzParseComment`)
- `internal/sensor/semantic/presets.go` — Função `ResolvePreset` e constante `yagniQuestion`
- `internal/sensor/semantic/presets_test.go` — Testes unitários `TestResolvePreset`
- `cmd/pipeline-ai/main.go` — Subcomando `pipeline-ai debt` e flag `-preset` em `pipeline-ai eval`
- `cmd/pipeline-ai/main_test.go` — Testes `TestCLIDebtCommand` e `TestCLIEvalPresetFlag`
- `internal/mcp/server.go` — Tool `pipeline_list_debt` e suporte a `preset` em `pipeline_eval_semantic`
- `internal/mcp/server_test.go` — Testes de contrato MCP para `pipeline_list_debt` e `preset`
- `internal/dashboard/dashboard.go` — Campo `Debt` em `DashboardData` e renderização no HTML e Markdown
- `internal/dashboard/dashboard_test.go` — Teste `TestGenerateDataIncludesDebtSummary`
- `.agents/skills/architecture-guardrails/SKILL.md` — Inclusão da Decision Ladder
- `.agents/skills/dev-implementation/SKILL.md` — Minimalismo estrito e anotação `// shortcut:`
- `.agents/skills/code-reviewer/SKILL.md` — Checklist anti-overengineering e validação de atalhos deliberados
- `internal/scaffold/templates/agents/skills/architecture-guardrails/SKILL.md` — Sincronização do template
- `internal/scaffold/templates/agents/skills/dev-implementation/SKILL.md` — Sincronização do template
- `internal/scaffold/templates/agents/skills/code-reviewer/SKILL.md` — Sincronização do template
- `plugin/skills/architecture-guardrails/SKILL.md` — Sincronização do plugin
- `plugin/skills/dev-implementation/SKILL.md` — Sincronização do plugin
- `plugin/skills/code-reviewer/SKILL.md` — Sincronização do plugin
- `docs-ai/03-MAPA-RAPIDO-MODULOS.md` — Documentação de `internal/debt`, preset `yagni`, CLI `debt` e MCP `pipeline_list_debt`
- `docs-ai/deliveries/INDEX.md` — Registro da `DEL-034` e atualização de status da `DEL-033`
- `docs-ai/deliveries/DEL-034/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia, exit code 0)

$ go build ./...
(saída vazia, exit code 0)

$ go vet ./...
(saída vazia, exit code 0)

$ go test ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	24.162s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-doctor	0.236s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.415s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-sensor	0.388s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	0.531s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/audit	0.971s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.668s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.961s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/debt	0.465s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.822s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	1.077s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook	1.112s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	8.955s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	1.196s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/modules	0.962s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	10.742s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.564s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	2.309s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	1.321s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	1.458s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/syncer	1.644s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validator	1.737s

$ go test -fuzz=FuzzParseComment -fuzztime=5s ./internal/debt
fuzz: elapsed: 0s, gathering baseline coverage: 0/5 completed
fuzz: elapsed: 0s, gathering baseline coverage: 5/5 completed, now fuzzing with 10 workers
fuzz: elapsed: 5s, execs: 695412 (139074/sec), new interesting: 14 (total: 19)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/debt	5.214s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Extração e fuzzing de comentários `// shortcut:` (`ParseComment`) | `TestParseComment` e `FuzzParseComment` em `internal/debt/debt_test.go` |
| Varredura recursiva e determinística de diretórios (`ScanDir`) | `TestScanDir` em `internal/debt/debt_test.go` |
| Resolução do preset semântico `yagni` e aliases (`ResolvePreset`) | `TestResolvePreset` em `internal/sensor/semantic/presets_test.go` |
| Subcomando `pipeline-ai debt` e flag `eval -preset` na CLI | `TestCLIDebtCommand` e `TestCLIEvalPresetFlag` em `cmd/pipeline-ai/main_test.go` |
| Ferramenta MCP `pipeline_list_debt` e argumento `preset` em `pipeline_eval_semantic` | `TestMCPServerListDebtTool` e `TestMCPServerEvalSemanticPreset` em `internal/mcp/server_test.go` |
| Seção Debt Ledger no Dashboard Generative UI e Markdown | `TestGenerateDataIncludesDebtSummary` em `internal/dashboard/dashboard_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Sincronização dos 3 arquivos `SKILL.md` atualizados também para o diretório `plugin/skills/` durante a etapa de documentação.

**Sugestões recusadas:** nenhuma

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Alvos de fuzz para todo parser/validador/desserializador novo
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
