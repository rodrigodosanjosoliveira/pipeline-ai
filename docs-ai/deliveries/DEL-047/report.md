---
delivery_id: DEL-047
task_code: TASK-047
title: "Workflow de Deploy do Guia Interativo no GitHub Pages"
status: draft
require_security: true
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

O Guia Interativo Step-by-Step do `pipeline-ai` (`docs/interactive/index.html`, entregue na `DEL-046`) estava acessível apenas localmente a partir do repositório. Para disponibilizá-lo publicamente com atualização contínua a cada alteração em `docs/interactive/**`, é necessário publicar o diretório via GitHub Pages através de um workflow dedicado no GitHub Actions com superfície de segurança estritamente controlada.

Esta delivery cria o workflow `.github/workflows/deploy-pages.yml`, atualiza o `README.md` com o link público do GitHub Pages ao lado do link local e sincroniza o histórico de deliveries (`DEL-046` para `merged` e registro de `DEL-047`).

# Escopo

- **CI/CD — GitHub Pages (`.github/workflows/deploy-pages.yml`):**
  - Gatilhos em `push` para `main` restritos aos caminhos `docs/interactive/**` e `.github/workflows/deploy-pages.yml`, além de `workflow_dispatch:`.
  - Permissões mínimas explícitas: `contents: read`, `pages: write`, `id-token: write`.
  - Controle de concorrência: `group: "pages"` e `cancel-in-progress: false`.
  - Uso exclusivo das actions oficiais: `actions/checkout@v4`, `actions/configure-pages@v5`, `actions/upload-pages-artifact@v3` (`path: "docs/interactive"`) e `actions/deploy-pages@v4`.
  - Sem `pull_request_target`, sem segredos e sem scripts inline.
- **Documentação e Governança:**
  - `README.md`: Adição do link público `https://rodrigodosanjosoliveira.github.io/pipeline-ai/` ao lado do link local `docs/interactive/index.html`.
  - `docs-ai/deliveries/DEL-046/report.md`: Atualização de `status: draft` para `status: merged`.
  - `docs-ai/deliveries/INDEX.md`: Atualização de `DEL-046` para `merged` e registro de `DEL-047`.
  - `docs-ai/deliveries/DEL-047/report.md`: Relatório completo desta delivery com `require_security: true`.

**Fora de escopo:**
- Alteração de código Go em `internal/` ou `cmd/`.
- Alteração do conteúdo HTML/CSS/JS de `docs/interactive/index.html`.
- Alteração de `go.mod` ou `go.sum`.

# Implementação

1. **Workflow GitHub Pages (`.github/workflows/deploy-pages.yml`):**
   - Criado workflow declarativo mínimo com `permissions` restritas (`contents: read`, `pages: write`, `id-token: write`), `concurrency` configurada para evitar race conditions no deploy (`group: "pages"`, `cancel-in-progress: false`) e job único `deploy` sobre `ubuntu-latest` no ambiente `github-pages` utilizando apenas actions oficiais da organização `actions/*`.
2. **README (`README.md`):**
   - Atualizada a chamada do Guia Interativo Step-by-Step para incluir o link público do GitHub Pages (`https://rodrigodosanjosoliveira.github.io/pipeline-ai/`) junto ao link local `docs/interactive/index.html`.
3. **Sincronização de Deliveries (`docs-ai/deliveries/DEL-046/report.md`, `docs-ai/deliveries/INDEX.md`, `docs-ai/deliveries/DEL-047/report.md`):**
   - Fechamento do status de `DEL-046` para `merged` e registro completo da `DEL-047`.

**Arquivos alterados:**
- `.github/workflows/deploy-pages.yml` — Workflow de deploy automático do Guia Interativo no GitHub Pages com permissões mínimas e actions oficiais
- `README.md` — Adição do link público do GitHub Pages (`https://rodrigodosanjosoliveira.github.io/pipeline-ai/`) ao lado do link local `docs/interactive/index.html`
- `docs-ai/deliveries/DEL-046/report.md` — Atualização de `status: draft` para `status: merged`
- `docs-ai/deliveries/INDEX.md` — Atualização de `DEL-046` para `merged` e registro de `DEL-047`
- `docs-ai/deliveries/DEL-047/report.md` — Relatório completo desta delivery

# Testes

**Gates:**

Delivery estritamente de CI/documentação (sem alteração de código Go). Validada integralmente pelo `pipeline-validate`:

```text
$ go run ./cmd/pipeline-validate docs-ai/deliveries/DEL-047/report.md
(validado com status APROVADO)
```

**Cobertura do escopo:**

| Item do escopo | Verificação |
|---|---|
| `.github/workflows/deploy-pages.yml` com permissões mínimas e actions oficiais | Revisão estrutural e validação `SEC-CI-01` |
| Link público do GitHub Pages no `README.md` | Verificado em `README.md` |
| Sincronização de `DEL-046` e registro de `DEL-047` no `INDEX.md` | `go run ./cmd/pipeline-validate docs-ai/deliveries/DEL-047/report.md` |

# Security

- **SEC-CI-01 (Superfície de CI/CD — `.github/workflows/deploy-pages.yml`):**
  - `.github/workflows/` pertence a `security_surface_paths`, portanto `require_security: true` está habilitado.
  - **Princípio do menor privilégio:** Bloco `permissions` declarado explicitamente no nível do workflow com apenas `contents: read`, `pages: write` e `id-token: write`.
  - **Ausência de vetores de escalação de privilégio:** Não utiliza `pull_request_target`, não expõe segredos (`secrets.*`) e não executa scripts shell inline (`run:`).
  - **Cadeia de suprimentos de Actions:** Utiliza exclusivamente actions oficiais mantidas pelo GitHub (`actions/checkout@v4`, `actions/configure-pages@v5`, `actions/upload-pages-artifact@v3`, `actions/deploy-pages@v4`), publicando apenas o diretório estático `docs/interactive`.

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Permissões mínimas explícitas, concorrência `cancel-in-progress: false` para não interromper deploys em andamento, ausência de scripts inline e uso exclusivo de `actions/*`.

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
