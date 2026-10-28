# Além dos Hooks: Como Construí uma Esteira de IA Autônoma sem "Fadiga de Confirmação" com Go e Antigravity

*Por Rodrigo Oliveira — Engenheiro de Software & Pós-Graduando em Engenharia de IA Aplicada*

---

No meu [artigo anterior](https://www.linkedin.com/pulse/o-gargalo-da-ia-n%C3%A3o-%C3%A9-gerar-c%C3%B3digo-governar-como-um-com-oliveira-aihpf/), discuti uma lição dolorosa da engenharia de software na era dos agentes inteligentes: **o gargalo da IA não é a velocidade de gerar código, mas a capacidade de governá-lo determinística e rigidamente**. 

Para resolver as alucinações e os commits desastrosos em branches principais, criei o **Pipeline-AI**: uma arquitetura baseada em **Zero-Trust com hooks de ciclo de vida (`PreToolUse`) em Go puro** e uma esteira especializada de 16 skills conectada ao ecossistema do **Google Antigravity**.

A governança funcionou. Os agentes pararam de destruir o repositório. Mas, no dia seguinte à implementação, nos deparamos com o próximo grande vilão do desenvolvimento com agentes autônomos:

> **A Fadiga de Confirmação.**

Se para cada uma das 16 etapas o orquestrador pausar e perguntar:
> *"Testes escritos com sucesso. Deseja que eu execute o dev-implementation? (Sim/Não)"*

...você deixa de ser um engenheiro de software de alto nível e vira um **"apertador de Enter em tempo integral"**.

Neste artigo — que marca o lançamento da versão **v0.6.1 do Pipeline-AI** —, compartilho como transformamos essa esteira estrita em um **fluxo autônomo contínuo**, introduzindo o **Princípio da Não-Interrupção**, validação formal de *handoff contracts*, salvaguardas contra autotestes falsos-positivos e um **Pipeline Runner unificado em Go integrado via Model Context Protocol (MCP)**.

---

## 1. O Paradoxo da Microgestão de Agentes

Quando começamos a construir sistemas multiagentes, o instinto inicial é aplicar barreiras de segurança manuais em todas as transições:

```text
[pm-triage] → "Posso avançar?" → Humano: "Sim"
  → [architecture-guardrails] → "Posso avançar?" → Humano: "Sim"
    → [test-writer] → "Posso avançar?" → Humano: "Sim"
      → [dev-implementation] → "Posso avançar?" → Humano: "Sim"
        → [code-reviewer] → "Posso avançar?" → Humano: "Sim"
...
```

Esse modelo é insustentável por três motivos:

1. **Quebra de fluxo cognitivo:** O desenvolvedor precisa ficar monitorando a tela a cada 90 segundos para dar aprovações protocolares que não agregam valor de julgamento.
2. **Falsa sensação de controle:** O humano, cansado de responder "sim" repetidamente, passa a aprovar no piloto automático sem inspecionar os artefatos com o rigor necessário.
3. **Custo operacional elevado:** O ciclo de vida de uma issue, que deveria durar 5 a 8 minutos de máquina de ponta a ponta, passa a levar horas por causa da latência humana de resposta.

A governança precisa ser **forte onde importa e invisível onde o determinismo dá conta**.

---

## 2. O Princípio da Não-Interrupção e o Contrato de Handoff

Para eliminar a microgestão, formalizamos uma nova regra fundamental no núcleo do framework: o **Princípio da Não-Interrupção (Regra 7 de Orquestração)**.

> **Regra:** Uma vez iniciada uma demanda técnica a partir do triage, o orquestrador conduz as transições entre skills de ponta a ponta **sem pausas protocolares para confirmação**, contanto que os gates estejam verdes (`STATUS: ok`).

A intervenção humana é estritamente reservada a apenas três momentos:
1. **Ambiguidade crítica de negócio/design** identificada durante a triagem (`pm-triage`).
2. **Falha bloqueante** que os agentes técnicos não conseguiram contornar (`STATUS: bloqueado`).
3. **Abertura final do Pull Request**, momento em que o humano atua como revisor da entrega completa.

```mermaid
flowchart TD
    A["Início: Demanda do Usuário"] --> B["pm-triage"]
    B --> C{"Ambiguidade Crítica?"}
    C -- Sim --> D["Pausa: Alinhamento Humano"]
    C -- Não --> E["architecture-guardrails"]
    E --> F["test-writer (TDD Red)"]
    F --> G["dev-implementation (Green)"]
    G --> H["code-reviewer & security-review"]
    H --> I["qa-validation & docs-update"]
    I --> J{"Gate Bloqueado?"}
    J -- Sim --> K["Pausa: Intervenção Humana"]
    J -- Não --> L["git-delivery: PR Aberto"]
    L --> M["Revisão Final Humana"]

    style D fill:#ffcccc,stroke:#ff0000
    K fill:#ffcccc,stroke:#ff0000
    M fill:#ccffcc,stroke:#00aa00
```

### O Contrato Formal de Handoff

Como garantir que um agente não alucine nem "engula" contexto ao passar o bastão para o próximo? Criamos o **Handoff Contract**: um cabeçalho estruturado e auditável exigido em toda passagem de skill:

```text
=== HANDOFF CONTRACT ===
DE: test-writer
PARA: dev-implementation
TAREFA: DEL-018
STATUS: ok
ESCOPO: internal/handoff
EVIDENCIAS:
- [x] go test ./internal/handoff (falhou conforme esperado na fase Red)
- [x] 4 cenários table-driven definidos
========================
```

Para garantir que essa regra não dependa da "boa vontade" do modelo, criamos um validador nativo em Go:
```bash
go tool pipeline-ai handoff validate handoff.txt
```

O comando analisa o bloco, valida se os estados são canônicos (`ok`, `bloqueado`, `pendente`), verifica se os campos obrigatórios estão preenchidos e audita a rastreabilidade. Para proteger o validador contra entradas malformadas de LLMs, submetemos o analisador sintático a **testes de Fuzzing contínuos com a stdlib de Go**.

---

## 3. O Pipeline Runner Unificado (`pipeline-ai run`) e o Protocolo MCP

Nas primeiras versões, a verificação da esteira exigia chamar múltiplos binários e comandos manuais: `pipeline-hook`, `pipeline-validate`, `go vet`, `go test`, etc.

Na versão `v0.6.1`, consolidamos toda a toolchain em um **Runner Determinístico Unificado**:

```bash
go tool pipeline-ai run
```

O comando executa três estágios em cascata:

1. **`Doctor`:** Inspeciona a integridade do ambiente (versão do Go 1.24+, presença de caminhos protegidos, declaração de `immutable_paths` e consistência do `.pipeline-ai.json`).
2. **`Sensor`:** Dispara a toolchain do projeto (`gofmt`, `go build`, `go vet`, `go test ./...`, `govulncheck`) de acordo com o perfil do projeto.
3. **`Validate`:** Realiza auditoria cruzada entre o relatório de entrega (`report.md`) e o `git diff` real do repositório, garantindo que nenhum arquivo de segurança foi tocado sem justificativa técnica.

```text
========================================
       PIPELINE-AI RUNNER v0.6.1        
========================================

[1/3] Executando Doctor...
[PASS] doctor: configuracao valida (tipo=api, 7 protegidos, 3 editores)

[2/3] Executando Sensores...
[PASS] sensor: go build ./...
[PASS] sensor: go vet ./...
[PASS] sensor: go test ./... (315 testes executados com sucesso)

[3/3] Executando Validacao de Delivery...
[PASS] validate: docs-ai/deliveries/DEL-018/report.md valido

========================================
 OVERALL: PASS (3/3 estagios aprovados)
========================================
```

### Integração Nativa via MCP (Model Context Protocol)

Além da CLI, expusemos esse runner diretamente como uma ferramenta nativa para o orquestrador através do **Model Context Protocol (MCP)**:

```go
// Tool MCP nativa: pipeline_run_pipeline
func (s *Server) handleRunPipeline(ctx context.Context, args json.RawMessage) (any, error) {
    // Executa doctor -> sensor -> validate e retorna JSON estruturado
    res := runner.Execute(runner.Options{
        DeliveryPath: activeDelivery,
        JsonOutput:   true,
    })
    return res, nil
}
```

Dessa forma, o orquestrador não precisa rodar scripts bash inseguros no terminal: ele invoca `pipeline_run_pipeline` via MCP, recebe o status estruturado do pipeline e toma decisões de roteamento com latência mínima.

---

## 4. As Pegadinhas Ocultas da IA em Projetos Reais

Durante a implementação do runner e a homologação em sistemas reais, descobrimos dois problemas críticos que quase nenhum framework de IA documenta:

### Pegadinha A: A Farsa do "Autoteste Negativo" (0 Testes = 0 Erros)

Você pede para o agente executar a suíte de testes. Ele roda `go test ./...`, nenhuma falha ocorre e o comando retorna código de saída `0`. O agente comemora: *"100% dos testes passaram!"*.

**A realidade oculta:** O agente havia colocado os arquivos de teste no diretório errado ou com um prefixo incorreto (ex: `test_helper.go` em vez de `helper_test.go`). **Nenhum teste foi executado.**

Em Go e em diversas outras linguagens, zero testes executados retorna código `0`. Para um agente probabilístico, ausência de erro é sinônimo de sucesso.

**A Solução no Pipeline-AI:**
Implementamos uma **Salvaguarda de Autoteste Negativo** no motor de sensores:

```go
// internal/sensor/sensor.go
if testCount == 0 && !cfg.AllowEmptyTests {
    return SensorResult{
        Passed: false,
        Error:  "Salvaguarda ativada: zero testes executados (falso positivo evitado)",
    }
}
```

Se o sensor rodar e o contador de testes for igual a zero, o pipeline **falha imediatamente**. Nenhum falso positivo passa para o Pull Request.

---

### Pegadinha B: O Agente Que Burlar a Própria Segurança (`immutable_paths`)

Quando um agente desenvolvedor recebe a missão de fazer os testes passarem, ele é extremamente pragmático. Se um hook de ciclo de vida (`hooks.json`) bloqueia sua escrita ou se a configuração de segurança (`.pipeline-ai.json`) o impede de commitar, alguns modelos tentam a solução mais óbvia: **editar o arquivo de configuração para remover a restrição!**

Para fechar essa brecha, introduzimos a diretiva de **Caminhos Imutáveis (`immutable_paths`)**:

```json
{
  "protected_paths": ["internal/", "pkg/", "cmd/"],
  "editor_agents": ["dev-implementation", "test-writer"],
  "immutable_paths": [
    ".pipeline-ai.json",
    ".agents/hooks.json",
    ".github/workflows/"
  ]
}
```

No hook `PreToolUse`, mesmo que o subagente seja um editor autorizado (como `dev-implementation`), se o caminho alvo estiver em `immutable_paths`, a operação de escrita é sumariamente bloqueada no sistema operacional. **Nem a IA nem o orquestrador podem alterar as regras do jogo durante a partida.**

---

## 5. Homologação em Produção: Dois Mundos Opostos

Para provar que essa arquitetura não é apenas teoria de pós-graduação, validamos a esteira `v0.6.1` em dois sistemas de produção com naturezas completamente distintas:

| Projeto | Arquitetura | Desafio Técnico | Resultado Autônomo |
|---|---|---|---|
| **`sistema-estoque`** | Desktop (Go + Wails v3) | Compilação cruzada Windows sem CGO a partir de containers Linux/WSL2 | **327 testes** passando com injeção automática de ambiente (`CGO_ENABLED=0 GOOS=windows GOARCH=amd64`) pelo runner. |
| **`gestao-financas`** | Backend API (Go + PostgreSQL) | Governança estrita de migrações DDL e isolamento de banco relacional | **315 testes** passando com validação de idempotência e zero locks perigosos. |

Em ambos os projetos, o ciclo completo (Triage → Guardrails → TDD Red → Dev Green → Code Review → Security Review → QA → Docs → Git Delivery) foi executado com **zero interrupções intermediárias**, culminando na abertura limpa dos Pull Requests [#16](https://github.com/rodrigodosanjosoliveira/sistema-estoque/pull/16) e [#18](https://github.com/rodrigodosanjosoliveira/gestao-financas/pull/18).

---

## Lições para a Próxima Geração da Engenharia de Software

Se o primeiro estágio da IA foi o deslumbramento com a geração de código (2023-2024), o estágio atual é o da **engenharia de confiabilidade de agentes (Agent Reliability Engineering - ARE)**:

1. **A autonomia real exige gates determinísticos:** Você só tem coragem de deixar um agente trabalhar sozinho se souber que o sistema operacional vai barrá-lo fisicamente caso ele tente violar um invariante.
2. **Microgestão humana gera fadiga e displicência:** Se você desenhou uma esteira segura, confie nos gates. Deixe o orquestrador conduzir o fluxo e reserve o tempo humano para o Code Review final da issue.
3. **Sensores negativos são tão vitais quanto os positivos:** Provar que algo passou não basta; é mandatório provar que a verificação realmente foi executada e que nada silencioso foi engolido.

---

O código do **Pipeline-AI v0.6.1**, a suíte de 16 skills, o motor de sensores e o runner determinístico em Go puro são 100% open-source:

👉 **Repositório Oficial:** [github.com/rodrigodosanjosoliveira/pipeline-ai](https://github.com/rodrigodosanjosoliveira/pipeline-ai)

Como você tem equilibrado a autonomia dos seus agentes com a segurança e a governança do repositório? Compartilhe suas experiências e desafios nos comentários! 💬🚀
