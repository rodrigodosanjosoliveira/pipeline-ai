package plugin_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin"
)

func TestValidateManifest(t *testing.T) {
	validManifest := plugin.Manifest{
		Name:        "pipeline-ai",
		Description: "Governança determinística de esteiras com múltiplos agentes em IA",
		DisplayName: "Pipeline AI",
		Version:     "0.9.0",
		SuggestedPrompts: []string{
			"Iniciar nova entrega com pm-triage",
			"Validar delivery antes do PR",
		},
		Repository: "https://github.com/rodrigodosanjosoliveira/pipeline-ai",
	}

	validJSON, err := json.Marshal(validManifest)
	if err != nil {
		t.Fatalf("falha ao serializar manifesto válido: %v", err)
	}

	makeJSONWithModifier := func(modify func(m *plugin.Manifest)) []byte {
		m := validManifest
		modify(&m)
		data, _ := json.Marshal(m)
		return data
	}

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "manifesto válido completo com 2 prompts",
			data:    validJSON,
			wantErr: false,
		},
		{
			name: "manifesto válido com suggestedPrompts vazio",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.SuggestedPrompts = []string{}
			}),
			wantErr: false,
		},
		{
			name: "manifesto válido com limite de 3 suggestedPrompts",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.SuggestedPrompts = []string{"prompt 1", "prompt 2", "prompt 3"}
			}),
			wantErr: false,
		},
		{
			name: "inválido quando suggestedPrompts excede 3 itens",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.SuggestedPrompts = []string{"p1", "p2", "p3", "p4"}
			}),
			wantErr: true,
			errMsg:  "suggestedPrompts",
		},
		{
			name: "inválido quando campo name está vazio",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.Name = ""
			}),
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "inválido quando campo name contém apenas espaços",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.Name = "   "
			}),
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "inválido quando campo description está vazio",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.Description = ""
			}),
			wantErr: true,
			errMsg:  "description",
		},
		{
			name: "inválido quando campo displayName está vazio",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.DisplayName = ""
			}),
			wantErr: true,
			errMsg:  "displayName",
		},
		{
			name: "inválido quando campo version está vazio",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.Version = ""
			}),
			wantErr: true,
			errMsg:  "version",
		},
		{
			name: "inválido quando campo repository está vazio",
			data: makeJSONWithModifier(func(m *plugin.Manifest) {
				m.Repository = ""
			}),
			wantErr: true,
			errMsg:  "repository",
		},
		{
			name:    "JSON malformado",
			data:    []byte(`{"name": "teste", "description": `),
			wantErr: true,
		},
		{
			name:    "payload vazio",
			data:    []byte(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := plugin.ValidateManifest(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateManifest() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
					t.Errorf("ValidateManifest() erro = %q não contém esperado %q", err.Error(), tt.errMsg)
				}
				return
			}
			if m == nil {
				t.Fatalf("ValidateManifest() retornou manifest nil para caso válido")
			}
			if m.Name != "pipeline-ai" {
				t.Errorf("manifest.Name = %q, esperado %q", m.Name, "pipeline-ai")
			}
		})
	}
}

func FuzzValidateManifest(f *testing.F) {
	// Seed inicial válido
	validManifest := plugin.Manifest{
		Name:             "pipeline-ai",
		Description:      "Governança determinística de esteiras com múltiplos agentes em IA",
		DisplayName:      "Pipeline AI",
		Version:          "0.9.0",
		SuggestedPrompts: []string{"prompt 1", "prompt 2"},
		Repository:       "https://github.com/rodrigodosanjosoliveira/pipeline-ai",
	}
	validJSON, _ := json.Marshal(validManifest)
	f.Add(validJSON)

	// Seeds com bordas
	f.Add([]byte("{}"))
	f.Add([]byte(`{"name": "", "version": "1.0.0"}`))
	f.Add([]byte(`{"suggestedPrompts": ["1", "2", "3", "4"]}`))
	f.Add([]byte(`invalid json text string`))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		// O validador nunca deve entrar em pânico com dados arbitrários
		_, _ = plugin.ValidateManifest(data)
	})
}

// helper para criar estrutura de bundle completa e válida em diretório temporário
func setupValidBundle(t *testing.T) string {
	t.Helper()
	bundleDir := t.TempDir()

	// 1. plugin.json
	manifest := plugin.Manifest{
		Name:             "pipeline-ai",
		Description:      "Kit de governança",
		DisplayName:      "Pipeline AI",
		Version:          "0.9.0",
		SuggestedPrompts: []string{"Iniciar entrega"},
		Repository:       "https://github.com/rodrigodosanjosoliveira/pipeline-ai",
	}
	mData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("falha ao gerar json do manifesto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "plugin.json"), mData, 0644); err != nil {
		t.Fatalf("falha ao gravar plugin.json: %v", err)
	}

	// 2. rules/AGENTS.md
	rulesDir := filepath.Join(bundleDir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("falha ao criar pasta rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "AGENTS.md"), []byte("# Regras do bundle\n"), 0644); err != nil {
		t.Fatalf("falha ao gravar AGENTS.md: %v", err)
	}

	// 3. mcp_config.json
	if err := os.WriteFile(filepath.Join(bundleDir, "mcp_config.json"), []byte(`{"mcpServers":{}}`), 0644); err != nil {
		t.Fatalf("falha ao gravar mcp_config.json: %v", err)
	}

	// 4. hooks.json
	if err := os.WriteFile(filepath.Join(bundleDir, "hooks.json"), []byte(`{"hooks":{}}`), 0644); err != nil {
		t.Fatalf("falha ao gravar hooks.json: %v", err)
	}

	// 5. skills/ com catálogo de skills (pelo menos 1 skill com SKILL.md)
	skillDir := filepath.Join(bundleDir, "skills", "pm-triage")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("falha ao criar pasta de skill: %v", err)
	}
	skillContent := "---\nname: pm-triage\ndescription: Triage de demandas\n---\n# PM Triage\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("falha ao gravar SKILL.md: %v", err)
	}

	return bundleDir
}

func TestVerifyBundle(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
		errMsg  string
	}{
		{
			name: "bundle completo e válido",
			setup: func(t *testing.T) string {
				return setupValidBundle(t)
			},
			wantErr: false,
		},
		{
			name: "bundle sem plugin.json",
			setup: func(t *testing.T) string {
				dir := setupValidBundle(t)
				_ = os.Remove(filepath.Join(dir, "plugin.json"))
				return dir
			},
			wantErr: true,
			errMsg:  "plugin.json",
		},
		{
			name: "bundle sem rules/AGENTS.md",
			setup: func(t *testing.T) string {
				dir := setupValidBundle(t)
				_ = os.Remove(filepath.Join(dir, "rules", "AGENTS.md"))
				return dir
			},
			wantErr: true,
			errMsg:  "AGENTS.md",
		},
		{
			name: "bundle sem mcp_config.json",
			setup: func(t *testing.T) string {
				dir := setupValidBundle(t)
				_ = os.Remove(filepath.Join(dir, "mcp_config.json"))
				return dir
			},
			wantErr: true,
			errMsg:  "mcp_config.json",
		},
		{
			name: "bundle sem hooks.json",
			setup: func(t *testing.T) string {
				dir := setupValidBundle(t)
				_ = os.Remove(filepath.Join(dir, "hooks.json"))
				return dir
			},
			wantErr: true,
			errMsg:  "hooks.json",
		},
		{
			name: "bundle sem catálogo de skills",
			setup: func(t *testing.T) string {
				dir := setupValidBundle(t)
				_ = os.RemoveAll(filepath.Join(dir, "skills"))
				return dir
			},
			wantErr: true,
			errMsg:  "skills",
		},
		{
			name: "bundle com pasta skills vazia",
			setup: func(t *testing.T) string {
				dir := setupValidBundle(t)
				_ = os.RemoveAll(filepath.Join(dir, "skills"))
				_ = os.Mkdir(filepath.Join(dir, "skills"), 0755)
				return dir
			},
			wantErr: true,
			errMsg:  "skills",
		},
		{
			name: "bundle apontando para diretório inexistente",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "diretorio-inexistente")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundleDir := tt.setup(t)
			err := plugin.VerifyBundle(bundleDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("VerifyBundle() erro = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
					t.Errorf("VerifyBundle() erro = %q não contém %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestInstallToDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (srcDir, targetDir string)
		verify  func(t *testing.T, srcDir, targetDir string)
		wantErr bool
	}{
		{
			name: "instalação completa em diretório limpo",
			setup: func(t *testing.T) (string, string) {
				src := setupValidBundle(t)
				target := filepath.Join(t.TempDir(), "installed-plugin")
				return src, target
			},
			verify: func(t *testing.T, srcDir, targetDir string) {
				// Verifica arquivos essenciais instalados no target
				requiredFiles := []string{
					"plugin.json",
					filepath.Join("rules", "AGENTS.md"),
					"mcp_config.json",
					"hooks.json",
					filepath.Join("skills", "pm-triage", "SKILL.md"),
				}
				for _, rf := range requiredFiles {
					p := filepath.Join(targetDir, rf)
					info, err := os.Stat(p)
					if err != nil {
						t.Errorf("arquivo obrigatório não instalado: %s (erro: %v)", rf, err)
						continue
					}
					if info.IsDir() {
						t.Errorf("esperado arquivo, obtido diretório: %s", rf)
					}
				}

				// Compara conteúdo do plugin.json
				srcContent, _ := os.ReadFile(filepath.Join(srcDir, "plugin.json"))
				targetContent, _ := os.ReadFile(filepath.Join(targetDir, "plugin.json"))
				if string(srcContent) != string(targetContent) {
					t.Errorf("conteúdo de plugin.json diverge entre src e target")
				}
			},
			wantErr: false,
		},
		{
			name: "instalação idempotente em diretório já existente com arquivos",
			setup: func(t *testing.T) (string, string) {
				src := setupValidBundle(t)
				target := filepath.Join(t.TempDir(), "idempotent-plugin")
				// Primeira instalação
				if err := plugin.InstallToDir(src, target); err != nil {
					t.Fatalf("primeira chamada de InstallToDir falhou: %v", err)
				}
				return src, target
			},
			verify: func(t *testing.T, srcDir, targetDir string) {
				// Após segunda instalação (executada no teste), arquivos devem continuar íntegros
				manifestPath := filepath.Join(targetDir, "plugin.json")
				if _, err := os.Stat(manifestPath); err != nil {
					t.Fatalf("plugin.json ausente após instalação idempotente: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "origem inexistente deve retornar erro",
			setup: func(t *testing.T) (string, string) {
				return filepath.Join(t.TempDir(), "origem-inexistente"), filepath.Join(t.TempDir(), "target")
			},
			verify:  nil,
			wantErr: true,
		},
		{
			name: "origem com bundle corrompido/incompleto deve retornar erro",
			setup: func(t *testing.T) (string, string) {
				src := setupValidBundle(t)
				_ = os.Remove(filepath.Join(src, "plugin.json"))
				target := filepath.Join(t.TempDir(), "target")
				return src, target
			},
			verify:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcDir, targetDir := tt.setup(t)
			err := plugin.InstallToDir(srcDir, targetDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("InstallToDir() erro = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, srcDir, targetDir)
			}
		})
	}
}

func TestOfficialPluginBundleVersion(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "plugin", "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("falha ao ler %s: %v", manifestPath, err)
	}
	m, err := plugin.ValidateManifest(data)
	if err != nil {
		t.Fatalf("ValidateManifest(%s) retornou erro: %v", manifestPath, err)
	}
	if m.Version != "0.9.0" {
		t.Errorf("plugin/plugin.json Version = %q, esperado %q", m.Version, "0.9.0")
	}
}
