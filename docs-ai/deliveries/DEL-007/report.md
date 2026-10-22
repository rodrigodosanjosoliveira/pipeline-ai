---
delivery_id: DEL-007
task_code: TASK-007
title: Suporte ao formato de projeto cli e tui no pipeline-ai init
status: draft
require_security: false
require_migration: false
require_performance: false
---

# Contexto

O comando `pipeline-ai init` suportava apenas a criação de projetos nos formatos `desktop` (Wails v3 + React) e `api` (Go API + UI). Desenvolvedores que utilizam o pipeline-ai para construir utilitários de linha de comando (CLI) ou ferramentas de terminal (TUI) em Go puro precisavam adaptar manualmente a estrutura gerada, removendo arquivos de frontend desnecessários e ajustando a governança.

Esta entrega adiciona o formato `cli` (e o alias `tui`) como opção nativa de primeiro nível no utilitário de scaffolding `pipeline-ai init`, com estrutura idiomática em `cmd/<nome>/main.go`, `Taskfile.yml` focado em CLI, templates e governança sem frontend.

# Escopo

- Suporte ao formato `cli` e alias `tui` no método `Normalize()` de `internal/scaffold`.
- Geração de templates dedicados: `internal/scaffold/templates/pipeline-ai.cli.json`, `Taskfile.cli.yml` e `main.cli.go.tmpl`.
- Adaptação de `createDirSkeleton` e `renderMarkdownDoc` para omitir `frontend/src` e gerar o ponto de entrada em `cmd/<nome>/main.go`.
- Atualização do parser de configuração em `internal/config/config.go` para aceitar `project_type: "cli"`.
- Atualização do prompt interativo e flags (`-type cli`) em `cmd/pipeline-ai/main.go`.
- Testes automatizados cobrindo normalização, scaffolding completo e comandos de CLI.

**Fora de escopo:**
- Não alterar contratos ou templates existentes dos tipos `desktop` e `api`.
- Não adicionar dependências de terceiros ao módulo `pipeline-ai` (permanece 100% Go stdlib).
- Não alterar retrocompatibilidade das flags de invocação do `init`.

# Implementação

A implementação estendeu `internal/scaffold` e `internal/config`:
1. `internal/config/config.go`: `ProjectType` passou a aceitar `"cli"`, mantendo a validação estrita contra tipos inválidos.
2. `internal/scaffold/scaffold.go`: `Options.Normalize()` converte `"tui"` em `"cli"` e valida `"desktop"`, `"api"` ou `"cli"`. A função `createDirSkeleton` recebeu o nome do projeto para provisionar `cmd/<nome>` sem criar diretórios de interface gráfica (`frontend/src/`).
3. Templates adicionados:
   - `pipeline-ai.cli.json`: define caminhos protegidos (`internal/`, `cmd/`, `go.mod`, `go.sum`, `Taskfile.yml`) e módulos ativos para CLI.
   - `Taskfile.cli.yml`: tarefas de compilação, execução e testes voltadas para executáveis de terminal.
   - `main.cli.go.tmpl`: aplicação CLI Go idiomática com suporte a flags, streams (`stdout`/`stderr`) e flag `--version`.
4. `cmd/pipeline-ai/main.go`: adicionada opção `3) cli` no menu interativo e documentação da flag `-type`.

**Arquivos alterados:**
- `internal/config/config.go` — Adicionado suporte a `project_type: "cli"` na validação
- `internal/config/config_test.go` — Teste unitário para validação de config com tipo `cli`
- `internal/scaffold/scaffold.go` — Suporte a `cli`/`tui` na normalização, esqueleto e templates
- `internal/scaffold/scaffold_test.go` — Testes de scaffolding CLI, alias TUI e validações
- `internal/scaffold/templates/pipeline-ai.cli.json` — Template de configuração para projetos CLI
- `internal/scaffold/templates/Taskfile.cli.yml` — Taskfile para projetos CLI
- `internal/scaffold/templates/main.cli.go.tmpl` — Template de código Go para CLI
- `cmd/pipeline-ai/main.go` — Prompt interativo e documentação de flag para tipo `cli`
- `cmd/pipeline-ai/main_test.go` — Teste de integração do comando init com `-type cli`
- `docs-ai/03-MAPA-RAPIDO-MODULOS.md` — Inclusão do formato CLI e roteamento
- `docs-ai/GUIA-CRIACAO-PROJETOS.md` — Documentação prática de inicialização de projetos CLI
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-007

# Testes

**Gates:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso)

$ go vet ./...
(análise estática sem avisos)

$ go test ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.180s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.007s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	0.191s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.013s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.025s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.030s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.012s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.100s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Scaffolding completo do formato CLI | `TestScaffoldCLI` em `internal/scaffold/scaffold_test.go` |
| Suporte e normalização do alias TUI | `TestScaffoldCLIAliasTUI` em `internal/scaffold/scaffold_test.go` |
| Validação de tipos permitidos no scaffold | `TestScaffoldValidation` em `internal/scaffold/scaffold_test.go` |
| Validação do formato CLI em `.pipeline-ai.json` | `TestLoadValidacao` em `internal/config/config_test.go` |
| Inicialização CLI via comando `pipeline-ai init` | `TestCLIInitCLI` em `cmd/pipeline-ai/main_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Ajustado template de entrada para `cmd/<nome>/main.go` isolado e garantida ausência de pasta `frontend/` em projetos do tipo CLI.

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
