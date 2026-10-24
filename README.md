# Candidate Allocation

Aplicação desktop e CLI para importar candidatos, restrições e avaliadores de uma planilha, revisar os dados, configurar critérios e executar uma alocação exata em grupos.

## Branch principal

A branch principal é **`mainjoao`**. Novas branches devem partir de `origin/mainjoao`, e os pull requests devem apontar para `mainjoao`. Trabalhe em uma branch/worktree própria e entregue o PR para revisão. Consulte também [AGENTS.md](AGENTS.md).

## Operar a agenda

Depois de parar/concluir a busca, cada candidato oferece **Realocar**, **Passou**, **Não passou** e **Avisar**. Cada mesa oferece realocação, avaliação em lote e envio. Os indicadores de pendências filtram candidatos sem avaliação, com aviso pendente/falho, com alerta ou sem mesa.

A realocação manual é livre: os melhores destinos são ordenados usando os contratos do solver e os conflitos aparecem como alertas. Horário novo pode criar uma mesa sem avaliadores. O Excel acompanha a agenda e as avaliações atuais. CPF e preferência de horários são opcionais; quando não há preferências, informe os horários na configuração. Adicione Telefone/WhatsApp como coluna adicional e inclua-a no mapeamento; matrícula não é usada como telefone.

Em **Canais e mensagens**, conecte e-mail/WhatsApp, escolha uma mensagem inicial e use variáveis como `{{nome}}`, `{{data}}`, `{{horario}}` e `{{mesa}}`. O envio exige confirmação com prévia dos destinatários. O envio automático ao concluir outra alocação tem autorização própria, desligada por padrão.

- Google: conexão pelo navegador com OAuth e PKCE. A instalação precisa de um cliente OAuth **Desktop** com Gmail API habilitada, configurado em `CANDIDATE_GOOGLE_CLIENT_ID` e `CANDIDATE_GOOGLE_CLIENT_SECRET`. O consentimento pede envio de e-mail e identificação da conta, sem ler a caixa postal. A distribuição pública pode exigir verificação do aplicativo pelo Google. Credenciais reais são configuradas fora do Git.
- Gmail por senha de aplicativo: informe o remetente e a senha de aplicativo; servidor e porta são preenchidos automaticamente. SMTP de outros provedores fica na configuração avançada.
- WhatsApp Business: conexão guiada com ID do número, token e nome do modelo aprovado da Meta. Número e modelo precisam estar habilitados na conta Business. Não há conexão de conta pessoal por QR Code. O texto livre do editor é usado para e-mail/webhook; a Meta usa o modelo aprovado e suas variáveis na ordem configurada.

**Salvar e testar conexão** não envia mensagens. Aceitação pelo provedor não é confirmação de entrega. O histórico é paginado e consulta detalhes de destinatários apenas sob demanda; atualiza ao abrir e por eventos de conclusão, sem consulta periódica quando parado. Repetição considera somente falhas confirmadas. Envios interrompidos durante a chamada ao provedor ficam para conferência, para evitar repetir algo possivelmente aceito.

Agendas e configurações ficam em `operations.db` no diretório de configuração do usuário, ou em `CANDIDATE_ALLOCATOR_DATA_DIR`. Cada mensagem é atualizada em seu próprio registro SQLite, sem regravar a agenda/histórico inteiro. O cache guarda até oito agendas e descarrega itens após quinze minutos de inatividade; agendas salvas são recuperadas sob demanda. Faça backup desse diretório; as credenciais são mascaradas ao consultar a configuração. No desktop, o envio em segundo plano exige manter o aplicativo aberto.

CLI e desktop usam o mesmo armazenamento no mesmo usuário/máquina:

```bash
go run . cli agenda resultado
go run . cli agenda sugestoes -pessoa 1
go run . cli agenda realocar-pessoa -pessoa 1 -destino 2
go run . cli agenda realocar-dia -de quarta -para sexta -avisar
go run . cli agenda avaliar -pessoa 1 -resultado aprovado
go run . cli agenda canais -arquivo canais.json
go run . cli agenda testar-canais
go run . cli agenda previa -mesa 2
go run . cli agenda enviar -mesa 2
go run . cli agenda historico
go run . cli agenda detalhes -lote 1
go run . cli agenda repetir-falhas -lote 1
go run . cli agenda exportar -arquivo agenda.xlsx
```

Os comandos de configuração devolvem senha/token mascarados. Não versione arquivos de configuração com credenciais. `historico` e `detalhes` retornam um cursor para a página seguinte (`-cursor`).

## Preview em servidor

Em um host sem as bibliotecas nativas GTK/WebKit exigidas pelo Wails, é possível validar o frontend com o backend real:

```bash
cd frontend && npm install && npm run build && cd ..
CANDIDATE_ALLOCATOR_DATA_DIR=/tmp/agenda-preview go run . preview
UI_BASE_URL=http://127.0.0.1:8128 tools/ui-testing/runner.sh
```

O adaptador chama os mesmos métodos `App` usados pelo desktop e fica em `127.0.0.1:8128` por padrão. É um preview de desenvolvimento. Para revisão em rede privada, escolha explicitamente `preview -listen IP:PORTA` e use dados de teste. Para separar também o banco de importação, configure `CANDIDATE_ALLOCATOR_DATABASE`.

## Stack

- Go 1.25
- Wails 2.10
- SQLite
- React, TypeScript, Vite e Tailwind CSS
- Playwright para testes de UI

## Executar

Pré-requisitos: Go, Node.js/npm, Wails CLI e um compilador C compatível com `go-sqlite3`.

```bash
wails dev
```

Modo CLI:

```bash
go run . cli -file caminho/para/arquivo.xlsx
go run . cli -file caminho/para/arquivo.xlsx -opcoes 5
```

## Validar

```bash
go test ./...
make lint
```

Para a UI, mantenha `wails dev` ativo e execute:

```bash
cd tools/ui-testing
./runner.sh
```

## Documentação

- [Índice completo](docs/README.md)
- [Visão geral da arquitetura](docs/architecture/system-overview.md)
- [Fluxo de importação](docs/features/import-workflow.md)
- [Execução da alocação](docs/features/allocation-execution.md)
- [Desenvolvimento local](docs/operations/development.md)
- [Testes](docs/operations/testing.md)
- [Limitações conhecidas](docs/reference/known-limitations.md)

Para atualizar a documentação depois de mudanças no código, comece por [`docs/manifest.yaml`](docs/manifest.yaml) e siga [`docs/operations/documentation-update.md`](docs/operations/documentation-update.md).

Depois de concluir um change de código no Jujutsu, execute uma única vez:

```bash
make docs-update
```

O comando encontra todos os changes posteriores ao marcador versionado e inicia `codex exec` para revisar e corrigir somente a documentação afetada.
