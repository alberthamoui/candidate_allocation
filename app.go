package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Session guarda o estado de um usuário durante o fluxo de alocação.
type Session struct {
	db          *sql.DB
	excelData   []byte
	nOpcoes     int
	emailDomain string
	lastResult  *AlocacaoResponse
	updatedAt   time.Time
}

// SaveUsuarios persiste candidatos no banco da sessão.
func (s *Session) SaveUsuarios(data []Usuario) error {
	fillDb(s.db, data)
	return nil
}

// SaveAvaliadores persiste avaliadores no banco da sessão.
func (s *Session) SaveAvaliadores(data []AvaliadorInfo) error {
	fillDb(s.db, data)
	return nil
}

// SaveRestricoes persiste restrições no banco da sessão.
func (s *Session) SaveRestricoes(data []Restricao) error {
	fillDb(s.db, data)
	return nil
}

// Reset recria o banco em memória e limpa o estado.
func (s *Session) Reset() {
	s.db.Close()
	db, _ := sql.Open("sqlite3", ":memory:")
	db.SetMaxOpenConns(1)
	setupConn(db)
	s.db = db
	s.excelData = nil
	s.lastResult = nil
	s.nOpcoes = 0
	s.emailDomain = ""
}

// ==================================================
// =================== SESSION STORE ================
// ==================================================

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	s := &SessionStore{sessions: make(map[string]*Session)}
	go s.cleanup()
	return s
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Create abre um banco :memory: exclusivo para a sessão e inicializa o schema.
func (s *SessionStore) Create() (string, *Session) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	// MaxOpenConns=1 garante que o pool sempre reutilize a mesma conexão,
	// preservando o banco em memória entre queries.
	db.SetMaxOpenConns(1)
	setupConn(db)

	id := newSessionID()
	sess := &Session{db: db, updatedAt: time.Now()}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return id, sess
}

// Get retorna a sessão e atualiza o timestamp de acesso.
func (s *SessionStore) Get(id string) *Session {
	s.mu.Lock()
	sess := s.sessions[id]
	if sess != nil {
		sess.updatedAt = time.Now()
	}
	s.mu.Unlock()
	return sess
}

// Delete encerra o banco e remove a sessão do mapa.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	if sess, ok := s.sessions[id]; ok {
		sess.db.Close()
		delete(s.sessions, id)
	}
	s.mu.Unlock()
}

// cleanup remove sessões inativas a cada 10 minutos (TTL: 90 minutos).
func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		cutoff := time.Now().Add(-90 * time.Minute)
		s.mu.Lock()
		for id, sess := range s.sessions {
			if sess.updatedAt.Before(cutoff) {
				sess.db.Close()
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// sessionFromRequest extrai a sessão do header X-Session-Id.
func (s *SessionStore) sessionFromRequest(r *http.Request) *Session {
	id := r.Header.Get("X-Session-Id")
	if id == "" {
		return nil
	}
	return s.Get(id)
}
