# Candidate Allocation

Sistema web para alocação de candidatos em mesas de entrevista, desenvolvido em Go (servidor HTTP) + React/TypeScript (frontend embarcado).

**Branch principal: `main`. Todo novo PR deve ter `main` como destino; o merge depende da revisão do mantenedor.**

App e CLI compartilham o servidor HTTP, a mesma sessão e as regras Go. A CLI não possui solver ou banco paralelo.

**Acesso online:** [https://candidate-allocation.onrender.com/](https://candidate-allocation.onrender.com/)

---

## Como funciona

O sistema guia o usuário em 5 etapas:

1. **Upload** — envia o arquivo `.xlsx` com candidatos, avaliadores e restrições e configura parâmetros iniciais
2. **Candidatos** — mapeamento de colunas, revisão e correção dos dados (duplicatas, campos inválidos)
3. **Avaliadores** — mapeamento e confirmação da aba de avaliadores
4. **Restrições** — mapeamento e confirmação da aba de restrições
5. **Parâmetros** — mesas por horário, avaliadores por mesa e mínimo/máximo de candidatos por mesa, com uma prévia da capacidade (mesas que cabem com os avaliadores, vagas, quantos candidatos cabem pelos horários que escolheram e avisos) e **critérios adicionais** opcionais sobre curso ou semestre

Ao confirmar os parâmetros, o algoritmo de alocação é executado. A tela de resultado mostra:

- **resumo** (mesas, horários, alocados, pontuação) e **relatório de qualidade**: quantos ficaram em cada opção de horário, quantos têm avaliador "prefiro não" na mesa e quantos ficaram sem mesa — clicar num item destaca esses candidatos;
- **busca** por candidato (nome, email, curso) ou avaliador (nome, sigla) e **filtros** por horário, curso e semestre;
- as **mesas agrupadas por horário** e a lista de quem ficou sem mesa, com os horários que escolheu;
- **detalhes** de qualquer pessoa ao clicar nela: horários escolhidos (e em qual ficou), restrições e, para avaliadores, as mesas em que está.

É possível voltar e ajustar os parâmetros, exportar o resultado para `.xlsx` ou reiniciar do zero.

### Algoritmo de alocação

Cada candidato vai para uma mesa de um dos horários que escolheu. Por padrão, cada horário tem até 5 mesas (limitado por `avaliadores / avaliadores por mesa`, pois cada avaliador fica em uma mesa por horário), cada mesa tem 5 avaliadores distintos e só é formada com 5 a 8 candidatos. Esses quatro valores são editáveis na etapa de parâmetros. A pontuação (pesos em [alocate.go](alocate.go), cálculo em [pontuacao.go](pontuacao.go)) penaliza:

| Situação | Pontos |
|---|---|
| Alocado na 1ª / 2ª / 3ª / 4ª / 5ª opção | 0 / −1 / −3 / −5 / −7 (−2 a cada opção seguinte) |
| Avaliador "prefiro não" na mesa | −5 por avaliador |
| Candidato sem mesa | −1000 |
| Avaliador "não posso" na mesa | proibido |
| Critério adicional não atendido | −1 / −3 / −10 por desvio (importância baixa / média / alta) |

Os **critérios adicionais** (até 5, em [criterios.go](criterios.go)) valem para a coluna curso ou semestre e contam desvios em cada mesa:

| Critério | Desvio |
|---|---|
| Misturar | cada par de candidatos com o mesmo valor na mesa |
| Agrupar | cada par de candidatos com valores diferentes na mesa |
| No máximo N por mesa | cada candidato de um dos valores escolhidos acima de N |
| Se aparecer, pelo menos N por mesa | o que falta para N quando um dos valores escolhidos aparece |
| Pelo menos um de cada | cada valor escolhido que não aparece na mesa |

Eles pesam junto com as preferências de horário: com importância alta, a alocação pode trocar a opção de horário de alguém para cumprir a regra. O resultado mostra, para cada critério, quantas mesas não o atendem.

A busca é um **simulated annealing** sobre a divisão dos candidatos em mesas (mover um candidato de mesa ou trocar dois candidatos). Os avaliadores não são sorteados: para cada divisão, a melhor escolha de avaliadores de um horário é um problema de atribuição, resolvido de forma exata pelo **algoritmo húngaro** a cada movimento. Mesas incompletas e conflitos "não posso" recebem penalidades que começam brandas e endurecem ao longo da busca, para que ela consiga montar mesas aos poucos. Rodam 8 buscas independentes em paralelo (sementes fixas, então a mesma planilha gera o mesmo resultado) e fica a melhor.

Um limite inferior exato (fluxo de custo mínimo sobre a escolha de horários) encerra a busca antes quando a solução é comprovadamente ótima.

---

## Formatação do arquivo Excel

O arquivo `.xlsx` deve ter exatamente **3 abas**, nesta ordem. Um arquivo de exemplo pode ser baixado diretamente na tela inicial do app (botão "Baixar exemplo").

### Aba 1 — Candidatos

Uma linha por candidato. Os nomes e a ordem das colunas não precisam ser exatos — o app sugere o mapeamento pelo nome de cada coluna (ignorando maiúsculas, acentos e pontuação, e entendendo variações como "Primeira Opção" ou "E-mail institucional"), ajustável na interface. CPF e telefone sem correspondência ficam vazios. Os demais campos sem correspondência podem receber colunas restantes; confira o mapeamento antes de confirmar.

| Campo | Descrição | Validação |
|---|---|---|
| Timestamp | Data/hora do preenchimento (opcional) | — |
| Nome | Nome completo | — |
| CPF | Opcional; pode omitir a coluna | Se preenchido, 11 dígitos numéricos |
| Número | RA ou matrícula; nunca usado para enviar WhatsApp | Opcional |
| Telefone / WhatsApp | Contato separado da matrícula, opcional | DDD + número, ou formato internacional com `+` |
| Semestre | Semestre atual do candidato | Valor entre 1 e 10 |
| Curso | Nome do curso | — |
| Email Institucional | Email institucional do candidato | Deve terminar com o domínio configurado (ex: `@al.insper.edu.br`) |
| Email Pessoal | Email pessoal | Qualquer email válido |
| Opção 1 … Opção N | Opcionais quando a preferência de horários estiver desligada | Sem preferência, o administrador define os horários nos parâmetros |

Dois parâmetros são configurados antes do upload:

- **Número de opções de horário** (N): quantas colunas de disponibilidade existem na planilha. A primeira opção tem maior prioridade no algoritmo.
- **Domínio do email institucional**: sufixo que todos os emails institucionais devem ter (ex: `@al.insper.edu.br`).

Exemplo de horário: `quarta 14-16`

### Aba 2 — Avaliadores

Uma linha por avaliador.

| Campo | Descrição |
|---|---|
| Nome | Nome completo do avaliador |
| Email | Email do avaliador |
| Sigla | Identificador curto e único (ex: `ABC`) — usado nas restrições |

### Aba 3 — Restricoes

Uma linha por candidato que possui restrição. As siglas devem corresponder exatamente ao campo **Sigla** dos avaliadores. Múltiplas siglas são separadas por vírgula ou espaço.

| Campo | Descrição |
|---|---|
| Candidato | Nome do candidato (deve coincidir com o cadastrado) |
| NaoPosso | Siglas de avaliadores com quem o candidato **não pode** ser alocado (restrição absoluta) |
| PrefiroNao | Siglas de avaliadores com quem o candidato **prefere não** ser alocado (restrição suave) |

---

## Rodando localmente

### Pré-requisitos

- [Go](https://golang.org/) 1.24.2+
- [Node.js](https://nodejs.org/) 18+
- Compilador C (necessário para `go-sqlite3`) — no Windows, instale o [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)
- `make` (opcional, para os atalhos abaixo) — no Windows: `scoop install make` ou `choco install make`

### Atalhos (`make`)

O [Makefile](Makefile) junta os comandos do dia a dia; `make` sozinho lista todos.

| Comando | O que faz |
|---|---|
| `make instalar` | instala as dependências do frontend e do Go |
| `make rodar` | compila o frontend e sobe o servidor em http://localhost:8080 |
| `make build` | gera o executável `server` (`server.exe` no Windows) com o frontend embutido |
| `make testar` | testes do Go e de interface (`make testar-go` / `make testar-interface` para só um deles) |
| `make lint` | analisa o código Go com o [golangci-lint](https://golangci-lint.run/) (regras em [.golangci.yml](.golangci.yml)) |
| `make formatar` | formata o código Go |
| `make verificar` | lint + todos os testes — rode antes de abrir um PR |
| `make limpar` | apaga o executável e os relatórios de teste |

O golangci-lint não precisa ser instalado: o `make lint` roda uma versão fixa via `go run` (baixada e guardada em cache na primeira vez, o que leva ~2 min). Os arquivos `.go` usam fim de linha LF também no Windows ([.gitattributes](.gitattributes)); com CRLF, o formatador acusaria todos os arquivos.

### Desenvolvimento

```bash
git clone https://github.com/alberthamoui/candidate_allocation.git
cd candidate_allocation

# instalar dependências do frontend e gerar o build estático
cd frontend && npm install && npm run build && cd ..

# rodar o servidor Go (porta 8080)
go run .
```

Acesse em [http://localhost:8080](http://localhost:8080).

Para hot-reload do frontend durante desenvolvimento:

```bash
# terminal 1 — frontend com Vite
cd frontend && npm run dev

# terminal 2 — servidor Go
go run .
```

### Testes

```bash
# backend: algoritmo, mapeamento, API
go test ./...

# interface: abre um navegador e usa o app como uma pessoa (Playwright)
cd frontend && npm run test:e2e
```

Os testes de interface compilam o frontend, sobem o servidor Go na porta 8099 (sem conflitar com um servidor aberto na 8080) e percorrem as telas com as planilhas de `Excels/`: fluxo completo até a exportação, sugestão e troca de colunas no mapeamento, parâmetros e prévia de capacidade, erros de upload e o selo de versão. Na primeira vez, instale o navegador com `npx playwright install chromium`. Para ver o que falhou, `npx playwright show-report` abre um relatório com screenshot e passo a passo de cada teste.

### Docker

```bash
docker build -t candidate-allocation .
docker run -p 8080:8080 candidate-allocation
```

### Produção (Render)

O app está publicado no Render, com deploy da branch `main` pelo `Dockerfile` — um merge `main ← dev` publica a versão. O servidor lê quanta CPU o container tem (cota do cgroup) e se ajusta: no plano gratuito (0,1 CPU) cada alocação roda **uma** busca e só **uma** alocação roda por vez; as outras esperam na fila ("Aguardando outra alocação terminar..."). Se o usuário fecha a página, a alocação dele para. As agendas agora são persistidas no diretório `ALOCACAO_DADOS`; configure um volume persistente no Render para recuperá-las após reinícios ou recriação do container. A expiração em memória não apaga os dados do disco.

Variáveis de ambiente opcionais (no painel do Render):

| Variável | Padrão | O que faz |
|---|---|---|
| `ALOCACAO_EXECUCOES` | uma por CPU, até 8 | buscas em paralelo por alocação |
| `ALOCACAO_ITERACOES` | 200000 | movimentos por busca (menos = mais rápido, pode piorar um pouco o resultado) |
| `ALOCACOES_SIMULTANEAS` | CPUs ÷ buscas (mínimo 1) | alocações ao mesmo tempo |
| `MAX_SESSOES` | 200 | sessões carregadas na memória ao mesmo tempo |
| `MAX_UPLOAD_MB` | 10 | tamanho máximo da planilha |

Referência medida com 1 núcleo: a planilha oficial (98 candidatos) leva ~0,5 s de CPU por busca — cerca de 6 s com 0,1 CPU — e chega ao mesmo resultado com 1 busca ou com 8. Com 300 candidatos, uma busca leva ~35 s com 0,1 CPU. O selo no canto da tela mostra branch e commit (no Render, via `RENDER_GIT_BRANCH`/`RENDER_GIT_COMMIT`).

---

## Estrutura do projeto

```
candidate_allocation/
├── main.go         -- entrypoint: servidor HTTP, roteamento
├── handlers.go     -- handlers das rotas HTTP e router
├── app.go          -- SessionStore e lógica de sessão
├── alocate.go      -- algoritmo de alocação (parâmetros e simulated annealing)
├── carregar.go     -- leitura do banco da sessão para a alocação
├── hungaro.go      -- algoritmo húngaro (escolha dos avaliadores de cada horário)
├── pontuacao.go    -- pontuação da alocação e impressão no log
├── capacidade.go   -- prévia de capacidade para os parâmetros da alocação
├── criterios.go    -- critérios adicionais (curso/semestre) usados na alocação
├── recursos.go     -- limites do servidor conforme a CPU disponível (produção)
├── versao.go       -- branch e commit exibidos no canto da tela
├── processa.go     -- parsing do arquivo Excel
├── mapping.go      -- lógica de mapeamento de colunas
├── sugestao_mapping.go -- sugestão de mapeamento pelo nome das colunas
├── export.go       -- execução da alocação e geração do Excel de resultado
├── resultado.go    -- resultado para a tela: dados das pessoas e relatório de qualidade
├── models.go       -- structs de dados
├── setup.go        -- inicialização do banco SQLite
├── db/             -- funções auxiliares de banco
├── Dockerfile      -- build multi-stage (Node → Go → Alpine)
├── Makefile        -- atalhos: rodar, testar, lint...
├── .golangci.yml   -- regras do golangci-lint
├── Excels/         -- planilha de exemplo (download) e planilha oficial (testes)
└── frontend/       -- app React/TypeScript (Vite + Tailwind)
    ├── e2e/                -- testes de interface (Playwright)
    ├── playwright.config.ts
    └── src/
        ├── main.tsx            -- roteamento e estado global
        ├── api.ts              -- cliente HTTP/SSE e gestão do sessionId
        ├── fluxo.ts            -- etapas do fluxo e textos de cada tela (título e ajuda "?")
        ├── nomesCampos.ts      -- nomes legíveis dos campos no mapeamento e na revisão
        ├── index.css           -- variáveis de cor e classes base do visual
        ├── components/
        │   ├── Layout.tsx          -- moldura das telas: topo com etapa e ajuda, etapas no rodapé
        │   ├── ui.tsx              -- peças visuais compartilhadas (botões, cartões, barra de ações...)
        │   ├── EtapaAba.tsx        -- lê uma aba da planilha (avaliadores ou restrições) e segue para o mapeamento
        │   ├── BranchBadge.tsx     -- selo com a branch e o commit no topo da tela
        │   ├── PainelResultado.tsx -- conteúdo da tela de resultado
        │   ├── EditorCriterios.tsx -- critérios adicionais nos parâmetros
        │   ├── UserCard.tsx        -- cartão de candidato na revisão
        │   └── EditableCell.tsx    -- valor editável com um clique
        └── pages/
            ├── Home.tsx            -- tela inicial e upload
            ├── MappingPage.tsx     -- mapeamento de colunas (reutilizado nas 3 etapas)
            ├── VerifyUsers.tsx     -- revisão de candidatos
            ├── UploadAvaliador.tsx -- etapa de avaliadores (usa EtapaAba)
            ├── UploadRestricao.tsx -- etapa de restrições (usa EtapaAba)
            ├── Parametros.tsx      -- parâmetros da alocação e prévia de capacidade
            └── Resultado.tsx       -- resultado da alocação e exportação
```

## API

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/api/upload` | Recebe o `.xlsx` e cria uma sessão |
| `POST` | `/api/build-usuarios` | Aplica mapeamento de colunas e retorna candidatos parseados |
| `POST` | `/api/save-usuarios` | Salva candidatos revisados na sessão |
| `POST` | `/api/suggest-avaliador` | Sugere mapeamento para a aba de avaliadores |
| `POST` | `/api/build-avaliadores` | Aplica mapeamento e retorna avaliadores parseados |
| `POST` | `/api/save-avaliadores` | Salva avaliadores na sessão |
| `POST` | `/api/suggest-restricao` | Sugere mapeamento para a aba de restrições |
| `POST` | `/api/build-restricoes` | Aplica mapeamento e retorna restrições parseadas |
| `POST` | `/api/save-restricoes` | Salva restrições na sessão |
| `GET` | `/api/capacidade` | Prévia de capacidade para os parâmetros (query opcional: `mesas_por_horario`, `avaliadores_por_mesa`, `min_pessoas_por_mesa`, `max_pessoas_por_mesa`; sem eles, usa os padrões) |
| `GET` | `/api/alocar?sessionId=` | Executa alocação via Server-Sent Events (streaming de progresso); aceita os mesmos parâmetros de `/api/capacidade` |
| `GET` | `/api/export?sessionId=` | Download do resultado em `.xlsx` |
| `GET` | `/api/exemplo` | Download do arquivo de exemplo |
| `GET` | `/api/versao` | Branch e commit que o servidor está rodando (exibidos no canto da tela) |
| `DELETE` | `/api/session` | Encerra e limpa a sessão atual (aguarda operações/envios ativos) |
| `GET` | `/api/resultado` | Recupera a agenda atual |
| `GET/POST` | `/api/avisos` | Consulta configuração sem segredos / configura canais |
| `POST` | `/api/avisos/testar`, `/api/avisos/previa` | Testa conexão / mostra prévia sem enviar |
| `POST` | `/api/avisos/enviar` | Autoriza aviso para todos, mesa ou pessoa |
| `GET` | `/api/avisos/historico` | Resumos paginados (`antes`, `limite`) |
| `GET` | `/api/avisos/lotes/{id}` | Detalhes do lote (`resumo=true` evita carregar mensagens) |
| `POST` | `/api/avisos/repetir` | Repete somente falhas confirmadas |
| `POST` | `/api/realocar` | Realoca pessoa, mesa ou dia |
| `GET` | `/api/realocar/sugestoes` | Ordena destinos para uma pessoa |
| `POST` | `/api/avaliar` | Avalia um ou vários candidatos atomicamente |

Todas as rotas de sessão recebem o `sessionId` pelo header `X-Session-Id`.


## Operar a agenda no app e na CLI

Na entrada, escolha **Configurar avisos** ou **Agora não**. CPF e canais são opcionais. Desmarque **Usar preferências de horário dos candidatos** para importar uma planilha sem opções e informar os horários disponíveis nos parâmetros. Todos os horários terão o mesmo custo no algoritmo nesse modo.

No resultado, cada pessoa tem ações **Realocar**, **Passou / Não passou / pendente** e **Avisar**. Cada mesa tem **Realocar mesa** e **Avisar mesa**. No cabeçalho do horário, **Realocar horário ou dia** permite mudar só aquele intervalo ou todas as mesas do dia: informe apenas a data na origem e no destino para preservar os intervalos.

A realocação manual permite qualquer destino, inclusive uma mesa nova ou acima da capacidade. O servidor ordena sugestões por vagas, preferências, equipe, restrições e critérios, e exibe alertas depois da alteração. O administrador escolhe se autoriza avisos aos participantes afetados. Avaliadores recebem e-mail quando esse canal está ativo. O Excel sempre usa a agenda e as avaliações atuais.

Compile com `make build`. Para iniciar o servidor e compartilhar a mesma agenda entre navegador e terminal:

```bash
ALOCACAO_DADOS=/caminho/privado/agendas PORT=8080 ./server servir
./server ajuda
./server alocar -planilha Excels/teste_oficial.xlsx -servidor http://localhost:8080
```

O último comando devolve o link da agenda e o identificador. Copie-o para os próximos comandos (também aparece em **Usar esta agenda pela CLI** no app):

```bash
./server resultado -sessao ID
./server realocar-pessoa -sessao ID -pessoa 12 -mesa 3
./server sugestoes -sessao ID -pessoa 12
./server realocar-mesa -sessao ID -mesa 3 -para '29/10/2026 14:00-16:00' -avisar
./server realocar-dia -sessao ID -de '29/10/2026' -para '30/10/2026' -avisar
./server avaliar -sessao ID -id 12 -resultado passou
./server avaliar -sessao ID -id 13 -resultado 'não passou'
./server avaliar -sessao ID -id 12 -resultado pendente
./server exportar -sessao ID -saida agenda.xlsx
./server avisos -sessao ID
./server avisos -sessao ID -config /caminho/privado/canais.json
./server avisos -sessao ID -testar
./server avisos -sessao ID -previa
./server enviar -sessao ID -pessoa 12
./server enviar -sessao ID -mesa 3
./server enviar -sessao ID
./server historico -sessao ID -limite 20
./server envio -sessao ID -lote LOTE
./server repetir-falhas -sessao ID -lote LOTE
```

Todos aceitam `-servidor URL` (padrão `http://localhost:8080`). `historico -antes CURSOR` consulta a página seguinte. `enviar` é autorização explícita; realocações só notificam com `-avisar`. A CLI retorna erro se algum envio falhar ou ficar incerto e imprime separadamente a operação já salva.

Para importar sem CPF e sem preferências:

```bash
./server alocar -planilha candidatos.xlsx -sem-preferencia \
  -horarios '28/10/2026 10:00-12:00,28/10/2026 14:00-16:00'
```

`alocar -avisos /caminho/privado/canais.json` permite configurar os canais na importação. O campo `enviar_ao_alocar: true` autoriza o aviso automático depois de concluir a alocação. Sem essa autorização, gerar uma agenda não envia mensagens. Não coloque senhas em argumentos do terminal; proteja o arquivo de configuração com `chmod 600`. A configuração retornada pela API/CLI oculta senha e token; valores vazios preservam o segredo somente na mesma conta.

## Conectar os canais

- **Gmail / Google Workspace:** informe o remetente e uma [senha de aplicativo do Google](https://support.google.com/accounts/answer/185833). O app preenche host, porta e usuário. A conta precisa permitir senhas de aplicativo; a senha normal da conta não serve. Para outros serviços, abra a configuração SMTP avançada. O envio exige TLS.
- **WhatsApp Business (Meta):** informe o ID do número, token e nome do modelo aprovado; configure idioma e a ordem das variáveis do corpo do modelo. O WhatsApp usa o texto aprovado pela Meta, conforme a [documentação oficial de modelos](https://whatsapp.github.io/WhatsApp-Nodejs-SDK/api-reference/messages/template/). O texto livre do editor é usado no e-mail e no webhook avançado.
- **Webhook avançado:** recebe POST JSON com `to`, `message`, `para` e `mensagem`; o token vai em `Authorization: Bearer`. Use HTTPS. Um HTTP 2xx significa aceitação pelo integrador, que é responsável pelo encaminhamento.

No editor, clique em **nome**, **data**, **horario**, **mesa**, **email**, **curso** ou **numero** para inserir variáveis. Veja a prévia antes de enviar. **Testar conexão** verifica autenticação SMTP ou acesso ao número da Meta, sem enviar uma mensagem. Credenciais inválidas e falta de telefone ficam visíveis.

## Persistência e acompanhamento dos envios

`ALOCACAO_DADOS` é o diretório privado de dados (padrão `alocacao-dados`). Cada sessão usa SQLite com arquivo `0600`, incluindo agenda, avaliações, configuração e os envios. Use um volume persistente em servidor/container e faça backup desse diretório. Execute uma única instância por diretório. Sem volume persistente, a hospedagem pode apagar os dados ao recriar o container.

Sessões inativas são descarregadas da memória após 15 minutos e recuperadas do disco pelo ID. O limite `MAX_SESSOES` se aplica às sessões carregadas; sessões com operações ou envios ativos não são descarregadas. O ID funciona como chave de acesso: mantenha-o privado. Para uma instalação exposta à internet, coloque autenticação administrativa e HTTPS no proxy antes de conectar contas reais.

A agenda e a autorização de envio são gravadas na mesma transação. Cada mensagem/canal tem estado próprio: pendente, em transmissão, aceito pelo provedor, falhou ou aceitação incerta. O histórico é paginado e carrega detalhes sob demanda; durante o envio, a interface consulta apenas o resumo daquele lote. Aceitação não confirma entrega ou leitura. Em caso de interrupção durante a transmissão, confira o provedor antes de preparar outro aviso; mensagens incertas não são repetidas automaticamente. **Repetir falhas** cria uma única nova tentativa das falhas confirmadas, com contatos e horários atuais.

As conexões precisam das contas do administrador. Os testes automatizados usam provedores simulados, sem disparos externos; validar autenticação e entrega com contas e destinatários reais permanece uma etapa de instalação.
