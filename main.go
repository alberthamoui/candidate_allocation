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

	store := NewSessionStore()
	mux := buildRouter(store, distFS)

	log.Printf("Servidor iniciado em http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
