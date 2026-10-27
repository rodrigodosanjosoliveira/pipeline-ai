---
delivery_id: DEL-013
task_code: TASK-013
title: Sensores com saída estruturada em JSON e autoteste negativo contra falsos positivos
status: merged
require_security: false
require_migration: false
require_performance: false
---

# Contexto

O contraditório técnico do framework identificou duas fragilidades críticas nos mecanismos de validação do pipeline:
1. **Falta de Sensores Estruturados com Saída Consumível por Máquina (C13):** As ferramentas de validação emitiam saídas textuais voltadas apenas para humanos, dificultando o parsing determinístico por agents de IA, scripts de CI/CD e clientes MCP.
2. **Fragilidade a Falsos Positivos e Ausência de Testes Negativos (C15):** Relatórios de entrega podiam ser aprovados mesmo que a seção `# Testes` contivesse saída vazia ou nenhum teste real tivesse sido executado (`[no test files]`). Além disso, comandos de teste que retornavam exit code 0 sem rodar nenhum teste induziam a falsos positivos.

Esta entrega implementa sensores com saída estruturada em JSON (`pipeline-ai sensor -json`, `pipeline-validate -json`, tool MCP `pipeline_run_sensors`) e salvaguardas explícitas contra falsos positivos (autoteste negativo que reprova execuções com 0 testes e validação de evidência real de testes no `report.md`).

# Escopo

- Criação do pacote `internal/sensor`:
  - Execução determinística dos sensores `fmt` (`gofmt -l .`), `vet` (`go vet ./...`) e `test` (`go test -v ./...`).
  - Parsing de testes executados, passados, falhos e pulados.
  - Salvaguarda contra falsos positivos: reprovação mandatória (`Overall: FAIL`, `OK: false`) caso zero testes sejam executados (`tests_run == 0`), alertando explicitamente sobre falso positivo evitado.
  - Estruturas de dados `Result`, `CheckResult`, `SensorStatus`.
- Saída JSON no validador (`internal/validate`):
  - Struct `ValidationResult` contendo status, ok, report, contagem de erros, avisos e lista de achados.
  - Função `RunJSON` serializando resultado em formato JSON indentado.
- Suporte à flag `-json`:
  - `cmd/pipeline-validate`: aceita `-json` emitindo `ValidationResult`.
  - `cmd/pipeline-ai validate`: aceita `-json` emitindo `ValidationResult`.
  - `cmd/pipeline-ai sensor`: novo subcomando executando sensores com opções `-dir` e `-json`.
- Salvaguarda de evidência de testes no relatório (`internal/report`):
  - Função `checkTestEvidence` reprovando relatórios com seção `# Testes` desprovida de logs de execução reais (apenas comando sem saída, apenas `[no test files]` ou declarações genéricas).
- Nova ferramenta MCP:
  - `pipeline_run_sensors`: registrada em `tools/list` e tratada em `tools/call`, permitindo que agents executem a bateria de sensores com retorno JSON.

**Fora de escopo:**
- Alterações em regras de governança de entrega (DEL-014 / C16).
- Adição de dependências externas (100% Go 1.24+ standard library).

# Implementação

1. **`internal/sensor/sensor.go`:**
   - Implementada a função `Run(dir string) (*Result, error)`.
   - Localização resiliente dos binários `go` e `gofmt` via `findBinary()`.
   - Execução sequencial dos sensores `fmt`, `vet` e `test`.
   - Contabilização precisa via contagem de marcadores `--- PASS:`, `--- FAIL:`, `--- SKIP:`.
   - Verificação mandatória `testsRun == 0` forçando `StatusFail`.
2. **`internal/report/report.go`:**
   - Implementado `checkTestEvidence()` no método `Validate()`, exigindo evidências reais de execução de testes (`PASS`, `FAIL`, `ok  \t`, `=== RUN`, etc.) e rejeitando declarações vazias ou genéricas.
3. **`internal/validate/validate.go`:**
   - Adicionada a struct `ValidationResult` e implementada a função `RunJSON(...)`.
4. **`cmd/pipeline-validate/main.go` & `cmd/pipeline-ai/main.go`:**
   - Flag `-json` integrada em ambos os comandos de validação.
   - Novo subcomando `sensor` adicionado ao CLI `pipeline-ai`.
5. **`internal/mcp/server.go`:**
   - Tool `pipeline_run_sensors` declarada e vinculada à execução de `sensor.Run`.

**Arquivos alterados:**
- `internal/sensor/sensor.go` — Implementação do motor de sensores e salvaguarda negativa
- `internal/sensor/sensor_test.go` — Testes unitários do pacote sensor (sucesso, falhas e autoteste negativo)
- `internal/report/report.go` — Salvaguarda contra falso positivo na seção de testes
- `internal/report/report_test.go` — Testes da verificação de evidência de execução
- `internal/validate/validate.go` — Struct ValidationResult e função RunJSON
- `internal/validate/validate_test.go` — Testes de saída estruturada em JSON
- `cmd/pipeline-validate/main.go` — Flag -json
- `cmd/pipeline-validate/main_test.go` — Testes do pipeline-validate -json
- `cmd/pipeline-ai/main.go` — Subcomando sensor e flag -json no validate
- `cmd/pipeline-ai/main_test.go` — Testes do pipeline-ai validate -json e sensor -json
- `internal/mcp/server.go` — Tool pipeline_run_sensors
- `internal/mcp/server_test.go` — Testes da tool pipeline_run_sensors
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-013
- `docs-ai/deliveries/DEL-013/report.md` — Relatório de entrega da delivery

# Testes

**Gates:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso)

$ go vet ./...
(análise estática sem avisos)

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	2.511s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.113s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	2.871s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.204s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.066s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.367s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.028s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.504s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	4.726s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.135s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.790s
PASS
```

**Execução do Sensor CLI (`pipeline-ai sensor -json`):**

```json
{
  "overall": "PASS",
  "ok": true,
  "tests_run": 329,
  "sensors": [
    {
      "name": "fmt",
      "status": "PASS",
      "message": "Todos os arquivos estão formatados segundo o padrão gofmt."
    },
    {
      "name": "vet",
      "status": "PASS",
      "message": "go vet executado com sucesso sem avisos ou erros."
    },
    {
      "name": "test",
      "status": "PASS",
      "message": "Todos os testes passaram (329 testes executados, 0 pulados)."
    }
  ],
  "message": "Todos os sensores passaram com sucesso (329 testes executados)."
}
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Sensores com saída estruturada e autoteste negativo | `TestRunSensors_Success`, `TestRunSensors_ZeroTestsNegativeCheck`, `TestRunSensors_FmtFailure`, `TestRunSensors_VetFailure` em `internal/sensor/sensor_test.go` |
| Salvaguarda contra falsos positivos na seção `# Testes` | `TestSalvaguardaSecaoTestes_EvidenciaExecucao` em `internal/report/report_test.go` |
| Validação de relatório em formato JSON | `TestValidateJSONOutput` em `internal/validate/validate_test.go` |
| CLI `pipeline-validate -json` | `TestPipelineValidateJSON` em `cmd/pipeline-validate/main_test.go` |
| CLI `pipeline-ai validate -json` e `pipeline-ai sensor -json` | `TestCLIValidateJSON`, `TestCLISensorJSON` em `cmd/pipeline-ai/main_test.go` |
| Tool MCP `pipeline_run_sensors` | `TestMCPToolsRunSensors` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Resolução de binários `go` e `gofmt` com fallback para `/usr/local/go/bin` e `~/go/bin` para robustez em subshells e pipelines CI.

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
