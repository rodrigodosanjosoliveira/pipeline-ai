// Command pipeline-ai é a CLI unificada do toolkit de governança de agentes pipeline-ai.
//
// Suporta os subcomandos:
//   - init: inicializa novo projeto no padrão (desktop ou api)
//   - hook: executa o hook PreToolUse que bloqueia edição direta
//   - validate: valida relatório de delivery e salvaguardas de git
//   - sync: sincroniza/atualiza skills e regras (.agents/) em projetos existentes
//   - delivery: gerencia a delivery ativa (open, close, status)
//   - doctor: diagnostica o ambiente local e configurações do projeto
//   - version: exibe a versão instalada
//   - help: exibe mensagem de ajuda
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate"
)

const version = "v0.5.0"

func main() {
	os.Exit(runWithStdin(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithStdin(args, os.Stdin, stdout, stderr)
}

func runWithStdin(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	cmd := strings.ToLower(strings.TrimSpace(args[0]))
	cmdArgs := args[1:]

	switch cmd {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0

	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "pipeline-ai %s\n", version)
		return 0

	case "hook":
		return hook.Run(stdin, stdout, stderr)

	case "validate":
		return runValidate(cmdArgs, stdout, stderr)

	case "init":
		return runInit(cmdArgs, stdout, stderr)

	case "sync":
		return runSync(cmdArgs, stdout, stderr)

	case "delivery":
		return runDelivery(cmdArgs, stdout, stderr)

	case "doctor":
		return runDoctor(cmdArgs, stdout, stderr)

	case "mcp":
		return runMCP(cmdArgs, stdin, stdout, stderr)

	default:
		fmt.Fprintf(stderr, "pipeline-ai: comando desconhecido: %q\nExecute 'pipeline-ai help' para ver a lista de comandos.\n", cmd)
		return 1
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Uso: pipeline-ai <comando> [argumentos]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Comandos disponíveis:")
	fmt.Fprintln(w, "  init       Inicializa um novo projeto com o padrão pipeline-ai")
	fmt.Fprintln(w, "  hook       Executa o hook PreToolUse que impede edição direta em caminhos protegidos")
	fmt.Fprintln(w, "  validate   Valida o relatório de delivery e salvaguardas de segurança antes do PR")
	fmt.Fprintln(w, "  sync       Sincroniza skills, regras e hooks em .agents/ a partir dos templates")
	fmt.Fprintln(w, "  delivery   Gerencia o ciclo de vida da delivery ativa (open, close, status)")
	fmt.Fprintln(w, "  doctor     Diagnostica o ambiente local (Go 1.24+, git, gh, hooks e .pipeline-ai.json)")
	fmt.Fprintln(w, "  mcp        Executa o servidor MCP sobre stdio (JSON-RPC 2.0)")
	fmt.Fprintln(w, "  version    Exibe a versão do pipeline-ai")
	fmt.Fprintln(w, "  help       Exibe esta mensagem de ajuda")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Execute 'pipeline-ai <comando> --help' para detalhes sobre opções específicas.")
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	baseRef := fs.String("base", "", "referência git para o diff (default: base_ref do config)")
	noGit := fs.Bool("no-git", false, "pula as checagens que dependem do git")

	if err := fs.Parse(args); err != nil {
		return validate.ExitBadUsage
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(stderr, "uso: pipeline-ai validate [flags] <caminho-do-relatorio.md>\n")
		return validate.ExitBadUsage
	}

	return validate.Run(fs.Arg(0), *baseRef, *noGit, stdout, stderr)
}

func runSync(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto a ser sincronizado")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := scaffold.Sync(*dir, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "ERRO: %v\n", err)
		return 1
	}
	return 0
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto a ser verificado")
	asJSON := fs.Bool("json", false, "emite o diagnóstico estruturado em formato JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *asJSON {
		return doctor.RunJSON(*dir, stdout)
	}
	return doctor.Run(*dir, stdout, stderr)
}

func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	s := mcp.NewServer(*dir)
	if err := s.ServeStream(stdin, stdout, stderr); err != nil && err != io.EOF {
		fmt.Fprintf(stderr, "mcp: erro no servidor: %v\n", err)
		return 1
	}
	return 0
}

func runDelivery(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printDeliveryUsage(stderr)
		return 1
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	subArgs := args[1:]

	switch sub {
	case "open":
		return runDeliveryOpen(subArgs, stdout, stderr)
	case "close":
		return runDeliveryClose(subArgs, stdout, stderr)
	case "status":
		return runDeliveryStatus(subArgs, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "pipeline-ai delivery: subcomando desconhecido %q (use open, close ou status)\n", sub)
		return 1
	}
}

func printDeliveryUsage(w io.Writer) {
	fmt.Fprintln(w, "Uso: pipeline-ai delivery <open|close|status> [opções]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcomandos:")
	fmt.Fprintln(w, "  open <DEL-ID> [-task <TASK_CODE>] [-ttl <duration>] [-dir <dir>]")
	fmt.Fprintln(w, "  close [-dir <dir>]")
	fmt.Fprintln(w, "  status [-dir <dir>]")
}

func runDeliveryOpen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("delivery open", flag.ContinueOnError)
	fs.SetOutput(stderr)

	task := fs.String("task", "", "código da tarefa (opcional)")
	ttlStr := fs.String("ttl", "168h", "tempo de validade (TTL), ex: 2h, 48h, 7d")
	dir := fs.String("dir", ".", "diretório raiz do projeto")

	// Reorganiza argumentos para permitir flags antes ou depois do delivery_id
	var flagArgs []string
	var posArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	if len(posArgs) == 0 || strings.TrimSpace(posArgs[0]) == "" {
		fmt.Fprintln(stderr, "ERRO: delivery_id é obrigatório. Exemplo: pipeline-ai delivery open DEL-001")
		return 1
	}

	delivID := strings.TrimSpace(posArgs[0])

	ttl, err := time.ParseDuration(*ttlStr)
	if err != nil {
		fmt.Fprintf(stderr, "ERRO: ttl %q inválido: %v\n", *ttlStr, err)
		return 1
	}

	if err := state.Open(*dir, delivID, *task, ttl); err != nil {
		fmt.Fprintf(stderr, "ERRO ao abrir delivery: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Delivery %s aberta com sucesso (TTL: %v).\n", delivID, ttl)
	return 0
}

func runDeliveryClose(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("delivery close", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := state.Close(*dir); err != nil {
		fmt.Fprintf(stderr, "ERRO ao fechar delivery: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Delivery ativa fechada com sucesso.")
	return 0
}

func runDeliveryStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("delivery status", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	d, err := state.Get(*dir)
	if err != nil || d == nil {
		fmt.Fprintln(stdout, "nenhuma delivery ativa")
		return 0
	}

	active, actErr := state.IsActive(*dir)
	if !active {
		if errors.Is(actErr, state.ErrExpired) {
			fmt.Fprintf(stdout, "Delivery ativa: %s (EXPIRADA em %s)\n", d.DeliveryID, d.ExpiresAt.Format(time.RFC3339))
			if d.TaskCode != "" {
				fmt.Fprintf(stdout, "Task:           %s\n", d.TaskCode)
			}
			return 0
		}
		fmt.Fprintln(stdout, "nenhuma delivery ativa")
		return 0
	}

	fmt.Fprintf(stdout, "Delivery ativa: %s\n", d.DeliveryID)
	if d.TaskCode != "" {
		fmt.Fprintf(stdout, "Task:           %s\n", d.TaskCode)
	}
	fmt.Fprintf(stdout, "Aberta em:      %s\n", d.OpenedAt.Format(time.RFC3339))
	fmt.Fprintf(stdout, "Expira em:      %s\n", d.ExpiresAt.Format(time.RFC3339))
	return 0
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts scaffold.Options
	fs.StringVar(&opts.Name, "name", "", "Nome do projeto (obrigatório)")
	fs.StringVar(&opts.Type, "type", "", "Formato do projeto: desktop, api ou cli")
	fs.StringVar(&opts.Database, "db", "", "Banco de dados: postgres, mongo ou dynamo (default: postgres)")
	fs.StringVar(&opts.Module, "module", "", "Caminho do módulo Go (ex: github.com/user/repo)")
	fs.StringVar(&opts.TargetDir, "dir", ".", "Diretório de destino (padrão: diretório atual)")
	fs.StringVar(&opts.Summary, "summary", "", "Uma frase descritiva sobre o projeto")
	noGit := fs.Bool("no-git", false, "Não inicializa repositório Git")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	opts.InitGit = !*noGit
	opts.Stdout = stdout
	opts.Stderr = stderr

	// Se executado sem flags interativas e name não foi fornecido
	if opts.Name == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Nome do projeto: ")
		text, _ := reader.ReadString('\n')
		opts.Name = strings.TrimSpace(text)
	}

	if opts.Type == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("\nEscolha o formato do projeto:")
		fmt.Println("  1) desktop (Go + Wails v3 + React num binário único)")
		fmt.Println("  2) api     (Servidor Go API + UI React)")
		fmt.Println("  3) cli     (Aplicação de linha de comando ou terminal em Go)")
		fmt.Print("Opção [1/2/3, default: 1]: ")
		text, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(text)
		switch choice {
		case "2", "api":
			opts.Type = "api"
		case "3", "cli", "tui":
			opts.Type = "cli"
		default:
			opts.Type = "desktop"
		}
	}

	if opts.Database == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("\nEscolha o banco de dados:")
		fmt.Println("  1) PostgreSQL (Relacional)")
		fmt.Println("  2) MongoDB    (Documento)")
		fmt.Println("  3) DynamoDB   (NoSQL Key-Value)")
		fmt.Print("Opção [1/2/3, default: 1]: ")
		text, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(text)
		switch choice {
		case "2", "mongo", "mongodb":
			opts.Database = "MongoDB"
		case "3", "dynamo", "dynamodb":
			opts.Database = "DynamoDB"
		default:
			opts.Database = "PostgreSQL"
		}
	}

	if opts.Module == "" {
		opts.Module = opts.Name
	}

	fmt.Fprintf(stdout, "\nInicializando projeto %q (%s, %s)...\n", opts.Name, opts.Type, opts.Database)
	if err := scaffold.Scaffold(opts); err != nil {
		fmt.Fprintf(stderr, "\nERRO ao inicializar projeto: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "\nSucesso! Projeto configurado com o padrão pipeline-ai.")
	fmt.Fprintf(stdout, "Diretório: %s\n\n", opts.TargetDir)
	return 0
}
