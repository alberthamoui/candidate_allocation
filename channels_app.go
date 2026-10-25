package main

import (
	"candidate_alocator/back/operations"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

func (a *App) noticeConfig() (ConfigAvisos, error) {
	s, err := a.store()
	if err != nil {
		return ConfigAvisos{}, err
	}
	data, err := s.Settings("channels")
	if errors.Is(err, sql.ErrNoRows) {
		return ConfigAvisos{Mensagem: mensagemPadrao, Email: EmailConfig{Provedor: "gmail", Host: "smtp.gmail.com", Porta: 587}, WhatsApp: WhatsConfig{Provedor: "meta", Idioma: "pt_BR", VariaveisTemplate: []string{"nome", "data", "horario", "mesa"}}}, nil
	}
	if err != nil {
		return ConfigAvisos{}, err
	}
	var cfg ConfigAvisos
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func (a *App) GetChannelConfig() (ConfigAvisos, error) {
	cfg, err := a.noticeConfig()
	cfg.Email.SenhaDefinida = cfg.Email.Senha != ""
	cfg.Email.GoogleConnected = cfg.Email.RefreshToken != ""
	cfg.Email.Senha = ""
	cfg.Email.RefreshToken = ""
	cfg.WhatsApp.TokenDefinido = cfg.WhatsApp.Token != ""
	cfg.WhatsApp.Token = ""
	return cfg, err
}

func (a *App) SaveChannelConfig(cfg ConfigAvisos) error {
	a.channelsMu.Lock()
	defer a.channelsMu.Unlock()
	previous, err := a.noticeConfig()
	if err != nil {
		return err
	}
	if cfg.Email.Provedor == "gmail" {
		cfg.Email.Host = "smtp.gmail.com"
		cfg.Email.Porta = 587
		cfg.Email.Usuario = cfg.Email.Remetente
	}
	if cfg.Email.Senha == "" && cfg.Email.Provedor == previous.Email.Provedor && cfg.Email.Usuario == previous.Email.Usuario && cfg.Email.Host == previous.Email.Host {
		cfg.Email.Senha = previous.Email.Senha
	}
	// OAuth secrets can only come from the private token exchange, never the UI.
	cfg.Email.RefreshToken = ""
	if cfg.Email.Provedor == "google" && cfg.Email.Remetente == previous.Email.Remetente {
		cfg.Email.RefreshToken = previous.Email.RefreshToken
	}
	if cfg.WhatsApp.Token == "" && cfg.WhatsApp.Provedor == previous.WhatsApp.Provedor && cfg.WhatsApp.PhoneID == previous.WhatsApp.PhoneID && cfg.WhatsApp.URL == previous.WhatsApp.URL {
		cfg.WhatsApp.Token = previous.WhatsApp.Token
	}
	if err := validarMensagem(cfg.Mensagem); err != nil {
		return err
	}
	if cfg.Email.Ativo {
		if _, err := mail.ParseAddress(cfg.Email.Remetente); err != nil {
			return errors.New("informe um e-mail remetente válido")
		}
	}
	if cfg.WhatsApp.Ativo && cfg.WhatsApp.Provedor == "meta" {
		if !regexp.MustCompile(`^\d+$`).MatchString(cfg.WhatsApp.PhoneID) || cfg.WhatsApp.Token == "" || cfg.WhatsApp.Template == "" {
			return errors.New("conecte o número WhatsApp Business e escolha um modelo aprovado")
		}
	}
	if cfg.WhatsApp.Ativo && cfg.WhatsApp.Provedor == "webhook" {
		if !strings.HasPrefix(cfg.WhatsApp.URL, "https://") {
			return errors.New("o endereço do provedor deve usar HTTPS")
		}
	}
	s, err := a.store()
	if err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.SaveSettings("channels", data)
}

func (a *App) DisconnectGoogle() error {
	a.channelsMu.Lock()
	defer a.channelsMu.Unlock()
	cfg, err := a.noticeConfig()
	if err != nil {
		return err
	}
	cfg.Email.RefreshToken = ""
	cfg.Email.GoogleConnected = false
	cfg.Email.Ativo = false
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	s, err := a.store()
	if err != nil {
		return err
	}
	return s.SaveSettings("channels", data)
}

func (a *App) PreviewAgendaNotices(ids []int, groupID int) ([]operations.Delivery, error) {
	snap, err := a.readAgenda()
	if err != nil {
		return nil, err
	}
	cfg, err := a.noticeConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Ativo || !cfg.Email.Ativo && !cfg.WhatsApp.Ativo {
		return nil, errors.New("configure e ative um canal em Canais e mensagens")
	}
	wanted := map[int]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	out := []operations.Delivery{}
	seen := map[string]bool{}
	for _, m := range snap.Result.Mesas {
		affected := false
		for _, c := range m.Candidatos {
			if groupID != 0 && m.ID != groupID || len(wanted) > 0 && !wanted[c.ID] {
				continue
			}
			affected = true
			out = append(out, prepareDeliveries(cfg, c, m)...)
		}
		if affected && cfg.Email.Ativo {
			for _, e := range m.Avaliadores {
				key := fmt.Sprintf("%d:%s", e.ID, m.Horario)
				if seen[key] {
					continue
				}
				seen[key] = true
				c := UICandidate{ID: -e.ID, Nome: e.Nome, EmailPessoal: e.Email}
				for _, d := range prepareDeliveries(cfg, c, m) {
					if d.Channel == "email" {
						out = append(out, d)
					}
				}
			}
		}
	}
	return out, nil
}

func prepareDeliveries(cfg ConfigAvisos, c UICandidate, m UIMesa) []operations.Delivery {
	base := operations.Delivery{PersonID: c.ID, GroupID: m.ID, Name: c.Nome, Message: renderMensagem(cfg.Mensagem, c, m), Variables: variaveisMensagem(c, m)}
	base.Variables["agenda_horario"] = m.Horario
	out := []operations.Delivery{}
	if cfg.Email.Ativo {
		d := base
		d.Channel = "email"
		d.Destination = emailDe(c)
		if _, err := mail.ParseAddress(d.Destination); err != nil {
			d.Error = "sem e-mail válido; corrija o cadastro"
		}
		out = append(out, d)
	}
	if cfg.WhatsApp.Ativo {
		d := base
		d.Channel = "whatsapp"
		number, err := normalizarTelefone(c.Telefone)
		d.Destination = number
		if err != nil {
			d.Error = err.Error()
		}
		out = append(out, d)
	}
	return out
}

func (a *App) SendAgendaNotices(ids []int, groupID int) (int64, error) {
	items, err := a.PreviewAgendaNotices(ids, groupID)
	if err != nil {
		return 0, err
	}
	s, err := a.store()
	if err != nil {
		return 0, err
	}
	id, err := s.CreateBatch(currentAgenda, items)
	if err != nil {
		return 0, err
	}
	a.emitNoticeChange(id)
	if a.synchronousNotices {
		a.executeNoticeBatch(id)
	} else {
		go a.executeNoticeBatch(id)
	}
	return id, nil
}

func (a *App) maybeAutoNotify() {
	cfg, err := a.noticeConfig()
	if err == nil && cfg.Ativo && cfg.EnviarAoAlocar {
		_, _ = a.SendAgendaNotices(nil, 0)
	}
}

func (a *App) GetNoticeHistory(before int64) (operations.HistoryPage, error) {
	s, err := a.store()
	if err != nil {
		return operations.HistoryPage{}, err
	}
	return s.History(currentAgenda, before, 10)
}
func (a *App) GetNoticeDetails(id, after int64) (operations.DeliveryPage, error) {
	s, err := a.store()
	if err != nil {
		return operations.DeliveryPage{}, err
	}
	return s.Deliveries(currentAgenda, id, after, 50)
}
func (a *App) RetryNoticeFailures(id int64) (int64, error) {
	s, err := a.store()
	if err != nil {
		return 0, err
	}
	current, err := a.PreviewAgendaNotices(nil, 0)
	if err != nil {
		return 0, err
	}
	byRecipient := map[string]operations.Delivery{}
	key := func(d operations.Delivery) string {
		group := 0
		if d.PersonID < 0 {
			group = d.GroupID
		}
		return fmt.Sprintf("%d:%d:%s", d.PersonID, group, d.Channel)
	}
	for _, d := range current {
		byRecipient[key(d)] = d
	}
	next, err := s.Retry(currentAgenda, id, func(d operations.Delivery) operations.Delivery {
		if fresh, ok := byRecipient[key(d)]; ok {
			return fresh
		}
		d.Error = "Destinatário ou canal não está na agenda atual; confira antes de enviar"
		return d
	})
	if err == nil {
		if a.synchronousNotices {
			a.executeNoticeBatch(next)
		} else {
			go a.executeNoticeBatch(next)
		}
	}
	return next, err
}

func (a *App) GetNoticeStates() (map[int]string, error) {
	s, err := a.store()
	if err != nil {
		return nil, err
	}
	snap, err := a.readAgenda()
	if err != nil {
		return map[int]string{}, nil
	}
	schedules := map[int]string{}
	for _, m := range snap.Result.Mesas {
		for _, c := range m.Candidatos {
			schedules[c.ID] = m.Horario
		}
	}
	return s.NoticeStates(currentAgenda, schedules)
}

func (a *App) executeNoticeBatch(id int64) {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	s, err := a.store()
	if err != nil {
		return
	}
	cfg, err := a.noticeConfig()
	if err != nil {
		return
	}
	var after int64
	for {
		page, err := s.Deliveries(currentAgenda, id, after, 50)
		if err != nil {
			return
		}
		for _, d := range page.Items {
			claimed, err := s.Claim(d.ID)
			if err != nil {
				return
			}
			if !claimed {
				continue
			}
			if d.Destination == "" {
				err = errors.New("destinatário ausente; corrija o cadastro")
			} else if a.noticeSender != nil {
				err = a.noticeSender(cfg, d)
			} else if d.Channel == "email" {
				if cfg.Email.Provedor == "google" {
					err = a.sendGoogleEmail(cfg.Email, d)
				} else {
					err = enviarEmailSMTP(cfg.Email, d.Destination, assuntoAviso, d.Message)
				}
			} else {
				err = enviarWhatsAppVariaveis(cfg.WhatsApp, d.Destination, d.Message, d.Variables)
			}
			if err := s.Complete(d.ID, err); err != nil {
				return
			}
		}
		if page.Next == 0 {
			break
		}
		after = page.Next
	}
	a.emitNoticeChange(id)
}

func (a *App) emitNoticeChange(id int64) {
	if a.noticeEvent != nil {
		a.noticeEvent(id)
	}
	a.allocationMu.Lock()
	ctx := a.ctx
	a.allocationMu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "notices:changed", map[string]int64{"batchId": id})
	}
}

func (a *App) TestChannels() (map[string]string, error) {
	cfg, err := a.noticeConfig()
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	if cfg.Email.Ativo {
		if cfg.Email.Provedor == "google" {
			_, err := a.googleClient(cfg.Email)
			if err != nil {
				result["email"] = err.Error()
			} else {
				result["email"] = "Conta Google autorizada para envio"
			}
		} else {
			client, err := clienteSMTP(cfg.Email)
			if err != nil {
				result["email"] = err.Error()
			} else {
				_ = client.Close()
				result["email"] = "Conexão verificada; nenhuma mensagem enviada"
			}
		}
	}
	if cfg.WhatsApp.Ativo {
		if cfg.WhatsApp.Provedor == "meta" {
			req, err := http.NewRequest(http.MethodGet, "https://graph.facebook.com/v23.0/"+cfg.WhatsApp.PhoneID+"?fields=display_phone_number,verified_name", nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+cfg.WhatsApp.Token)
			resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
			if err != nil {
				result["whatsapp"] = "Não foi possível conectar à Meta"
			} else {
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
				_ = resp.Body.Close()
				if resp.StatusCode >= 300 {
					result["whatsapp"] = "A Meta recusou o acesso; verifique a conexão"
				} else {
					result["whatsapp"] = "Número conectado; o envio usará o modelo aprovado"
				}
			}
		} else {
			result["whatsapp"] = "Provedor avançado configurado; confira o contrato de envio"
		}
	}
	if len(result) == 0 {
		result["configuracao"] = "Ative pelo menos um canal antes de testar"
	}
	return result, nil
}
