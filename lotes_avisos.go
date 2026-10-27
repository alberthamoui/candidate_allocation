package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

type LoteAvisos struct {
	config      ConfigAvisos
	Total       int           `json:"total"`
	Pendentes   int           `json:"pendentes"`
	Incertos    int           `json:"incertos"`
	Origem      string        `json:"origem,omitempty"`
	RepetidoPor string        `json:"repetido_por,omitempty"`
	ID          string        `json:"id"`
	Criado      time.Time     `json:"criado"`
	Status      string        `json:"status"`
	Avisos      []AvisoGerado `json:"avisos,omitempty"`
	Enviados    int           `json:"enviados"`
	Falhas      int           `json:"falhas"`
}

func (s *Session) prepararLoteLocked(ids []int) LoteAvisos {
	lote := LoteAvisos{ID: newSessionID(), Criado: time.Now(), config: s.avisos, Status: "enviando", Avisos: []AvisoGerado{}}
	alvo := map[int]bool{}
	for _, id := range ids {
		alvo[id] = true
	}
	avaliadores := map[string]bool{}
	cfg := s.avisos
	for _, m := range s.lastResult.Mesas {
		afetada := false
		for _, c := range m.Candidatos {
			if alvo[c.ID] {
				afetada = true
				lote.Avisos = append(lote.Avisos, prepararAvisos(cfg, c, m)...)
			}
		}
		if afetada && cfg.Email.Ativo {
			for _, a := range m.Avaliadores {
				key := strconv.Itoa(a.ID) + "|" + m.DiaNome
				if avaliadores[key] {
					continue
				}
				avaliadores[key] = true
				c := CandidatoResultado{ID: -a.ID, Nome: a.Nome, EmailPessoal: a.Email}
				lista := prepararAvisos(cfg, c, m)
				for _, av := range lista {
					if av.Canal == "email" {
						lote.Avisos = append(lote.Avisos, av)
					}
				}
			}
		}
	}
	return lote
}
func prepararAvisos(cfg ConfigAvisos, c CandidatoResultado, m MesaResult) []AvisoGerado {
	texto := renderMensagem(cfg.Mensagem, c, m)
	base := AvisoGerado{PessoaID: c.ID, Horario: m.DiaNome, MesaID: m.ID, Nome: c.Nome, Mensagem: texto, Variaveis: variaveisMensagem(c, m)}
	if !cfg.Ativo {
		base.Canal = "nenhum"
		base.Erro = "avisos estão desligados"
		return []AvisoGerado{base}
	}
	out := []AvisoGerado{}
	if cfg.Email.Ativo {
		a := base
		a.Canal = "email"
		a.Destino = emailDe(c)
		if a.Destino == "" {
			a.Erro = "destinatário sem e-mail"
		}
		out = append(out, a)
	}
	if cfg.WhatsApp.Ativo {
		a := base
		a.Canal = "whatsapp"
		telefone := c.Telefone
		var err error
		a.Destino, err = normalizarTelefone(telefone)
		if err != nil {
			a.Erro = err.Error()
		}
		out = append(out, a)
	}
	if len(out) == 0 {
		base.Canal = "nenhum"
		base.Erro = "ative um canal de envio"
		out = append(out, base)
	}
	return out
}

// A agenda e o lote autorizado são gravados na mesma transação. Cada entrega
// muda de estado isoladamente; o histórico nunca faz parte do snapshot da agenda.
func setupAvisos(db *sql.DB) error {
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS estado_sessao(id INTEGER PRIMARY KEY, estado BLOB NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS lotes(id TEXT PRIMARY KEY, criado TEXT NOT NULL, origem TEXT UNIQUE, repetido_por TEXT NOT NULL DEFAULT '', config BLOB NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS entregas(id INTEGER PRIMARY KEY, lote TEXT NOT NULL REFERENCES lotes(id), payload BLOB NOT NULL, status TEXT NOT NULL, erro TEXT NOT NULL DEFAULT '')`,
		`CREATE INDEX IF NOT EXISTS entregas_lote_status ON entregas(lote,status)`,
	} {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
func inserirLote(tx *sql.Tx, l LoteAvisos) error {
	var origem any
	if l.Origem != "" {
		origem = l.Origem
	}
	rawCfg, err := json.Marshal(l.config)
	if err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO lotes(id,criado,origem,config) VALUES(?,?,?,?)", l.ID, l.Criado.UTC().Format(time.RFC3339Nano), origem, rawCfg); err != nil {
		return err
	}
	if l.Origem != "" {
		if _, err := tx.Exec("UPDATE lotes SET repetido_por=? WHERE id=? AND repetido_por=''", l.ID, l.Origem); err != nil {
			return err
		}
	}
	for _, a := range l.Avisos {
		raw, err := json.Marshal(a)
		if err != nil {
			return err
		}
		status := "pendente"
		if a.Erro != "" {
			status = "falhou"
		}
		if _, err := tx.Exec("INSERT INTO entregas(lote,payload,status,erro) VALUES(?,?,?,?)", l.ID, raw, status, a.Erro); err != nil {
			return err
		}
	}
	return nil
}

// Erros depois de começar a transmitir podem significar aceitação pelo provedor.
// Esses registros ficam incertos e nunca entram na repetição automática de falhas.
type envioIncerto struct{ err error }

func (e envioIncerto) Error() string { return "Aceitação incerta: " + e.err.Error() }
func (e envioIncerto) Unwrap() error { return e.err }
func (s *Session) iniciarLote(id string, cfg ConfigAvisos) {
	s.busy.Add(1)
	go func() { defer s.busy.Add(-1); s.executarLote(id, cfg) }()
}
func (s *Session) retomarPendentes() {
	rows, err := s.db.Query("SELECT DISTINCT lote FROM entregas WHERE status='pendente'")
	if err != nil {
		log.Printf("[ERRO] consultando envios pendentes: %v", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		log.Printf("[ERRO] recuperando envios: %v", err)
		return
	}
	for _, id := range ids {
		s.iniciarLote(id, s.avisos)
	}
}
func (s *Session) executarLote(id string, cfg ConfigAvisos) {
	var rawCfg []byte
	if err := s.db.QueryRow("SELECT config FROM lotes WHERE id=?", id).Scan(&rawCfg); err != nil {
		log.Printf("[ERRO] lendo canal autorizado: %v", err)
		return
	}
	var autorizado ConfigAvisos
	if err := json.Unmarshal(rawCfg, &autorizado); err != nil {
		log.Printf("[ERRO] lendo configuração autorizada: %v", err)
		return
	}
	autorizado.enviarEmail, autorizado.enviarWhats = cfg.enviarEmail, cfg.enviarWhats
	cfg = autorizado
	for {
		var deliveryID int64
		var raw []byte
		err := s.db.QueryRow("SELECT id,payload FROM entregas WHERE lote=? AND status='pendente' ORDER BY id LIMIT 1", id).Scan(&deliveryID, &raw)
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			log.Printf("[ERRO] lendo envio: %v", err)
			return
		}
		var a AvisoGerado
		if err = json.Unmarshal(raw, &a); err != nil {
			log.Printf("[ERRO] lendo mensagem: %v", err)
			return
		}
		claimed, err := s.db.Exec("UPDATE entregas SET status='enviando' WHERE id=? AND status='pendente'", deliveryID)
		if err != nil {
			log.Printf("[ERRO] reservando envio: %v", err)
			return
		}
		n, err := claimed.RowsAffected()
		if err != nil {
			log.Printf("[ERRO] reservando envio: %v", err)
			return
		}
		if n == 0 {
			continue
		}
		switch a.Canal {
		case "email":
			enviar := cfg.enviarEmail
			if enviar == nil {
				enviar = enviarEmailSMTP
			}
			err = enviar(cfg.Email, a.Destino, assuntoAviso, a.Mensagem)
		case "whatsapp":
			if cfg.enviarWhats != nil {
				err = cfg.enviarWhats(cfg.WhatsApp, a.Destino, a.Mensagem)
			} else {
				err = enviarWhatsAppVariaveis(cfg.WhatsApp, a.Destino, a.Mensagem, a.Variaveis)
			}
		default:
			err = fmt.Errorf("nenhum canal configurado")
		}
		status, msg := "aceito", ""
		if err != nil {
			status = "falhou"
			msg = err.Error()
			var uncertain envioIncerto
			if errors.As(err, &uncertain) {
				status = "incerto"
			}
		}
		if _, err := s.db.Exec("UPDATE entregas SET status=?,erro=? WHERE id=? AND status='enviando'", status, msg, deliveryID); err != nil {
			log.Printf("[ERRO] registrando retorno do provedor; envio %d permanece incerto: %v", deliveryID, err)
			return
		}
	}
}
func (s *Session) Lote(id string) (LoteAvisos, error) { return s.lote(id, true) }
func (s *Session) lote(id string, detalhes bool) (LoteAvisos, error) {
	l := LoteAvisos{ID: id}
	var criado string
	err := s.db.QueryRow("SELECT criado,repetido_por,COALESCE(origem,'') FROM lotes WHERE id=?", id).Scan(&criado, &l.RepetidoPor, &l.Origem)
	if err != nil {
		return l, fmt.Errorf("envio não encontrado: %w", err)
	}
	l.Criado, err = time.Parse(time.RFC3339Nano, criado)
	if err != nil {
		return l, err
	}
	rows, err := s.db.Query("SELECT status,COUNT(*) FROM entregas WHERE lote=? GROUP BY status", id)
	if err != nil {
		return l, err
	}
	for rows.Next() {
		var status string
		var n int
		if err = rows.Scan(&status, &n); err != nil {
			break
		}
		l.Total += n
		switch status {
		case "aceito":
			l.Enviados += n
		case "falhou":
			l.Falhas += n
		case "incerto":
			l.Incertos += n
		default:
			l.Pendentes += n
		}
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return l, err
	}
	l.Status = "concluido"
	switch {
	case l.Pendentes > 0:
		l.Status = "enviando"
	case l.Incertos > 0:
		l.Status = "interrompido"
	case l.Falhas == l.Total && l.Total > 0:
		l.Status = "falhou"
	case l.Falhas > 0:
		l.Status = "parcial"
	}
	if detalhes {
		rows, err = s.db.Query("SELECT id,payload,status,erro FROM entregas WHERE lote=? ORDER BY id", id)
		if err != nil {
			return l, err
		}
		defer rows.Close()
		l.Avisos = []AvisoGerado{}
		for rows.Next() {
			var a AvisoGerado
			var raw []byte
			var did int64
			var status, msg string
			if err := rows.Scan(&did, &raw, &status, &msg); err != nil {
				return l, err
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return l, err
			}
			a.ID = did
			a.Status = status
			a.Enviado = status == "aceito"
			a.Erro = msg
			l.Avisos = append(l.Avisos, a)
		}
		if err := rows.Err(); err != nil {
			return l, err
		}
	}
	return l, nil
}

type PaginaHistorico struct {
	Lotes   []LoteAvisos `json:"lotes"`
	Proximo int64        `json:"proximo,omitempty"`
}

func (s *Session) HistoricoPagina(antes int64, limite int) (PaginaHistorico, error) {
	if limite < 1 || limite > 50 {
		limite = 20
	}
	if antes <= 0 {
		antes = 1 << 62
	}
	rows, err := s.db.Query("SELECT rowid,id FROM lotes WHERE rowid<? ORDER BY rowid DESC LIMIT ?", antes, limite+1)
	if err != nil {
		return PaginaHistorico{}, err
	}
	type ref struct {
		row int64
		id  string
	}
	var refs []ref
	for rows.Next() {
		var r ref
		if err = rows.Scan(&r.row, &r.id); err != nil {
			break
		}
		refs = append(refs, r)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return PaginaHistorico{}, err
	}
	out := PaginaHistorico{Lotes: []LoteAvisos{}}
	if len(refs) > limite {
		out.Proximo = refs[limite-1].row
		refs = refs[:limite]
	}
	for _, r := range refs {
		l, err := s.lote(r.id, false)
		if err != nil {
			return out, err
		}
		out.Lotes = append(out.Lotes, l)
	}
	return out, nil
}
func (s *Session) RepetirFalhas(id string, async bool) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) {
		if err := s.exigirResultado(); err != nil {
			return ResultadoOperacao{}, err
		}
		l, err := s.Lote(id)
		if err != nil {
			return ResultadoOperacao{}, err
		}
		if l.Pendentes > 0 {
			return ResultadoOperacao{}, fmt.Errorf("envio ainda em andamento")
		}
		if l.RepetidoPor != "" {
			return ResultadoOperacao{}, fmt.Errorf("falhas já encaminhadas ao envio %s", l.RepetidoPor)
		}
		novo := LoteAvisos{ID: newSessionID(), Origem: id, Criado: time.Now(), config: s.avisos, Status: "enviando", Avisos: []AvisoGerado{}}
		for _, a := range l.Avisos {
			if a.Status != "falhou" {
				continue
			}
			// Recria usando contato, mensagem e agenda atuais. Nunca reutiliza data antiga.
			var c CandidatoResultado
			var mesa MesaResult
			found := false
			for _, m := range s.lastResult.Mesas {
				for _, cand := range m.Candidatos {
					if cand.ID == a.PessoaID {
						c, mesa, found = cand, m, true
					}
				}
				for _, av := range m.Avaliadores {
					if -av.ID == a.PessoaID {
						c, mesa, found = CandidatoResultado{ID: -av.ID, Nome: av.Nome, EmailPessoal: av.Email}, m, true
					}
				}
			}
			if !found {
				return ResultadoOperacao{}, fmt.Errorf("%s não está mais alocado; prepare um novo aviso", a.Nome)
			}
			for _, fresh := range prepararAvisos(s.avisos, c, mesa) {
				if fresh.Canal == a.Canal {
					novo.Avisos = append(novo.Avisos, fresh)
				}
			}
		}
		if len(novo.Avisos) == 0 {
			return ResultadoOperacao{}, fmt.Errorf("não há falhas seguras para repetir; confira os envios incertos no provedor")
		}
		s.novoLote = &novo
		return ResultadoOperacao{Resumo: "Preparadas somente as falhas confirmadas, com a agenda atual.", LoteID: novo.ID, Envio: &novo}, nil
	}, async)
}
