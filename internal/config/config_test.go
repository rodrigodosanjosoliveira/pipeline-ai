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
