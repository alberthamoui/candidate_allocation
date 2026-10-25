package main

import (
	"candidate_alocator/back/allocation"
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
	if len(args) > 0 && args[0] == "agenda" {
		return runAgendaCLI(args[1:])
	}
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

	a := NewApp()
	a.synchronousNotices = true
	defer a.clearTransientState()
	return workflow.RunCLIWithResult(context.Background(), *filePath, *optionCount, func(run allocation.ConfiguredAllocationResult) error {
		if err := a.saveConfiguredAgenda(run, buildUIAllocationResult(run)); err != nil {
			return err
		}
		a.maybeAutoNotify()
		return nil
	})
}
