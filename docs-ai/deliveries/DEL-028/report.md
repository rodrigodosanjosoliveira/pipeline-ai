---
delivery_id: DEL-028
task_code: TASK-028
title: "Resolução recursiva de symlinks ancestrais em Config.relToRoot (macOS TMPDIR /var -> /private/var) e atualização do artigo DORA Metrics"
status: draft
require_security: true
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

1. **Falha na Resolução de Symlinks Ancestrais em `Config.relToRoot` (macOS):**
   Em ambientes macOS, `os.TempDir()` / `t.TempDir()` retorna caminhos sob `/var/folders/...`, onde `/var` é um link simbólico do sistema operacional apontando para `/private/var`. Quando `Config.relToRoot(absPath)` avaliava um caminho de arquivo ainda não criado cujos diretórios intermediários também não existiam em disco (por exemplo, `/var/folders/.../root/internal/sub1/sub2/novo.go` ou `docs-ai/deliveries/DEL-001/report.md`), tanto `filepath.EvalSymlinks(clean)` quanto `filepath.EvalSymlinks(filepath.Dir(clean))` falhavam com `ENOENT`. Com isso, `clean` permanecia com prefixo `/var/folders/...` enquanto `rootClean` (`filepath.EvalSymlinks(c.root)`) era resolvido para `/private/var/folders/...`, fazendo `filepath.Rel(rootClean, clean)` produzir `../../../../var/folders/...` (iniciado por `..`) e retornar `("", false)`. Isso impedia o reconhecimento de caminhos protegidos e imutáveis para arquivos novos em subdiretórios aninhados ainda inexistentes sob uma árvore com ancestral symlinkado.
2. **Atualização do Artigo Técnico DORA Metrics & Telemetria de Agentes de IA:**
   Sincronização e refinamento editorial dos materiais de divulgação técnica sobre Métricas DORA e Telemetria de Agentes de IA (`docs-ai/artigos/`).

# Escopo

- Implementação da função auxiliar privada `resolveSymlinkAncestors(p string) string` em `internal/config/config.go` e seu uso em `Config.relToRoot` para tanto `c.root` quanto `clean`.
- Cobertura de testes unitários e adversariais de segurança em `TestIsProtectedSymlinks` (`internal/config/config_test.go`), cobrindo `SEC-01` (subdiretórios múltiplos inexistentes sob symlink e sob raiz symlinkada estilo macOS `/var -> /private/var`) e `SEC-02` (symlink apontando para fora da raiz do projeto).
- Atualização dos arquivos do artigo sobre DORA Metrics e Telemetria de IA em `docs-ai/artigos/`.
- Sincronização do status da delivery anterior (`DEL-027`) para `merged` e registro da `DEL-028` em `docs-ai/deliveries/INDEX.md`.

**Fora de escopo:**
- Adição de dependências externas (100% Go 1.24+ standard library).
- Alteração de contratos públicos exportados de `internal/config` ou protocolo de comunicação do `pipeline-hook`.

# Implementação

1. **Resolução Recursiva de Ancestrais (`internal/config/config.go`):**
   - Criada a função `resolveSymlinkAncestors(p string) string`, que sobe recursivamente a árvore de diretórios (`filepath.Split`) até encontrar o ancestral mais próximo existente no filesystem onde `filepath.EvalSymlinks` tenha sucesso, reconstruindo em seguida os segmentos finais inexistentes com `filepath.Join`.
   - Atualizado `Config.relToRoot(absPath string)` para normalizar tanto `c.root` quanto `filepath.Clean(absPath)` via `resolveSymlinkAncestors`, garantindo que ambos compartilhem o mesmo prefixo físico canônico independentemente de quantos níveis de subdiretórios ainda não existam em disco.
2. **Testes Unitários e Adversariais (`internal/config/config_test.go`):**
   - Adicionados subtestes em `TestIsProtectedSymlinks` validando múltiplos níveis de subdiretórios inexistentes abaixo de diretório symlinkado, raiz de projeto acessada via symlink (`symlinked_root/internal/sub1/sub2/novo.go` e `symlinked_root/.agents/hooks/sub/novo.sh`), e bloqueio de escape por symlink externo (`SEC-02`).
3. **Artigos Técnicos (`docs-ai/artigos/`):**
   - Atualização dos arquivos HTML, Markdown (PT-BR e EN/Dev.to) e post de LinkedIn sobre DORA Metrics e Telemetria de IA.

**Arquivos alterados:**
- `internal/config/config.go` — Adição de `resolveSymlinkAncestors` e uso em `Config.relToRoot`
- `internal/config/config_test.go` — Novos casos em `TestIsProtectedSymlinks` para `SEC-01` e `SEC-02`
- `docs-ai/artigos/artigo-dora-metrics-telemetria-ia.html` — Atualização do artigo DORA Metrics em HTML
- `docs-ai/artigos/artigo-dora-metrics-telemetria-ia.md` — Atualização do artigo DORA Metrics em Markdown (PT-BR)
- `docs-ai/artigos/article-dora-metrics-ai-agents-devto.md` — Atualização do artigo DORA Metrics para Dev.to (EN)
- `docs-ai/artigos/post-feed-linkedin-dora-metrics.txt` — Atualização do texto para feed do LinkedIn
- `docs-ai/deliveries/DEL-027/report.md` — Atualização de status de `draft` para `merged`
- `docs-ai/deliveries/INDEX.md` — Atualização de `DEL-027` para `merged` e registro de `DEL-028`
- `docs-ai/deliveries/DEL-028/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia, exit code 0)

$ go build ./...
(saída vazia, exit code 0)

$ go vet ./...
(saída vazia, exit code 0)

$ go test -v ./internal/config ./cmd/pipeline-hook
=== RUN   TestLoadAndHelpers
--- PASS: TestLoadAndHelpers (0.00s)
=== RUN   TestIsProtectedSymlinks
--- PASS: TestIsProtectedSymlinks (0.00s)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.200s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.625s

$ go test -race -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	7.142s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	1.094s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	3.612s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.541s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.612s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.954s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	1.320s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	3.710s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.688s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.790s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.580s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	14.920s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	2.740s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	7.890s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	0.610s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.510s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	1.680s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Resolução recursiva de symlinks ancestrais para caminhos com múltiplos diretórios inexistentes (`SEC-01`) | `TestIsProtectedSymlinks` em `internal/config/config_test.go` |
| Resolução quando a própria raiz do projeto contém symlink ancestral como `/var -> /private/var` no macOS (`SEC-01`) | `TestIsProtectedSymlinks` em `internal/config/config_test.go` e `TestActiveDeliveryGate` em `cmd/pipeline-hook/main_test.go` |
| Bloqueio de escape para fora da raiz via symlink (`SEC-02`) | `TestIsProtectedSymlinks` em `internal/config/config_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** nenhuma

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- `internal/config/config.go` (`Config.relToRoot` e `resolveSymlinkAncestors`) — Cálculo de caminho relativo canônico utilizado por `IsProtected`, `IsImmutable` e `IsSecuritySurface`.

**Validação dos requisitos de segurança (`REQUISITOS_SEGURANCA`):**
- `SEC-01` (Prevenção de bypass de proteção em subdiretórios inexistentes sob symlink ou raiz symlinkada): Aprovado. `resolveSymlinkAncestors` garante que qualquer número de níveis de diretórios ainda não criados em disco seja reconstruído sobre o ancestral real resolvido por `filepath.EvalSymlinks`, impedindo falsos negativos em `IsProtected` e `IsImmutable`.
- `SEC-02` (Prevenção de path traversal via symlink externo): Aprovado. Symlinks dentro do workspace que apontam para fora de `c.root` resolvem para seu destino externo real e produzem prefixo `..` em `filepath.Rel`, retornando `("", false)` com terminação garantida no nó raiz do filesystem (`parent == cur`).

# Security Review

Todos os requisitos `SEC-01` e `SEC-02` definidos no `architecture-guardrails` foram auditados no diff e validados por testes automatizados em `internal/config/config_test.go` e `cmd/pipeline-hook/main_test.go`, sem achados residuais.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
