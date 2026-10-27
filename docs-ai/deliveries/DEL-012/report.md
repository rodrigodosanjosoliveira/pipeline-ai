---
delivery_id: DEL-012
task_code: TASK-012
title: Suporte a relatórios customizados, prefixos de entrega flexíveis e índice opcional
status: merged
require_security: false
require_migration: false
require_performance: false
---

# Contexto

O contraditório técnico do framework identificou duas restrições de formato que geravam fricção na adoção do pipeline-ai em projetos corporativos reais (como o `ris-backend` e o `GuardIA`):
1. **Esquema de Relatório Fixo (C10):** As seções do `report.md` eram fixas no binário (`internal/report/report.go`). Projetos que exigiam seções de governança de domínio específico (como `Tenant Isolation`, `Frontend Handoff`, `Checkpoints` ou `Plano de Incrementos`) não tinham como declarar sua obrigatoriedade nem serem validados pelo `pipeline-validate`.
2. **Identificador e Índice Engessados (C11):** O validador exigia incondicionalmente o formato `DEL-NNN` e a presença do registro no arquivo `INDEX.md`. Equipes que utilizam identificadores corporativos diretos (como `Wrike`, `Jira`, `RIS-1234`, `PLAT-004`) ou que não mantêm um índice centralizado enfrentavam falhas artificiais de validação.

Esta entrega introduz a flexibilização declarativa dessas regras no `.pipeline-ai.json`: suporte a regexes customizadas para IDs de entrega e tarefas (`delivery_id_pattern` e `task_code_pattern`), declaração de seções obrigatórias adicionais (`custom_sections`) e desativação opcional da exigência do `INDEX.md` (`require_index: false`), mantendo 100% de compatibilidade com os padrões existentes.

# Escopo

- Configuração declarativa em `internal/config/config.go`:
  - `delivery_id_pattern`: expressão regular para formato de ID de entrega (padrão: `^DEL-[0-9]{3,}$`).
  - `task_code_pattern`: expressão regular para código de tarefa (padrão: `^[A-Z0-9]+-[0-9]+$`).
  - `custom_sections`: lista de títulos de seções obrigatórias adicionais exigidas pelo projeto.
  - `require_index`: booleano para definir se o arquivo `INDEX.md` é obrigatório (padrão: `true`).
- Validação e compilação de regexes no carregamento do `.pipeline-ai.json`.
- Método `ValidateWithConfig(cfg *config.Config)` em `internal/report/report.go` validando IDs e preenchimento de seções customizadas.
- Adaptação do validador em `internal/validate/validate.go` para consultar `IsIndexRequired()` e executar `ValidateWithConfig`.
- Tolerância à ausência de `INDEX.md` em `internal/mcp/scaffold.go` quando `require_index: false`.

**Fora de escopo:**
- Sensores estruturados em JSON e autoteste negativo (C13/C15), escopo da próxima etapa.
- Dependências externas de terceiros (100% Go stdlib pura).

# Implementação

1. **`internal/config/config.go`:**
   - Adicionados os campos `DeliveryIDPattern`, `TaskCodePattern`, `CustomSections` e `RequireIndex` à struct `Config`.
   - Adicionada validação de sintaxe das regexes durante o método `validate()`.
   - Implementados os métodos auxiliares `IsIndexRequired()`, `ValidateDeliveryID()` e `ValidateTaskCode()`.
2. **`internal/report/report.go`:**
   - Implementado o método `ValidateWithConfig(cfg *config.Config)`: invoca as checagens padrão de `Validate()`, valida `delivery_id` e `task_code` contra as expressões regulares configuradas e percorre `cfg.CustomSections` garantindo que cada seção exista no mapa normalizado e possua conteúdo não-vazio.
3. **`internal/validate/validate.go`:**
   - Atualizado o método `Run` para chamar `ValidateWithConfig(cfg)`.
   - Ajustada a função `checkIndex` para retornar imediatamente sem achados caso `cfg.IsIndexRequired() == false`.
4. **`internal/mcp/scaffold.go`:**
   - Atualizado `ScaffoldDelivery` para calcular o sequencial de entrega a partir da leitura dos diretórios existentes em `delivDir` caso `require_index: false` e o arquivo `INDEX.md` não exista.

**Arquivos alterados:**
- `internal/config/config.go` — Campos de configuração de padrões e seções customizadas
- `internal/config/config_test.go` — Testes de carregamento, validação e fuzzing de padrões
- `internal/report/report.go` — Implementação do método ValidateWithConfig
- `internal/report/report_test.go` — Testes de validação de IDs e seções customizadas
- `internal/validate/validate.go` — Integração de ValidateWithConfig e respeito a require_index
- `internal/validate/validate_test.go` — Testes de validação condicional com require_index false/true
- `internal/mcp/scaffold.go` — Suporte a scaffolding sem INDEX.md quando não obrigatório
- `internal/mcp/server_test.go` — Testes de scaffolding sem arquivo de índice
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-012
- `docs-ai/deliveries/DEL-012/report.md` — Relatório de entrega da delivery

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.284s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.164s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.199s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.225s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.077s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.206s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.041s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.712s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.140s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.510s

$ go test -fuzz=FuzzValidateDeliveryID -fuzztime=5s ./internal/config
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	6.310s

$ go test -fuzz=FuzzValidateWithConfig -fuzztime=5s ./internal/report
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	6.296s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Carregamento e validação de padrões flexíveis | `TestLoadFlexibleDeliveryConfig`, `TestConfigPatternValidation`, `TestIsIndexRequired`, `TestValidateDeliveryID`, `TestValidateTaskCode` e `FuzzValidateDeliveryID` em `internal/config/config_test.go` |
| Validação de relatório com regras customizadas | `TestValidateWithConfig` e `FuzzValidateWithConfig` em `internal/report/report_test.go` |
| Validação condicional do índice | `TestValidateRequireIndex` em `internal/validate/validate_test.go` |
| MCP Scaffold sem índice | `TestScaffoldDeliveryWithoutIndex` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Estratégia de fallback no MCP Scaffold escaneando pastas de entrega pré-existentes caso `require_index: false`.

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
