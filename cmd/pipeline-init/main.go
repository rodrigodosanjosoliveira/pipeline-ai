// Command pipeline-init inicializa um novo projeto configurado para o padrão pipeline-ai.
//
// Pode ser executado via:
//
//	go run github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init@latest
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold"
)

func main() {
	var opts scaffold.Options

	flag.StringVar(&opts.Name, "name", "", "Nome do projeto (obrigatório)")
	flag.StringVar(&opts.Type, "type", "", "Formato do projeto: desktop ou api")
	flag.StringVar(&opts.Database, "db", "", "Banco de dados: postgres, mongo ou dynamo (default: postgres)")
	flag.StringVar(&opts.Module, "module", "", "Caminho do módulo Go (ex: github.com/user/repo)")
	flag.StringVar(&opts.TargetDir, "dir", ".", "Diretório de destino (padrão: diretório atual)")
	flag.StringVar(&opts.Summary, "summary", "", "Uma frase descritiva sobre o projeto")

	noGit := flag.Bool("no-git", false, "Não inicializa repositório Git")
	flag.Parse()

	opts.InitGit = !*noGit
	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr

	// Modo interativo caso flags essenciais não tenham sido fornecidas
	reader := bufio.NewReader(os.Stdin)

	if opts.Name == "" {
		fmt.Print("Nome do projeto: ")
		text, _ := reader.ReadString('\n')
		opts.Name = strings.TrimSpace(text)
	}

	if opts.Type == "" {
		fmt.Println("\nEscolha o formato do projeto:")
		fmt.Println("  1) desktop (Go + Wails v3 + React num binário único)")
		fmt.Println("  2) api     (Servidor Go API + UI React)")
		fmt.Print("Opção [1/2, default: 1]: ")
		text, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(text)
		if choice == "2" || strings.ToLower(choice) == "api" {
			opts.Type = "api"
		} else {
			opts.Type = "desktop"
		}
	}

	if opts.Database == "" {
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
		fmt.Printf("\nCaminho do módulo Go [default: %s]: ", opts.Name)
		text, _ := reader.ReadString('\n')
		mod := strings.TrimSpace(text)
		if mod != "" {
			opts.Module = mod
		} else {
			opts.Module = opts.Name
		}
	}

	fmt.Printf("\nInicializando projeto %q (%s, %s)...\n", opts.Name, opts.Type, opts.Database)
	if err := scaffold.Scaffold(opts); err != nil {
		fmt.Fprintf(os.Stderr, "\nERRO ao inicializar projeto: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nSucesso! Projeto configurado com o padrão pipeline-ai.")
	fmt.Printf("Diretório: %s\n\n", opts.TargetDir)
	fmt.Println("Próximos passos:")
	fmt.Println("  1. cd", opts.TargetDir)
	fmt.Println("  2. Abra no Antigravity ou Antigravity IDE")
	fmt.Println("  3. Abra sua primeira demanda com a skill pm-triage")
}
