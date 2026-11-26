# Changelog

Todas as mudanças notáveis neste projeto são documentadas neste arquivo, seguindo o padrão [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/) e [Semantic Versioning](https://semver.org/lang/pt-BR/).

## [0.12.0] - 2026-11-26

### Adicionado
- **Bounded Output, Smart Test Filtering & Partial Results (`DEL-037`, D-39):** Teto padrão de 8 KB por sensor (`-max-output-bytes`, `-verbose-output`), supressão de blocos `--- PASS`/`--- SKIP` em testes aprovados, preservação integral de `--- FAIL` e metadados explícitos (`truncated`, `total_bytes`, `omitted_bytes`, `failed_tests`) na CLI e MCP (`pipeline_run_sensors`, `pipeline_run_pipeline`).
- **Agent Mode, Detecção de TTY & Cobertura JSON Integral (`DEL-038`, D-40):** Flags globais `-agent` / `-no-agent` e variáveis `PIPELINE_AGENT_MODE` / `AI_AGENT`, detecção de terminal via `isTTY` (`os.ModeCharDevice`), hardening não-interativo no `init` (`-non-interactive`, `-json`) e saída JSON estruturada em 100% dos subcomandos (`version`, `delivery status`, `module list`, `plugin verify`, `plugin install`, `sync`, `init`).
- **Gemini 4 Hydrogen Routing & Native Semantic Provider (`DEL-039`, D-41):** Matriz `model_routing` em `.pipeline-ai.json` (`DefaultModelRouting` e `ResolveSkillModel`) exposta em `pipeline://catalog`, e provider nativo `"gemini"` em `internal/sensor/semantic`, `pipeline-ai eval` e `pipeline_eval_semantic` usando `generateContent`, `responseSchema`, `thinkingConfig.thinkingLevel` e header `x-goog-api-key` em 100% Go stdlib.
- **Discovery & Schema Introspection (`DEL-040`, D-42):** Subcomando `pipeline-ai schema [comando]` e flag global `-schema` / `--schema` com catálogo declarativo (`cmd/pipeline-ai/schema.go`) de flags globais, variáveis de ambiente e todos os 17 comandos sem efeitos colaterais.
- **Helpful Errors & Guided Recovery (`DEL-041`, D-43):** Sugestão automática de comandos por distância de Levenshtein (`suggestCommand`), respostas estruturadas (`cause`, `hint`, `fix_command`) na CLI (`writeCLIError`) e no servidor MCP (`sendToolError`).

### Documentação
- **Ajustes no 5º Artigo Técnico (`DEL-036`):** Ajustes de formatação e legibilidade no pacote do 5º artigo técnico (`post-feed-linkedin-yagni.txt`).

## [0.11.0] - 2026-11-09

### Adicionado
- **Decision Ladder Anti-Overengineering (`DEL-034`):** Escada de decisão de 5 degraus (`1. YAGNI → 2. Reuso no repo → 3. Stdlib → 4. Dependência já instalada → 5. Código novo mínimo`) incorporada nas skills `architecture-guardrails`, `dev-implementation` e `code-reviewer` (em `.agents/skills/`, templates embutidos de `scaffold` e `plugin/skills/`).
- **Debt Ledger Determinístico (`internal/debt` — `DEL-034`):** Novo pacote em Go 1.24 puro (`ParseComment` e `ScanDir`) para rastrear atalhos deliberados de engenharia anotados com `// shortcut: <limite> | upgrade: <gatilho>`, blindado por `FuzzParseComment`.
- **Subcomando CLI `pipeline-ai debt` e Tool MCP `pipeline_list_debt` (`DEL-034`):** Listagem determinística do inventário de atalhos deliberados em formato legível ou JSON (`-json`), ignorando arquivos `*_test.go` e diretórios ocultos/`vendor`/`node_modules`.
- **Preset Semântico YAGNI (`pipeline-ai eval -preset yagni` — `DEL-034`):** Função `semantic.ResolvePreset` (`yagni`, `minimalism`, `ponytail`) integrada à CLI (`-preset`) e à tool MCP `pipeline_eval_semantic` (`preset`).
- **Seção Debt Ledger e Painel Jev no Dashboard Generative UI (`DEL-032`, `DEL-034`):** Painel de telemetria semântica Jev (`SemanticSummary`) na Aba 1 e seção "Atalhos Deliberados (Debt Ledger)" no HTML e Markdown do `pipeline-ai dashboard`.
- **Integração Automática de `EventGate`, `delivery close` e MTTR Inferido nas Métricas DORA (`DEL-033`):** Síntese automática de métricas DORA a partir de eventos do `pipeline-ai run`, registro automático em `pipeline-ai delivery close` (`RecordDeliveryClose`) e cálculo automático de `TimeToRestore` (MTTR) em transições `EventGate(fail → pass)`.
- **5º Artigo Técnico Oficial (`DEL-035`):** Pacote completo em `docs-ai/artigos/artigo-governanca-anti-overengineering-yagni.*` (PT-BR Markdown, LinkedIn HTML sem tabelas, Dev.to EN e post de feed).

## [0.10.0] - 2026-11-05

### Adicionado
- Primitivas semânticas calibradas `EvalChoice` (`/v1/choice` com trava anti-alucinação `SEC-01`) e `EvalScore` (`/v1/score` com dupla trava `SEC-02`) em `internal/sensor/semantic` (`DEL-029`).
- Telemetria semântica (`SemanticEvalEvent` e `SemanticSummary`) acoplada às métricas DORA e 4º artigo técnico oficial (`DEL-029`, `DEL-030`).

### Corrigido
- Resolução recursiva de symlinks ancestrais no macOS (`resolveSymlinkAncestors`) em `internal/config` (`DEL-028`).
