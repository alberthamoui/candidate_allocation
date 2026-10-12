package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed frontend/dist
var frontendDist embed.FS

//go:embed Excels/base_exemplo.xlsx
var exemploXLSX []byte

func main() {
	if err := prepararRecursos(); err != nil {
		log.Fatal(err)
	}
	args := os.Args[1:]
	if len(args) > 0 && args[0] != "servir" && args[0] != "serve" {
		if err := executarCLI(args); err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(1)
		}
		return
	}

	distFS, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := NewSessionStore()
	mux := buildRouter(store, distFS)

	log.Printf("Servidor iniciado em http://localhost:%s (branch %q, commit %s)", port, versaoAtual.Branch, versaoAtual.Commit)
	if versaoAtual.Desatualizado {
		log.Printf("[WARN] O binário foi compilado de outro commit que o atual do repositório; recompile.")
	}
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func prepararRecursos() error {
	versaoAtual = carregarVersao()
	r, err := configurarRecursos(os.Getenv, cotaCPU())
	if err != nil {
		return err
	}
	aplicarRecursos(r)
	return nil
}
