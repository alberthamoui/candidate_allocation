package main

import (
	"bytes"
	"candidate_alocator/back/operations"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"net"
	"net/http"
	"net/mail"
	"os"
	"sync"
	"time"
)

type googleConnection struct {
	cancel   context.CancelFunc
	state    string
	verifier string
	status   string
	message  string
	server   *http.Server
	config   oauth2.Config
}
type GoogleConnectionStatus struct {
	Available bool   `json:"available"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

func googleOAuthConfig() oauth2.Config {
	// #nosec G101 -- provider endpoints are constants; credentials come from environment.
	return oauth2.Config{ClientID: os.Getenv("CANDIDATE_GOOGLE_CLIENT_ID"), ClientSecret: os.Getenv("CANDIDATE_GOOGLE_CLIENT_SECRET"), Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"}, Scopes: []string{"https://www.googleapis.com/auth/gmail.send", "https://www.googleapis.com/auth/userinfo.email"}}
}

func (a *App) GetGoogleConnectionStatus() GoogleConnectionStatus {
	a.googleMu.Lock()
	defer a.googleMu.Unlock()
	status := GoogleConnectionStatus{Available: googleOAuthConfig().ClientID != "", Status: "idle"}
	if a.googlePending != nil {
		status.Status = a.googlePending.status
		status.Message = a.googlePending.message
	}
	return status
}

// BeginGoogleConnection returns a system-browser URL. PKCE, a random state and
// a loopback-only callback bind the consent to this desktop instance.
func (a *App) BeginGoogleConnection() (string, error) {
	cfg := googleOAuthConfig()
	if cfg.ClientID == "" {
		return "", errors.New("conexão Google ainda não habilitada nesta instalação; use Gmail com senha de aplicativo ou SMTP")
	}
	a.googleMu.Lock()
	defer a.googleMu.Unlock()
	if a.googlePending != nil && a.googlePending.status == "waiting" {
		return "", errors.New("uma autorização Google já está em andamento")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	cfg.RedirectURL = "http://" + listener.Addr().String() + "/google/callback"
	connection := &googleConnection{state: oauth2.GenerateVerifier(), verifier: oauth2.GenerateVerifier(), status: "waiting", config: cfg}
	waitDone := make(chan struct{})
	var stopOnce sync.Once
	cancelWait := func() { stopOnce.Do(func() { close(waitDone) }) }
	connection.cancel = cancelWait
	mux := http.NewServeMux()
	mux.HandleFunc("GET /google/callback", func(w http.ResponseWriter, r *http.Request) { a.finishGoogleConnection(connection, w, r) })
	connection.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 5 * time.Second}
	a.googlePending = connection
	go func() { _ = connection.server.Serve(listener) }()
	go func() {
		timer := time.NewTimer(3 * time.Minute)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-waitDone:
			return
		}
		a.googleMu.Lock()
		if connection.status == "waiting" {
			connection.status = "error"
			connection.message = "Autorização expirou; tente conectar novamente"
		}
		a.googleMu.Unlock()
		_ = connection.server.Close()
	}()
	return cfg.AuthCodeURL(connection.state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"), oauth2.S256ChallengeOption(connection.verifier)), nil
}

func (a *App) finishGoogleConnection(c *googleConnection, w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("state")), []byte(c.state)) != 1 {
		http.Error(w, "Autorização inválida", http.StatusBadRequest)
		return
	}
	a.googleMu.Lock()
	if c.status != "waiting" {
		a.googleMu.Unlock()
		http.Error(w, "Autorização já utilizada", http.StatusBadRequest)
		return
	}
	c.status = "exchanging"
	a.googleMu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	token, err := c.config.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(c.verifier))
	if err == nil && token.RefreshToken == "" {
		err = errors.New("o Google não autorizou acesso contínuo; tente novamente")
	}
	if err == nil {
		err = a.saveGoogleToken(ctx, &c.config, token)
	}
	a.googleMu.Lock()
	if err != nil {
		c.status = "error"
		c.message = "Não foi possível conectar. Confira o consentimento e tente novamente."
	} else {
		c.status = "connected"
		c.message = "Conta Google conectada"
	}
	a.googleMu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, "<!doctype html><title>Candidate Allocation</title><p>Autorização concluída. Volte ao Candidate Allocation para conferir a conexão.</p>")
	// #nosec G118 -- listener shutdown intentionally outlives the callback request.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = c.server.Shutdown(ctx)
	}()
}

func (a *App) saveGoogleToken(ctx context.Context, oauth *oauth2.Config, token *oauth2.Token) error {
	client := oauth.Client(ctx, token)
	client.Timeout = 15 * time.Second
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return errors.New("não foi possível identificar a conta autorizada")
	}
	var user struct {
		Email    string `json:"email"`
		Verified bool   `json:"verified_email"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&user); err != nil {
		return err
	}
	if _, err := mail.ParseAddress(user.Email); err != nil || !user.Verified {
		return errors.New("conta Google sem e-mail verificado")
	}
	a.channelsMu.Lock()
	defer a.channelsMu.Unlock()
	cfg, err := a.noticeConfig()
	if err != nil {
		return err
	}
	cfg.Ativo = true
	cfg.Email = EmailConfig{Provedor: "google", Ativo: true, Remetente: user.Email, Usuario: user.Email, RefreshToken: token.RefreshToken, GoogleConnected: true}
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

func (a *App) googleClient(cfg EmailConfig) (*http.Client, error) {
	if cfg.RefreshToken == "" {
		return nil, errors.New("conecte sua conta Google")
	}
	oauth := googleOAuthConfig()
	if oauth.ClientID == "" {
		return nil, errors.New("conexão Google não habilitada nesta instalação")
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: 15 * time.Second})
	source := oauth.TokenSource(ctx, &oauth2.Token{RefreshToken: cfg.RefreshToken})
	token, err := source.Token()
	if err != nil {
		return nil, errors.New("a conexão Google expirou; conecte novamente")
	}
	client := oauth.Client(ctx, token)
	client.Timeout = 15 * time.Second
	return client, nil
}

func (a *App) sendGoogleEmail(cfg EmailConfig, d operations.Delivery) error {
	to, err := mail.ParseAddress(d.Destination)
	if err != nil {
		return errors.New("destinatário inválido")
	}
	from, err := mail.ParseAddress(cfg.Remetente)
	if err != nil {
		return errors.New("remetente inválido")
	}
	client, err := a.googleClient(cfg)
	if err != nil {
		return err
	}
	mime := "From: " + from.String() + "\r\nTo: " + to.String() + "\r\nSubject: Sua entrevista\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + d.Message
	body, err := json.Marshal(map[string]string{"raw": base64.URLEncoding.EncodeToString([]byte(mime))})
	if err != nil {
		return err
	}
	resp, err := client.Post("https://gmail.googleapis.com/gmail/v1/users/me/messages/send", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: %v", operations.ErrUncertain, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("google recusou o envio (%d); confira a conexão", resp.StatusCode)
	}
	var accepted struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&accepted); err != nil || accepted.ID == "" {
		return operations.ErrUncertain
	}
	return nil
}
