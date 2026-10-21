# Contrato de Handoff

Todo agent termina seu turno com este bloco. Sem ele, o próximo agent não tem
por onde começar e o `agent-auditor` marca `HANDOFF_RASO`.

Resolve a divergência 4 do backlog: esta é a única versão válida. As variantes
que existiam na skill, no command e no guia estão descartadas.

## Formato

```
DELIVERY_ID: {DEL-000}
TASK_CODE: {TASK-000}
AGENT: {nome-do-agent}
STATUS: ok | bloqueado

CONFIRMADO:
  - {o que foi verificado no código, com caminho ou comando}

ASSUNCOES:
  - {o que foi suposto e não confirmado — vazio só se nada foi suposto}

ARQUIVOS:
  - {caminho} — {o que mudou}

RISCOS:
  - {risco residual específico, não boilerplate}

SINAIS:
  SECURITY: sim|nao — {motivo}
  SUPPLY_CHAIN: sim|nao — {motivo}
  MIGRATION: sim|nao — {motivo}
  PERFORMANCE: sim|nao — {motivo}

PROXIMO: {agent}
MOTIVO: {por que este agent e não outro}
```

## Regras

1. **`CONFIRMADO` exige evidência.** Caminho de arquivo, comando executado ou
   resultado real. "Verifiquei e está ok" não é evidência.
2. **`ASSUNCOES` vazio precisa ser verdade.** Se havia ambiguidade e o campo
   está vazio, é `DRIFT`.
3. **Os `SINAIS` determinam o roteamento.** `SECURITY: sim` obriga a passagem
   por `security-review` antes do QA. `SUPPLY_CHAIN: sim` obriga
   `supply-chain-review`. Os outros dois ativam módulos opcionais, se o projeto
   os tiver habilitados.
4. **`STATUS: bloqueado` interrompe o fluxo.** O `PROXIMO` passa a ser quem pode
   desbloquear, e o `MOTIVO` diz o que falta.
5. **Um handoff não repete o anterior.** Carrega o que o receptor precisa, não o
   histórico inteiro.
