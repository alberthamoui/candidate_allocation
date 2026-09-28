package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ==================================================
// =================== HELPERS ======================
// ==================================================

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[WARN] erro ao escrever resposta JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// ==================================================
// =================== HANDLERS =====================
// ==================================================

// POST /api/upload — recebe o arquivo Excel e cria uma sessão.
// Retorna { sessionId, mapping }.
func (store *SessionStore) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, 400, "erro ao parsear form: "+err.Error())
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "campo 'file' não encontrado: "+err.Error())
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, 400, "erro ao ler arquivo: "+err.Error())
		return
	}

	nOpcoes := 5
	if v := r.FormValue("nOpcoes"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 10 {
			writeError(w, 400, fmt.Sprintf("número de opções inválido: %q (esperado de 1 a 10)", v))
			return
		}
		nOpcoes = n
	}
	emailDomain := r.FormValue("emailDomain")
	if emailDomain == "" {
		emailDomain = "@al.insper.edu.br"
	}

	id, sess := store.Create()
	mapping, err := sess.SuggestMapping(data, nOpcoes, emailDomain)
	if err != nil {
		store.Delete(id)
		writeError(w, 400, "erro ao processar arquivo: "+err.Error())
		return
	}

	writeJSON(w, 200, map[string]any{
		"sessionId": id,
		"mapping":   mapping,
	})
}

// POST /api/build-usuarios
func (store *SessionStore) handleBuildUsuarios(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	var items []MappingItem
	if err := decodeJSON(r, &items); err != nil {
		writeError(w, 400, "body inválido: "+err.Error())
		return
	}
	result, err := sess.BuildUsuariosWithMapping(items)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

// POST /api/save-usuarios
func (store *SessionStore) handleSaveUsuarios(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	var data []Usuario
	if err := decodeJSON(r, &data); err != nil {
		writeError(w, 400, "body inválido: "+err.Error())
		return
	}
	if err := sess.SaveUsuarios(data); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// POST /api/suggest-avaliador
func (store *SessionStore) handleSuggestAvaliador(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	result, err := sess.SuggestMappingAvaliador()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

// POST /api/build-avaliadores
func (store *SessionStore) handleBuildAvaliadores(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	var items []MappingItem
	if err := decodeJSON(r, &items); err != nil {
		writeError(w, 400, "body inválido")
		return
	}
	result, err := sess.BuildAvaliadoresWithMapping(items)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

// POST /api/save-avaliadores
func (store *SessionStore) handleSaveAvaliadores(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	var data []AvaliadorInfo
	if err := decodeJSON(r, &data); err != nil {
		writeError(w, 400, "body inválido")
		return
	}
	if err := sess.SaveAvaliadores(data); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// POST /api/suggest-restricao
func (store *SessionStore) handleSuggestRestricao(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	result, err := sess.SuggestMappingRestricao()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

// POST /api/build-restricoes
func (store *SessionStore) handleBuildRestricoes(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	var items []MappingItem
	if err := decodeJSON(r, &items); err != nil {
		writeError(w, 400, "body inválido")
		return
	}
	result, err := sess.BuildRestricoesWithMapping(items)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

// POST /api/save-restricoes
func (store *SessionStore) handleSaveRestricoes(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	var data []Restricao
	if err := decodeJSON(r, &data); err != nil {
		writeError(w, 400, "body inválido")
		return
	}
	if err := sess.SaveRestricoes(data); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// GET /api/alocar?sessionId=xxx — Server-Sent Events com progresso + resultado final.
func (store *SessionStore) handleAlocar(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Query().Get("sessionId")
	sess := store.Get(sessionId)
	if sess == nil {
		http.Error(w, "sessão não encontrada", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	emit := func(v any) {
		b, _ := json.Marshal(v)
		// se o cliente desconectou, não há a quem avisar
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	param, err := lerParametros(r.URL.Query())
	if err != nil {
		emit(map[string]string{"error": err.Error()})
		return
	}
	result, err := sess.RunAlocacao(param, emit)
	if err != nil {
		emit(map[string]string{"error": err.Error()})
		return
	}

	type doneMsg struct {
		Done   bool             `json:"done"`
		Result AlocacaoResponse `json:"result"`
	}
	emit(doneMsg{Done: true, Result: result})
}

// lerParametros lê os parâmetros da alocação da query string; os que faltarem
// ficam com o valor padrão.
func lerParametros(q url.Values) (ParametrosAlocacao, error) {
	pa := parametrosAlocacaoPadrao()
	campos := []struct {
		nome string
		dst  *int
	}{
		{"mesas_por_horario", &pa.MesasPorHorario},
		{"min_pessoas_por_mesa", &pa.MinPessoasPorMesa},
		{"max_pessoas_por_mesa", &pa.MaxPessoasPorMesa},
		{"avaliadores_por_mesa", &pa.AvaliadoresPorMesa},
	}
	for _, c := range campos {
		if v := q.Get(c.nome); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return pa, fmt.Errorf("parâmetro %s inválido: %q", c.nome, v)
			}
			*c.dst = n
		}
	}
	return pa, pa.validar()
}

// GET /api/capacidade?mesas_por_horario=...&min_pessoas_por_mesa=...
// Prévia do que cabe com os parâmetros; sem parâmetros, usa os padrões.
func (store *SessionStore) handleCapacidade(w http.ResponseWriter, r *http.Request) {
	sess := store.sessionFromRequest(r)
	if sess == nil {
		writeError(w, 401, "sessão não encontrada")
		return
	}
	param, err := lerParametros(r.URL.Query())
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, sess.PreviaCapacidade(param))
}

// GET /api/export?sessionId=xxx — download do arquivo Excel.
func (store *SessionStore) handleExport(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Query().Get("sessionId")
	if sessionId == "" {
		sessionId = r.Header.Get("X-Session-Id")
	}
	sess := store.Get(sessionId)
	if sess == nil {
		http.Error(w, "sessão não encontrada", http.StatusNotFound)
		return
	}
	data, err := sess.ExportResultado()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="alocacao.xlsx"`)
	w.Write(data)
}

// GET /api/exemplo — download do arquivo Excel de exemplo.
func handleExemplo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="base_exemplo.xlsx"`)
	w.Write(exemploXLSX)
}

// DELETE /api/session — encerra a sessão atual.
func (store *SessionStore) handleReset(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Session-Id")
	if id != "" {
		store.Delete(id)
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ==================================================
// =================== ROUTER =======================
// ==================================================

func buildRouter(store *SessionStore, distFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/upload", store.handleUpload)
	mux.HandleFunc("POST /api/build-usuarios", store.handleBuildUsuarios)
	mux.HandleFunc("POST /api/save-usuarios", store.handleSaveUsuarios)
	mux.HandleFunc("POST /api/suggest-avaliador", store.handleSuggestAvaliador)
	mux.HandleFunc("POST /api/build-avaliadores", store.handleBuildAvaliadores)
	mux.HandleFunc("POST /api/save-avaliadores", store.handleSaveAvaliadores)
	mux.HandleFunc("POST /api/suggest-restricao", store.handleSuggestRestricao)
	mux.HandleFunc("POST /api/build-restricoes", store.handleBuildRestricoes)
	mux.HandleFunc("POST /api/save-restricoes", store.handleSaveRestricoes)
	mux.HandleFunc("GET /api/capacidade", store.handleCapacidade)
	mux.HandleFunc("GET /api/alocar", store.handleAlocar)
	mux.HandleFunc("GET /api/export", store.handleExport)
	mux.HandleFunc("GET /api/exemplo", handleExemplo)
	mux.HandleFunc("GET /api/versao", handleVersao)
	mux.HandleFunc("DELETE /api/session", store.handleReset)

	// SPA: serve index.html para rotas do React Router, static assets direto do FS
	fileServer := http.FileServer(http.FS(distFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			http.ServeFileFS(w, r, distFS, "index.html")
			return
		}
		f, err := distFS.Open(path)
		if err != nil {
			// Arquivo não encontrado → SPA routing
			http.ServeFileFS(w, r, distFS, "index.html")
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})

	return mux
}
