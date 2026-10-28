# pipeline-ai

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](go.mod)
[![CI](https://github.com/rodrigodosanjosoliveira/pipeline-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/rodrigodosanjosoliveira/pipeline-ai/actions/workflows/ci.yml)

Toolkit e padrão reutilizável para projetos de desenvolvimento assistido por agentes de IA, integrando nativamente Go no backend e o ecossistema **Antigravity**.

A ferramenta central é a CLI unificada **`pipeline-ai`**, compilada estaticamente em Go puro sem dependências externas:

| Comando | Função |
|---|---|
| `pipeline-ai init` | Utilitário de *scaffolding* que inicializa novos projetos completos em 1 comando com templates embutidos. |
| `pipeline-ai run` | Pipeline Runner determinístico que executa em lote `doctor` → `sensor` → `validate` com auto-detecção de delivery e telemetria (D-27). |
| `pipeline-ai plugin` | Validador e instalador oficial do plugin do Antigravity (`verify`, `install`) (DEL-023). |
| `pipeline-ai dashboard` | Gera dashboard HTML interativo e responsivo (Generative UI) com métricas DORA, esteira e entregas (DEL-022). |
| `pipeline-ai metrics` | Motor de telemetria e observabilidade com cálculo determinístico das quatro métricas DORA (`summary`, `list`, `-json`) (DEL-020). |
| `pipeline-ai handoff` | Parser e validador automatizado de contratos de transição entre skills (`validate`) com salvaguarda de Fuzzing (C12). |
| `pipeline-ai hook` | Hook `PreToolUse` que impede edição direta em caminhos protegidos fora dos agents/skills autorizados (D-11). |
| `pipeline-ai validate` | Valida o relatório de delivery antes do commit e abertura de PR, garantindo salvaguarda estrita de Git (D-14). Suporta flag `-json`. |
| `pipeline-ai sync` | Sincroniza e atualiza skills (`.agents/skills`), regras e hooks em projetos existentes sem tocar em código de negócio. |
| `pipeline-ai delivery` | Gerencia o ciclo da delivery ativa (`open`, `close`, `status`), com suporte a TTL e governança temporal (D-21). |
| `pipeline-ai sensor` | Executa sensores de qualidade (`fmt`, `vet`, `test`) com saída estruturada e salvaguarda contra falsos positivos (suporta `-json`) (D-24). |
| `pipeline-ai module` | Gerencia módulos opcionais (`list`, `enable`, `disable`) dinamicamente no `.pipeline-ai.json` (D-25). |
| `pipeline-ai doctor` | Diagnostica a integridade do ambiente local (Go 1.24+, git, gh, pre-push hook e `.pipeline-ai.json`). Suporta `--json` para automações e CI. |
| `pipeline-ai mcp` | Servidor MCP nativo (JSON-RPC 2.0 sobre stdio) expondo ferramentas de governança para Antigravity e IDEs. |

*(Os binários individuais `pipeline-hook`, `pipeline-validate` e `pipeline-init` permanecem disponíveis para retrocompatibilidade com diretivas `tool` existentes).*

---

## Fluxo de Execução

![Fluxo de Execução do Pipeline AI](docs-ai/pipeline-ai.png)

---

## Estado

- Compilado e testado com **100% de sucesso em Go 1.24+** (stdlib pura, compatível com detector de concorrência `-race`).
- Integrado nativamente ao ecossistema **Antigravity** via `.agents/hooks.json`, regras e **16 skills modulares**.
- Automação via **`Taskfile.yml`** e workflow completo no **GitHub Actions** (`.github/workflows/ci.yml`).
- Versão estável atual: **`v0.7.0`**.

---

## Como Criar um Novo Projeto (1 Comando)

Crie o diretório do novo projeto e execute o `pipeline-ai init` (ou `pipeline-init`) via `go run`:

```bash
mkdir ~/workspace/meu-novo-app && cd ~/workspace/meu-novo-app

go run github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai@latest init
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
go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai@v0.6.0
```

*(Os comandos legados `cmd/pipeline-hook@v0.6.0` e `cmd/pipeline-validate@v0.6.0` também permanecem disponíveis).*

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

Ou com saída estruturada JSON para automações, bots e esteiras de CI:

```bash
go tool pipeline-validate -json docs-ai/deliveries/DEL-001/report.md
```

- **Exit code 0:** Aprovado.
- **Exit code 1:** Reprovado (bloqueia o commit/PR caso haja marcadores `TODO`, checklist incompleto ou salvaguarda de segurança violada).

---

## Gestão de Entregas, Sensores e Módulos (v0.6.0)

A CLI unificada expande os recursos de governança autônoma e determinística:

### Ciclo de Delivery (`pipeline-ai delivery`)
Controla o estado e ciclo temporal das entregas (D-21):
```bash
pipeline-ai delivery open DEL-015 TASK-015 "Release v0.6.0" 168h  # Abre delivery com TTL de 7 dias
pipeline-ai delivery status                                      # Exibe status e tempo restante
pipeline-ai delivery close                                       # Fecha a delivery ativa
```

### Sensores de Qualidade (`pipeline-ai sensor`)
Executa a validação de formato, vet e testes com proteção contra falsos positivos e saída estruturada (D-24):
```bash
pipeline-ai sensor          # Execução padrão em formato legível
pipeline-ai sensor -json    # Saída estruturada em JSON para integrações
```

### Gestão Dinâmica de Módulos (`pipeline-ai module`)
Permite inspecionar e alterar módulos opcionais em `.pipeline-ai.json` de forma segura (D-25):
```bash
pipeline-ai module list                # Lista módulos opcionais e status de ativação
pipeline-ai module enable release-ops  # Ativa módulo opcional com validação de esquema
pipeline-ai module disable release-ops # Desativa módulo opcional
```

---

## Pipeline Runner Determinístico (`pipeline-ai run`)

Consolida a execução em lote da esteira de validação em um único comando de alta performance:

```bash
pipeline-ai run
```

O Runner executa automaticamente a cascata em 3 estágios:
1. **`Doctor`:** Valida runtime (Go 1.24+), integridade de hooks, imutabilidade (`immutable_paths`) e configuração.
2. **`Sensor`:** Executa a toolchain (`gofmt`, `go vet`, `go test ./...`, injeção automática de ambiente sem CGO para projetos desktop).
3. **`Validate`:** Localiza a delivery ativa (ou via flag `-delivery` / `-report`) e audita o `report.md` contra o `git diff` real.

Suporta a flag `-json` para emissão do payload estruturado com duração e status de cada estágio.

---

## Plugin Oficial do Google Antigravity (`pipeline-ai plugin`)

O `pipeline-ai` é distribuído oficialmente como um plugin nativo do Antigravity (`plugin/`), agrupando em um único bundle: as 16 skills especializadas, as regras de orquestração (`rules/AGENTS.md`), os hooks de ciclo de vida (`hooks.json`) e o servidor MCP:

### Instalação em 1 Comando no Antigravity:
```bash
# Instala o plugin oficial na configuração do usuário (~/.gemini/config/plugins/pipeline-ai):
pipeline-ai plugin install

# Ou especificando origem e destino customizados:
pipeline-ai plugin install -src plugin -target ~/.gemini/config/plugins/pipeline-ai
```

### Validação do Bundle:
```bash
# Valida a integridade do manifesto plugin.json e a presença de todas as skills e configs:
pipeline-ai plugin verify -dir plugin
```

---

## Métricas DORA & Telemetria de Agentes (`pipeline-ai metrics`)

Para medir com precisão a performance e estabilidade de múltiplos agentes de IA sem depender de SaaS externo ou bibliotecas de terceiros, o `pipeline-ai` integra um motor de telemetria determinístico em Go puro (DEL-020).

As quatro métricas DORA são calculadas e adaptadas para velocidade de máquina:

| Métrica DORA | Significado para Agentes de IA |
|---|---|
| **Deployment Frequency (DF)** | Taxa de micro-entregas diárias concluídas com sucesso. |
| **Lead Time for Changes (LT)** | Tempo médio e mediana decorrido da abertura da issue no `pm-triage` ao PR validado no `git-delivery` (medido em minutos). |
| **Change Failure Rate (CFR)** | Percentual de falhas e bloqueios nos gates determinísticos da IA (*doctor*, *sensor*, *validate*). |
| **Time to Restore Service (MTTR)** | Tempo médio para a esteira se autorrecuperar de um bloqueio técnico (`STATUS: bloqueado`). |

### Uso na CLI:

```bash
# Resumo legível das 4 métricas DORA:
pipeline-ai metrics summary

# Emissão em JSON estruturado para integrações ou dashboards:
pipeline-ai metrics summary -json

# Lista detalhada de entregas e durações por estágio:
pipeline-ai metrics list
```

Persistência atômica e fail-safe em `.pipeline-state/metrics.json`, com parser blindado por **Fuzz Testing contínuo** (mais de 1.1M de iterações sem panics).

---

## Validação de Contratos de Handoff (`pipeline-ai handoff`)

Garante a integridade das passagens de bastão entre skills no modelo autônomo (Princípio da Não-Interrupção), evitando alucinações e perda de contexto:

```bash
# Valida um arquivo de handoff contract:
pipeline-ai handoff validate handoff.txt

# Validação com saída estruturada JSON:
pipeline-ai handoff validate -json handoff.txt
```

---

## Servidor MCP Nativo (`pipeline-ai mcp`)

O `pipeline-ai` inclui um servidor compatível com a especificação **Model Context Protocol (MCP)** sobre transporte `stdio` (JSON-RPC 2.0). Ele permite que IDEs e assistentes (como o **Antigravity**) invoquem ferramentas de governança diretamente:

```json
{
  "mcpServers": {
    "pipeline-ai": {
      "command": "pipeline-ai",
      "args": ["mcp"]
    }
  }
}
```

### Recursos MCP (Resources)
- **`pipeline://invariants`**: Fornece os invariantes de arquitetura e governança do projeto.
- **`pipeline://active-delivery`**: Retorna o status da entrega ativa e tabela do `INDEX.md`.
- **`pipeline://catalog`**: Lista de skills nativas e módulos disponíveis.
- **`pipeline://delivery/active`**: Metadados, status e expiração da delivery atualmente ativa em JSON (D-21).

### Ferramentas MCP (Tools)
- **`pipeline_doctor`**: Executa diagnóstico completo de ambiente e configuração (`dir`).
- **`pipeline_validate`**: Valida relatórios de entrega (`report_path`, `no_git`, `base_ref`) e salvaguardas de Git.
- **`pipeline_run_pipeline`**: Executa a esteira determinística completa em lote (`doctor` → `sensor` → `validate`) com telemetria automática (`delivery_id`, `report_path`, `dir`).
- **`pipeline_get_dashboard_html`**: Gera o dashboard HTML interativo, autocontido e responsivo com métricas DORA, esteira e histórico (`dir`).
- **`pipeline_get_metrics`**: Retorna as métricas DORA e o histórico de telemetria dos agentes em formato JSON (`dir`, `window_days`).
- **`pipeline_validate_handoff`**: Valida sintaxe e completude de blocos de Handoff Contract (`contract`, `file`).
- **`pipeline_scaffold_delivery`**: Cria o esqueleto da próxima entrega (`title`, `type`, `task_code`), gera `report.md` com seções obrigatórias e registra no `INDEX.md`.
- **`pipeline_sync_skills`**: Sincroniza e atualiza `.agents/skills` e regras a partir dos templates embutidos (`dir`).
- **`pipeline_close_delivery`**: Encerra a delivery ativa no repositório.
- **`pipeline_run_sensors`**: Executa sensores de qualidade (`fmt`, `vet`, `test`) com saída estruturada e salvaguarda contra falsos positivos (`dir`).
- **`pipeline_manage_modules`**: Gerencia módulos opcionais (`action`: list, enable, disable; `module`, `dir`).

---

## Como Contribuir

Contribuições são muito bem-vindas! Se você deseja contribuir com novas funcionalidades, melhorias nas skills ou correções de bugs:

1. Consulte o [Guia de Contribuição](CONTRIBUTING.md) para detalhes sobre invariantes de arquitetura, fluxo de branch e estilo de código.
2. Abra uma [Issue](https://github.com/rodrigodosanjosoliveira/pipeline-ai/issues) para discutir propostas antes de submeter grandes alterações.
3. Certifique-se de que `task check` passa com sucesso antes de abrir o seu Pull Request.

---

## Licença

Este projeto é distribuído sob os termos da licença **MIT**. Consulte o arquivo [LICENSE](LICENSE) para mais detalhes.
