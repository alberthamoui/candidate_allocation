package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// ConfigAvisos é o que o administrador define na entrada: a mensagem, com
// variáveis, e como ela sai (e-mail por SMTP e WhatsApp por um webhook).
// Os dois canais são opcionais. Sem servidor ou URL, a mensagem é montada
// mesmo assim e a tela mostra o texto, mas nada é enviado.
type ConfigAvisos struct {
	Ativo       bool                                                     `json:"ativo"`
	Mensagem    string                                                   `json:"mensagem"`
	Email       EmailConfig                                              `json:"email"`
	WhatsApp    WhatsConfig                                              `json:"whatsapp"`
	enviarEmail func(cfg EmailConfig, para, assunto, corpo string) error `json:"-"`
	enviarWhats func(cfg WhatsConfig, para, corpo string) error          `json:"-"`
}

// EmailConfig é o mínimo para mandar e-mail: servidor, porta, remetente e,
// se o servidor pedir, usuário e senha. A porta 587 é a usual.
type EmailConfig struct {
	Ativo     bool   `json:"ativo"`
	Host      string `json:"host"`
	Porta     int    `json:"porta"`
	Usuario   string `json:"usuario"`
	Senha     string `json:"senha"`
	Remetente string `json:"remetente"`
}

// WhatsConfig aponta para um endereço que recebe POST JSON
// {"to","message","para","mensagem"}. Qualquer API simples de WhatsApp que
// aceite isso serve. Token, se houver, vai no header Authorization.
type WhatsConfig struct {
	Ativo bool   `json:"ativo"`
	URL   string `json:"url"`
	Token string `json:"token"`
}

// AvisoGerado é uma mensagem já com o nome e a data preenchidos.
type AvisoGerado struct {
	Nome     string `json:"nome"`
	Canal    string `json:"canal"`
	Destino  string `json:"destino"`
	Mensagem string `json:"mensagem"`
	Enviado  bool   `json:"enviado"`
	Erro     string `json:"erro,omitempty"`
}

const mensagemPadrao = "Olá {{nome}}, sua entrevista é em {{data}}, no horário {{horario}} ({{mesa}})."

const assuntoAviso = "Sua entrevista"

// variaveisAviso são as que o administrador pode colocar na mensagem.
var variaveisAviso = []string{"nome", "data", "horario", "mesa", "email", "curso", "numero"}

func renderMensagem(modelo string, c CandidatoResultado, mesa MesaResult) string {
	if strings.TrimSpace(modelo) == "" {
		modelo = mensagemPadrao
	}
	data, hora := dividirDataHorario(mesa.DiaNome)
	repl := map[string]string{
		"nome":    c.Nome,
		"data":    data,
		"horario": hora,
		"mesa":    mesa.Descricao,
		"email":   emailDe(c),
		"curso":   c.Curso,
		"numero":  c.Numero,
	}
	out := modelo
	for _, k := range variaveisAviso {
		v := repl[k]
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// dividirDataHorario separa "quarta 14-16" em data "quarta" e horário "14-16".
// Com uma palavra só, os dois ficam iguais.
func dividirDataHorario(dia string) (string, string) {
	dia = strings.TrimSpace(dia)
	partes := strings.Fields(dia)
	switch len(partes) {
	case 0:
		return "", ""
	case 1:
		return dia, dia
	default:
		return partes[0], strings.Join(partes[1:], " ")
	}
}

func emailDe(c CandidatoResultado) string {
	if strings.TrimSpace(c.EmailInsper) != "" {
		return strings.TrimSpace(c.EmailInsper)
	}
	return strings.TrimSpace(c.EmailPessoal)
}

func textoAvaliacao(resultado string) string {
	switch resultado {
	case "aprovado":
		return "passou"
	case "reprovado":
		return "não passou"
	default:
		return ""
	}
}

func (s *Session) enviarPara(pessoas []CandidatoResultado, mesaDe map[int]MesaResult) []AvisoGerado {
	var avisos []AvisoGerado
	for _, c := range pessoas {
		mesa := mesaDe[c.ID]
		texto := renderMensagem(s.avisos.Mensagem, c, mesa)
		avisos = append(avisos, s.despachar(c, texto)...)
	}
	if avisos == nil {
		avisos = []AvisoGerado{}
	}
	return avisos
}

func (s *Session) despachar(c CandidatoResultado, texto string) []AvisoGerado {
	cfg := s.avisos
	if !cfg.Ativo {
		return []AvisoGerado{{
			Nome: c.Nome, Canal: "nenhum", Mensagem: texto,
			Erro: "avisos estão desligados: ative na entrada ou na tela de resultado",
		}}
	}
	var out []AvisoGerado
	if cfg.Email.Ativo {
		destino := emailDe(c)
		item := AvisoGerado{Nome: c.Nome, Canal: "email", Destino: destino, Mensagem: texto}
		switch destino {
		case "":
			item.Erro = "candidato sem e-mail"
		default:
			enviar := cfg.enviarEmail
			if enviar == nil {
				enviar = enviarEmailSMTP
			}
			if err := enviar(cfg.Email, destino, assuntoAviso, texto); err != nil {
				item.Erro = err.Error()
			} else {
				item.Enviado = true
			}
		}
		out = append(out, item)
	}
	if cfg.WhatsApp.Ativo {
		destino := strings.TrimSpace(c.Numero)
		item := AvisoGerado{Nome: c.Nome, Canal: "whatsapp", Destino: destino, Mensagem: texto}
		switch destino {
		case "":
			item.Erro = "candidato sem número"
		default:
			enviar := cfg.enviarWhats
			if enviar == nil {
				enviar = enviarWhatsAppHTTP
			}
			if err := enviar(cfg.WhatsApp, destino, texto); err != nil {
				item.Erro = err.Error()
			} else {
				item.Enviado = true
			}
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		out = append(out, AvisoGerado{
			Nome: c.Nome, Canal: "nenhum", Mensagem: texto,
			Erro: "ative e-mail ou WhatsApp na configuração de avisos",
		})
	}
	return out
}

func (cfg EmailConfig) endereco() string {
	porta := cfg.Porta
	if porta == 0 {
		porta = 587
	}
	return fmt.Sprintf("%s:%d", cfg.Host, porta)
}

func enviarEmailSMTP(cfg EmailConfig, para, assunto, corpo string) error {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.Remetente) == "" {
		return fmt.Errorf("e-mail não configurado: informe o servidor SMTP e o remetente")
	}
	porta := cfg.Porta
	if porta == 0 {
		porta = 587
	}
	msg := []byte("From: " + cfg.Remetente + "\r\n" +
		"To: " + para + "\r\n" +
		"Subject: " + assunto + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + corpo)
	var auth smtp.Auth
	if cfg.Usuario != "" {
		auth = smtp.PlainAuth("", cfg.Usuario, cfg.Senha, cfg.Host)
	}
	if porta == 465 {
		return enviarEmailTLS(cfg, para, msg, auth)
	}
	return smtp.SendMail(cfg.endereco(), auth, cfg.Remetente, []string{para}, msg)
}

func enviarEmailTLS(cfg EmailConfig, para string, msg []byte, auth smtp.Auth) error {
	conn, err := tls.Dial("tcp", cfg.endereco(), &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return err
	}
	cliente, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = cliente.Close() }()
	if auth != nil {
		if err := cliente.Auth(auth); err != nil {
			return err
		}
	}
	if err := cliente.Mail(cfg.Remetente); err != nil {
		return err
	}
	if err := cliente.Rcpt(para); err != nil {
		return err
	}
	w, err := cliente.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return cliente.Quit()
}

func enviarWhatsAppHTTP(cfg WhatsConfig, para, corpo string) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("WhatsApp não configurado: informe a URL do webhook")
	}
	body, err := json.Marshal(map[string]string{
		"to": para, "message": corpo, "para": para, "mensagem": corpo,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	cliente := &http.Client{Timeout: 15 * time.Second}
	resp, err := cliente.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("WhatsApp respondeu %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}
