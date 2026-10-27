package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type clienteRemoto struct {
	base, sessao string
	client       *http.Client
}

func extrairRemoto(args []string) ([]string, *clienteRemoto, error) {
	out := []string{}
	base, sid := "http://localhost:8080", ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-servidor" || a == "-sessao" {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("%s exige um valor", a)
			}
			i++
			if a == "-servidor" {
				base = args[i]
			} else {
				sid = args[i]
			}
		} else if strings.HasPrefix(a, "-servidor=") {
			base = strings.TrimPrefix(a, "-servidor=")
		} else if strings.HasPrefix(a, "-sessao=") {
			sid = strings.TrimPrefix(a, "-sessao=")
		} else {
			out = append(out, a)
		}
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, nil, fmt.Errorf("URL de servidor inválida")
	}
	return out, &clienteRemoto{base: strings.TrimRight(base, "/"), sessao: sid, client: &http.Client{Timeout: 3 * time.Minute}}, nil
}
func (c *clienteRemoto) request(method, path string, data any) ([]byte, error) {
	var b []byte
	var err error
	if data != nil {
		b, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Session-Id", c.sessao)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		var out struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &out)
		if out.Error == "" {
			out.Error = string(raw)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, out.Error)
	}
	return raw, nil
}
func (c *clienteRemoto) operacao(path string, data any) error {
	raw, err := c.request(http.MethodPost, path, data)
	if err != nil {
		return err
	}
	var op ResultadoOperacao
	if err := json.Unmarshal(raw, &op); err != nil {
		return err
	}
	if op.LoteID != "" {
		fmt.Fprintf(os.Stderr, "Operação salva: %s Envio: %s.\n", op.Resumo, op.LoteID)
		deadline := time.Now().Add(3 * time.Minute)
		for time.Now().Before(deadline) {
			raw, err := c.request(http.MethodGet, "/api/avisos/lotes/"+url.PathEscape(op.LoteID)+"?resumo=true", nil)
			if err != nil {
				return err
			}
			var lote LoteAvisos
			if err := json.Unmarshal(raw, &lote); err != nil {
				return err
			}
			if lote.Pendentes == 0 {
				op.Envio = &lote
				imprimirOperacao(op)
				return erroEnvio(op)
			}
			time.Sleep(time.Second)
		}
		return fmt.Errorf("envio continua no servidor; consulte historico para acompanhar")
	}
	imprimirOperacao(op)
	return nil
}
func imprimirJSON(raw []byte) error {
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return err
	}
	fmt.Println(out.String())
	return nil
}
func executarRemoto(args []string, c *clienteRemoto) error {
	if len(args) == 0 {
		return fmt.Errorf("informe um comando")
	}
	cmd := args[0]
	if cmd == "alocar" {
		return c.alocar(args[1:])
	}
	if c.sessao == "" {
		return fmt.Errorf("informe -sessao (copie o identificador na aba Avisos do site)")
	}
	fs := novoFlagSet(cmd)
	switch cmd {
	case "resultado", "mesas", "historico", "sugestoes", "envio", "capacidade":
		pessoa := fs.String("pessoa", "", "nome ou id do candidato")
		lote := fs.String("lote", "", "identificador do envio")
		antes := fs.Int64("antes", 0, "cursor da página anterior")
		limite := fs.Int("limite", 20, "envios por página (máximo 50)")
		if _, err := parseFlags(fs, args[1:]); err != nil {
			return err
		}
		path := "/api/resultado"
		if cmd == "historico" {
			path = fmt.Sprintf("/api/avisos/historico?antes=%d&limite=%d", *antes, *limite)
		}
		if cmd == "envio" {
			path = "/api/avisos/lotes/" + url.PathEscape(*lote)
		}
		if cmd == "capacidade" {
			path = "/api/capacidade"
		}
		if cmd == "sugestoes" {
			path = "/api/realocar/sugestoes?pessoa=" + url.QueryEscape(*pessoa)
		}
		b, err := c.request(http.MethodGet, path, nil)
		if err != nil {
			return err
		}
		return imprimirJSON(b)
	case "realocar-dia", "realocar-mesa", "realocar-pessoa":
		de := fs.String("de", "", "dia ou intervalo de origem")
		para := fs.String("para", "", "novo dia/horário")
		horario := fs.String("horario", "", "horário da pessoa")
		pessoa := fs.String("pessoa", "", "nome ou id")
		mesa := fs.Int("mesa", 0, "id da mesa")
		avisar := fs.Bool("avisar", false, "avisar participantes afetados")
		if _, err := parseFlags(fs, args[1:]); err != nil {
			return err
		}
		tipo := strings.TrimPrefix(cmd, "realocar-")
		return c.operacao("/api/realocar", pedidoRealocar{Tipo: tipo, De: *de, Para: *para, Horario: *horario, Pessoa: *pessoa, MesaID: *mesa, Avisar: *avisar})
	case "avaliar":
		pessoa := fs.String("pessoa", "", "nome do candidato")
		id := fs.Int("id", 0, "id do candidato")
		res := fs.String("resultado", "", "passou, não passou ou pendente")
		listar := fs.Bool("listar", false, "listar avaliações")
		if _, err := parseFlags(fs, args[1:]); err != nil {
			return err
		}
		if *listar {
			b, err := c.request(http.MethodGet, "/api/resultado", nil)
			if err != nil {
				return err
			}
			return imprimirJSON(b)
		}
		return c.operacao("/api/avaliar", pedidoAvaliar{Itens: []itemAvaliacao{{ID: *id, Pessoa: *pessoa, Resultado: *res}}})
	case "enviar":
		pessoa := fs.String("pessoa", "", "nome ou id (vazio: todos)")
		mesa := fs.Int("mesa", 0, "mesa específica (zero: todos)")
		if _, err := parseFlags(fs, args[1:]); err != nil {
			return err
		}
		return c.operacao("/api/avisos/enviar", pedidoEnvio{Pessoa: *pessoa, MesaID: *mesa})
	case "repetir-falhas":
		id := fs.String("lote", "", "id do envio")
		if _, err := parseFlags(fs, args[1:]); err != nil {
			return err
		}
		return c.operacao("/api/avisos/repetir", map[string]string{"id": *id})
	case "avisos":
		arquivo := fs.String("config", "", "JSON com canais e mensagem; segredos vazios preservam os salvos")
		testar := fs.Bool("testar", false, "verificar conexão sem enviar")
		previa := fs.Bool("previa", false, "prévia da mensagem")
		if _, err := parseFlags(fs, args[1:]); err != nil {
			return err
		}
		raw, err := c.request(http.MethodGet, "/api/avisos", nil)
		if err != nil {
			return err
		}
		if *arquivo != "" {
			raw, err = os.ReadFile(*arquivo)
			if err != nil {
				return err
			}
		}
		var cfg ConfigAvisos
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return err
		}
		path := "/api/avisos"
		if *testar {
			path = "/api/avisos/testar"
		} else if *previa {
			path = "/api/avisos/previa"
		} else if *arquivo == "" {
			return imprimirJSON(raw)
		}
		raw, err = c.request(http.MethodPost, path, cfg)
		if err != nil {
			return err
		}
		return imprimirJSON(raw)
	case "exportar", "excel":
		saida := fs.String("saida", "alocacao.xlsx", "arquivo Excel")
		if _, err := parseFlags(fs, args[1:]); err != nil {
			return err
		}
		b, err := c.request(http.MethodGet, "/api/export", nil)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(*saida), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(*saida, b, 0o600); err != nil {
			return err
		}
		fmt.Println("Excel salvo em", *saida)
		return nil
	default:
		return fmt.Errorf("comando remoto desconhecido: %s", cmd)
	}
}
func (c *clienteRemoto) alocar(args []string) error {
	fs := novoFlagSet("alocar")
	arquivo := fs.String("planilha", "", "arquivo Excel")
	sem := fs.Bool("sem-preferencia", false, "desliga preferências")
	horarios := fs.String("horarios", "", "horários separados por vírgula")
	opcoes := fs.Int("opcoes", 5, "colunas de opção")
	canais := fs.String("avisos", "", "JSON privado dos canais (opcional, com autorização para envio automático)")
	dominio := fs.String("dominio", "@al.insper.edu.br", "domínio de e-mail")
	mesas := fs.Int("mesas", 5, "mesas por horário")
	minP := fs.Int("min", 5, "mínimo por mesa")
	maxP := fs.Int("max", 8, "máximo por mesa")
	avaliadores := fs.Int("avaliadores", 5, "avaliadores por mesa")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	var canaisJSON []byte
	if *canais != "" {
		var err error
		canaisJSON, err = os.ReadFile(*canais)
		if err != nil {
			return err
		}
	}
	data, err := os.ReadFile(*arquivo)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", filepath.Base(*arquivo))
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	for k, v := range map[string]string{"usarPreferencia": strconv.FormatBool(!*sem), "nOpcoes": strconv.Itoa(*opcoes), "emailDomain": *dominio} {
		if err := writer.WriteField(k, v); err != nil {
			return err
		}
	}
	if len(canaisJSON) > 0 {
		if err := writer.WriteField("avisos", string(canaisJSON)); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.base+"/api/upload", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload: %s", raw)
	}
	var up struct {
		SessionID string        `json:"sessionId"`
		Mapping   []MappingItem `json:"mapping"`
	}
	if err := json.Unmarshal(raw, &up); err != nil {
		return err
	}
	c.sessao = up.SessionID
	raw, err = c.request(http.MethodPost, "/api/build-usuarios", up.Mapping)
	if err != nil {
		return err
	}
	var users UsuariosResponse
	if err := json.Unmarshal(raw, &users); err != nil {
		return err
	}
	if len(users.Duplicates) > 0 {
		_ = imprimirJSON(raw)
		return fmt.Errorf("candidatos duplicados; corrija a planilha e importe novamente. Sessão: %s", c.sessao)
	}
	cands := []Usuario{}
	for i := 1; i <= len(users.Usuarios); i++ {
		u := users.Usuarios[i]
		if len(u.Erros) > 0 {
			fmt.Fprintf(os.Stderr, "Avisos de validação (%s): %v\n", u.Usuario.Nome, u.Erros)
		}
		cands = append(cands, u.Usuario)
	}
	if _, err := c.request(http.MethodPost, "/api/save-usuarios", cands); err != nil {
		return err
	}
	for _, aba := range []string{"avaliador", "restricao"} {
		m, err := c.request(http.MethodPost, "/api/suggest-"+aba, nil)
		if err != nil {
			return err
		}
		plural := map[string]string{"avaliador": "avaliadores", "restricao": "restricoes"}[aba]
		b, err := c.request(http.MethodPost, "/api/build-"+plural, json.RawMessage(m))
		if err != nil {
			return err
		}
		if _, err := c.request(http.MethodPost, "/api/save-"+plural, json.RawMessage(b)); err != nil {
			return err
		}
	}
	hs, _ := json.Marshal(normalizarHorarios([]string{*horarios}))
	q := url.Values{"sessionId": {c.sessao}, "sem_preferencia": {strconv.FormatBool(*sem)}, "horarios_livres": {string(hs)}, "mesas_por_horario": {strconv.Itoa(*mesas)}, "min_pessoas_por_mesa": {strconv.Itoa(*minP)}, "max_pessoas_por_mesa": {strconv.Itoa(*maxP)}, "avaliadores_por_mesa": {strconv.Itoa(*avaliadores)}}
	b, err := c.request(http.MethodGet, "/api/alocar?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	done := false
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "data: ") {
			var ev struct {
				Done  bool   `json:"done"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
				return err
			}
			if ev.Error != "" {
				return fmt.Errorf("alocação: %s", ev.Error)
			}
			done = done || ev.Done
		}
	}
	if !done {
		return fmt.Errorf("alocação não concluída")
	}
	fmt.Printf("Alocação disponível no site: %s/resultado?sessao=%s\nCLI: -servidor %s -sessao %s\n", c.base, c.sessao, c.base, c.sessao)
	return nil
}

func novoFlagSet(nome string) *flag.FlagSet { return flag.NewFlagSet(nome, flag.ContinueOnError) }
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		return nil, fmt.Errorf("argumentos inesperados: %v", fs.Args())
	}
	return fs.Args(), nil
}
func imprimirOperacao(op ResultadoOperacao) {
	raw, err := json.MarshalIndent(op, "", "  ")
	if err == nil {
		fmt.Println(string(raw))
	}
}
func executarCLI(args []string) error {
	if len(args) == 0 || args[0] == "ajuda" || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Uso: alocador <comando> [-servidor http://localhost:8080] [-sessao ID] [opções]
Comandos: alocar, resultado, capacidade, realocar-dia, realocar-mesa,
realocar-pessoa, sugestoes, avaliar, avisos, enviar, historico, envio,
repetir-falhas, exportar. Use <comando> -h para opções.
O servidor (alocador servir) compartilha a mesma agenda com app e CLI.`)
		return nil
	}
	rest, c, err := extrairRemoto(args)
	if err != nil {
		return err
	}
	return executarRemoto(rest, c)
}
