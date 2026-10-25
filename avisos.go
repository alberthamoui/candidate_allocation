package main

import (
	"bytes"
	"candidate_alocator/back/operations"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
	"time"
)

// ConfigAvisos é o que o administrador define na entrada: a mensagem, com
// variáveis, e como ela sai (e-mail por SMTP e WhatsApp por um webhook).
// Os dois canais são opcionais. Sem servidor ou URL, a mensagem é montada
// mesmo assim e a tela mostra o texto, mas nada é enviado.
type ConfigAvisos struct {
	Ativo          bool        `json:"ativo"`
	EnviarAoAlocar bool        `json:"enviar_ao_alocar"`
	Mensagem       string      `json:"mensagem"`
	Email          EmailConfig `json:"email"`
	WhatsApp       WhatsConfig `json:"whatsapp"`
}

// EmailConfig é o mínimo para mandar e-mail: servidor, porta, remetente e,
// se o servidor pedir, usuário e senha. A porta 587 é a usual.
type EmailConfig struct {
	Provedor        string `json:"provedor,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	GoogleConnected bool   `json:"google_connected"`
	SenhaDefinida   bool   `json:"senha_definida,omitempty"`
	Ativo           bool   `json:"ativo"`
	Host            string `json:"host"`
	Porta           int    `json:"porta"`
	Usuario         string `json:"usuario"`
	Senha           string `json:"senha"`
	Remetente       string `json:"remetente"`
}

// WhatsConfig aponta para um endereço que recebe POST JSON
// {"to","message","para","mensagem"}. Qualquer API simples de WhatsApp que
// aceite isso serve. Token, se houver, vai no header Authorization.
type WhatsConfig struct {
	httpClient        *http.Client
	Provedor          string   `json:"provedor,omitempty"`
	PhoneID           string   `json:"phone_id,omitempty"`
	Template          string   `json:"template,omitempty"`
	Idioma            string   `json:"idioma,omitempty"`
	VariaveisTemplate []string `json:"variaveis_template,omitempty"`
	TokenDefinido     bool     `json:"token_definido,omitempty"`
	Ativo             bool     `json:"ativo"`
	URL               string   `json:"url"`
	Token             string   `json:"token"`
}

// AvisoGerado é uma mensagem já com o nome e a data preenchidos.
type AvisoGerado struct {
	Variaveis map[string]string `json:"variaveis,omitempty"`
	Nome      string            `json:"nome"`
	Canal     string            `json:"canal"`
	Destino   string            `json:"destino"`
	Mensagem  string            `json:"mensagem"`
	Enviado   bool              `json:"enviado"`
	Erro      string            `json:"erro,omitempty"`
}

const mensagemPadrao = "Olá {{nome}}, sua entrevista é em {{data}}, no horário {{horario}} ({{mesa}})."

const assuntoAviso = "Sua entrevista"

// variaveisAviso são as que o administrador pode colocar na mensagem.
var variaveisAviso = []string{"nome", "data", "horario", "mesa", "email", "curso", "numero"}

func renderMensagem(modelo string, c UICandidate, mesa UIMesa) string {
	if strings.TrimSpace(modelo) == "" {
		modelo = mensagemPadrao
	}
	repl := variaveisMensagem(c, mesa)
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
	for i, p := range partes {
		if horarioMensagem.MatchString(p) {
			return strings.Join(partes[:i], " "), strings.Join(partes[i:], " ")
		}
	}
	if len(partes) <= 1 {
		return dia, dia
	}
	return strings.Join(partes[:len(partes)-1], " "), partes[len(partes)-1]
}

var horarioMensagem = regexp.MustCompile(`^\d{1,2}(?::\d{2})?[-h:]`)

func variaveisMensagem(c UICandidate, m UIMesa) map[string]string {
	data, hora := dividirDataHorario(m.Horario)
	return map[string]string{"nome": c.Nome, "data": data, "horario": hora, "mesa": m.Descricao, "email": emailDe(c), "curso": c.Curso, "numero": c.Numero}
}
func validarMensagem(msg string) error {
	for _, v := range regexp.MustCompile(`\{\{([^{}]+)\}\}`).FindAllStringSubmatch(msg, -1) {
		ok := false
		for _, k := range variaveisAviso {
			if v[1] == k {
				ok = true
			}
		}
		if !ok {
			return fmt.Errorf("variável desconhecida: %s", v[0])
		}
	}
	return nil
}
func normalizarTelefone(s string) (string, error) {
	n := regexp.MustCompile(`[^0-9]`).ReplaceAllString(s, "")
	if strings.HasPrefix(strings.TrimSpace(s), "+") {
		if len(n) >= 8 && len(n) <= 15 {
			return n, nil
		}
	} else {
		if len(n) == 10 || len(n) == 11 {
			return "55" + n, nil
		}
		if len(n) >= 12 && len(n) <= 15 {
			return n, nil
		}
	}
	return "", fmt.Errorf("informe um telefone com DDD; matrícula não é telefone de WhatsApp")
}

func emailDe(c UICandidate) string {
	if strings.TrimSpace(c.EmailSecundario) != "" {
		return strings.TrimSpace(c.EmailSecundario)
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

func (cfg EmailConfig) endereco() string {
	porta := cfg.Porta
	if porta == 0 {
		porta = 587
	}
	return fmt.Sprintf("%s:%d", cfg.Host, porta)
}

func clienteSMTP(cfg EmailConfig) (*smtp.Client, error) {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.Remetente) == "" {
		return nil, fmt.Errorf("informe o servidor SMTP e o remetente")
	}
	if _, err := mail.ParseAddress(cfg.Remetente); err != nil {
		return nil, fmt.Errorf("remetente inválido")
	}
	var conn net.Conn
	var err error
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if cfg.Porta == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", cfg.endereco(), &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.Dial("tcp", cfg.endereco())
	}
	if err != nil {
		return nil, err
	}
	if err = conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if cfg.Porta != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				_ = c.Close()
				return nil, err
			}
		}
	}
	if cfg.Usuario != "" {
		if err := c.Auth(smtp.PlainAuth("", cfg.Usuario, cfg.Senha, cfg.Host)); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("autenticação SMTP falhou; verifique a senha de aplicativo e o usuário: %w", err)
		}
	}
	return c, nil
}
func enviarEmailSMTP(cfg EmailConfig, para, assunto, corpo string) error {
	destino, err := mail.ParseAddress(para)
	if err != nil {
		return fmt.Errorf("destinatário inválido")
	}
	remetente, err := mail.ParseAddress(cfg.Remetente)
	if err != nil {
		return fmt.Errorf("remetente inválido")
	}
	c, err := clienteSMTP(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if err := c.Mail(remetente.Address); err != nil {
		return err
	}
	if err := c.Rcpt(destino.Address); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	msg := "From: " + remetente.String() + "\r\nTo: " + destino.String() + "\r\nSubject: " + assunto + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + corpo
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("%w: %v", operations.ErrUncertain, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("%w: %v", operations.ErrUncertain, err)
	}
	// DATA's positive response is the acceptance. A lost QUIT response cannot
	// turn a message already accepted by SMTP into a retryable failure.
	_ = c.Quit()
	return nil
}

func enviarWhatsAppVariaveis(cfg WhatsConfig, para, corpo string, vars map[string]string) error {
	var payload any
	destino := cfg.URL
	if cfg.Provedor == "meta" {
		if cfg.PhoneID == "" || cfg.Token == "" || cfg.Template == "" {
			return fmt.Errorf("configure o número, token e modelo aprovado do WhatsApp Business")
		}
		if !regexp.MustCompile(`^\d+$`).MatchString(cfg.PhoneID) {
			return fmt.Errorf("ID do número inválido")
		}
		destino = "https://graph.facebook.com/v23.0/" + cfg.PhoneID + "/messages"
		idioma := cfg.Idioma
		if idioma == "" {
			idioma = "pt_BR"
		}
		params := []map[string]string{}
		for _, v := range cfg.VariaveisTemplate {
			value, ok := vars[v]
			if !ok {
				return fmt.Errorf("variável de modelo inválida: %s", v)
			}
			params = append(params, map[string]string{"type": "text", "text": value})
		}
		components := []map[string]any{}
		if len(params) > 0 {
			components = append(components, map[string]any{"type": "body", "parameters": params})
		}
		payload = map[string]any{"messaging_product": "whatsapp", "to": para, "type": "template", "template": map[string]any{"name": cfg.Template, "language": map[string]string{"code": idioma}, "components": components}}
	} else {
		if strings.TrimSpace(destino) == "" {
			return fmt.Errorf("WhatsApp não configurado: informe o provedor ou a URL do webhook")
		}
		payload = map[string]string{"to": para, "message": corpo, "para": para, "mensagem": corpo}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, destino, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	client := cfg.httpClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", operations.ErrUncertain, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("provedor WhatsApp recusou o envio (%s); verifique token, telefone e modelo aprovado", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return fmt.Errorf("%w: %v", operations.ErrUncertain, err)
	}
	if cfg.Provedor == "meta" {
		var out struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(b, &out); err != nil || len(out.Messages) == 0 || out.Messages[0].ID == "" {
			return fmt.Errorf("%w: WhatsApp não confirmou a aceitação da mensagem", operations.ErrUncertain)
		}
	}
	return nil
}
