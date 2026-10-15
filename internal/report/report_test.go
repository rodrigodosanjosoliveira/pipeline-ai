package report

import (
	"strings"
	"testing"
)

const relatorioOK = `---
delivery_id: DEL-001
task_code: TASK-42
title: Ajuste no parser de tokens
status: done
require_security: false
---

# Contexto
Parser rejeitava tokens com padding.

# Escopo
internal/auth/parser.go

# Implementação
Normalização antes do decode.

# Testes
Table-driven cobrindo 6 casos.

# Code Review
Aprovado sem ressalvas.

# Final Checklist
- [x] build passa
- [x] testes passam
- [x] vet limpo
`

func hasError(fs []Finding, substr string) bool {
	for _, f := range fs {
		if f.Level == "ERRO" && strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}

func TestRelatorioValido(t *testing.T) {
	r, err := Parse(relatorioOK)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if fs := r.Validate(); len(fs) != 0 {
		t.Errorf("esperava zero findings, obteve %d: %v", len(fs), fs)
	}
}

func TestParseErros(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"sem front matter", "# Contexto\ntexto"},
		{"front matter não fechado", "---\ndelivery_id: X\n# Contexto"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse(tt.content); err == nil {
				t.Error("esperava erro de parse, obteve nil")
			}
		})
	}
}

func TestFrontMatterObrigatorio(t *testing.T) {
	content := strings.Replace(relatorioOK, "task_code: TASK-42\n", "", 1)
	r, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !hasError(r.Validate(), "task_code") {
		t.Error("esperava erro sobre task_code ausente")
	}
}

func TestSecaoObrigatoriaAusente(t *testing.T) {
	content := strings.Replace(relatorioOK, "# Code Review\nAprovado sem ressalvas.\n", "", 1)
	r, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !hasError(r.Validate(), "code review") {
		t.Error("esperava erro sobre a seção Code Review")
	}
}

// TestPlaceholderParcial fixa a divergência 2: a versão em bash só rejeitava
// quando o valor inteiro era TODO. Aqui o marcador aparece no meio da frase.
func TestPlaceholderParcial(t *testing.T) {
	content := strings.Replace(
		relatorioOK,
		"Table-driven cobrindo 6 casos.",
		"Table-driven cobrindo 6 casos. TODO: falta o caso de expiração.",
		1,
	)
	r, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !hasError(r.Validate(), "TODO") {
		t.Errorf("esperava erro sobre marcador TODO, obteve: %v", r.Validate())
	}
}

// TestPortuguesNaoEhMarcador fixa o falso positivo que a primeira versão tinha:
// a detecção por substring reprovava "todos" (contém "todo") e "independente"
// (contém "pendente"), inviabilizando qualquer relatório em português.
func TestPortuguesNaoEhMarcador(t *testing.T) {
	frases := []string{
		"Todos os testes passaram.",
		"O módulo é independente do storage.",
		"Cobre todo o fluxo de autenticação.",
		"Nenhum item pendente no checklist.",
	}
	for _, frase := range frases {
		t.Run(frase, func(t *testing.T) {
			content := strings.Replace(relatorioOK, "Table-driven cobrindo 6 casos.", frase, 1)
			r, err := Parse(content)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if fs := r.Validate(); len(fs) != 0 {
				t.Errorf("texto português legítimo foi reprovado: %v", fs)
			}
		})
	}
}

func TestMarcadoresDeCodigo(t *testing.T) {
	tests := []struct {
		frase string
		erro  bool
	}{
		{"Falta cobrir expiração. FIXME antes do merge.", true},
		{"Prazo TBD com o time.", true},
		{"Deixado como XXX por ora.", true},
		{"Ainda WIP.", true},
		{"Seção a definir.", true},
		{"Lorem ipsum dolor.", true},
		{"O fix me pareceu correto.", false},
		{"Rodou em ambiente Windows XP.", false},
	}
	for _, tt := range tests {
		t.Run(tt.frase, func(t *testing.T) {
			content := strings.Replace(relatorioOK, "Table-driven cobrindo 6 casos.", tt.frase, 1)
			r, err := Parse(content)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			fs := r.Validate()
			if tt.erro && len(fs) == 0 {
				t.Error("esperava reprovação, passou")
			}
			if !tt.erro && len(fs) != 0 {
				t.Errorf("esperava aprovação, reprovou: %v", fs)
			}
		})
	}
}

func TestChecklistIncompleto(t *testing.T) {
	content := strings.Replace(relatorioOK, "- [x] vet limpo", "- [ ] vet limpo", 1)
	r, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !hasError(r.Validate(), "não concluído") {
		t.Error("esperava erro sobre item não concluído")
	}
}

func TestFlagsCondicionais(t *testing.T) {
	tests := []struct {
		name     string
		front    string
		secao    string
		querErro string
	}{
		{
			name:     "security ligado sem seção",
			front:    "require_security: true",
			secao:    "",
			querErro: `seção "security"`,
		},
		{
			name:     "security ligado com N/A",
			front:    "require_security: true",
			secao:    "\n# Security\nN/A\n",
			querErro: "N/A",
		},
		{
			name:     "security ligado e preenchido",
			front:    "require_security: true",
			secao:    "\n# Security\ngovulncheck limpo; sem novos símbolos vulneráveis.\n",
			querErro: "",
		},
		{
			name:     "security desligado sem seção",
			front:    "require_security: false",
			secao:    "",
			querErro: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.Replace(relatorioOK, "require_security: false", tt.front, 1) + tt.secao
			r, err := Parse(content)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			fs := r.Validate()
			if tt.querErro == "" {
				if len(fs) != 0 {
					t.Errorf("esperava zero findings, obteve: %v", fs)
				}
				return
			}
			if !hasError(fs, tt.querErro) {
				t.Errorf("esperava erro contendo %q, obteve: %v", tt.querErro, fs)
			}
		})
	}
}

func TestFlag(t *testing.T) {
	tests := []struct {
		valor string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"sim", true},
		{"1", true},
		{"false", false},
		{"", false},
		{"talvez", false},
	}
	for _, tt := range tests {
		t.Run(tt.valor, func(t *testing.T) {
			r := &Report{Front: map[string]string{"require_security": tt.valor}}
			if got := r.Flag("require_security"); got != tt.want {
				t.Errorf("Flag(%q) = %v, quer %v", tt.valor, got, tt.want)
			}
		})
	}
}

func TestFlagAusenteEhFalse(t *testing.T) {
	r := &Report{Front: map[string]string{}}
	if r.Flag("require_security") {
		t.Error("flag ausente deve valer false (D-14)")
	}
}

func TestNormalizeTitulos(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Implementação", "implementacao"},
		{"# Code Review", "code review"},
		{"  TESTES  ", "testes"},
		{"Migração:", "migracao"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalize(strings.TrimLeft(tt.in, "# ")); got != tt.want {
				t.Errorf("normalize(%q) = %q, quer %q", tt.in, got, tt.want)
			}
		})
	}
}
