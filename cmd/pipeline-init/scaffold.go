package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed all:templates
var templateFS embed.FS

// Options define os parâmetros de criação de um novo projeto.
type Options struct {
	Name      string
	Type      string // "desktop" ou "api"
	Database  string // "PostgreSQL", "MongoDB", "DynamoDB"
	Module    string // caminho do módulo Go
	TargetDir string // diretório de destino
	InitGit   bool   // inicializar repositório Git
	Summary   string // frase de resumo do projeto
	Stdout    io.Writer
	Stderr    io.Writer
}

// Normalize valida e preenche valores padrão.
func (o *Options) Normalize() error {
	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" {
		return fmt.Errorf("o nome do projeto é obrigatório")
	}

	o.Type = strings.ToLower(strings.TrimSpace(o.Type))
	if o.Type != "desktop" && o.Type != "api" {
		return fmt.Errorf("tipo de projeto inválido %q: use \"desktop\" ou \"api\"", o.Type)
	}

	db := strings.ToLower(strings.TrimSpace(o.Database))
	switch {
	case strings.Contains(db, "mongo"):
		o.Database = "MongoDB"
	case strings.Contains(db, "dynamo"):
		o.Database = "DynamoDB"
	default:
		o.Database = "PostgreSQL"
	}

	if o.Module == "" {
		o.Module = o.Name
	}

	if o.TargetDir == "" {
		o.TargetDir = "."
	}

	if o.Summary == "" {
		o.Summary = fmt.Sprintf("Aplicação %s desenvolvida com o padrão pipeline-ai.", o.Type)
	}

	if o.Stdout == nil {
		o.Stdout = io.Discard
	}
	if o.Stderr == nil {
		o.Stderr = io.Discard
	}

	return nil
}

// Scaffold gera todo o projeto a partir dos templates embutidos.
func Scaffold(opts Options) error {
	if err := opts.Normalize(); err != nil {
		return err
	}

	absTarget, err := filepath.Abs(opts.TargetDir)
	if err != nil {
		return fmt.Errorf("caminho de destino inválido: %w", err)
	}

	if err := os.MkdirAll(absTarget, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório destino: %w", err)
	}

	// 1. Criar estrutura de diretórios do formato
	if err := createDirSkeleton(absTarget, opts.Type); err != nil {
		return err
	}

	// 2. Copiar assets estáticos embutidos (.agents e docs-ai)
	if err := copyEmbeddedDir("templates/agents", filepath.Join(absTarget, ".agents")); err != nil {
		return fmt.Errorf("falha ao copiar .agents: %w", err)
	}
	if err := copyEmbeddedDir("templates/docs-ai", filepath.Join(absTarget, "docs-ai")); err != nil {
		return fmt.Errorf("falha ao copiar docs-ai: %w", err)
	}

	// 3. Copiar .pipeline-ai.json específico
	jsonTemplate := "templates/pipeline-ai.desktop.json"
	if opts.Type == "api" {
		jsonTemplate = "templates/pipeline-ai.api.json"
	}
	if err := copyEmbeddedFile(jsonTemplate, filepath.Join(absTarget, ".pipeline-ai.json")); err != nil {
		return fmt.Errorf("falha ao copiar .pipeline-ai.json: %w", err)
	}

	// 4. Copiar .gitignore e .githooks/pre-push
	if err := copyEmbeddedFile("templates/gitignore.tmpl", filepath.Join(absTarget, ".gitignore")); err != nil {
		return fmt.Errorf("falha ao copiar .gitignore: %w", err)
	}
	githooksDir := filepath.Join(absTarget, ".githooks")
	if err := os.MkdirAll(githooksDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar .githooks: %w", err)
	}
	prePushDest := filepath.Join(githooksDir, "pre-push")
	if err := copyEmbeddedFile("templates/pre-push.tmpl", prePushDest); err != nil {
		return fmt.Errorf("falha ao copiar pre-push hook: %w", err)
	}
	_ = os.Chmod(prePushDest, 0755)

	// 5. Gerar go.mod
	goModContent := fmt.Sprintf(`module %s

go 1.24

tool (
	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook
	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate
)
`, opts.Module)
	if err := os.WriteFile(filepath.Join(absTarget, "go.mod"), []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("falha ao escrever go.mod: %w", err)
	}

	// 6. Gerar Taskfile.yml
	taskfileTemplate := "templates/Taskfile.desktop.yml"
	if opts.Type == "api" {
		taskfileTemplate = "templates/Taskfile.api.yml"
	}
	if err := copyEmbeddedFile(taskfileTemplate, filepath.Join(absTarget, "Taskfile.yml")); err != nil {
		return fmt.Errorf("falha ao copiar Taskfile.yml: %w", err)
	}

	// 7. Gerar main.go
	if opts.Type == "desktop" {
		data, err := templateFS.ReadFile("templates/main.desktop.go.tmpl")
		if err != nil {
			return err
		}
		rendered := strings.ReplaceAll(string(data), "{{NOME_DO_PROJETO}}", opts.Name)
		if err := os.WriteFile(filepath.Join(absTarget, "main.go"), []byte(rendered), 0644); err != nil {
			return err
		}
	} else {
		data, err := templateFS.ReadFile("templates/main.api.go.tmpl")
		if err != nil {
			return err
		}
		rendered := strings.ReplaceAll(string(data), "{{NOME_DO_PROJETO}}", opts.Name)
		if err := os.WriteFile(filepath.Join(absTarget, "cmd", "api", "main.go"), []byte(rendered), 0644); err != nil {
			return err
		}
	}

	// 8. Gerar AGENTS.md e GEMINI.md interpolados
	if err := renderMarkdownDoc("templates/AGENTS.md.tmpl", filepath.Join(absTarget, "AGENTS.md"), opts); err != nil {
		return err
	}
	if err := renderMarkdownDoc("templates/GEMINI.md.tmpl", filepath.Join(absTarget, "GEMINI.md"), opts); err != nil {
		return err
	}

	// 9. Inicializar Git se solicitado
	if opts.InitGit {
		if err := initGitRepo(absTarget, opts.Stdout, opts.Stderr); err != nil {
			fmt.Fprintf(opts.Stderr, "aviso: não foi possível inicializar git: %v\n", err)
		}
	}

	return nil
}

func createDirSkeleton(root, projType string) error {
	dirs := []string{
		"frontend/src",
		"internal/domain",
		"internal/storage",
		"internal/config",
		"docs-ai/deliveries/_template",
	}

	if projType == "desktop" {
		dirs = append(dirs, "build", "internal/app")
	} else {
		dirs = append(dirs, "cmd/api", "internal/http", "migrations")
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			return fmt.Errorf("falha ao criar pasta %s: %w", d, err)
		}
	}
	return nil
}

func copyEmbeddedFile(srcPath, destPath string) error {
	data, err := templateFS.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("falha ao ler template %s: %w", srcPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0644)
}

func copyEmbeddedDir(srcDir, destDir string) error {
	return fs.WalkDir(templateFS, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func renderMarkdownDoc(tmplPath, destPath string, opts Options) error {
	data, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("falha ao ler %s: %w", tmplPath, err)
	}

	tree := ""
	if opts.Type == "desktop" {
		tree = `.agents/  .pipeline-ai.json  docs-ai/  build/
frontend/src/
internal/{app,domain,storage,config}/
main.go  Taskfile.yml  go.mod  go.sum`
	} else {
		tree = `.agents/  .pipeline-ai.json  docs-ai/
cmd/api/main.go
internal/{http,domain,storage,config}/
frontend/src/  migrations/
Taskfile.yml  go.mod  go.sum`
	}

	s := string(data)
	s = strings.ReplaceAll(s, "{{NOME_DO_PROJETO}}", opts.Name)
	s = strings.ReplaceAll(s, "{{UMA_FRASE_SOBRE_O_QUE_O_PROJETO_FAZ}}", opts.Summary)
	s = strings.ReplaceAll(s, "{{desktop | api}}", opts.Type)
	s = strings.ReplaceAll(s, "{{PostgreSQL | MongoDB | DynamoDB}}", opts.Database)
	s = strings.ReplaceAll(s, "{{ARVORE_DE_DIRETORIOS}}", tree)

	return os.WriteFile(destPath, []byte(s), 0644)
}

func initGitRepo(root string, stdout, stderr io.Writer) error {
	gitDir := filepath.Join(root, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return nil // já inicializado
	}

	runCmd := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		return cmd.Run()
	}

	if err := runCmd("init", "-b", "main"); err != nil {
		if err := runCmd("init"); err != nil {
			return err
		}
	}
	_ = runCmd("config", "core.hooksPath", ".githooks")
	_ = runCmd("add", ".")
	_ = runCmd("commit", "-m", "feat: inicializar projeto com padrao pipeline-ai")
	return nil
}
