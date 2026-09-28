package main

import (
	"embed"
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
	distFS, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	versaoAtual = carregarVersao()
	r, err := configurarRecursos(os.Getenv, cotaCPU())
	if err != nil {
		log.Fatal(err)
	}
	aplicarRecursos(r)
	store := NewSessionStore()
	mux := buildRouter(store, distFS)

	log.Printf("Servidor iniciado em http://localhost:%s (branch %q, commit %s)", port, versaoAtual.Branch, versaoAtual.Commit)
	if versaoAtual.Desatualizado {
		log.Printf("[WARN] O binário foi compilado de outro commit que o atual do repositório; recompile.")
	}
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
