---
delivery_id: DEL-001
task_code: TASK-001
title: CLI unificada pipeline-ai com subcomandos sync e doctor
type: feat
status: merged
require_security: false
require_migration: false
require_performance: false
date: 2026-10-21
---

# Contexto

Anteriormente, o toolkit de governança do pipeline-ai dependia de múltiplos binários Go isolados (`pipeline-init`, `pipeline-hook` e `pipeline-validate`). Projetos inicializados pelo utilitário ficavam desconectados de melhorias subsequentes no ecossistema de skills e regras (`.agents/`), exigindo cópias manuais propensas a erros.

Esta entrega unifica as ferramentas em uma CLI única `pipeline-ai`, adiciona o subcomando `sync` para atualização segura das 16 skills e regras em projetos existentes, e o subcomando `doctor` para diagnóstico determinístico do ambiente.

# Escopo

- Criação da CLI unificada em `cmd/pipeline-ai/main.go` suportando os subcomandos:
  - `init`: scaffolding de novos projetos no padrão desktop ou api.
  - `hook`: PreToolUse hook para bloqueio de escrita não autorizada.
  - `validate`: validação de relatórios de entrega e salvaguardas de diff git.
  - `sync`: sincronização/atualização segura de `.agents/` em projetos existentes.
  - `doctor`: diagnóstico de ambiente local (Go 1.24+, git, gh, hooks e configuração).
  - `version` e `help`.
- Modularização das rotinas reutilizáveis em `internal/scaffold`, `internal/hook`, `internal/validate` e `internal/doctor`.
- Preservação da retrocompatibilidade total com as diretivas `tool` existentes em `cmd/pipeline-*`.
- Suíte completa de testes unitários para a CLI e novos pacotes.
- Atualização da documentação no `README.md`.

**Fora de escopo:**
- Novos templates de aplicação (CLI/TUI, event-driven - pertencem ao médio prazo).
- Orquestrador de subagents autônomo (longo prazo).
- Servidor MCP dedicado (longo prazo).
- Remoção dos comandos individuais legados.

# Implementação

A implementação foi conduzida estritamente dentro da biblioteca padrão do Go (Go 1.24+ stdlib pura), sem adição de dependências externas ao `go.mod`.

A lógica de scaffolding e templates embutidos (`//go:embed all:templates`) foi concentrada em `internal/scaffold`, expondo as funções `Scaffold` e `Sync`. O comando `sync` atua de forma cirúrgica: atualiza apenas `.agents/skills/`, `.agents/rules/` e `.agents/hooks.json`, nunca tocando em código de negócio (`internal/`, `frontend/`, `main.go`), configurações locais (`.pipeline-ai.json`) ou relatórios de delivery (`docs-ai/deliveries/`).

O pacote `internal/doctor` implementa checagens determinísticas:
1. Validação de versão do Go via regex (`>= 1.24`).
2. Presença dos binários essenciais da toolchain (`gofmt`, `git`, `gh`).
3. Existência e integridade do `.pipeline-ai.json`.
4. Existência de repositório Git e hook `pre-push`.
5. Integridade do diretório `.agents/` e catálogo de skills.

Os binários legados `cmd/pipeline-hook`, `cmd/pipeline-init` e `cmd/pipeline-validate` foram convertidos em chamadas delegadas diretas aos pacotes `internal/`, mantendo estabilidade retroativa de 100%.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — ponto de entrada e roteamento da CLI unificada.
- `cmd/pipeline-ai/main_test.go` — testes de roteamento, argumentos, validação e execução dos subcomandos.
- `cmd/pipeline-hook/main.go` — wrapper para `internal/hook`.
- `cmd/pipeline-init/main.go` — wrapper para `internal/scaffold`.
- `cmd/pipeline-validate/main.go` — wrapper para `internal/validate`.
- `internal/scaffold/scaffold.go` — scaffolding de projetos e templates embutidos.
- `internal/scaffold/sync.go` — rotina de sincronização de `.agents`.
- `internal/scaffold/sync_test.go` — teste de sincronização sem efeitos colaterais.
- `internal/doctor/doctor.go` — verificador de ambiente e conformidade.
- `internal/doctor/doctor_test.go` — testes unitários para parsing de versão e diagnósticos.
- `internal/hook/hook.go` — lógica reutilizável do hook PreToolUse.
- `internal/validate/validate.go` — lógica reutilizável do validador de entregas.
- `README.md` — documentação da CLI unificada e seus subcomandos.
- `docs-ai/deliveries/INDEX.md` — registro da delivery DEL-001.

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.150s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.013s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	0.302s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.014s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.185s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.012s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.106s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Roteamento da CLI unificada | `TestCLIRouting` em `cmd/pipeline-ai/main_test.go` |
| Subcomandos sync e doctor | `TestCLISyncAndDoctor` em `cmd/pipeline-ai/main_test.go` |
| Subcomando validate | `TestCLIValidate` em `cmd/pipeline-ai/main_test.go` |
| Diagnóstico doctor e parsing de Go | `TestParseGoVersion`, `TestCheckConfig`, `TestCheckGit`, `TestRunDoctor` em `internal/doctor/doctor_test.go` |
| Sincronização não-destrutiva de skills | `TestSyncAgents` em `internal/scaffold/sync_test.go` |
| Scaffolding e templates | `TestScaffoldDesktop`, `TestScaffoldAPI`, `TestScaffoldWithGit` em `internal/scaffold/scaffold_test.go` |
| PreToolUse Hook e protocolo Antigravity | `TestRun`, `TestAntigravityProtocol` em `cmd/pipeline-hook/main_test.go` |
| Validação de entregas e salvaguardas | `TestRelatorioAprovado`, `TestSalvaguardaSegurancaReprova` em `cmd/pipeline-validate/main_test.go` |

# Code Review

**Bloqueios:** Nenhum.

**Sugestões acatadas:**
- Encapsulamento de constantes de exit code e métodos nos pacotes `internal/` para evitar acoplamento.
- Criação de testes de integração na CLI verificando ciclo completo de init, sync, doctor e validate.

**Sugestões recusadas:**
- Flag `--json` no `pipeline-ai doctor`: registrada como sugestão futura de melhoria técnica para não inflar o escopo da entrega atual.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
