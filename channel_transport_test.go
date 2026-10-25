package main

import (
	"candidate_alocator/back/operations"
	"context"
	"encoding/json"
	"golang.org/x/oauth2"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleTokenStorageRefreshAndGmailSend(t *testing.T) {
	a := operationsApp(t)
	t.Setenv("CANDIDATE_GOOGLE_CLIENT_ID", "unit-test-client")
	old := http.DefaultTransport
	http.DefaultTransport = testTransport(func(r *http.Request) (*http.Response, error) {
		body := "{}"
		switch {
		case strings.Contains(r.URL.Path, "token"):
			body = `{"access_token":"unit-access","refresh_token":"unit-refresh","token_type":"Bearer","expires_in":3600}`
		case strings.Contains(r.URL.Path, "userinfo"):
			body = `{"email":"organization@example.test","verified_email":true}`
		case strings.Contains(r.URL.Path, "messages/send"):
			if r.Header.Get("Authorization") != "Bearer unit-access" {
				t.Error("missing OAuth authorization")
			}
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload["raw"] == "" {
				t.Error("missing MIME payload")
			}
			body = `{"id":"unit-message"}`
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = old })
	config := googleOAuthConfig()
	if err := a.saveGoogleToken(context.Background(), &config, &oauth2.Token{AccessToken: "unit-access", RefreshToken: "unit-refresh", TokenType: "Bearer"}); err != nil {
		t.Fatal(err)
	}
	private, _ := a.noticeConfig()
	public, _ := a.GetChannelConfig()
	if public.Email.RefreshToken != "" || !public.Email.GoogleConnected || private.Email.RefreshToken == "" {
		t.Fatal("Google token handling incorrect")
	}
	if err := a.sendGoogleEmail(private.Email, operations.Delivery{Destination: "candidate@example.test", Message: "Entrevista"}); err != nil {
		t.Fatal(err)
	}
	if result, err := a.TestChannels(); err != nil || !strings.Contains(result["email"], "autorizada") {
		t.Fatal(result, err)
	}
	connection := &googleConnection{state: "unit-state", verifier: "unit-verifier", status: "waiting", config: config, server: &http.Server{ReadHeaderTimeout: time.Second}}
	response := httptest.NewRecorder()
	a.finishGoogleConnection(connection, response, httptest.NewRequest("GET", "/google/callback?state=unit-state&code=unit-code", nil))
	if connection.status != "connected" {
		t.Fatal(connection.status)
	}
	if err := a.DisconnectGoogle(); err != nil {
		t.Fatal(err)
	}
	public, _ = a.GetChannelConfig()
	if public.Email.GoogleConnected || public.Email.Ativo {
		t.Fatal("Google account remains connected")
	}
}

func TestMetaUsesApprovedTemplateAndValidatesAcceptance(t *testing.T) {
	var payload map[string]any
	client := &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer unit-token" {
			t.Error("authorization missing")
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"messages":[{"id":"unit-message"}]}`))}, nil
	})}
	cfg := WhatsConfig{Provedor: "meta", PhoneID: "123", Template: "entrevista", Token: "unit-token", VariaveisTemplate: []string{"nome", "horario"}, httpClient: client}
	if err := enviarWhatsAppVariaveis(cfg, "5511999999999", "texto diferente", map[string]string{"nome": "Ana", "horario": "14-16"}); err != nil {
		t.Fatal(err)
	}
	if payload["type"] != "template" {
		t.Fatal(payload)
	}
	template := payload["template"].(map[string]any)
	if template["name"] != "entrevista" {
		t.Fatal(template)
	}
	cfg.VariaveisTemplate = []string{"unknown"}
	if err := enviarWhatsAppVariaveis(cfg, "5511999999999", "", nil); err == nil {
		t.Fatal("invalid template variable accepted")
	}
	cfg.PhoneID = "invalid"
	if err := enviarWhatsAppVariaveis(cfg, "5511999999999", "", nil); err == nil {
		t.Fatal("invalid number ID accepted")
	}
}

func TestSMTPConnectionAndLocalDelivery(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = io.WriteString(conn, "220 unit SMTP\r\n")
		buffer := make([]byte, 4096)
		var content strings.Builder
		data := false
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				return
			}
			line := string(buffer[:n])
			if data {
				content.WriteString(line)
				if strings.Contains(content.String(), "\r\n.\r\n") {
					received <- content.String()
					_, _ = io.WriteString(conn, "250 accepted\r\n")
					return // SMTP accepted DATA; simulate a lost QUIT response.
				}
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				_, _ = io.WriteString(conn, "250 unit\r\n")
			case strings.HasPrefix(line, "DATA"):
				data = true
				_, _ = io.WriteString(conn, "354 content\r\n")
			case strings.HasPrefix(line, "QUIT"):
				_, _ = io.WriteString(conn, "221 bye\r\n")
				return
			default:
				_, _ = io.WriteString(conn, "250 ok\r\n")
			}
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := enviarEmailSMTP(EmailConfig{Host: "127.0.0.1", Porta: port, Remetente: "org@example.test"}, "candidate@example.test", "Entrevista", "Olá Ana"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(<-received, "Olá Ana") {
		t.Fatal("message body absent")
	}
	if _, err := clienteSMTP(EmailConfig{}); err == nil {
		t.Fatal("empty SMTP config accepted")
	}
}
