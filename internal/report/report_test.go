package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
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

func TestValidateWithConfig(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		reportContent string
		wantErrSubstr string
		customEmpty   bool
		wantNoError   bool
	}{
		{
			name:          "cfg nil aprova relatorioOK",
			cfg:           nil,
			reportContent: relatorioOK,
			wantNoError:   true,
		},
		{
			name: "delivery_id em desacordo com cfg.DeliveryIDPattern",
			cfg: &config.Config{
				DeliveryIDPattern: `^REL-\d+$`,
			},
			reportContent: relatorioOK, // delivery_id: DEL-001
			wantErrSubstr: "delivery_id",
		},
		{
			name: "delivery_id em acordo com cfg.DeliveryIDPattern",
			cfg: &config.Config{
				DeliveryIDPattern: `^DEL-\d+$`,
			},
			reportContent: relatorioOK,
			wantNoError:   true,
		},
		{
			name: "task_code em desacordo com cfg.TaskCodePattern",
			cfg: &config.Config{
				TaskCodePattern: `^ISSUE-\d+$`,
			},
			reportContent: relatorioOK, // task_code: TASK-42
			wantErrSubstr: "task_code",
		},
		{
			name: "task_code em acordo com cfg.TaskCodePattern",
			cfg: &config.Config{
				TaskCodePattern: `^TASK-\d+$`,
			},
			reportContent: relatorioOK,
			wantNoError:   true,
		},
		{
			name: "seção obrigatória customizada ausente",
			cfg: &config.Config{
				CustomSections: []string{"Arquitetura"},
			},
			reportContent: relatorioOK, // não contém # Arquitetura
			wantErrSubstr: "seção obrigatória customizada ausente: arquitetura",
		},
		{
			name: "seção obrigatória customizada vazia",
			cfg: &config.Config{
				CustomSections: []string{"Arquitetura"},
			},
			reportContent: relatorioOK + "\n# Arquitetura\n\n",
			customEmpty:   true,
		},
		{
			name: "todas as seções customizadas preenchidas aprova",
			cfg: &config.Config{
				CustomSections: []string{"Arquitetura", "Rollback Plan"},
			},
			reportContent: relatorioOK + "\n# Arquitetura\nDecisão técnica sobre arquitetura.\n\n# Rollback Plan\nProcedimento detalhado de rollback.\n",
			wantNoError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Parse(tt.reportContent)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			fs := r.ValidateWithConfig(tt.cfg)
			if tt.wantNoError {
				for _, f := range fs {
					if f.Level == "ERRO" {
						t.Errorf("esperava nenhum erro, obteve finding: %v", f)
					}
				}
			} else if tt.customEmpty {
				found := false
				for _, f := range fs {
					if f.Level == "ERRO" && strings.Contains(f.Message, "vazia") && strings.Contains(strings.ToLower(f.Message), "arquitetura") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("esperava erro de seção customizada vazia para arquitetura, obteve findings: %v", fs)
				}
			} else {
				if !hasError(fs, tt.wantErrSubstr) {
					t.Errorf("esperava erro contendo %q, obteve findings: %v", tt.wantErrSubstr, fs)
				}
			}
		})
	}
}

func FuzzValidateWithConfig(f *testing.F) {
	f.Add(relatorioOK)
	f.Add("---\ndelivery_id: DEL-001\n---\n")
	f.Add("")
	cfg := &config.Config{
		DeliveryIDPattern: `^DEL-\d+$`,
		TaskCodePattern:   `^TASK-\d+$`,
		CustomSections:    []string{"Arquitetura"},
	}
	f.Fuzz(func(t *testing.T, content string) {
		r, err := Parse(content)
		if err != nil {
			return
		}
		_ = r.ValidateWithConfig(cfg)
	})
}

func TestSalvaguardaSecaoTestes_EvidenciaExecucao(t *testing.T) {
	tests := []struct {
		name          string
		testesContent string
		wantError     bool
	}{
		{
			name:          "evidência válida com ok e PASS",
			testesContent: "```\n$ go test ./...\nok  \tgithub.com/rodrigodosanjosoliveira/pipeline-ai/internal/report\t0.041s\nPASS\n```",
			wantError:     false,
		},
		{
			name:          "seção testes vazia",
			testesContent: "",
			wantError:     true,
		},
		{
			name:          "apenas comando digitado sem saída de testes",
			testesContent: "```bash\n$ go test ./...\n```",
			wantError:     true,
		},
		{
			name:          "apenas no test files sem testes executados",
			testesContent: "```\n$ go test ./...\n?   \tgithub.com/rodrigodosanjosoliveira/pipeline-ai/foo\t[no test files]\n```",
			wantError:     true,
		},
		{
			name:          "apenas texto genérico sem log de execução",
			testesContent: "Os testes de unidade foram rodados com sucesso.",
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.Replace(
				relatorioOK,
				"Table-driven cobrindo 6 casos.",
				tt.testesContent,
				1,
			)
			r, err := Parse(content)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			fs := r.Validate()

			hasTestErr := false
			for _, f := range fs {
				if f.Level == "ERRO" && (strings.Contains(strings.ToLower(f.Message), "teste") || strings.Contains(strings.ToLower(f.Message), "evidência") || strings.Contains(strings.ToLower(f.Message), "evidencia")) {
					hasTestErr = true
					break
				}
			}

			if tt.wantError && !hasTestErr {
				t.Errorf("esperava erro de falta de evidência de execução de testes na seção Testes, obteve findings: %v", fs)
			}
			if !tt.wantError && hasTestErr {
				t.Errorf("não esperava erro de teste para conteúdo legítimo, obteve: %v", fs)
			}
		})
	}
}

func TestValidate_RequireRollback(t *testing.T) {
	baseFront := func(requireRollback string) string {
		var extra string
		if requireRollback != "" {
			extra = "\nrequire_rollback: " + requireRollback
		}
		return `---
delivery_id: DEL-001
task_code: TASK-42
title: Ajuste no parser de tokens
status: done
require_security: false` + extra + `
---

# Contexto
Parser rejeitava tokens com padding.

# Escopo
internal/auth/parser.go

# Implementação
Normalização antes do decode.

# Testes
` + "```\n$ go test ./...\nok  \tpkg\t0.01s\nPASS\n```" + `

# Code Review
Aprovado sem ressalvas.

# Final Checklist
- [x] build passa
- [x] testes passam
- [x] vet limpo
`
	}

	tests := []struct {
		name            string
		frontRollback   string
		rollbackSection string
		rollbackBody    string
		wantErr         bool
	}{
		{
			name:            "require_rollback true com Rollback Plan válido",
			frontRollback:   "true",
			rollbackSection: "Rollback Plan",
			rollbackBody:    "1. Reverter commit via git revert\n2. Executar script de rollback do banco de dados.",
			wantErr:         false,
		},
		{
			name:            "require_rollback true com Plano de Rollback válido",
			frontRollback:   "true",
			rollbackSection: "Plano de Rollback",
			rollbackBody:    "Comandos de rollback e reversão de migração.",
			wantErr:         false,
		},
		{
			name:            "require_rollback true com Reversibilidade válida",
			frontRollback:   "true",
			rollbackSection: "Reversibilidade",
			rollbackBody:    "Desativar flag de feature via painel de administração.",
			wantErr:         false,
		},
		{
			name:            "require_rollback true com Plano de Reversibilidade válido",
			frontRollback:   "true",
			rollbackSection: "Plano de Reversibilidade",
			rollbackBody:    "Passos de mitigação e reversão.",
			wantErr:         false,
		},
		{
			name:            "require_rollback true sem seção de rollback",
			frontRollback:   "true",
			rollbackSection: "",
			wantErr:         true,
		},
		{
			name:            "require_rollback true com seção de rollback vazia",
			frontRollback:   "true",
			rollbackSection: "Rollback Plan",
			rollbackBody:    "   \n",
			wantErr:         true,
		},
		{
			name:            "require_rollback true com seção contendo N/A",
			frontRollback:   "true",
			rollbackSection: "Rollback Plan",
			rollbackBody:    "N/A",
			wantErr:         true,
		},
		{
			name:            "require_rollback true com seção contendo não se aplica",
			frontRollback:   "true",
			rollbackSection: "Plano de Rollback",
			rollbackBody:    "não se aplica",
			wantErr:         true,
		},
		{
			name:            "require_rollback false sem seção de rollback",
			frontRollback:   "false",
			rollbackSection: "",
			wantErr:         false,
		},
		{
			name:            "require_rollback omitido no frontmatter sem seção",
			frontRollback:   "",
			rollbackSection: "",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := baseFront(tt.frontRollback)
			if tt.rollbackSection != "" {
				raw += "\n# " + tt.rollbackSection + "\n" + tt.rollbackBody + "\n"
			}

			rep, err := Parse(raw)
			if err != nil {
				t.Fatalf("Parse falhou: %v", err)
			}

			findings := rep.Validate()
			hasRollbackErr := false
			for _, f := range findings {
				if f.Level == "ERRO" && strings.Contains(strings.ToLower(f.Message), "rollback") {
					hasRollbackErr = true
					break
				}
			}

			if tt.wantErr && !hasRollbackErr {
				t.Errorf("esperava erro de rollback em Validate(), obteve findings: %v", findings)
			}
			if !tt.wantErr && hasRollbackErr {
				t.Errorf("não esperava erro de rollback em Validate(), obteve: %v", findings)
			}
		})
	}

	t.Run("politica de projeto require_rollback via ValidateWithConfig", func(t *testing.T) {
		// 1. Projeto exige rollback no .pipeline-ai.json mas relatório omitiu flag e seção
		tmpDir := t.TempDir()
		cfgJSON := `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_rollback": true}`
		if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfgJSON), 0644); err != nil {
			t.Fatal(err)
		}
		cfgRequired, err := config.Load(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		rawNoSection := baseFront("")
		rep1, err := Parse(rawNoSection)
		if err != nil {
			t.Fatal(err)
		}
		fs1 := rep1.ValidateWithConfig(cfgRequired)
		hasErr1 := false
		for _, f := range fs1 {
			if f.Level == "ERRO" && strings.Contains(strings.ToLower(f.Message), "rollback") {
				hasErr1 = true
				break
			}
		}
		if !hasErr1 {
			t.Errorf("esperava erro ao exigir rollback pelo config do projeto, findings: %v", fs1)
		}

		// 2. Projeto exige rollback e relatório possui Rollback Plan preenchido
		rawWithSection := baseFront("") + "\n# Rollback Plan\nComandos de reversão: git revert HEAD\n"
		rep2, err := Parse(rawWithSection)
		if err != nil {
			t.Fatal(err)
		}
		fs2 := rep2.ValidateWithConfig(cfgRequired)
		for _, f := range fs2 {
			if f.Level == "ERRO" && strings.Contains(strings.ToLower(f.Message), "rollback") {
				t.Errorf("erro inesperado de rollback com seção válida: %v", f)
			}
		}

		// 3. Projeto com require_rollback: false e relatório sem seção
		tmpDirFalse := t.TempDir()
		cfgFalseJSON := `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_rollback": false}`
		if err := os.WriteFile(filepath.Join(tmpDirFalse, ".pipeline-ai.json"), []byte(cfgFalseJSON), 0644); err != nil {
			t.Fatal(err)
		}
		cfgNotRequired, err := config.Load(tmpDirFalse)
		if err != nil {
			t.Fatal(err)
		}

		fs3 := rep1.ValidateWithConfig(cfgNotRequired)
		for _, f := range fs3 {
			if f.Level == "ERRO" && strings.Contains(strings.ToLower(f.Message), "rollback") {
				t.Errorf("erro inesperado de rollback quando require_rollback=false: %v", f)
			}
		}
	})
}

func FuzzValidate_RequireRollback(f *testing.F) {
	f.Add("true", "Rollback Plan", "passos de rollback")
	f.Add("false", "", "")
	f.Add("", "Reversibilidade", "reverter imediatamente")
	f.Add("true", "Plano de Rollback", "N/A")
	f.Add("1", "Rollback Plan", "123")
	f.Add("invalid", "Rollback Plan", "todo")

	f.Fuzz(func(t *testing.T, reqRollback, secName, body string) {
		raw := `---
delivery_id: DEL-001
task_code: TASK-42
title: Fuzz Rollback
status: done
require_rollback: ` + reqRollback + `
---

# Contexto
Contexto

# Escopo
internal/test.go

# Implementação
Implementacao

# Testes
` + "```\nPASS\n```" + `

# Code Review
OK

# Final Checklist
- [x] item 1
`
		if secName != "" {
			raw += "\n# " + secName + "\n" + body + "\n"
		}

		rep, err := Parse(raw)
		if err != nil {
			return
		}
		_ = rep.Validate()
	})
}
