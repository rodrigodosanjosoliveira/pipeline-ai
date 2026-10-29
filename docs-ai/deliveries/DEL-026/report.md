---
delivery_id: DEL-026
task_code: TASK-026
title: "Integração Operacional do Jev: Subcomando CLI eval e Ferramenta MCP pipeline_eval_semantic"
status: draft
require_security: true
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A entrega anterior (DEL-025) introduziu o pacote fundamental `internal/sensor/semantic`, permitindo inferências rápidas, calibradas e tipadas a partir de modelos "System One" (Jev) em Go puro (stdlib). No entanto, essas capacidades residiam apenas como biblioteca Go interna sem superfícies de invocação direta.

Para viabilizar o uso operacional tanto por desenvolvedores humanos quanto por agentes autônomos em esteiras de software (Antigravity, Claude Desktop, Cursor), fazia-se necessária a exposição dessas funcionalidades em duas frentes de primeira classe:
1. Uma interface de linha de comando (`pipeline-ai eval`), permitindo testar hipóteses, critérios de aceite ou conformidade semântica diretamente do terminal ou de scripts shell/CI;
2. Uma ferramenta no servidor MCP (`pipeline_eval_semantic`), permitindo que agentes de IA consultem o sensoriamento semântico de forma estruturada via JSON-RPC 2.0.

Ambas as superfícies devem aplicar rigorosamente salvaguardas operacionais de segurança (SEC-01 token redaction, SEC-02 mitigação de DoS com limite de 512KB em arquivos/strings/stdin, e SEC-03 timeout mandatório).

# Escopo

- Implementação do subcomando CLI `pipeline-ai eval` em `cmd/pipeline-ai/main.go`:
  - Flags de configuração: `-state` (arquivo ou string), `-question`, `-threshold` (default 0.8), `-json`, `-endpoint`, `-api-key`, `-timeout` (default 10s).
  - Suporte polimórfico a fontes de estado: arquivo no disco, string inline ou leitura transparente via `stdin`.
  - Saída legível e amigável por padrão e saída estruturada JSON com `-json`.
  - Salvaguardas operacionais: redação de segredos (SEC-01), limite de 512KB no estado (SEC-02) e timeout estrito configurável via context (SEC-03).
- Implementação da ferramenta MCP `pipeline_eval_semantic` em `internal/mcp/server.go`:
  - Registro no catálogo de ferramentas do servidor MCP (Tools List) com schema tipado (`state`, `question`, `threshold`, `endpoint`, `api_key`, `timeout_seconds`).
  - Manipulador de execução (Tool Call) integrado ao cliente semântico.
  - Comportamento fail-closed gracioso: se o sensor não estiver configurado, retorna mensagem instrutiva sem quebrar a sessão do agente.
  - Salvaguardas operacionais: redação de segredos (SEC-01), limite de 512KB no estado (SEC-02) e deadline de contexto (SEC-03).
- Cobertura de testes unitários e de integração com mock servers HTTP em `cmd/pipeline-ai/main_test.go` e `internal/mcp/server_test.go`.
- Atualização da documentação arquitetural e de uso:
  - Registro da Decisão D-34 na seção 12 de `docs-ai/DECISOES.md`.
  - Documentação da CLI `eval` e da ferramenta MCP `pipeline_eval_semantic` em `README.md`.
  - Registro de DEL-026 no índice central `docs-ai/deliveries/INDEX.md`.

**Fora de escopo:**
- Adição de bibliotecas externas ao `go.mod` (preservação de 100% Go standard library).
- Alteração nos contratos de domínio existentes em `internal/sensor/semantic`.
- Dependência de servidores Jev em nuvem para execução de testes (uso exclusivo de `net/http/httptest`).

# Implementação

1. **Subcomando CLI `eval` (`cmd/pipeline-ai/main.go`):**
   - Roteamento no `runWithStdin` e documentação de ajuda em `printUsage`.
   - Implementação de `runEval(args []string, stdin io.Reader, stdout, stderr io.Writer) int`:
     - Processamento de flags via `flag.FlagSet`.
     - Resolução polimórfica de `state`: inspeciona se aponta para arquivo existente no disco (com checagem de tamanho até 512KB); caso contrário, interpreta como string literal (validando tamanho máximo de 512KB); caso ausente, lê a partir de `stdin` limitado a 512KB via `io.LimitReader`.
     - Resolução de credenciais e endpoint via flags ou variáveis de ambiente de fallback (`PIPELINE_SEMANTIC_ENDPOINT`, `PIPELINE_SEMANTIC_API_KEY`, `JEV_ENDPOINT`, `JEV_API_KEY`).
     - Execução via `semantic.Client.EvalNoul` envolvida por `context.WithTimeout`.
     - Sanitização e redação de API keys em qualquer saída de erro (SEC-01).
     - Formatação legível no stdout (Decisão, Confiança, Latência, Razão) ou formatação JSON com `-json`.

2. **Ferramenta MCP `pipeline_eval_semantic` (`internal/mcp/server.go`):**
   - Declaração no schema de ferramentas com validação estrita de campos requeridos (`state`, `question`) e opcionais (`threshold`, `endpoint`, `api_key`, `timeout_seconds`).
   - Handler em `handleCallTool`:
     - Validação de presença e tipos dos parâmetros.
     - Validação de tamanho máximo de 512KB para o campo `state` (SEC-02).
     - Fallback seguro para variáveis de ambiente caso credenciais não sejam passadas no payload.
     - Detecção de sensor não configurado retornando resposta informativa com `isError: false` sem romper o fluxo do assistente.
     - Execução da avaliação com timeout via `context.WithTimeout` (SEC-03).
     - Redação de tokens em mensagens de erro MCP (SEC-01).

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Roteamento e implementação do subcomando CLI `eval`
- `cmd/pipeline-ai/main_test.go` — Testes unitários do subcomando `eval` e salvaguardas
- `internal/mcp/server.go` — Definição do schema e handler de `pipeline_eval_semantic`
- `internal/mcp/server_test.go` — Testes da ferramenta MCP, schema e salvaguardas
- `docs-ai/DECISOES.md` — Formalização da Decisão D-34
- `README.md` — Documentação do comando `eval` e da ferramenta MCP
- `docs-ai/deliveries/INDEX.md` — Registro da entrega DEL-026
- `docs-ai/deliveries/DEL-026/report.md` — Relatório formal de entrega

# Testes

**Gates de Qualidade:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso, exit code 0)

$ go vet ./...
(análise estática sem avisos, exit code 0)

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	3.454s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.136s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.536s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.257s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.029s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.106s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.008s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.147s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.106s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.245s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.037s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	5.325s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.934s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	2.924s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	0.280s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.156s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.592s
```

**Bateria de Testes Unitários e de Integração:**

```
$ go test -v ./cmd/pipeline-ai -run "TestCLIEval"
=== RUN   TestCLIEval
=== RUN   TestCLIEval/eval_sem_argumentos
=== RUN   TestCLIEval/eval_com_state_ou_question_vazios
=== RUN   TestCLIEval/eval_quando_sensor_nao_configurado
=== RUN   TestCLIEval/eval_com_mock_server_HTTP_sucesso_e_saida_formatada
=== RUN   TestCLIEval/eval_com_flag_-json
=== RUN   TestCLIEval/eval_com_-state_apontando_para_arquivo_no_disco
=== RUN   TestCLIEval/eval_seguranca_SEC-01_token_redaction_em_caso_de_erro
=== RUN   TestCLIEval/eval_seguranca_SEC-02_limitacao_de_tamanho_do_state_max_512KB
=== RUN   TestCLIEval/eval_seguranca_SEC-03_timeout_mandatorio_via_context
--- PASS: TestCLIEval (0.34s)
PASS

$ go test -v ./internal/mcp -run "TestMCPEvalSemanticTool"
=== RUN   TestMCPEvalSemanticTool
=== RUN   TestMCPEvalSemanticTool/tools/list_lista_pipeline_eval_semantic_com_schemas_validos
=== RUN   TestMCPEvalSemanticTool/tools/call_com_parametros_invalidos
=== RUN   TestMCPEvalSemanticTool/tools/call_quando_API_nao_configurada
=== RUN   TestMCPEvalSemanticTool/tools/call_com_mock_server_httptest_e_resposta_200
=== RUN   TestMCPEvalSemanticTool/tools/call_seguranca_SEC-01_token_redaction
=== RUN   TestMCPEvalSemanticTool/tools/call_seguranca_SEC-02_limite_maximo_de_state_512KB
--- PASS: TestMCPEvalSemanticTool (0.03s)
PASS
```

**Matriz de Rastreabilidade de Testes:**

| Item do Escopo | Teste Automatizado |
|---|---|
| Validação de flags e argumentos obrigatórios da CLI | `TestCLIEval/eval_sem_argumentos`, `TestCLIEval/eval_com_state_ou_question_vazios` |
| Fallback gracioso para sensor não configurado na CLI | `TestCLIEval/eval_quando_sensor_nao_configurado` |
| Execução com sucesso e formatação padrão na CLI | `TestCLIEval/eval_com_mock_server_HTTP_sucesso_e_saida_formatada` |
| Emissão de saída estruturada JSON na CLI | `TestCLIEval/eval_com_flag_-json` |
| Resolução de estado a partir de arquivo em disco na CLI | `TestCLIEval/eval_com_-state_apontando_para_arquivo_no_disco` |
| Redação de credenciais em erros na CLI (SEC-01) | `TestCLIEval/eval_seguranca_SEC-01_token_redaction_em_caso_de_erro` |
| Limite de 512KB no estado na CLI (SEC-02) | `TestCLIEval/eval_seguranca_SEC-02_limitacao_de_tamanho_do_state_max_512KB` |
| Timeout estrito com context na CLI (SEC-03) | `TestCLIEval/eval_seguranca_SEC-03_timeout_mandatorio_via_context` |
| Exposição correta de schema da ferramenta MCP | `TestMCPEvalSemanticTool/tools/list_lista_pipeline_eval_semantic_com_schemas_validos` |
| Validação de parâmetros inválidos no MCP | `TestMCPEvalSemanticTool/tools/call_com_parametros_invalidos` |
| Resposta informativa se API desconfigurada no MCP | `TestMCPEvalSemanticTool/tools/call_quando_API_nao_configurada` |
| Invocação com sucesso e retorno estruturado no MCP | `TestMCPEvalSemanticTool/tools/call_com_mock_server_httptest_e_resposta_200` |
| Redação de segredos em mensagens de erro do MCP (SEC-01) | `TestMCPEvalSemanticTool/tools/call_seguranca_SEC-01_token_redaction` |
| Limite de 512KB no estado no MCP (SEC-02) | `TestMCPEvalSemanticTool/tools/call_seguranca_SEC-02_limite_maximo_de_state_512KB` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:**
- Suporte polimórfico a `state` permitindo leitura direta de arquivo, string inline ou via pipe do `stdin`.
- Resolução e sanitização uniforme de variáveis de ambiente de compatibilidade (`PIPELINE_SEMANTIC_*` e `JEV_*`).
- Resposta não bloqueante no MCP quando o sensor não estiver configurado (`isError: false` com mensagem descritiva), orientando o agente sem interromper a orquestração da sessão.
- Preservação estrita da invariante de zero dependências externas no `go.mod`.

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- Entrada e validação de comandos CLI (`cmd/pipeline-ai/main.go`).
- Manipulação e exposição de ferramentas no servidor MCP (`internal/mcp/server.go`).
- Tratamento e trânsito de segredos de autenticação (API keys / tokens).
- Consumo de memória e processamento de arquivos e streams de entrada (`stdin`).

**Requisitos de Segurança e Mitigações:**
- **SEC-01 (Token Redaction):** Chaves de API e tokens informados via flags, variáveis de ambiente ou argumentos de ferramentas MCP são ativamente redatados (`[REDACTED]`) em todas as mensagens de erro, saídas de stderr e responses de erro do MCP antes de serem emitidos.
- **SEC-02 (Mitigação de DoS / Limite de 512KB):** O tamanho máximo de qualquer `state` avaliado (seja via arquivo em disco, string literal ou leitura de stream `stdin`) é limitado deterministicamente a 512KB (524.288 bytes). Entradas que excedem este limiar são rejeitadas imediatamente antes de qualquer alocação excessiva ou envio HTTP.
- **SEC-03 (Timeout Mandatório):** Todas as chamadas de avaliação semântica são orquestradas sob `context.WithTimeout` (com timeout padrão seguro de 10 segundos na CLI e MCP, ajustável via flags/argumentos), prevenindo travamento de conexões e exaustão de descritores de arquivo.
- **SEC-04 (Supply Chain & Invariantes):** O código mantém zero dependências externas no `go.mod`, utilizando exclusivamente recursos nativos da standard library do Go 1.24+.

# Final Checklist

- [x] `gofmt -l .` sem saída: código formatado de acordo com as convenções oficiais
- [x] `go build ./...` limpo: compilação estática de todos os binários e pacotes sem erros
- [x] `go vet ./...` limpo: análise estática sem advertências
- [x] `go test ./...` verde: todos os pacotes passando com 100% de sucesso
- [x] Escopo coberto por teste: testes unitários e de integração cobrindo CLI, MCP e salvaguardas SEC-01, SEC-02, SEC-03
- [x] Nada do fora-de-escopo foi implementado: nenhuma dependência externa adicionada
- [x] Documentação dos módulos tocados atualizada: DECISOES.md e README.md atualizados
- [x] Delivery registrada no INDEX.md: linha da DEL-026 registrada no topo do índice
