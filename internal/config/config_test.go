package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

const validBody = `{
  "version": 1,
  "project_type": "desktop",
  "protected_paths": ["internal/", "cmd/", "frontend/src/"],
  "editor_agents": ["dev-implementation", "test-writer"],
  "security_surface_paths": ["internal/auth/", "go.mod"]
}`

func TestFindSobeAteAraiz(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, validBody)
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := Find(deep)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	want := filepath.Join(root, FileName)
	if got != want {
		t.Errorf("Find = %q, quer %q", got, want)
	}
}

func TestFindSemConfig(t *testing.T) {
	// t.TempDir() fica sob /tmp; se existir um .pipeline-ai.json acima, o teste
	// perde o sentido. Checamos o tipo de erro, não a ausência absoluta.
	dir := t.TempDir()
	if _, err := Find(dir); err == nil {
		t.Skip("existe um " + FileName + " acima de " + dir + "; nada a testar")
	} else if err != ErrNotFound {
		t.Errorf("erro = %v, quer ErrNotFound", err)
	}
}

func TestLoadValidacao(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "config válida",
			body: validBody,
		},
		{
			name: "config válida cli",
			body: `{"version":1,"project_type":"cli","protected_paths":["internal/","cmd/"]}`,
		},
		{
			name: "enabled_modules válidos",
			body: `{"version":1,"project_type":"api","protected_paths":["internal/"],"enabled_modules":["data-model","release-ops"]}`,
		},
		{
			name:    "enabled_modules com módulo inválido",
			body:    `{"version":1,"project_type":"api","protected_paths":["internal/"],"enabled_modules":["modulo-inexistente"]}`,
			wantErr: true,
		},
		{
			name:    "enabled_modules com item vazio",
			body:    `{"version":1,"project_type":"api","protected_paths":["internal/"],"enabled_modules":[""]}`,
			wantErr: true,
		},
		{
			name:    "project_type ausente",
			body:    `{"version":1,"protected_paths":["internal/"]}`,
			wantErr: true,
		},
		{
			name:    "project_type inválido",
			body:    `{"version":1,"project_type":"mobile","protected_paths":["internal/"]}`,
			wantErr: true,
		},
		{
			name:    "protected_paths vazio",
			body:    `{"version":1,"project_type":"api","protected_paths":[]}`,
			wantErr: true,
		},
		{
			name:    "version futura",
			body:    `{"version":99,"project_type":"api","protected_paths":["internal/"]}`,
			wantErr: true,
		},
		{
			name:    "JSON malformado",
			body:    `{"version":1,`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.body)
			_, err := Load(dir)
			if tt.wantErr && err == nil {
				t.Fatal("esperava erro, obteve nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
		})
	}
}

func TestDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, validBody)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DeliveriesDir != "docs-ai/deliveries" {
		t.Errorf("DeliveriesDir = %q", c.DeliveriesDir)
	}
	if c.BaseRef != "origin/main" {
		t.Errorf("BaseRef = %q", c.BaseRef)
	}
	if filepath.ToSlash(c.IndexFile) != "docs-ai/deliveries/INDEX.md" {
		t.Errorf("IndexFile = %q", c.IndexFile)
	}
}

func TestIsProtected(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, validBody)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"dentro de internal", filepath.Join(dir, "internal", "auth", "token.go"), true},
		{"dentro de cmd", filepath.Join(dir, "cmd", "app", "main.go"), true},
		{"dentro do frontend", filepath.Join(dir, "frontend", "src", "App.tsx"), true},
		{"fora dos protegidos", filepath.Join(dir, "docs-ai", "nota.md"), false},
		{"raiz do projeto", filepath.Join(dir, "README.md"), false},
		{"prefixo parcial não casa", filepath.Join(dir, "internals", "x.go"), false},
		{"fora da raiz", filepath.Join(filepath.Dir(dir), "outro", "internal", "x.go"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.IsProtected(tt.path); got != tt.want {
				t.Errorf("IsProtected(%q) = %v, quer %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsEditorAgent(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, validBody)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name  string
		agent string
		want  bool
	}{
		{"agent autorizado", "dev-implementation", true},
		{"outro autorizado", "test-writer", true},
		{"case-insensitive", "Dev-Implementation", true},
		{"com espaços", "  test-writer  ", true},
		{"agent não autorizado", "pm-triage", false},
		{"sessão principal", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.IsEditorAgent(tt.agent); got != tt.want {
				t.Errorf("IsEditorAgent(%q) = %v, quer %v", tt.agent, got, tt.want)
			}
		})
	}
}

func TestIsSecuritySurface(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, validBody)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"diretório de auth", "internal/auth/jwt.go", true},
		{"arquivo exato go.mod", "go.mod", true},
		{"fora da superfície", "internal/ui/render.go", false},
		{"go.sum não listado", "go.sum", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.IsSecuritySurface(tt.path); got != tt.want {
				t.Errorf("IsSecuritySurface(%q) = %v, quer %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsModuleEnabled(t *testing.T) {
	const body = `{
  "version": 1,
  "project_type": "api",
  "protected_paths": ["internal/"],
  "enabled_modules": ["data-model", "release-ops", "query-performance"]
}`
	dir := t.TempDir()
	writeConfig(t, dir, body)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name   string
		module string
		want   bool
	}{
		{"módulo ativo", "data-model", true},
		{"outro ativo", "release-ops", true},
		{"case-insensitive", "Query-Performance", true},
		{"com espaços", "  data-model  ", true},
		{"módulo válido mas não ativo", "migration-reviewer", false},
		{"módulo inexistente", "desconhecido", false},
		{"string vazia", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.IsModuleEnabled(tt.module); got != tt.want {
				t.Errorf("IsModuleEnabled(%q) = %v, quer %v", tt.module, got, tt.want)
			}
		})
	}
}

func TestIsModuleEnabledSemModulos(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, validBody)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.IsModuleEnabled("data-model") {
		t.Errorf("IsModuleEnabled devolveu true sem enabled_modules configurado")
	}
}

func TestIsProtectedSymlinks(t *testing.T) {
	root := t.TempDir()
	internalDir := filepath.Join(root, "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatalf("mkdir internal: %v", err)
	}
	secretFile := filepath.Join(internalDir, "secret.go")
	if err := os.WriteFile(secretFile, []byte("package auth"), 0o644); err != nil {
		t.Fatalf("write secret.go: %v", err)
	}

	symlinkDir := filepath.Join(root, "link_to_internal")
	if err := os.Symlink(internalDir, symlinkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	symlinkRoot := filepath.Join(t.TempDir(), "symroot")
	if err := os.Symlink(root, symlinkRoot); err != nil {
		t.Fatalf("symlink root: %v", err)
	}

	outsideDir := t.TempDir()
	escapeLink := filepath.Join(internalDir, "escape_link")
	if err := os.Symlink(outsideDir, escapeLink); err != nil {
		t.Fatalf("symlink escape: %v", err)
	}

	writeConfig(t, root, `{
  "version": 1,
  "project_type": "desktop",
  "protected_paths": ["internal/"]
}`)

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "arquivo existente via symlink de diretório protegido",
			path: filepath.Join(root, "link_to_internal", "secret.go"),
			want: true,
		},
		{
			name: "arquivo novo ou inexistente via symlink de diretório protegido",
			path: filepath.Join(root, "link_to_internal", "new_file.go"),
			want: true,
		},
		{
			name: "arquivo em subdiretório aninhado inexistente via symlink de diretório protegido",
			path: filepath.Join(root, "link_to_internal", "sub", "deep", "new_file.go"),
			want: true,
		},
		{
			name: "arquivo em subdiretório aninhado inexistente quando root é carregado via symlink",
			path: filepath.Join(symlinkRoot, "internal", "auth", "token.go"),
			want: true,
		},
		{
			name: "arquivo direto existente em internal",
			path: secretFile,
			want: true,
		},
		{
			name: "arquivo direto inexistente em internal",
			path: filepath.Join(internalDir, "new_file_direct.go"),
			want: true,
		},
		{
			name: "symlink dentro de diretório protegido apontando para fora da raiz",
			path: filepath.Join(internalDir, "escape_link", "outside.go"),
			want: false,
		},
		{
			name: "symlink dentro de diretório protegido com subdiretório inexistente apontando para fora da raiz",
			path: filepath.Join(internalDir, "escape_link", "sub", "deep", "outside.go"),
			want: false,
		},
		{
			name: "arquivo fora do escopo protegido",
			path: filepath.Join(root, "README.md"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.IsProtected(tt.path); got != tt.want {
				t.Errorf("IsProtected(%q) = %v, quer %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsImmutable(t *testing.T) {
	const cfgWithImmutable = `{
  "version": 1,
  "project_type": "desktop",
  "protected_paths": ["internal/", "cmd/"],
  "immutable_paths": [
    ".pipeline-ai.json",
    "AGENTS.md",
    "internal/core/",
    "docs-ai/01-INVARIANTES-GLOBAIS.md"
  ]
}`
	dir := t.TempDir()
	writeConfig(t, dir, cfgWithImmutable)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "arquivo exato .pipeline-ai.json",
			path: filepath.Join(dir, ".pipeline-ai.json"),
			want: true,
		},
		{
			name: "arquivo exato AGENTS.md",
			path: filepath.Join(dir, "AGENTS.md"),
			want: true,
		},
		{
			name: "arquivo dentro de diretorio imutavel internal/core/",
			path: filepath.Join(dir, "internal", "core", "guard.go"),
			want: true,
		},
		{
			name: "arquivo em subdiretorio dentro de internal/core/",
			path: filepath.Join(dir, "internal", "core", "sub", "rule.go"),
			want: true,
		},
		{
			name: "arquivo exato em docs-ai imutavel",
			path: filepath.Join(dir, "docs-ai", "01-INVARIANTES-GLOBAIS.md"),
			want: true,
		},
		{
			name: "outro arquivo em docs-ai nao configurado em immutable_paths",
			path: filepath.Join(dir, "docs-ai", "DECISOES.md"),
			want: false,
		},
		{
			name: "arquivo em internal mas fora de internal/core (apenas protegido, nao imutavel)",
			path: filepath.Join(dir, "internal", "auth", "token.go"),
			want: false,
		},
		{
			name: "arquivo na raiz nao imutavel",
			path: filepath.Join(dir, "README.md"),
			want: false,
		},
		{
			name: "prefixo parcial nao casa com diretorio imutavel",
			path: filepath.Join(dir, "internal", "core_extra", "file.go"),
			want: false,
		},
		{
			name: "caminho fora da raiz do projeto",
			path: filepath.Join(filepath.Dir(dir), "outro", "AGENTS.md"),
			want: false,
		},
		{
			name: "caminho vazio",
			path: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.IsImmutable(tt.path); got != tt.want {
				t.Errorf("IsImmutable(%q) = %v, quer %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadImmutablePaths(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
	}{
		{
			name: "com immutable_paths configurado",
			body: `{
				"version": 1,
				"project_type": "api",
				"protected_paths": ["internal/"],
				"immutable_paths": [".pipeline-ai.json", "AGENTS.md"]
			}`,
			wantCount: 2,
		},
		{
			name: "sem immutable_paths configurado",
			body: `{
				"version": 1,
				"project_type": "api",
				"protected_paths": ["internal/"]
			}`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.body)
			c, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if len(c.ImmutablePaths) != tt.wantCount {
				t.Errorf("len(c.ImmutablePaths) = %d, quer %d", len(c.ImmutablePaths), tt.wantCount)
			}
		})
	}
}

func TestLoadFlexibleDeliveryConfig(t *testing.T) {
	tests := []struct {
		name                string
		body                string
		wantDeliveryPattern string
		wantTaskPattern     string
		wantCustomSections  []string
		wantIsIndexRequired bool
	}{
		{
			name: "campos flexíveis configurados",
			body: `{
				"version": 1,
				"project_type": "api",
				"protected_paths": ["internal/"],
				"delivery_id_pattern": "^DEL-[0-9]{3,}$",
				"task_code_pattern": "^TASK-[0-9]+$",
				"custom_sections": ["Arquitetura", "Rollback Plan"],
				"require_index": false
			}`,
			wantDeliveryPattern: "^DEL-[0-9]{3,}$",
			wantTaskPattern:     "^TASK-[0-9]+$",
			wantCustomSections:  []string{"Arquitetura", "Rollback Plan"},
			wantIsIndexRequired: false,
		},
		{
			name: "campos flexíveis omitidos (valores default)",
			body: `{
				"version": 1,
				"project_type": "desktop",
				"protected_paths": ["internal/"]
			}`,
			wantDeliveryPattern: "",
			wantTaskPattern:     "",
			wantCustomSections:  nil,
			wantIsIndexRequired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.body)
			c, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			if c.DeliveryIDPattern != tt.wantDeliveryPattern {
				t.Errorf("DeliveryIDPattern = %q, quer %q", c.DeliveryIDPattern, tt.wantDeliveryPattern)
			}
			if c.TaskCodePattern != tt.wantTaskPattern {
				t.Errorf("TaskCodePattern = %q, quer %q", c.TaskCodePattern, tt.wantTaskPattern)
			}
			if len(c.CustomSections) != len(tt.wantCustomSections) {
				t.Fatalf("len(CustomSections) = %d, quer %d", len(c.CustomSections), len(tt.wantCustomSections))
			}
			for i, s := range tt.wantCustomSections {
				if c.CustomSections[i] != s {
					t.Errorf("CustomSections[%d] = %q, quer %q", i, c.CustomSections[i], s)
				}
			}
			if got := c.IsIndexRequired(); got != tt.wantIsIndexRequired {
				t.Errorf("IsIndexRequired() = %v, quer %v", got, tt.wantIsIndexRequired)
			}
		})
	}
}

func TestConfigPatternValidation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "delivery_id_pattern regex válida",
			body: `{
				"version": 1,
				"project_type": "api",
				"protected_paths": ["internal/"],
				"delivery_id_pattern": "^DEL-[0-9]+$"
			}`,
			wantErr: false,
		},
		{
			name: "delivery_id_pattern regex inválida",
			body: `{
				"version": 1,
				"project_type": "api",
				"protected_paths": ["internal/"],
				"delivery_id_pattern": "[a-z"
			}`,
			wantErr: true,
		},
		{
			name: "task_code_pattern regex válida",
			body: `{
				"version": 1,
				"project_type": "api",
				"protected_paths": ["internal/"],
				"task_code_pattern": "^[A-Z]+-[0-9]+$"
			}`,
			wantErr: false,
		},
		{
			name: "task_code_pattern regex inválida",
			body: `{
				"version": 1,
				"project_type": "api",
				"protected_paths": ["internal/"],
				"task_code_pattern": "(?P<"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.body)
			_, err := Load(dir)
			if tt.wantErr && err == nil {
				t.Errorf("esperava erro de validação para regex inválida, obteve nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("erro inesperado: %v", err)
			}
		})
	}
}

func TestIsIndexRequired(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "omitido no JSON retorna true",
			body: `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"]}`,
			want: true,
		},
		{
			name: "explicitamente true",
			body: `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_index": true}`,
			want: true,
		},
		{
			name: "explicitamente false",
			body: `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_index": false}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.body)
			c, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := c.IsIndexRequired(); got != tt.want {
				t.Errorf("IsIndexRequired() = %v, quer %v", got, tt.want)
			}
		})
	}
}

func TestValidateDeliveryID(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		deliveryID string
		wantErr    bool
	}{
		{
			name:       "default pattern caso válido",
			pattern:    "",
			deliveryID: "DEL-001",
			wantErr:    false,
		},
		{
			name:       "default pattern caso válido de 4 digitos",
			pattern:    "",
			deliveryID: "DEL-1234",
			wantErr:    false,
		},
		{
			name:       "default pattern com prefixo errado",
			pattern:    "",
			deliveryID: "REL-001",
			wantErr:    true,
		},
		{
			name:       "default pattern vazio",
			pattern:    "",
			deliveryID: "",
			wantErr:    true,
		},
		{
			name:       "default pattern minúsculas",
			pattern:    "",
			deliveryID: "del-001",
			wantErr:    true,
		},
		{
			name:       "custom pattern válido",
			pattern:    "^REL-[0-9]{4}$",
			deliveryID: "REL-2026",
			wantErr:    false,
		},
		{
			name:       "custom pattern não casa com DEL padrão",
			pattern:    "^REL-[0-9]{4}$",
			deliveryID: "DEL-001",
			wantErr:    true,
		},
		{
			name:       "custom pattern tamanho incompatível",
			pattern:    "^REL-[0-9]{4}$",
			deliveryID: "REL-12",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DeliveryIDPattern: tt.pattern,
			}
			err := cfg.ValidateDeliveryID(tt.deliveryID)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateDeliveryID(%q) com pattern %q esperava erro, obteve nil", tt.deliveryID, tt.pattern)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateDeliveryID(%q) com pattern %q erro inesperado: %v", tt.deliveryID, tt.pattern, err)
			}
		})
	}
}

func TestValidateTaskCode(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		taskCode string
		wantErr  bool
	}{
		{
			name:     "default pattern caso válido",
			pattern:  "",
			taskCode: "TASK-42",
			wantErr:  false,
		},
		{
			name:     "default pattern caso válido 3 dígitos",
			pattern:  "",
			taskCode: "TASK-001",
			wantErr:  false,
		},
		{
			name:     "default pattern vazio",
			pattern:  "",
			taskCode: "",
			wantErr:  true,
		},
		{
			name:     "default pattern com formato desconhecido",
			pattern:  "",
			taskCode: "INVALID",
			wantErr:  true,
		},
		{
			name:     "custom pattern válido Jira",
			pattern:  "^PROJ-[0-9]+$",
			taskCode: "PROJ-1234",
			wantErr:  false,
		},
		{
			name:     "custom pattern inválido contra TASK",
			pattern:  "^PROJ-[0-9]+$",
			taskCode: "TASK-42",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				TaskCodePattern: tt.pattern,
			}
			err := cfg.ValidateTaskCode(tt.taskCode)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateTaskCode(%q) com pattern %q esperava erro, obteve nil", tt.taskCode, tt.pattern)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateTaskCode(%q) com pattern %q erro inesperado: %v", tt.taskCode, tt.pattern, err)
			}
		})
	}
}

func FuzzValidateDeliveryID(f *testing.F) {
	f.Add("DEL-001")
	f.Add("DEL-9999")
	f.Add("TASK-001")
	f.Add("")
	f.Add("DEL-")
	f.Add("del-001")
	f.Add("DEL-001\n")
	f.Add("DEL-001; rm -rf /")

	cfg := &Config{}
	f.Fuzz(func(t *testing.T, s string) {
		// Não deve dar panic com nenhuma string arbitrária
		_ = cfg.ValidateDeliveryID(s)
	})
}

func TestConfig_RequireRollback(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantRequired bool
	}{
		{
			name:         "omitido assume default false",
			body:         `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"]}`,
			wantRequired: false,
		},
		{
			name:         "explicitamente false",
			body:         `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_rollback": false}`,
			wantRequired: false,
		},
		{
			name:         "explicitamente true",
			body:         `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_rollback": true}`,
			wantRequired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.body)

			cfg, err := Load(dir)
			if err != nil {
				t.Fatalf("Load falhou: %v", err)
			}

			if got := cfg.IsRollbackRequired(); got != tt.wantRequired {
				t.Errorf("cfg.IsRollbackRequired() = %v, quer %v", got, tt.wantRequired)
			}
		})
	}

	t.Run("nil receiver safe", func(t *testing.T) {
		var nilCfg *Config
		if nilCfg.IsRollbackRequired() != false {
			t.Errorf("nilCfg.IsRollbackRequired() deveria retornar false")
		}
	})
}

func TestConfig_ModuleManagement(t *testing.T) {
	t.Run("EnableModule", func(t *testing.T) {
		dir := t.TempDir()
		body := `{
			"version": 1,
			"project_type": "desktop",
			"protected_paths": ["internal/"],
			"enabled_modules": ["release-ops"]
		}`
		writeConfig(t, dir, body)

		cfg, err := Load(dir)
		if err != nil {
			t.Fatalf("Load falhou: %v", err)
		}

		// 1. Ativa módulo válido
		if err := cfg.EnableModule("data-model"); err != nil {
			t.Fatalf("EnableModule(\"data-model\") retornou erro: %v", err)
		}
		if !cfg.IsModuleEnabled("data-model") {
			t.Errorf("módulo data-model deveria estar habilitado")
		}

		// 2. Idempotência: ativar novamente não duplica
		countBefore := len(cfg.EnabledModules)
		if err := cfg.EnableModule("data-model"); err != nil {
			t.Fatalf("EnableModule repetido falhou: %v", err)
		}
		if len(cfg.EnabledModules) != countBefore {
			t.Errorf("EnableModule duplicou módulo: len antes=%d, depois=%d", countBefore, len(cfg.EnabledModules))
		}

		// 3. Normalização case-insensitive e trim
		if err := cfg.EnableModule("  agent-auditor  "); err != nil {
			t.Fatalf("EnableModule com espaços falhou: %v", err)
		}
		if !cfg.IsModuleEnabled("agent-auditor") {
			t.Errorf("módulo agent-auditor deveria estar habilitado")
		}

		// 4. Erro ao tentar ativar módulo inexistente/inválido
		invalidCases := []string{
			"modulo-desconhecido",
			"",
			"   ",
			"../escape",
		}
		for _, inv := range invalidCases {
			if err := cfg.EnableModule(inv); err == nil {
				t.Errorf("EnableModule(%q) esperava erro, retornou nil", inv)
			}
		}

		// 5. Nil receiver safe
		var nilCfg *Config
		if err := nilCfg.EnableModule("data-model"); err == nil {
			t.Errorf("nilCfg.EnableModule esperava erro, retornou nil")
		}
	})

	t.Run("DisableModule", func(t *testing.T) {
		dir := t.TempDir()
		body := `{
			"version": 1,
			"project_type": "desktop",
			"protected_paths": ["internal/"],
			"enabled_modules": ["release-ops", "data-model"]
		}`
		writeConfig(t, dir, body)

		cfg, err := Load(dir)
		if err != nil {
			t.Fatalf("Load falhou: %v", err)
		}

		// 1. Desativa módulo existente
		if err := cfg.DisableModule("release-ops"); err != nil {
			t.Fatalf("DisableModule(\"release-ops\") retornou erro: %v", err)
		}
		if cfg.IsModuleEnabled("release-ops") {
			t.Errorf("módulo release-ops deveria estar desabilitado")
		}

		// 2. Idempotência: desativar módulo já inativo não falha
		if err := cfg.DisableModule("release-ops"); err != nil {
			t.Errorf("DisableModule repetido retornou erro: %v", err)
		}
		if err := cfg.DisableModule("query-performance"); err != nil {
			t.Errorf("DisableModule em módulo nunca ativado retornou erro: %v", err)
		}

		// 3. Case-insensibilidade ao desativar
		if err := cfg.DisableModule("DATA-MODEL"); err != nil {
			t.Errorf("DisableModule case-insensitive retornou erro: %v", err)
		}
		if cfg.IsModuleEnabled("data-model") {
			t.Errorf("módulo data-model ainda está ativo")
		}

		// 4. Nil receiver safe
		var nilCfg *Config
		if err := nilCfg.DisableModule("data-model"); err == nil {
			t.Errorf("nilCfg.DisableModule esperava erro, retornou nil")
		}
	})

	t.Run("Save", func(t *testing.T) {
		dir := t.TempDir()
		body := `{
			"version": 1,
			"project_type": "desktop",
			"protected_paths": ["internal/", "cmd/"],
			"editor_agents": ["dev-implementation", "test-writer"],
			"enabled_modules": ["release-ops"]
		}`
		writeConfig(t, dir, body)

		cfg, err := Load(dir)
		if err != nil {
			t.Fatalf("Load falhou: %v", err)
		}

		// Habilita novo módulo e salva
		if err := cfg.EnableModule("context-loader"); err != nil {
			t.Fatalf("EnableModule falhou: %v", err)
		}

		if err := cfg.Save(); err != nil {
			t.Fatalf("Save falhou: %v", err)
		}

		// Recarrega do disco e valida persistência
		reloaded, err := Load(dir)
		if err != nil {
			t.Fatalf("Load após Save falhou: %v", err)
		}

		if !reloaded.IsModuleEnabled("context-loader") {
			t.Errorf("módulo context-loader não persistido após Save")
		}
		if !reloaded.IsModuleEnabled("release-ops") {
			t.Errorf("módulo release-ops perdido após Save")
		}
		if reloaded.ProjectType != "desktop" {
			t.Errorf("project_type corrompido: %q", reloaded.ProjectType)
		}
		if len(reloaded.ProtectedPaths) != 2 {
			t.Errorf("protected_paths corrompido: %v", reloaded.ProtectedPaths)
		}

		// Nil receiver e root vazio
		var nilCfg *Config
		if err := nilCfg.Save(); err == nil {
			t.Errorf("nilCfg.Save esperava erro, obteve nil")
		}

		emptyRootCfg := &Config{}
		if err := emptyRootCfg.Save(); err == nil {
			t.Errorf("Save com root vazio esperava erro, obteve nil")
		}
	})
}

func FuzzModuleManagement(f *testing.F) {
	f.Add("data-model")
	f.Add("release-ops")
	f.Add("query-performance")
	f.Add("migration-reviewer")
	f.Add("agent-auditor")
	f.Add("context-loader")
	f.Add("unknown-mod")
	f.Add("")
	f.Add("   ")
	f.Add("../path/traversal")
	f.Add("DATA-MODEL")

	dir := f.TempDir()
	cfg := &Config{
		root:           dir,
		Version:        1,
		ProjectType:    "desktop",
		ProtectedPaths: []string{"internal/"},
	}

	f.Fuzz(func(t *testing.T, mod string) {
		// Enable e Disable não devem entrar em panic com nenhuma entrada
		_ = cfg.EnableModule(mod)
		_ = cfg.DisableModule(mod)
	})
}
