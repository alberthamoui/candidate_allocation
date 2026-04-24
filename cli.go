package main

import (
	"candidate_alocator/back/workflow"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

const defaultCLIOptionCount = 5

func detectCLIMode(args []string) ([]string, bool) {
	if len(args) == 0 {
		return nil, false
	}

	switch args[0] {
	case "cli", "-cli", "--cli":
		return args[1:], true
	default:
		return nil, false
	}
}

func runCLI(args []string) error {
	flagSet := flag.NewFlagSet("cli", flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)

	filePath := flagSet.String("file", "", "caminho para o arquivo .xlsx")
	optionCount := flagSet.Int("opcoes", defaultCLIOptionCount, "quantidade de opcoes de horario esperadas na aba de candidatos")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintf(flagSet.Output(), "Uso: %s cli -file arquivo.xlsx [-opcoes %d]\n", os.Args[0], defaultCLIOptionCount)
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*filePath) == "" {
		flagSet.Usage()
		return errors.New("o parametro -file e obrigatorio no modo CLI")
	}
	if *optionCount <= 0 {
		return fmt.Errorf("o parametro -opcoes deve ser maior que zero")
	}

	return workflow.RunCLI(context.Background(), *filePath, *optionCount)
}
