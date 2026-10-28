---
delivery_id: DEL-023
task_code: TASK-023
title: Empacotamento oficial como Plugin do Google Antigravity (plugin.json, MCP e hooks)
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A distribuição do pipeline-ai até a versão v0.7.0 dependia da sincronização de arquivos soltos em `.agents/` projeto por projeto via `pipeline-ai sync`. Com o amadurecimento do ecossistema Google Antigravity e sua especificação formal de plugins, tornou-se estratégico disponibilizar o toolkit como um **Plugin Nativo de Primeira Classe**, permitindo instalação e ativação em 1 comando, agrupando regras, hooks, skills e servidor MCP em um único bundle declarativo.

Esta delivery introduz o pacote `internal/plugin`, o bundle oficial do plugin em `plugin/` e o subcomando CLI `pipeline-ai plugin [verify|install]`.

# Escopo

- Implementação do pacote `internal/plugin`:
  - `ValidateManifest`: validação estrita do `plugin.json` contra o schema do Antigravity (campos obrigatórios `name`, `displayName`, `version`, `description`, `repository`, `suggestedPrompts` <= 3).
  - `VerifyBundle`: verificação estrutural garantindo presença de `plugin.json`, `rules/AGENTS.md`, `mcp_config.json`, `hooks.json`, `skills/` e `assets/logo.svg`.
  - `InstallToDir`: rotina determinística de instalação e cópia recursiva do bundle com idempotência.
  - Alvo de Fuzzing (`FuzzValidateManifest`).
- Estruturação do bundle oficial em `plugin/`:
  - `plugin/plugin.json`: manifesto oficial com metadados do projeto.
  - `plugin/mcp_config.json`: configuração do servidor MCP `pipeline-ai mcp` sobre stdio.
  - `plugin/hooks.json`: hooks `PreToolUse` e `PostToolUse`.
  - `plugin/rules/AGENTS.md`: regras consolidadas (invariantes, handoff e não-interrupção).
  - `plugin/skills/`: catálogo completo das 16 skills especializadas.
  - `plugin/assets/logo.svg`: logotipo vetorial oficial.
  - `plugin/README.md`: documentação voltada ao usuário do plugin.
- Subcomando CLI `pipeline-ai plugin`:
  - `pipeline-ai plugin verify [-dir <dir>]`: validação de bundles.
  - `pipeline-ai plugin install [-src <dir>] [-target <dir>]`: instalação automatizada em `~/.gemini/config/plugins/pipeline-ai`.

**Fora de escopo:**
- Alterações em regras de interceptação de hooks.
- Adição de dependências externas no `go.mod`.

# Implementação

1. `internal/plugin/plugin.go`:
   - Estruturas `Manifest` e `BundleReport`.
   - Validador `ValidateManifest` com tratamento de campos obrigatórios e restrições de prompts.
   - Verificador estrutural `VerifyBundle`.
   - Instalador recursivo `InstallToDir` preservando permissões e idempotência.
2. `plugin/`:
   - Bundle completo criado e validado, pronto para distribuição ou instalação direta.
3. `cmd/pipeline-ai/main.go`:
   - Subcomando `plugin` implementado com tratamento de flags `-dir`, `-src`, `-target`.
   - Mensagem de ajuda (`printUsage`) atualizada.

**Arquivos alterados:**
- `internal/plugin/plugin.go` — implementação do validador e instalador de plugins
- `internal/plugin/plugin_test.go` — testes unitários table-driven
- `plugin/plugin.json` — manifesto oficial do plugin
- `plugin/mcp_config.json` — configuração do servidor MCP
- `plugin/hooks.json` — hooks de ciclo de vida
- `plugin/rules/AGENTS.md` — regras consolidadas do pipeline
- `plugin/skills/` — catálogo das 16 skills
- `plugin/assets/logo.svg` — logotipo SVG vetorial
- `plugin/README.md` — documentação de uso do plugin
- `cmd/pipeline-ai/main.go` — subcomando CLI pipeline-ai plugin
- `cmd/pipeline-ai/main_test.go` — testes unitários da CLI
- `README.md` — documentação do comando e seção do plugin
- `docs-ai/deliveries/INDEX.md` — registro da delivery DEL-023

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	7.408s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.312s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	2.991s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.547s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.136s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.263s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.029s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	4.511s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.174s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.451s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.070s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	13.672s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.864s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	6.624s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.288s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	1.238s
```

**Alvo de Fuzzing:**
```
$ go test -run=^FuzzValidateManifest$ -fuzz=FuzzValidateManifest -fuzztime=5s ./internal/plugin
fuzz: elapsed: 0s, gathering baseline coverage: 6/6 completed, now fuzzing with 18 workers
fuzz: elapsed: 6s, execs: 103660 (0/sec), new interesting: 182 (total: 188)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	6.511s
```

**Verificação do Bundle:**
```
$ pipeline-ai plugin verify -dir plugin
Bundle do plugin em "plugin" é válido [OK].
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Validação de manifesto plugin.json (campos e restrições) | `TestValidateManifest` em `internal/plugin/plugin_test.go` |
| Verificação da integridade do bundle do plugin | `TestVerifyBundle` em `internal/plugin/plugin_test.go` |
| Instalação recursiva e idempotência | `TestInstallToDir` em `internal/plugin/plugin_test.go` |
| Fuzzing contra entradas corrompidas no manifesto | `FuzzValidateManifest` em `internal/plugin/plugin_test.go` |
| Subcomando CLI pipeline-ai plugin verify e install | `TestCLIPluginCommand` em `cmd/pipeline-ai/main_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** tratamento seguro do diretório padrão do usuário (`~/.gemini/config/plugins/pipeline-ai`) expandindo `$HOME` automaticamente no comando `install`.

**Sugestões recusadas:** nenhuma

# Rollback Plan

Esta entrega adiciona o subdiretório `plugin/` e o pacote `internal/plugin` sem modificar comportamentos anteriores de CI ou sensores. Em caso de reversão:
1. Reverter o commit com `git revert <commit_hash>`.
2. Remover o diretório `plugin/` (se aplicável).
3. Recompilar o binário unificado via `go build -o /tmp/pipeline-ai ./cmd/pipeline-ai`.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
