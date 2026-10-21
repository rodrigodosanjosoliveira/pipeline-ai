# Contrato de Handoff

Entre qualquer etapa do pipeline, a passagem de contexto segue este bloco:

```text
DE: {skill/agent}
PARA: {skill/agent}
DELIVERY_ID: {ID}
STATUS: ok | bloqueado
ESCOPO: {módulos e caminhos}
ARQUIVOS: {lista dos tocados nesta etapa}
CONFIRMADO: {comandos rodados e saídas reais}
ASSUNCOES: {o que foi assumido; vazio se nada}
RISCOS: {o que pode quebrar adiante}
SINAIS:
  SECURITY: sim | nao
  SUPPLY_CHAIN: sim | nao
  MIGRATION: sim | nao
  PERFORMANCE: sim | nao
PROXIMO: {skill/agent}
MOTIVO: {por que esta skill/agent e não outra}
```

## Regras

1. **`CONFIRMADO` exige evidência.** Caminho de arquivo, comando executado ou resultado real. "Verifiquei e está ok" não é evidência aceitável.
2. **`ASSUNCOES` vazio precisa ser verdade.** Se havia ambiguidade e o campo está vazio, é considerado `DRIFT`.
3. **Os `SINAIS` determinam o roteamento.** `SECURITY: sim` obriga a passagem por `security-review` antes do QA. `SUPPLY_CHAIN: sim` obriga `supply-chain-review`. Os outros dois ativam módulos opcionais, se o projeto os tiver habilitados.
4. **`STATUS: bloqueado` interrompe o fluxo.** O `PROXIMO` passa a ser quem pode desbloquear, e o `MOTIVO` diz o que falta.
5. **Um handoff não repete o anterior.** Carrega o que o receptor precisa, não o histórico inteiro.
