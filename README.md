# pipeline-ai

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](go.mod)
[![CI](https://github.com/rodrigodosanjosoliveira/pipeline-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/rodrigodosanjosoliveira/pipeline-ai/actions/workflows/ci.yml)

Toolkit e padrão reutilizável para projetos de desenvolvimento assistido por agentes de IA, integrando nativamente Go no backend e o ecossistema **Antigravity**.

Três binários Go compilados estaticamente, sem dependências externas além da biblioteca padrão:

| Binário | Função |
|---|---|
| `pipeline-hook` | Hook `PreToolUse` que impede edição direta em caminhos protegidos fora dos agents/skills autorizados (D-11). |
| `pipeline-validate` | Valida o relatório de delivery antes do commit e abertura de PR, garantindo salvaguarda estrita de Git (D-14). |
| `pipeline-init` | Utilitário de *scaffolding* que inicializa novos projetos completos em 1 comando com templates embutidos. |

---

## Estado

- Compilado e testado com **100% de sucesso em Go 1.24+** (stdlib pura, compatível com detector de concorrência `-race`).
- Integrado nativamente ao ecossistema **Antigravity** via `.agents/hooks.json`, regras e **16 skills modulares**.
- Automação via **`Taskfile.yml`** e workflow completo no **GitHub Actions** (`.github/workflows/ci.yml`).
- Versão estável atual: **`v0.3.1`**.

---

## Como Criar um Novo Projeto (1 Comando)

Crie o diretório do novo projeto e execute o `pipeline-init` via `go run`:

```bash
mkdir ~/workspace/meu-novo-app && cd ~/workspace/meu-novo-app

go run github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init@latest
```

O utilitário guia você de forma interativa (ou via flags `-name`, `-type`, `-db`, `-module`, `-dir`) e entrega o projeto pronto:
- `git init` e hook `.githooks/pre-push` instalados (bloqueando pushes diretos na `main`).
- `go.mod` inicializado com diretivas `tool` apontando para a versão estável do `pipeline-ai`.
- `.agents/` configurado com `hooks.json`, regras e as 16 skills do padrão.
- `AGENTS.md` e `GEMINI.md` com placeholders substituídos.
- Estrutura completa de documentação e relatórios em `docs-ai/`.

Consulte o [Guia de Criação de Projetos](docs-ai/GUIA-CRIACAO-PROJETOS.md) para detalhes.

---

## Instalação em Projeto Existente

Requer Go 1.24 ou superior (a diretiva `tool` no `go.mod` é obrigatória):

```bash
go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook@v0.3.0
go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate@v0.3.0
```

*(Se o repositório for privado, configure `go env -w GOPRIVATE="github.com/rodrigodosanjosoliveira/*"` e autentique o Git via SSH)*.

---

## Catálogo de Skills Nativas (16 Skills)

O fluxo do Antigravity é orquestrado por skills especializadas em `.agents/skills/`:

### Núcleo Obrigatório (10)
| Skill | Responsabilidade | Roteamento |
|---|---|---|
| `pm-triage` | Ponto de entrada de toda demanda. Classifica escopo, atribui `DEL-NNN` e roteia. | `architecture-guardrails` |
| `architecture-guardrails` | Define a abordagem técnica, limites de implementação e bloqueia violação de invariantes. | `data-model` ou `test-writer` |
| `test-writer` | Escreve os testes antes da implementação (TDD) e alvos de fuzzing determinísticos. | `dev-implementation` |
| `dev-implementation` | Implementa o código estritamente dentro do escopo até os 4 gates passarem. | `code-reviewer` |
| `code-reviewer` | Revisa o diff contra os invariantes e escopo. Não possui permissão de edição direta. | `migration-reviewer` / `query-performance` / `security-review` / `qa-validation` |
| `security-review` | Audita superfícies de autenticação, criptografia, storage e CI quando tocadas. | `supply-chain-review` ou `qa-validation` |
| `supply-chain-review` | Audita alterações em `go.mod` e `go.sum` com `govulncheck` e `go mod verify`. | `qa-validation` |
| `qa-validation` | Valida o comportamento entregue contra o escopo inicial com suíte completa de testes. | `release-ops` ou `docs-update` |
| `docs-update` | Atualiza documentação afetada e monta o relatório em `docs-ai/deliveries/`. | `git-delivery` |
| `git-delivery` | Cria a branch `task/*`, o commit padronizado e abre o Pull Request (`gh pr create`). | Encerramento |

### Módulos Opcionais (6)
| Skill | Ativação | Responsabilidade |
|---|---|---|
| `release-ops` | `"release-ops"` | Versionamento SemVer, checklist de manifests e notas de release. |
| `data-model` | `"data-model"` | Modelagem de dados para PostgreSQL (SQL), MongoDB (BSON) ou DynamoDB (PK/SK). |
| `migration-reviewer` | `"migration-reviewer"` | Auditoria de scripts DDL em `migrations/` (idempotência, rollback, zero-downtime). |
| `query-performance` | `"query-performance"` | Análise de planos de execução (`EXPLAIN`), índices adequados e eliminação de N+1. |
| `context-loader` | `"context-loader"` | Carregamento seletivo de documentações externas e contratos de API sob demanda. |
| `agent-auditor` | `"agent-auditor"` | Auditoria de integridade contra alucinação, drift e scope creep. |

---

## Configuração (`.pipeline-ai.json`)

Exemplo de configuração para projeto desktop (`examples/desktop/.pipeline-ai.json`):

```json
{
  "version": 1,
  "project_type": "desktop",
  "protected_paths": [
    "internal/",
    "frontend/src/",
    "main.go",
    "go.mod",
    "go.sum",
    "Taskfile.yml"
  ],
  "editor_agents": [
    "dev-implementation",
    "test-writer",
    "docs-update"
  ],
  "enabled_modules": [
    "release-ops",
    "data-model"
  ],
  "security_surface_paths": [
    "internal/auth/",
    "internal/crypto/",
    "internal/storage/",
    "go.mod",
    "go.sum",
    "build/",
    ".github/workflows/"
  ],
  "deliveries_dir": "docs-ai/deliveries",
  "index_file": "docs-ai/deliveries/INDEX.md",
  "base_ref": "origin/main"
}
```

---

## Automação e Gates (`Taskfile.yml`)

O repositório inclui rotinas determinísticas:

```bash
task check      # Roda format check, go vet e suíte completa de testes
task test:race  # Executa testes com detector de concorrência ativo
task vuln       # Executa auditoria de vulnerabilidades com govulncheck
task build      # Compila todos os pacotes e binários
```

---

## Validação de Entregas (`pipeline-validate`)

Antes de abrir o Pull Request, toda entrega é validada pelo binário em Go:

```bash
go tool pipeline-validate docs-ai/deliveries/DEL-001/report.md
```

- **Exit code 0:** Aprovado.
- **Exit code 1:** Reprovado (bloqueia o commit/PR caso haja marcadores `TODO`, checklist incompleto ou salvaguarda de segurança violada).

---

## Como Contribuir

Contribuições são muito bem-vindas! Se você deseja contribuir com novas funcionalidades, melhorias nas skills ou correções de bugs:

1. Consulte o [Guia de Contribuição](CONTRIBUTING.md) para detalhes sobre invariantes de arquitetura, fluxo de branch e estilo de código.
2. Abra uma [Issue](https://github.com/rodrigodosanjosoliveira/pipeline-ai/issues) para discutir propostas antes de submeter grandes alterações.
3. Certifique-se de que `task check` passa com sucesso antes de abrir o seu Pull Request.

---

## Licença

Este projeto é distribuído sob os termos da licença **MIT**. Consulte o arquivo [LICENSE](LICENSE) para mais detalhes.
