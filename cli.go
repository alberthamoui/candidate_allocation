package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const ajudaCLI = `Uso:
  o programa                         sobe o site em http://localhost:8080
  o programa servir                  o mesmo

  o programa alocar -planilha ARQ [-sem-preferencia] [-horarios "qua 14-16,qui 10-12"]
  o programa avisos -mensagem "Olá {{nome}}, sua entrevista é em {{data}} ({{horario}})."
  o programa enviar                  manda a mensagem para todo mundo alocado
  o programa realocar-dia -de "quarta 14-16" -para "quinta 14-16" [-avisar]
  o programa realocar-mesa -mesa 301 -para "quinta 14-16" [-avisar]
  o programa realocar-pessoa -pessoa "Ana" -horario "quinta 10-12" [-avisar]
  o programa avaliar -pessoa "Ana" -resultado aprovado
  o programa avaliar -listar
  o programa exportar -saida resultado.xlsx

A pasta -workspace (padrão alocacao-cli) guarda a alocação entre os comandos.
Use -servidor URL -sessao ID para trabalhar na mesma alocação do site.
Inspeção: mesas, resultado, sugestoes -pessoa ID, historico, repetir-falhas -lote ID.
Configuração remota: avisos -config arquivo.json; avisos -testar; avisos -previa.
Variáveis da mensagem: {{nome}} {{data}} {{horario}} {{mesa}} {{email}} {{curso}} {{numero}}
-avisar e enviar só disparam e-mail/WhatsApp se você configurou avisos e aceitou o envio.
`

type estadoCLI struct {
	Resultado       *AlocacaoResponse  `json:"resultado,omitempty"`
	Avisos          ConfigAvisos       `json:"avisos"`
	UsarPreferencia bool               `json:"usar_preferencia"`
	Avaliacoes      map[int]string     `json:"avaliacoes,omitempty"`
	Parametros      ParametrosAlocacao `json:"parametros"`
	Historico       []LoteAvisos       `json:"historico"`
}

func executarCLI(args []string) error {
	args, remoto, err := extrairRemoto(args)
	if err != nil {
		return err
	}
	if remoto != nil {
		return executarRemoto(args, remoto)
	}
	if len(args) == 0 {
		fmt.Print(ajudaCLI)
		return nil
	}
	switch args[0] {
	case "ajuda", "help", "-h", "--help":
		fmt.Print(ajudaCLI)
		return nil
	case "mesas", "resultado", "sugestoes", "historico", "repetir-falhas":
		return cmdInspecionar(args)
	case "alocar":
		return cmdAlocar(args[1:])
	case "avisos":
		return cmdAvisos(args[1:])
	case "enviar":
		return cmdEnviar(args[1:])
	case "realocar-dia":
		return cmdRealocarDia(args[1:])
	case "realocar-mesa":
		return cmdRealocarMesa(args[1:])
	case "realocar-pessoa":
		return cmdRealocarPessoa(args[1:])
	case "avaliar":
		return cmdAvaliar(args[1:])
	case "exportar", "excel":
		return cmdExportar(args[1:])
	default:
		return fmt.Errorf("comando desconhecido %q\n\n%s", args[0], ajudaCLI)
	}
}

func novoFlagSet(nome string) *flag.FlagSet {
	fs := flag.NewFlagSet(nome, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, ajudaCLI)
		fs.PrintDefaults()
	}
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) (map[string]bool, error) {
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("argumento inesperado %q", fs.Arg(0))
	}
	definido := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { definido[f.Name] = true })
	return definido, nil
}

func caminhoEstado(dir string) string {
	return filepath.Join(dir, "estado.json")
}

func salvarEstado(dir string, s *Session) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	est := estadoCLI{
		Resultado:       s.lastResult,
		Avisos:          s.avisos,
		UsarPreferencia: s.usarPreferencia,
		Avaliacoes:      s.avaliacoes,
		Parametros:      s.parametros, Historico: s.historico,
	}
	b, err := json.MarshalIndent(est, "", "  ")
	if err != nil {
		return err
	}
	// 0600: o arquivo pode guardar a senha do SMTP.
	return gravarArquivoAtomico(caminhoEstado(dir), append(b, '\n'))
}

func carregarEstado(dir string) (*Session, error) {
	b, err := os.ReadFile(caminhoEstado(dir))
	if err != nil {
		return nil, fmt.Errorf("não achei a alocação em %s (rode alocar antes): %w", dir, err)
	}
	var est estadoCLI
	if err := json.Unmarshal(b, &est); err != nil {
		return nil, fmt.Errorf("lendo %s: %w", caminhoEstado(dir), err)
	}
	if est.Avaliacoes == nil {
		est.Avaliacoes = map[int]string{}
	}
	s := &Session{
		lastResult:      est.Resultado,
		avisos:          est.Avisos,
		usarPreferencia: est.UsarPreferencia,
		avaliacoes:      est.Avaliacoes,
		parametros:      est.Parametros, historico: est.Historico,
	}
	s.aplicarAvaliacoes(s.lastResult)
	return s, nil
}

func imprimirOperacao(op ResultadoOperacao) {
	fmt.Println(op.Resumo)
	for _, a := range op.Avisos {
		estado := "preparado, não enviado"
		if a.Enviado {
			estado = "aceito pelo provedor"
		}
		if a.Erro != "" {
			estado = "não enviado: " + a.Erro
		}
		fmt.Printf("  - %s [%s] %s — %s\n", a.Nome, a.Canal, a.Destino, estado)
		if a.Mensagem != "" {
			fmt.Printf("    %s\n", a.Mensagem)
		}
	}
}

func prepararSessaoArquivo(data []byte, nOpcoes int, dominio string, usarPreferencia bool) (*Session, error) {
	db, err := abrirBancoMemoria()
	if err != nil {
		return nil, err
	}
	if !usarPreferencia {
		nOpcoes = 0
	}
	s := &Session{
		db:              db,
		emailDomain:     dominio,
		usarPreferencia: usarPreferencia,
		avaliacoes:      map[int]string{},
	}
	mapping, err := s.SuggestMapping(data, nOpcoes, dominio)
	if err != nil {
		return nil, err
	}
	usuarios, err := s.BuildUsuariosWithMapping(mapping)
	if err != nil {
		return nil, err
	}
	if len(usuarios.Duplicates) > 0 {
		return nil, fmt.Errorf("há candidatos duplicados na planilha; resolva antes de alocar pela CLI")
	}
	var problemas []string
	var cands []Usuario
	for i := 1; i <= len(usuarios.Usuarios); i++ {
		item := usuarios.Usuarios[i]
		if len(item.Erros) > 0 {
			var msgs []string
			for _, e := range item.Erros {
				msgs = append(msgs, e.Field+": "+e.Msg)
			}
			problemas = append(problemas, fmt.Sprintf("linha %d (%s): %s", i, item.Usuario.Nome, strings.Join(msgs, "; ")))
		}
		cands = append(cands, item.Usuario)
	}
	if len(problemas) > 0 {
		return nil, fmt.Errorf("a planilha tem dados inválidos:\n%s", strings.Join(problemas, "\n"))
	}
	if err := s.SaveUsuarios(cands); err != nil {
		return nil, err
	}
	mAv, err := s.SuggestMappingAvaliador()
	if err != nil {
		return nil, err
	}
	avals, err := s.BuildAvaliadoresWithMapping(mAv)
	if err != nil {
		return nil, err
	}
	if err := s.SaveAvaliadores(avals); err != nil {
		return nil, err
	}
	mRe, err := s.SuggestMappingRestricao()
	if err != nil {
		return nil, err
	}
	restricoes, err := s.BuildRestricoesWithMapping(mRe)
	if err != nil {
		return nil, err
	}
	if err := s.SaveRestricoes(restricoes); err != nil {
		return nil, err
	}
	return s, nil
}

func abrirBancoMemoria() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := setupConn(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func cmdAlocar(args []string) error {
	fs := novoFlagSet("alocar")
	planilha := fs.String("planilha", "", "arquivo .xlsx com candidatos, avaliadores e restrições")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	opcoes := fs.Int("opcoes", 5, "colunas de preferência de horário (ignorado com -sem-preferencia)")
	dominio := fs.String("dominio", "@al.insper.edu.br", "domínio do e-mail institucional")
	semPref := fs.Bool("sem-preferencia", false, "não usar a preferência de horário dos candidatos")
	horarios := fs.String("horarios", "", "horários quando a preferência está desligada, separados por vírgula")
	mesas := fs.Int("mesas", 0, "mesas por horário (0 = padrão)")
	minP := fs.Int("min", 0, "mínimo de candidatos por mesa (0 = padrão)")
	maxP := fs.Int("max", 0, "máximo de candidatos por mesa (0 = padrão)")
	avals := fs.Int("avaliadores", 0, "avaliadores por mesa (0 = padrão)")
	saida := fs.String("saida", "", "caminho do Excel (padrão: workspace/alocacao.xlsx)")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	if *planilha == "" {
		return fmt.Errorf("informe -planilha")
	}
	data, err := os.ReadFile(*planilha)
	if err != nil {
		return err
	}
	s, err := prepararSessaoArquivo(data, *opcoes, *dominio, !*semPref)
	if err != nil {
		return err
	}
	defer func() { _ = s.db.Close() }()
	if anterior, errEstado := carregarEstado(*dir); errEstado == nil {
		s.avisos = anterior.avisos
	}
	param := parametrosAlocacaoPadrao()
	if *mesas > 0 {
		param.MesasPorHorario = *mesas
	}
	if *minP > 0 {
		param.MinPessoasPorMesa = *minP
	}
	if *maxP > 0 {
		param.MaxPessoasPorMesa = *maxP
	}
	if *avals > 0 {
		param.AvaliadoresPorMesa = *avals
	}
	param = s.comPreferenciaDaSessao(param)
	if param.SemPreferencia {
		param.HorariosLivres = normalizarHorarios([]string{*horarios})
	}
	res, err := s.RunAlocacao(context.Background(), param, func(v any) {
		if ev, ok := v.(progressEvent); ok && ev.Step != "" {
			fmt.Printf("%s (%d%%)\n", ev.Step, ev.Pct)
		}
	})
	if err != nil {
		return err
	}
	s.lastResult = &res
	for _, l := range s.HistoricoAvisos() {
		if l.Status == "enviando" {
			s.executarLote(l.ID, s.avisos)
		}
	}
	if err := salvarEstado(*dir, s); err != nil {
		return err
	}
	destino := *saida
	if destino == "" {
		destino = filepath.Join(*dir, "alocacao.xlsx")
	}
	arq, err := s.ExportResultado()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(destino, arq, 0o644); err != nil {
		return err
	}
	fmt.Printf("Alocados: %d. Estado em %s. Excel em %s.\n", res.TotalAlocados, caminhoEstado(*dir), destino)
	return nil
}

func cmdAvisos(args []string) error {
	fs := novoFlagSet("avisos")
	config := fs.String("config", "", "arquivo JSON com configuração completa")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	mensagem := fs.String("mensagem", "", "texto com variáveis, por exemplo {{nome}} e {{data}}")
	desligado := fs.Bool("desligado", false, "desliga o envio automático")
	emailOn := fs.Bool("email", false, "ativa o e-mail")
	host := fs.String("smtp-host", "", "servidor SMTP")
	porta := fs.Int("smtp-porta", 0, "porta SMTP (587 é a usual)")
	usuario := fs.String("smtp-usuario", "", "usuário do SMTP")
	senha := fs.String("smtp-senha", "", "senha do SMTP")
	remetente := fs.String("remetente", "", "endereço que aparece como remetente")
	whatsOn := fs.Bool("whatsapp", false, "ativa o WhatsApp")
	urlZap := fs.String("whatsapp-url", "", "URL que recebe POST JSON {to, message}")
	token := fs.String("whatsapp-token", "", "token opcional, enviado como Bearer")
	definido, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	s, errEstado := carregarEstado(*dir)
	if errEstado != nil {
		s = &Session{usarPreferencia: true, avaliacoes: map[int]string{}}
	}
	cfg := s.avisos
	if *config != "" {
		b, err := os.ReadFile(*config)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &cfg); err != nil {
			return err
		}
	}
	if definido["mensagem"] {
		cfg.Mensagem = *mensagem
		cfg.Ativo = true
	}
	if definido["desligado"] && *desligado {
		cfg.Ativo = false
	}
	if definido["email"] {
		cfg.Email.Ativo = *emailOn
		if *emailOn {
			cfg.Ativo = true
		}
	}
	if definido["smtp-host"] {
		cfg.Email.Host = *host
	}
	if definido["smtp-porta"] {
		cfg.Email.Porta = *porta
	}
	if definido["smtp-usuario"] {
		cfg.Email.Usuario = *usuario
	}
	if definido["smtp-senha"] {
		cfg.Email.Senha = *senha
	}
	if definido["remetente"] {
		cfg.Email.Remetente = *remetente
	}
	if definido["whatsapp"] {
		cfg.WhatsApp.Ativo = *whatsOn
		if *whatsOn {
			cfg.Ativo = true
		}
	}
	if definido["whatsapp-url"] {
		cfg.WhatsApp.URL = *urlZap
	}
	if definido["whatsapp-token"] {
		cfg.WhatsApp.Token = *token
	}
	if !definido["mensagem"] && !definido["email"] && !definido["whatsapp"] && !definido["desligado"] && cfg.Mensagem == "" {
		return fmt.Errorf("informe -mensagem. Exemplo: Olá {{nome}}, sua entrevista é em {{data}}, horário {{horario}}")
	}
	if err := s.ConfigurarAvisos(cfg); err != nil {
		return err
	}
	if err := salvarEstado(*dir, s); err != nil {
		return err
	}
	estado := "ligado"
	if !cfg.Ativo {
		estado = "desligado"
	}
	fmt.Printf("Avisos %s. Mensagem: %s\n", estado, cfg.Mensagem)
	fmt.Printf("  e-mail: %s | WhatsApp: %s\n", ligado(cfg.Email.Ativo), ligado(cfg.WhatsApp.Ativo))
	return nil
}

func ligado(v bool) string {
	if v {
		return "ativo"
	}
	return "desligado"
}

func cmdEnviar(args []string) error {
	fs := novoFlagSet("enviar")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	s, err := carregarEstado(*dir)
	if err != nil {
		return err
	}
	op, err := s.EnviarAvisosTodos()
	if err != nil {
		return err
	}
	if err := salvarEstado(*dir, s); err != nil {
		return err
	}
	imprimirOperacao(op)
	return erroEnvio(op)
}

func cmdRealocarDia(args []string) error {
	fs := novoFlagSet("realocar-dia")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	de := fs.String("de", "", "horário atual")
	para := fs.String("para", "", "novo horário")
	avisar := fs.Bool("avisar", false, "enviar e-mail e WhatsApp para todo mundo deste horário")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	return comEstado(*dir, func(s *Session) (ResultadoOperacao, error) {
		return s.RealocarDia(*de, *para, *avisar)
	})
}

func cmdRealocarMesa(args []string) error {
	fs := novoFlagSet("realocar-mesa")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	mesa := fs.Int("mesa", 0, "id da mesa")
	para := fs.String("para", "", "novo horário")
	avisar := fs.Bool("avisar", false, "enviar e-mail e WhatsApp para a mesa")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	if *mesa == 0 {
		return fmt.Errorf("informe -mesa")
	}
	return comEstado(*dir, func(s *Session) (ResultadoOperacao, error) {
		return s.RealocarMesa(*mesa, *para, *avisar)
	})
}

func cmdRealocarPessoa(args []string) error {
	fs := novoFlagSet("realocar-pessoa")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	pessoa := fs.String("pessoa", "", "nome ou id do candidato")
	horario := fs.String("horario", "", "novo horário")
	mesa := fs.Int("mesa", 0, "id da mesa de destino (opcional)")
	avisar := fs.Bool("avisar", false, "avisar esta pessoa")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	return comEstado(*dir, func(s *Session) (ResultadoOperacao, error) {
		return s.RealocarPessoa(*pessoa, *horario, *mesa, *avisar)
	})
}

func cmdAvaliar(args []string) error {
	fs := novoFlagSet("avaliar")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	pessoa := fs.String("pessoa", "", "nome do candidato")
	id := fs.Int("id", 0, "id do candidato")
	resultado := fs.String("resultado", "", "aprovado ou reprovado")
	listar := fs.Bool("listar", false, "mostra a avaliação de cada candidato")
	definido, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	s, err := carregarEstado(*dir)
	if err != nil {
		return err
	}
	if *listar || (!definido["resultado"] && !definido["pessoa"] && !definido["id"]) {
		imprimirAvaliacoes(s)
		return nil
	}
	op, err := s.Avaliar(*pessoa, *id, *resultado)
	if err != nil {
		return err
	}
	if err := salvarEstado(*dir, s); err != nil {
		return err
	}
	imprimirOperacao(op)
	return erroEnvio(op)
}

func imprimirAvaliacoes(s *Session) {
	if err := s.exigirResultado(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Avaliações:")
	for _, m := range s.lastResult.Mesas {
		for _, c := range m.Candidatos {
			marca := textoAvaliacao(c.Avaliacao)
			if marca == "" {
				marca = "pendente"
			}
			fmt.Printf("  %d  %s  %s  %s\n", c.ID, c.Nome, m.DiaNome, marca)
		}
	}
	for _, c := range s.lastResult.NaoAlocadosInfo {
		marca := textoAvaliacao(c.Avaliacao)
		if marca == "" {
			marca = "pendente"
		}
		fmt.Printf("  %d  %s  sem mesa  %s\n", c.ID, c.Nome, marca)
	}
}

func cmdExportar(args []string) error {
	fs := novoFlagSet("exportar")
	dir := fs.String("workspace", "alocacao-cli", "pasta da alocação")
	saida := fs.String("saida", "", "caminho do .xlsx (padrão: workspace/alocacao.xlsx)")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	s, err := carregarEstado(*dir)
	if err != nil {
		return err
	}
	destino := *saida
	if destino == "" {
		destino = filepath.Join(*dir, "alocacao.xlsx")
	}
	arq, err := s.ExportResultado()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(destino, arq, 0o644); err != nil {
		return err
	}
	fmt.Printf("Excel salvo em %s\n", destino)
	return nil
}

func comEstado(dir string, fn func(*Session) (ResultadoOperacao, error)) error {
	s, err := carregarEstado(dir)
	if err != nil {
		return err
	}
	op, err := fn(s)
	if err != nil {
		return err
	}
	if err := salvarEstado(dir, s); err != nil {
		return err
	}
	imprimirOperacao(op)
	return erroEnvio(op)
}
