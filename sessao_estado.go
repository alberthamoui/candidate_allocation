package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var idValido = regexp.MustCompile(`^[a-f0-9]{32}$`)

type estadoPersistido struct {
	Excel           []byte             `json:"excel,omitempty"`
	NOpcoes         int                `json:"n_opcoes"`
	Dominio         string             `json:"dominio"`
	UsarPreferencia bool               `json:"usar_preferencia"`
	Resultado       *AlocacaoResponse  `json:"resultado,omitempty"`
	Parametros      ParametrosAlocacao `json:"parametros"`
	Avisos          ConfigAvisos       `json:"avisos"`
	Avaliacoes      map[int]string     `json:"avaliacoes"`
	Historico       []LoteAvisos       `json:"historico"`
}

func NewPersistentSessionStore(dir string) (*SessionStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := NewSessionStore()
	s.dir = dir
	return s, nil
}

func (s *SessionStore) abrirBanco(id string) (*sql.DB, error) {
	path := ":memory:"
	if s.dir != "" {
		path = filepath.Join(s.dir, id+".db")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func (s *SessionStore) carregarSessao(id string) (*Session, error) {
	if _, err := os.Stat(filepath.Join(s.dir, id+".db")); err != nil {
		return nil, err
	}
	db, err := s.abrirBanco(id)
	if err != nil {
		return nil, err
	}
	if err := setupConn(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	var raw []byte
	if err := db.QueryRow("SELECT estado FROM estado_sessao WHERE id=1").Scan(&raw); err != nil {
		_ = db.Close()
		return nil, err
	}
	var e estadoPersistido
	if err := json.Unmarshal(raw, &e); err != nil {
		_ = db.Close()
		return nil, err
	}
	sess := &Session{db: db, id: id, persistente: true, excelData: e.Excel, nOpcoes: e.NOpcoes, emailDomain: e.Dominio, usarPreferencia: e.UsarPreferencia, lastResult: e.Resultado, parametros: e.Parametros, avisos: e.Avisos, avaliacoes: e.Avaliacoes, historico: e.Historico, updatedAt: time.Now()}
	if sess.avaliacoes == nil {
		sess.avaliacoes = map[int]string{}
	}
	// Um envio interrompido fica visível para repetição explícita; não reenviamos
	// mensagens possivelmente aceitas pelo provedor após um reinício.
	for i := range sess.historico {
		if sess.historico[i].Status == "enviando" {
			sess.historico[i].Status = "interrompido"
		}
	}
	return sess, nil
}

func (s *Session) persistirLocked() error {
	if !s.persistente {
		return nil
	}
	e := estadoPersistido{Excel: s.excelData, NOpcoes: s.nOpcoes, Dominio: s.emailDomain, UsarPreferencia: s.usarPreferencia, Resultado: s.lastResult, Parametros: s.parametros, Avisos: s.avisos, Avaliacoes: s.avaliacoes, Historico: s.historico}
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec("CREATE TABLE IF NOT EXISTS estado_sessao (id INTEGER PRIMARY KEY, estado BLOB NOT NULL)"); err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO estado_sessao(id,estado) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET estado=excluded.estado", raw)
	return err
}

func clonarResultado(r *AlocacaoResponse) *AlocacaoResponse {
	if r == nil {
		return nil
	}
	raw, _ := json.Marshal(r)
	var out AlocacaoResponse
	_ = json.Unmarshal(raw, &out)
	return &out
}
func (s *Session) ResultadoAtual() (*AlocacaoResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.exigirResultado(); err != nil {
		return nil, err
	}
	return clonarResultado(s.lastResult), nil
}
func (s *Session) parametrosAtuais() ParametrosAlocacao {
	p := s.parametros
	if p.MaxPessoasPorMesa == 0 {
		p = parametrosAlocacaoPadrao()
	}
	return p
}
func (s *Session) operar(fn func() (ResultadoOperacao, error), async bool) (ResultadoOperacao, error) {
	s.mu.Lock()
	antes := clonarResultado(s.lastResult)
	avaliacoes := make(map[int]string, len(s.avaliacoes))
	for k, v := range s.avaliacoes {
		avaliacoes[k] = v
	}
	historicoAntes := make([]LoteAvisos, len(s.historico))
	for i, l := range s.historico {
		historicoAntes[i] = clonarLote(l)
	}
	op, err := fn()
	if err == nil {
		err = s.persistirLocked()
	}
	if err != nil {
		s.lastResult = antes
		s.avaliacoes = avaliacoes
		s.historico = historicoAntes
	}
	cfg := s.avisos
	s.mu.Unlock()
	if err != nil {
		return ResultadoOperacao{}, err
	}
	if op.LoteID != "" {
		if async {
			go s.executarLote(op.LoteID, cfg)
		} else {
			s.executarLote(op.LoteID, cfg)
			lote, _ := s.Lote(op.LoteID)
			op.Avisos = lote.Avisos
			op.Envio = &lote
		}
	}
	return op, nil
}
func (s *Session) AvaliarLote(itens []itemAvaliacao) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) {
		var op ResultadoOperacao
		for _, item := range itens {
			var err error
			op, err = s.avaliarLocked(item.Pessoa, item.ID, item.Resultado)
			if err != nil {
				return op, err
			}
		}
		return op, nil
	}, false)
}
func (s *Session) ConfigurarAvisos(cfg ConfigAvisos) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.Email.Senha == "" && cfg.Email.Usuario == s.avisos.Email.Usuario && cfg.Email.Host == s.avisos.Email.Host {
		cfg.Email.Senha = s.avisos.Email.Senha
	}
	if cfg.WhatsApp.Token == "" && cfg.WhatsApp.Provedor == s.avisos.WhatsApp.Provedor {
		cfg.WhatsApp.Token = s.avisos.WhatsApp.Token
	}
	cfg.enviarEmail, cfg.enviarWhats = s.avisos.enviarEmail, s.avisos.enviarWhats
	if err := validarMensagem(cfg.Mensagem); err != nil {
		return err
	}
	antes := s.avisos
	s.avisos = cfg
	if err := s.persistirLocked(); err != nil {
		s.avisos = antes
		return err
	}
	return nil
}
func (s *Session) ConfigAtual() ConfigAvisos {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := s.avisos
	cfg.Email.SenhaDefinida = cfg.Email.Senha != ""
	cfg.Email.Senha = ""
	cfg.WhatsApp.TokenDefinido = cfg.WhatsApp.Token != ""
	cfg.WhatsApp.Token = ""
	return cfg
}
func erroEnvio(op ResultadoOperacao) error {
	if op.Envio != nil && op.Envio.Falhas > 0 {
		return fmt.Errorf("%d aviso(s) falharam; %d aceitos pelo provedor", op.Envio.Falhas, op.Envio.Enviados)
	}
	return nil
}

func (s *Session) ParametrosAtuais() ParametrosAlocacao {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.parametrosAtuais()
	p.SemPreferencia = p.SemPreferencia || !s.usarPreferencia
	raw, _ := json.Marshal(p)
	var out ParametrosAlocacao
	_ = json.Unmarshal(raw, &out)
	return out
}
