package handoff

import (
	"strings"
	"testing"
)

func validHandoffBase() *Handoff {
	return &Handoff{
		De:         "architecture-guardrails",
		Para:       "test-writer",
		DeliveryID: "DEL-018",
		TaskCode:   "TASK-018",
		Status:     "ok",
		Escopo:     "internal/handoff, cmd/pipeline-ai, internal/mcp",
		Arquivos: []string{
			"internal/handoff/handoff.go",
			"internal/handoff/handoff_test.go",
		},
		Confirmado: "go test -count=1 ./... executado com sucesso e 404 testes passando",
		Assuncoes:  "stdlib pura sem dependencias externas",
		Riscos:     "nenhum",
		Sinais: Signals{
			Security:    false,
			SupplyChain: false,
			Migration:   false,
			Performance: false,
		},
		Proximo: "test-writer",
		Motivo:  "escrita de testes automatizados antes da implementação",
	}
}

func TestParse_Success(t *testing.T) {
	tests := []struct {
		name              string
		raw               string
		wantDe            string
		wantPara          string
		wantDeliveryID    string
		wantTaskCode      string
		wantStatus        string
		wantEscopo        string
		wantArquivosCount int
		wantConfirmado    string
		wantAssuncoes     string
		wantRiscos        string
		wantSecurity      bool
		wantSupplyChain   bool
		wantMigration     bool
		wantPerformance   bool
		wantReqSec        string
		wantProximo       string
		wantMotivo        string
	}{
		{
			name: "bloco texto puro completo",
			raw: `DE: architecture-guardrails
PARA: test-writer
DELIVERY_ID: DEL-018
TASK_CODE: TASK-018
STATUS: ok
ESCOPO: internal/handoff, cmd/pipeline-ai, internal/mcp
ARQUIVOS: internal/handoff/handoff.go, internal/handoff/handoff_test.go
CONFIRMADO: go test -count=1 ./... executado com sucesso
ASSUNCOES: stdlib pura sem dependencias externas
RISCOS: nenhum
SINAIS:
  SECURITY: nao
  SUPPLY_CHAIN: nao
  MIGRATION: nao
  PERFORMANCE: nao
PROXIMO: test-writer
MOTIVO: escrita de testes automatizados antes da implementacao`,
			wantDe:            "architecture-guardrails",
			wantPara:          "test-writer",
			wantDeliveryID:    "DEL-018",
			wantTaskCode:      "TASK-018",
			wantStatus:        "ok",
			wantEscopo:        "internal/handoff, cmd/pipeline-ai, internal/mcp",
			wantArquivosCount: 2,
			wantConfirmado:    "go test -count=1 ./... executado com sucesso",
			wantAssuncoes:     "stdlib pura sem dependencias externas",
			wantRiscos:        "nenhum",
			wantSecurity:      false,
			wantSupplyChain:   false,
			wantMigration:     false,
			wantPerformance:   false,
			wantReqSec:        "",
			wantProximo:       "test-writer",
			wantMotivo:        "escrita de testes automatizados antes da implementacao",
		},
		{
			name: "envolvido em cerca de codigo markdown com security sim e requisitos",
			raw: "```text\n" +
				"DE: test-writer\n" +
				"PARA: dev-implementation\n" +
				"DELIVERY_ID: DEL-018\n" +
				"TASK_CODE: TASK-018\n" +
				"STATUS: ok\n" +
				"ESCOPO: internal/handoff\n" +
				"ARQUIVOS: internal/handoff/handoff_test.go\n" +
				"REQUISITOS_SEGURANCA: 1. Salvaguarda adversarial: rejeitar payloads maliciosos sem autenticacao\n" +
				"CONFIRMADO: go test ./internal/handoff retornou falhas esperadas na fase vermelha\n" +
				"ASSUNCOES: nenhuma\n" +
				"RISCOS: quebra de compatibilidade com versoes anteriores\n" +
				"SINAIS:\n" +
				"  SECURITY: sim\n" +
				"  SUPPLY_CHAIN: sim\n" +
				"  MIGRATION: nao\n" +
				"  PERFORMANCE: nao\n" +
				"PROXIMO: dev-implementation\n" +
				"MOTIVO: implementar funcionalidade do parser e validador\n" +
				"```",
			wantDe:            "test-writer",
			wantPara:          "dev-implementation",
			wantDeliveryID:    "DEL-018",
			wantTaskCode:      "TASK-018",
			wantStatus:        "ok",
			wantEscopo:        "internal/handoff",
			wantArquivosCount: 1,
			wantConfirmado:    "go test ./internal/handoff retornou falhas esperadas na fase vermelha",
			wantAssuncoes:     "nenhuma",
			wantRiscos:        "quebra de compatibilidade com versoes anteriores",
			wantSecurity:      true,
			wantSupplyChain:   true,
			wantMigration:     false,
			wantPerformance:   false,
			wantReqSec:        "1. Salvaguarda adversarial: rejeitar payloads maliciosos sem autenticacao",
			wantProximo:       "dev-implementation",
			wantMotivo:        "implementar funcionalidade do parser e validador",
		},
		{
			name: "envolvido em cerca markdown simples e status bloqueado",
			raw: "```\n" +
				"DE: dev-implementation\n" +
				"PARA: architecture-guardrails\n" +
				"DELIVERY_ID: DEL-018\n" +
				"STATUS: bloqueado\n" +
				"ESCOPO: internal/handoff\n" +
				"ARQUIVOS: internal/handoff/handoff.go\n" +
				"CONFIRMADO: go build ./... falhou com erro de sintaxe na linha 42\n" +
				"ASSUNCOES: \n" +
				"RISCOS: atraso na delivery\n" +
				"SINAIS:\n" +
				"  SECURITY: nao\n" +
				"  SUPPLY_CHAIN: nao\n" +
				"  MIGRATION: sim\n" +
				"  PERFORMANCE: sim\n" +
				"PROXIMO: architecture-guardrails\n" +
				"MOTIVO: definicao arquitetural ambigua quanto ao modelo de dados\n" +
				"```",
			wantDe:            "dev-implementation",
			wantPara:          "architecture-guardrails",
			wantDeliveryID:    "DEL-018",
			wantTaskCode:      "",
			wantStatus:        "bloqueado",
			wantEscopo:        "internal/handoff",
			wantArquivosCount: 1,
			wantConfirmado:    "go build ./... falhou com erro de sintaxe na linha 42",
			wantAssuncoes:     "",
			wantRiscos:        "atraso na delivery",
			wantSecurity:      false,
			wantSupplyChain:   false,
			wantMigration:     true,
			wantPerformance:   true,
			wantReqSec:        "",
			wantProximo:       "architecture-guardrails",
			wantMotivo:        "definicao arquitetural ambigua quanto ao modelo de dados",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := Parse(tt.raw)
			if err != nil {
				t.Fatalf("Parse() retornou erro inesperado: %v", err)
			}
			if h == nil {
				t.Fatalf("Parse() retornou h == nil")
			}

			if h.De != tt.wantDe {
				t.Errorf("h.De = %q, esperado %q", h.De, tt.wantDe)
			}
			if h.Para != tt.wantPara {
				t.Errorf("h.Para = %q, esperado %q", h.Para, tt.wantPara)
			}
			if h.DeliveryID != tt.wantDeliveryID {
				t.Errorf("h.DeliveryID = %q, esperado %q", h.DeliveryID, tt.wantDeliveryID)
			}
			if tt.wantTaskCode != "" && h.TaskCode != tt.wantTaskCode {
				t.Errorf("h.TaskCode = %q, esperado %q", h.TaskCode, tt.wantTaskCode)
			}
			if h.Status != tt.wantStatus {
				t.Errorf("h.Status = %q, esperado %q", h.Status, tt.wantStatus)
			}
			if h.Escopo != tt.wantEscopo {
				t.Errorf("h.Escopo = %q, esperado %q", h.Escopo, tt.wantEscopo)
			}
			if len(h.Arquivos) != tt.wantArquivosCount {
				t.Errorf("len(h.Arquivos) = %d, esperado %d (arquivos: %v)", len(h.Arquivos), tt.wantArquivosCount, h.Arquivos)
			}
			if h.Confirmado != tt.wantConfirmado {
				t.Errorf("h.Confirmado = %q, esperado %q", h.Confirmado, tt.wantConfirmado)
			}
			if h.Assuncoes != tt.wantAssuncoes {
				t.Errorf("h.Assuncoes = %q, esperado %q", h.Assuncoes, tt.wantAssuncoes)
			}
			if h.Riscos != tt.wantRiscos {
				t.Errorf("h.Riscos = %q, esperado %q", h.Riscos, tt.wantRiscos)
			}
			if h.Sinais.Security != tt.wantSecurity {
				t.Errorf("h.Sinais.Security = %v, esperado %v", h.Sinais.Security, tt.wantSecurity)
			}
			if h.Sinais.SupplyChain != tt.wantSupplyChain {
				t.Errorf("h.Sinais.SupplyChain = %v, esperado %v", h.Sinais.SupplyChain, tt.wantSupplyChain)
			}
			if h.Sinais.Migration != tt.wantMigration {
				t.Errorf("h.Sinais.Migration = %v, esperado %v", h.Sinais.Migration, tt.wantMigration)
			}
			if h.Sinais.Performance != tt.wantPerformance {
				t.Errorf("h.Sinais.Performance = %v, esperado %v", h.Sinais.Performance, tt.wantPerformance)
			}
			if tt.wantReqSec != "" && h.RequisitosSeguranca != tt.wantReqSec {
				t.Errorf("h.RequisitosSeguranca = %q, esperado %q", h.RequisitosSeguranca, tt.wantReqSec)
			}
			if h.Proximo != tt.wantProximo {
				t.Errorf("h.Proximo = %q, esperado %q", h.Proximo, tt.wantProximo)
			}
			if h.Motivo != tt.wantMotivo {
				t.Errorf("h.Motivo = %q, esperado %q", h.Motivo, tt.wantMotivo)
			}
		})
	}
}

func TestValidate_MandatoryFields(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(h *Handoff)
		wantValid   bool
		wantErrPart string
	}{
		{
			name:      "handoff base totalmente valido com status ok",
			mutate:    func(h *Handoff) {},
			wantValid: true,
		},
		{
			name: "handoff valido com status bloqueado",
			mutate: func(h *Handoff) {
				h.Status = "bloqueado"
			},
			wantValid: true,
		},
		{
			name: "falha quando DE esta ausente",
			mutate: func(h *Handoff) {
				h.De = ""
			},
			wantValid:   false,
			wantErrPart: "DE",
		},
		{
			name: "falha quando PARA esta ausente",
			mutate: func(h *Handoff) {
				h.Para = ""
			},
			wantValid:   false,
			wantErrPart: "PARA",
		},
		{
			name: "falha quando DELIVERY_ID esta ausente",
			mutate: func(h *Handoff) {
				h.DeliveryID = ""
			},
			wantValid:   false,
			wantErrPart: "DELIVERY_ID",
		},
		{
			name: "falha quando PROXIMO esta ausente",
			mutate: func(h *Handoff) {
				h.Proximo = ""
			},
			wantValid:   false,
			wantErrPart: "PROXIMO",
		},
		{
			name: "falha quando MOTIVO esta ausente",
			mutate: func(h *Handoff) {
				h.Motivo = ""
			},
			wantValid:   false,
			wantErrPart: "MOTIVO",
		},
		{
			name: "falha quando STATUS e vazio",
			mutate: func(h *Handoff) {
				h.Status = ""
			},
			wantValid:   false,
			wantErrPart: "STATUS",
		},
		{
			name: "falha quando STATUS e invalido",
			mutate: func(h *Handoff) {
				h.Status = "em-andamento"
			},
			wantValid:   false,
			wantErrPart: "STATUS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := validHandoffBase()
			tt.mutate(h)

			res := Validate(h)
			if res == nil {
				t.Fatalf("Validate() retornou res == nil")
			}

			if res.Valid != tt.wantValid {
				t.Errorf("res.Valid = %v, esperado %v (erros: %v)", res.Valid, tt.wantValid, res.Errors)
			}

			if !tt.wantValid {
				if len(res.Errors) == 0 {
					t.Fatalf("esperava erros de validacao, mas lista de erros esta vazia")
				}
				found := false
				for _, errStr := range res.Errors {
					if strings.Contains(strings.ToUpper(errStr), strings.ToUpper(tt.wantErrPart)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("erros %v nao contem a parte esperada %q", res.Errors, tt.wantErrPart)
				}
			}
		})
	}
}

func TestValidate_EvidenceRequirement(t *testing.T) {
	tests := []struct {
		name        string
		confirmado  string
		wantValid   bool
		wantErrPart string
	}{
		{
			name:        "falha com CONFIRMADO vazio",
			confirmado:  "",
			wantValid:   false,
			wantErrPart: "CONFIRMADO",
		},
		{
			name:        "falha com evasiva generica 'Verifiquei e está ok'",
			confirmado:  "Verifiquei e está ok",
			wantValid:   false,
			wantErrPart: "evidência",
		},
		{
			name:        "falha com evasiva generica 'tudo certo'",
			confirmado:  "tudo certo",
			wantValid:   false,
			wantErrPart: "evidência",
		},
		{
			name:        "falha com evasiva 'ok'",
			confirmado:  "ok",
			wantValid:   false,
			wantErrPart: "evidência",
		},
		{
			name:        "falha com evasiva 'testado'",
			confirmado:  "testado",
			wantValid:   false,
			wantErrPart: "evidência",
		},
		{
			name:        "falha com evasiva 'nenhum'",
			confirmado:  "nenhum",
			wantValid:   false,
			wantErrPart: "evidência",
		},
		{
			name:        "passa com comando real executado",
			confirmado:  "go test ./... PASS em 0.42s com 100% dos testes passando",
			wantValid:   true,
			wantErrPart: "",
		},
		{
			name:        "passa com saida de comando e arquivo real",
			confirmado:  "cat internal/handoff/handoff.go | wc -l retornou 45 linhas",
			wantValid:   true,
			wantErrPart: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := validHandoffBase()
			h.Confirmado = tt.confirmado

			res := Validate(h)
			if res == nil {
				t.Fatalf("Validate() retornou res == nil")
			}

			if res.Valid != tt.wantValid {
				t.Errorf("res.Valid = %v, esperado %v (erros: %v)", res.Valid, tt.wantValid, res.Errors)
			}

			if !tt.wantValid {
				if len(res.Errors) == 0 {
					t.Fatalf("esperava erros de validacao para CONFIRMADO=%q", tt.confirmado)
				}
				if tt.wantErrPart != "" {
					found := false
					for _, errStr := range res.Errors {
						if strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantErrPart)) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("erros %v nao mencionam %q", res.Errors, tt.wantErrPart)
					}
				}
			}
		})
	}
}

func TestValidate_SecurityRequirementsCondition(t *testing.T) {
	tests := []struct {
		name                string
		securitySignal      bool
		requisitosSeguranca string
		wantValid           bool
		wantErrPart         string
	}{
		{
			name:                "falha quando SECURITY=sim e REQUISITOS_SEGURANCA vazio",
			securitySignal:      true,
			requisitosSeguranca: "",
			wantValid:           false,
			wantErrPart:         "REQUISITOS_SEGURANCA",
		},
		{
			name:                "passa quando SECURITY=sim e REQUISITOS_SEGURANCA preenchido",
			securitySignal:      true,
			requisitosSeguranca: "1. Requisito adversarial: rejeitar payloads sem auth no endpoint de validacao",
			wantValid:           true,
			wantErrPart:         "",
		},
		{
			name:                "passa quando SECURITY=nao e REQUISITOS_SEGURANCA vazio",
			securitySignal:      false,
			requisitosSeguranca: "",
			wantValid:           true,
			wantErrPart:         "",
		},
		{
			name:                "passa quando SECURITY=nao e REQUISITOS_SEGURANCA preenchido opcionalmente",
			securitySignal:      false,
			requisitosSeguranca: "Salvaguardas adicionais opcionais",
			wantValid:           true,
			wantErrPart:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := validHandoffBase()
			h.Sinais.Security = tt.securitySignal
			h.RequisitosSeguranca = tt.requisitosSeguranca

			res := Validate(h)
			if res == nil {
				t.Fatalf("Validate() retornou res == nil")
			}

			if res.Valid != tt.wantValid {
				t.Errorf("res.Valid = %v, esperado %v (erros: %v)", res.Valid, tt.wantValid, res.Errors)
			}

			if !tt.wantValid {
				if len(res.Errors) == 0 {
					t.Fatalf("esperava erros de validacao")
				}
				found := false
				for _, errStr := range res.Errors {
					if strings.Contains(strings.ToUpper(errStr), strings.ToUpper(tt.wantErrPart)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("erros %v nao contem %q", res.Errors, tt.wantErrPart)
				}
			}
		})
	}
}

func FuzzParseHandoff(f *testing.F) {
	seeds := []string{
		`DE: architecture-guardrails
PARA: test-writer
DELIVERY_ID: DEL-018
TASK_CODE: TASK-018
STATUS: ok
ESCOPO: internal/handoff
ARQUIVOS: internal/handoff/handoff.go
CONFIRMADO: go test ./... PASS
ASSUNCOES: nenhuma
RISCOS: nenhum
SINAIS:
  SECURITY: nao
  SUPPLY_CHAIN: nao
  MIGRATION: nao
  PERFORMANCE: nao
PROXIMO: dev-implementation
MOTIVO: implementacao`,

		"```text\nDE: pm-triage\nPARA: architecture-guardrails\nDELIVERY_ID: DEL-001\nSTATUS: ok\nPROXIMO: architecture-guardrails\nMOTIVO: arquitetura\n```",
		"",
		"STATUS: invalido\nCONFIRMADO: tudo certo\n",
		"DE: a\nPARA: b\nDELIVERY_ID: DEL-999\nSTATUS: ok\nSINAIS:\n  SECURITY: sim\nREQUISITOS_SEGURANCA: 1. Adversarial\nCONFIRMADO: go test\nPROXIMO: c\nMOTIVO: d\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		h, err := Parse(data)
		if err == nil && h != nil {
			res := Validate(h)
			if res == nil {
				t.Errorf("Validate() retornou nil para handoff valido extraido")
			}
		}
	})
}
