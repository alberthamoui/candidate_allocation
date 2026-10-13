package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type LoteAvisos struct {
	RepetidoPor string        `json:"repetido_por,omitempty"`
	ID          string        `json:"id"`
	Criado      time.Time     `json:"criado"`
	Status      string        `json:"status"`
	Avisos      []AvisoGerado `json:"avisos"`
	Enviados    int           `json:"enviados"`
	Falhas      int           `json:"falhas"`
}

func (s *Session) prepararLoteLocked(ids []int) LoteAvisos {
	lote := LoteAvisos{ID: newSessionID(), Criado: time.Now(), Status: "enviando", Avisos: []AvisoGerado{}}
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
	base := AvisoGerado{Nome: c.Nome, Mensagem: texto, Variaveis: variaveisMensagem(c, m)}
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
		if telefone == "" && len(digitosNumero(c.Numero)) >= 10 {
			telefone = c.Numero
		}
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
func (s *Session) executarLote(id string, cfg ConfigAvisos) {
	lote, err := s.Lote(id)
	if err != nil {
		return
	}
	for i := range lote.Avisos {
		a := &lote.Avisos[i]
		if a.Destino == "" && a.Erro == "" {
			a.Erro = "destinatário ausente; corrija o cadastro antes de reenviar"
		}
		if a.Erro == "" {
			switch a.Canal {
			case "email":
				enviar := cfg.enviarEmail
				if enviar == nil {
					enviar = enviarEmailSMTP
				}
				err = enviar(cfg.Email, a.Destino, assuntoAviso, a.Mensagem)
			case "whatsapp":
				enviar := cfg.enviarWhats
				if enviar != nil {
					err = enviar(cfg.WhatsApp, a.Destino, a.Mensagem)
				} else {
					err = enviarWhatsAppVariaveis(cfg.WhatsApp, a.Destino, a.Mensagem, a.Variaveis)
				}
			default:
				err = fmt.Errorf("nenhum canal configurado")
			}
			if err != nil {
				a.Erro = err.Error()
			} else {
				a.Enviado = true
			}
		}
		if a.Enviado {
			lote.Enviados++
		} else {
			lote.Falhas++
		}
		s.mu.Lock()
		for j := range s.historico {
			if s.historico[j].ID == id {
				s.historico[j] = clonarLote(lote)
				break
			}
		}
		if err := s.persistirLocked(); err != nil {
			lote.Status = "interrompido"
		}
		s.mu.Unlock()
	}
	lote.Status = "concluido"
	if lote.Falhas > 0 {
		lote.Status = "parcial"
		if lote.Enviados == 0 {
			lote.Status = "falhou"
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.historico {
		if s.historico[i].ID == id {
			s.historico[i] = clonarLote(lote)
			break
		}
	}
	if err := s.persistirLocked(); err != nil {
		for i := range s.historico {
			if s.historico[i].ID == id {
				s.historico[i].Status = "interrompido"
			}
		}
	}
}
func (s *Session) Lote(id string) (LoteAvisos, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, l := range s.historico {
		if l.ID == id {
			return clonarLote(l), nil
		}
	}
	return LoteAvisos{}, fmt.Errorf("envio não encontrado")
}
func clonarLote(l LoteAvisos) LoteAvisos {
	b, _ := json.Marshal(l)
	var out LoteAvisos
	_ = json.Unmarshal(b, &out)
	return out
}
func (s *Session) HistoricoAvisos() []LoteAvisos {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []LoteAvisos{}
	for i := len(s.historico) - 1; i >= 0; i-- {
		out = append(out, clonarLote(s.historico[i]))
	}
	return out
}
func (s *Session) RepetirFalhas(id string, async bool) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) {
		for _, l := range s.historico {
			if l.ID == id {
				if l.RepetidoPor != "" {
					return ResultadoOperacao{}, fmt.Errorf("falhas já encaminhadas ao envio %s", l.RepetidoPor)
				}
				if l.Status == "enviando" {
					return ResultadoOperacao{}, fmt.Errorf("envio ainda em andamento")
				}
				novo := LoteAvisos{ID: newSessionID(), Criado: time.Now(), Status: "enviando", Avisos: []AvisoGerado{}}
				for _, a := range l.Avisos {
					if !a.Enviado {
						a.Erro = ""
						novo.Avisos = append(novo.Avisos, a)
					}
				}
				if len(novo.Avisos) == 0 {
					return ResultadoOperacao{}, fmt.Errorf("não há falhas para repetir")
				}
				for i := range s.historico {
					if s.historico[i].ID == id {
						s.historico[i].RepetidoPor = novo.ID
					}
				}
				s.historico = append(s.historico, novo)
				return ResultadoOperacao{Resumo: "Repetindo somente os avisos não enviados.", LoteID: novo.ID, Envio: &novo, Avisos: []AvisoGerado{}}, nil
			}
		}
		return ResultadoOperacao{}, fmt.Errorf("envio não encontrado")
	}, async)
}
