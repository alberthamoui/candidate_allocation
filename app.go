package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Session guarda o estado de um usuário durante o fluxo de alocação.
type Session struct {
	mu          sync.RWMutex
	id          string
	persistente bool
	parametros  ParametrosAlocacao
	novoLote    *LoteAvisos
	busy        atomic.Int64

	db                  *sql.DB
	excelData           []byte
	nOpcoes             int
	emailDomain         string
	preferenciaDefinida bool
	usarPreferencia     bool // false: o administrador define os horários, sem ranking dos candidatos
	avisos              ConfigAvisos
	avaliacoes          map[int]string // pessoa_id → "aprovado" ou "reprovado"
	lastResult          *AlocacaoResponse
	updatedAt           time.Time
}

// As três importações usam a mesma transação para dados e estado da agenda.
func (s *Session) salvarDados(data any) error {
	if !s.persistente {
		return fillDb(s.db, data)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	switch v := data.(type) {
	case []Usuario:
		err = gravarCandidatos(tx, v)
	case []AvaliadorInfo:
		err = gravarAvaliadores(tx, v)
	case []Restricao:
		err = gravarRestricoes(tx, v)
	default:
		return fmt.Errorf("dados não suportados")
	}
	if err != nil {
		return err
	}
	if err := s.persistirTxLocked(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Session) SaveUsuarios(data []Usuario) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	processed, duplicates := processData(data, s.emailDomain)
	if len(duplicates) > 0 {
		return fmt.Errorf("candidatos duplicados: %v; candidato %q precisa de CPF/e-mail distinto ou remoção", duplicates, data[duplicates[0][len(duplicates[0])-1]-1].Nome)
	}
	for i := range data {
		data[i] = processed[i+1].Usuario
	}
	return s.salvarDados(data)
}
func (s *Session) SaveAvaliadores(data []AvaliadorInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.salvarDados(data)
}
func (s *Session) SaveRestricoes(data []Restricao) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.salvarDados(data)
}

// ==================================================
// =================== SESSION STORE ================
// ==================================================

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	// vagasAlocacao limita quantas alocações rodam ao mesmo tempo
	vagasAlocacao chan struct{}
	dir           string
	stop          chan struct{}
	closeOnce     sync.Once
}

// errServidorCheio: já há recursos.MaxSessoes sessões abertas.
var errServidorCheio = errors.New("o servidor está com muitas sessões abertas; tente de novo em alguns minutos")

func NewSessionStore() *SessionStore {
	s := &SessionStore{
		sessions:      make(map[string]*Session),
		stop:          make(chan struct{}),
		vagasAlocacao: make(chan struct{}, recursos.AlocacoesSimultaneas),
	}
	go s.cleanup()
	return s
}

func newSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // sem aleatoriedade não dá para criar ids de sessão seguros
	}
	return hex.EncodeToString(b)
}

// Create abre um banco :memory: exclusivo para a sessão e inicializa o schema.
func (s *SessionStore) Create() (string, *Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictLocked(time.Now(), true)
	cheio := len(s.sessions) >= recursos.MaxSessoes
	if cheio {
		return "", nil, errServidorCheio
	}
	id := newSessionID()
	db, err := s.abrirBanco(id)
	if err != nil {
		return "", nil, err
	}
	// MaxOpenConns=1 garante que o pool sempre reutilize a mesma conexão,
	// preservando o banco em memória entre queries.
	db.SetMaxOpenConns(1)
	if err := setupConn(db); err != nil {
		_ = db.Close()
		return "", nil, err
	}

	sess := &Session{
		id: id, persistente: s.dir != "",
		db:              db,
		updatedAt:       time.Now(),
		usarPreferencia: true,
		avaliacoes:      map[int]string{},
	}

	if err := sess.persistirLocked(); err != nil {
		_ = db.Close()
		return "", nil, err
	}
	s.sessions[id] = sess
	return id, sess, nil
}

// Get retorna a sessão e atualiza o timestamp de acesso.
func (s *SessionStore) Get(id string) *Session {
	s.mu.Lock()
	sess := s.sessions[id]
	if sess == nil && s.dir != "" && idValido.MatchString(id) {
		s.evictLocked(time.Now(), true)
		if len(s.sessions) >= recursos.MaxSessoes {
			s.mu.Unlock()
			return nil
		}
		var err error
		sess, err = s.carregarSessao(id)
		if err != nil {
			log.Printf("[WARN] recuperando sessão: %v", err)
		} else {
			s.sessions[id] = sess
			sess.retomarPendentes()
		}
	}
	if sess != nil {
		sess.updatedAt = time.Now()
	}
	s.mu.Unlock()
	return sess
}

// Delete encerra o banco e remove a sessão do mapa.
func (s *SessionStore) Delete(id string) error {
	s.mu.Lock()
	if sess, ok := s.sessions[id]; ok {
		if sess.busy.Load() != 0 {
			s.mu.Unlock()
			return fmt.Errorf("aguarde a operação ou o envio terminar antes de apagar a agenda")
		}
		sess.mu.Lock()
		fecharBanco(sess)
		sess.mu.Unlock()
		delete(s.sessions, id)
	}
	if s.dir != "" && idValido.MatchString(id) {
		if err := os.Remove(filepath.Join(s.dir, id+".db")); err != nil && !os.IsNotExist(err) {
			s.mu.Unlock()
			return fmt.Errorf("não foi possível apagar a agenda: %w", err)
		}
	}
	s.mu.Unlock()
	return nil
}

// O TTL descarrega bancos persistidos sem apagar a agenda. Operações e envios
// ativos mantêm a sessão na memória até terminar.
func (s *SessionStore) evictLocked(now time.Time, capacity bool) {
	for id, sess := range s.sessions {
		if sess.busy.Load() != 0 || !sess.mu.TryLock() {
			continue
		}
		if sess.updatedAt.Before(now.Add(-15 * time.Minute)) {
			fecharBanco(sess)
			delete(s.sessions, id)
		}
		sess.mu.Unlock()
	}
	if !capacity || s.dir == "" || len(s.sessions) < recursos.MaxSessoes {
		return
	}
	var oldest string
	for id, sess := range s.sessions {
		if sess.busy.Load() == 0 && (oldest == "" || sess.updatedAt.Before(s.sessions[oldest].updatedAt)) {
			oldest = id
		}
	}
	if oldest != "" {
		sess := s.sessions[oldest]
		if sess.mu.TryLock() {
			fecharBanco(sess)
			delete(s.sessions, oldest)
			sess.mu.Unlock()
		}
	}
}
func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			s.mu.Lock()
			s.evictLocked(now, false)
			s.mu.Unlock()
		case <-s.stop:
			return
		}
	}
}
func (s *SessionStore) Close() {
	s.closeOnce.Do(func() { close(s.stop) })
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		sess.mu.Lock()
		fecharBanco(sess)
		delete(s.sessions, id)
		sess.mu.Unlock()
	}
}

func fecharBanco(sess *Session) {
	if err := sess.db.Close(); err != nil {
		log.Printf("[WARN] erro ao fechar o banco da sessão: %v", err)
	}
}

// sessionFromRequest extrai a sessão do header X-Session-Id.
func (s *SessionStore) sessionFromRequest(r *http.Request) *Session {
	if sess, ok := r.Context().Value(chaveSessao{}).(*Session); ok {
		return sess
	}
	id := r.Header.Get("X-Session-Id")
	if id == "" {
		id = r.URL.Query().Get("sessionId")
	}
	if id == "" {
		return nil
	}
	return s.Get(id)
}

type chaveSessao struct{}

func (store *SessionStore) protegerSessao(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Session-Id")
		if id == "" {
			id = r.URL.Query().Get("sessionId")
		}
		if id != "" && r.Method != http.MethodDelete {
			// Get e pin sob a mesma trava impedem o descarregamento entre os dois.
			for {
				sess := store.Get(id)
				if sess == nil {
					break
				}
				store.mu.Lock()
				if store.sessions[id] != sess {
					store.mu.Unlock()
					continue
				}
				sess.busy.Add(1)
				store.mu.Unlock()
				defer sess.busy.Add(-1)
				r = r.WithContext(context.WithValue(r.Context(), chaveSessao{}, sess))
				break
			}
		}
		if r.URL.Path != "/api/exemplo" && strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
