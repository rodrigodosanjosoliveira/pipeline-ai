---
delivery_id: DEL-009
task_code: TASK-009
title: Hardening do hook fail-closed, resolução de symlinks e MCP scaffold configurável
status: draft
require_security: true
require_migration: false
require_performance: false
---

# Contexto

Dando continuidade às melhorias de robustez apontadas no contraditório técnico do framework:
1. **Falha Aberta do Hook (C1):** O `pipeline-hook` liberava a execução silenciosamente (`exit 0` / `allow`) caso o stdin estivesse corrompido, ilegível ou contivesse JSON malformado. Essa abordagem fail-open colocava a governança em risco no caso de mudanças de formato da IDE ou falhas transitórias de leitura.
2. **Resolução Incompleta de Caminhos via Symlinks (C2):** O método `relToRoot` em `internal/config` usava apenas `filepath.Abs` e `filepath.Rel`, permitindo que um link simbólico criado fora da pasta protegida e apontando para um diretório protegido (ex: `link_dir` -> `internal/`) contornasse a checagem do `IsProtected`.
3. **Caminhos Fixos no MCP Scaffold (C17):** A ferramenta `ScaffoldDelivery` do servidor MCP assumia caminhos estáticos fixados no código Go (`docs-ai/deliveries` e `INDEX.md`), ignorando as configurações `deliveries_dir` e `index_file` de projetos com estruturas customizadas.

Esta entrega implementa as soluções do pacote P1: falha fechada com mensagens determinísticas para o usuário, resolução canônica de links simbólicos e suporte completo às opções de configuração de delivery no MCP.

# Escopo

- C1: Comportamento fail-closed determinístico no `internal/hook/hook.go` para stdin ilegível e JSON inválido, diferenciando protocolos Antigravity e CLI e sugerindo a execução de `pipeline-ai doctor`.
- C2: Resolução canônica de links simbólicos com `filepath.EvalSymlinks` em `internal/config/config.go` para diretórios e arquivos (inclusive novos arquivos).
- C17: Leitura e aplicação de `deliveries_dir` e `index_file` a partir de `.pipeline-ai.json` em `internal/mcp/scaffold.go`.

**Fora de escopo:**
- Refatorações conceituais de autorização baseada em TTL de delivery (C3) e reorganização de security no PRE (C8), planejadas para v0.6.0.
- Adicionar dependências externas de terceiros (projeto permanece 100% Go stdlib pura).

# Implementação

1. **`internal/hook/hook.go`:**
   - Em `Run()`, se `io.ReadAll(stdin)` falhar por I/O, emite mensagem em stderr orientando executar `pipeline-ai doctor` e bloqueia com `replyDeny(false, msg, stdout)`.
   - Se `json.Unmarshal(raw, &in)` falhar:
     - Caso o payload bruto contenha `"toolCall"`, responde via protocolo Antigravity com `replyDeny(true, msg, stdout)` (`decision: "deny"`, exit 0).
     - Caso contrário, bloqueia via CLI com `replyDeny(false, msg, stdout)` (exit code 2).
2. **`internal/config/config.go`:**
   - Em `relToRoot(absPath string)`, canonicidade resolvida via `filepath.EvalSymlinks(c.root)` e `filepath.EvalSymlinks(clean)`.
   - Para arquivos ainda não gravados em disco, avalia o symlink no diretório pai (`filepath.Dir(clean)`) antes de calcular a rota relativa contra a raiz canônica.
3. **`internal/mcp/scaffold.go`:**
   - Em `ScaffoldDelivery`, carrega `config.Load(rootDir)` e aplica `cfg.DeliveriesDir` e `cfg.IndexFile` com `filepath.FromSlash`, mantendo fallbacks idiomáticos padrão caso a configuração não esteja presente.

**Arquivos alterados:**
- `internal/hook/hook.go` — Comportamento fail-closed seguro no PreToolUse hook
- `cmd/pipeline-hook/main_test.go` — Testes de bloqueio em stdin ilegível e JSON inválido
- `internal/config/config.go` — Resolução canônica de links simbólicos em relToRoot
- `internal/config/config_test.go` — Testes de proteção com diretórios e arquivos via symlink
- `internal/mcp/scaffold.go` — Suporte a deliveries_dir e index_file customizados
- `internal/mcp/server_test.go` — Testes de scaffold de delivery com caminhos configuráveis
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-009
- `docs-ai/deliveries/DEL-009/report.md` — Relatório de entrega da delivery

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.195s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.050s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.060s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.114s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.066s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.115s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.021s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.536s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.366s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| C1 (Hook Fail-Closed Seguro) | `TestRun`, `TestAntigravityProtocol` e `TestStdinIlegivelBloqueia` em `cmd/pipeline-hook/main_test.go` |
| C2 (Resolução de Symlinks) | `TestIsProtectedSymlinks` em `internal/config/config_test.go` |
| C17 (MCP Scaffold Dinâmico) | `TestScaffoldDeliveryCustomPaths` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Resolução defensiva de symlinks no diretório pai para dar suporte a novos arquivos criados dentro de symlinks.

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- `internal/hook/hook.go` — Eliminação de falha aberta (fail-open) no controle de acesso de pré-execução
- `internal/config/config.go` — Mitigação de bypass via links simbólicos (CWE-59)
- `internal/mcp/scaffold.go` — Respeito estrito aos caminhos delimitados na configuração do projeto

**Achados:**
- Nenhum achado de vulnerabilidade residual (0 CRITICO, 0 ALTO, 0 MEDIO, 0 BAIXO).
- Zero dependências externas adicionadas.
- Zero segredos, credenciais ou tokens em código.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
