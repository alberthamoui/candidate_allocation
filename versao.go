package main

import (
	"net/http"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
)

// gitBranch pode ser fixado no build (ex.: no Docker, onde não há git):
//
//	go build -ldflags "-X main.gitBranch=dev" .
var gitBranch string

// VersaoInfo identifica o código que o servidor está rodando.
type VersaoInfo struct {
	Branch        string `json:"branch"`
	Commit        string `json:"commit"`
	Modificado    bool   `json:"modificado"`    // compilado com mudanças não commitadas
	Desatualizado bool   `json:"desatualizado"` // o repositório está em outro commit que o do binário
}

var versaoAtual VersaoInfo

// carregarVersao junta o commit gravado pelo `go build` no binário (o que foi
// de fato compilado) com a branch atual do repositório. Se o repositório
// estiver em outro commit, o binário está desatualizado. Sem git (ex.:
// Docker no Render) usa RENDER_GIT_BRANCH/RENDER_GIT_COMMIT; com `go run`,
// que não grava o commit, usa o que houver.
func carregarVersao() VersaoInfo {
	var v VersaoInfo
	var commitBinario string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				commitBinario = s.Value
			case "vcs.modified":
				v.Modificado = s.Value == "true"
			}
		}
	}
	commitRepo := git("rev-parse", "HEAD")

	v.Branch = gitBranch
	if v.Branch == "" {
		v.Branch = git("branch", "--show-current")
	}
	v.Commit = commitBinario
	if v.Commit == "" {
		v.Commit = commitRepo
	}
	// no Render não há git no container, mas ele informa branch e commit
	if v.Branch == "" {
		v.Branch = os.Getenv("RENDER_GIT_BRANCH")
	}
	if v.Commit == "" {
		v.Commit = os.Getenv("RENDER_GIT_COMMIT")
	}
	v.Commit = v.Commit[:min(7, len(v.Commit))]
	v.Desatualizado = commitBinario != "" && commitRepo != "" && commitBinario != commitRepo
	return v
}

// git roda um comando git na pasta atual; devolve "" se falhar.
func git(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GET /api/versao
func handleVersao(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, versaoAtual)
}
