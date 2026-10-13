package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (store *SessionStore) handleSugestoes(w http.ResponseWriter, r *http.Request) {
	s := store.sessionFromRequest(r)
	if s == nil {
		writeError(w, 401, msgSessaoExpirada)
		return
	}
	out, err := s.Sugestoes(r.URL.Query().Get("pessoa"))
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, out)
}
func (store *SessionStore) handleHistorico(w http.ResponseWriter, r *http.Request) {
	s := store.sessionFromRequest(r)
	if s == nil {
		writeError(w, 401, msgSessaoExpirada)
		return
	}
	writeJSON(w, 200, s.HistoricoAvisos())
}
func (store *SessionStore) handleRepetir(w http.ResponseWriter, r *http.Request) {
	s := store.sessionFromRequest(r)
	if s == nil {
		writeError(w, 401, msgSessaoExpirada)
		return
	}
	var p struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, 400, "pedido inválido")
		return
	}
	op, err := s.RepetirFalhas(p.ID, true)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, op)
}
func (store *SessionStore) configDoPedido(r *http.Request) (ConfigAvisos, error) {
	var cfg ConfigAvisos
	if err := decodeJSON(r, &cfg); err != nil {
		return cfg, err
	}
	if s := store.sessionFromRequest(r); s != nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if cfg.Email.Senha == "" && cfg.Email.Host == s.avisos.Email.Host && cfg.Email.Usuario == s.avisos.Email.Usuario {
			cfg.Email.Senha = s.avisos.Email.Senha
		}
		if cfg.WhatsApp.Token == "" && cfg.WhatsApp.Provedor == s.avisos.WhatsApp.Provedor {
			cfg.WhatsApp.Token = s.avisos.WhatsApp.Token
		}
	}
	return cfg, nil
}
func (store *SessionStore) handleTestar(w http.ResponseWriter, r *http.Request) {
	cfg, err := store.configDoPedido(r)
	if err != nil {
		writeError(w, 400, "configuração inválida")
		return
	}
	status := map[string]string{}
	if cfg.Email.Ativo {
		c, err := clienteSMTP(cfg.Email)
		if err != nil {
			status["email"] = err.Error()
		} else {
			_ = c.Close()
			status["email"] = "Conexão e autenticação verificadas. Nenhuma mensagem enviada."
		}
	}
	if cfg.WhatsApp.Ativo {
		if cfg.WhatsApp.Provedor != "meta" {
			status["whatsapp"] = "Webhook avançado: confirme o contrato com seu provedor; use a prévia antes de enviar."
		} else if cfg.WhatsApp.PhoneID == "" || cfg.WhatsApp.Token == "" {
			status["whatsapp"] = "Informe o ID do número e o token da Meta."
		} else {
			req, err := http.NewRequest(http.MethodGet, "https://graph.facebook.com/v23.0/"+cfg.WhatsApp.PhoneID+"?fields=display_phone_number,verified_name", nil)
			if err != nil {
				status["whatsapp"] = err.Error()
			} else {
				req.Header.Set("Authorization", "Bearer "+cfg.WhatsApp.Token)
				resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
				if err != nil {
					status["whatsapp"] = "Não foi possível conectar à Meta."
				} else {
					_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
					_ = resp.Body.Close()
					if resp.StatusCode >= 300 {
						status["whatsapp"] = "A Meta recusou o acesso; confira token e ID do número."
					} else {
						status["whatsapp"] = "Número conectado. O envio usa o modelo aprovado configurado."
					}
				}
			}
		}
	}
	if len(status) == 0 {
		status["configuracao"] = "Ative ao menos um canal para testar."
	}
	writeJSON(w, 200, status)
}
func (store *SessionStore) handlePreviaAvisos(w http.ResponseWriter, r *http.Request) {
	cfg, err := store.configDoPedido(r)
	if err != nil {
		writeError(w, 400, "configuração inválida")
		return
	}
	if err := validarMensagem(cfg.Mensagem); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	c := CandidatoResultado{Nome: "Maria", EmailPessoal: "maria@exemplo.com", Telefone: "11999999999", Curso: "Economia", Numero: "123456789"}
	m := MesaResult{DiaNome: "21/10/2026 14-16", Descricao: "21/10/2026 14-16 - mesa 1"}
	if s := store.sessionFromRequest(r); s != nil {
		s.mu.RLock()
		if s.lastResult != nil {
			for _, mesa := range s.lastResult.Mesas {
				if len(mesa.Candidatos) > 0 {
					m = mesa
					c = mesa.Candidatos[0]
					break
				}
			}
		}
		s.mu.RUnlock()
	}
	writeJSON(w, 200, map[string]any{"mensagem": renderMensagem(cfg.Mensagem, c, m), "destinatarios": prepararAvisos(cfg, c, m), "nota": fmt.Sprintf("Prévia para %s. %s", c.Nome, strings.TrimSpace("Nenhuma mensagem foi enviada."))})
}
