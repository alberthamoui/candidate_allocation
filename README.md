# Candidate Allocation

Sistema web para alocação de candidatos em mesas de entrevista, desenvolvido em Go (servidor HTTP) + React/TypeScript (frontend embarcado).

**Acesso online:** [https://candidate-allocation.onrender.com/](https://candidate-allocation.onrender.com/)

---

## Como funciona

O sistema guia o usuário em 5 etapas:

1. **Upload** — envia o arquivo `.xlsx` com candidatos, avaliadores e restrições. A preferência de horário e os avisos (e-mail e WhatsApp) são opcionais e se configuram nesta tela
2. **Candidatos** — mapeamento de colunas, revisão e correção dos dados (duplicatas, campos inválidos)
3. **Avaliadores** — mapeamento e confirmação da aba de avaliadores
4. **Restrições** — mapeamento e confirmação da aba de restrições
5. **Parâmetros** — mesas por horário, avaliadores por mesa e mínimo/máximo de candidatos por mesa, com uma prévia da capacidade (mesas que cabem com os avaliadores, vagas, quantos candidatos cabem pelos horários que escolheram e avisos) e **critérios adicionais** opcionais sobre curso ou semestre. Se a preferência de horário foi desligada no upload, esta etapa pede a lista de horários: todo mundo pode ir em qualquer um deles

Ao confirmar os parâmetros, o algoritmo de alocação é executado. A tela de resultado mostra:

- **resumo** (mesas, horários, alocados, pontuação) e **relatório de qualidade**: quantos ficaram em cada opção de horário, quantos têm avaliador "prefiro não" na mesa e quantos ficaram sem mesa — clicar num item destaca esses candidatos;
- **busca** por candidato (nome, email, curso) ou avaliador (nome, sigla) e **filtros** por horário, curso e semestre;
- as **mesas agrupadas por horário** e a lista de quem ficou sem mesa, com os horários que escolheu;
- **detalhes** de qualquer pessoa ao clicar nela: horários escolhidos (e em qual ficou), restrições e, para avaliadores, as mesas em que está.

É possível voltar e ajustar os parâmetros, exportar o resultado para `.xlsx` ou reiniciar do zero.

Na mesma tela, depois da alocação:

- **Imprevisto** — move um intervalo, um dia inteiro preservando os intervalos, ou uma mesa. Com confirmação, avisa candidatos e avaliadores afetados.
- **Uma pessoa** — mostra os melhores destinos, ordenados por restrições, avaliadores, vagas, preferência e critérios. A escolha manual é livre, inclusive mesa cheia ou horário novo: exceções aparecem como alertas.
- **Avaliação** — marca quem passou e quem não passou. A marca aparece na lista e no Excel (`passou` / `não passou`).
- **Avisos** — edita a mensagem (com `{{nome}}`, `{{data}}`, `{{horario}}`, `{{mesa}}`, `{{email}}`, `{{curso}}`, `{{numero}}`) e dispara o envio para todo mundo alocado.
- **Exportar Excel** — baixa a agenda atual com horário, e-mail, matrícula, telefone e avaliação. As duas abas acompanham as mudanças.

### Algoritmo de alocação

Cada candidato vai para uma mesa de um dos horários que escolheu. Com a preferência desligada, o custo de horário é zero e qualquer horário informado pelo administrador serve. Por padrão, cada horário tem até 5 mesas (limitado por `avaliadores / avaliadores por mesa`, pois cada avaliador fica em uma mesa por horário), cada mesa tem 5 avaliadores distintos e só é formada com 5 a 8 candidatos. Esses quatro valores são editáveis na etapa de parâmetros. A pontuação (em [alocate.go](alocate.go)) penaliza:

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

O arquivo `.xlsx` deve ter exatamente **3 abas**, nesta ordem. Um arquivo de exemplo pode ser baixado diretamente na tela inicial do app (botão "Baixar planilha de exemplo").

### Aba 1 — Candidatos

Uma linha por candidato. Os nomes e a ordem das colunas não precisam ser exatos — o app sugere o mapeamento pelo nome de cada coluna (ignorando maiúsculas, acentos e pontuação, e entendendo variações como "Primeira Opção" ou "E-mail institucional"), ajustável na interface. Campos sem correspondência pelo nome recebem as colunas que sobraram, na ordem da planilha.

| Campo | Descrição | Validação |
|---|---|---|
| Timestamp | Data/hora do preenchimento (opcional) | — |
| Nome | Nome completo | — |
| CPF | Apenas dígitos, sem pontuação. Opcional | Se vier preenchido, 11 dígitos numéricos |
| Número | RA ou número de matrícula | 9 dígitos (telefones legados também são aceitos) |
| Telefone | Telefone/WhatsApp separado da matrícula, opcional | DDD para brasileiros; `+` e código do país para internacionais. RA de 9 dígitos não é enviado ao WhatsApp |
| Semestre | Semestre atual do candidato | Valor entre 1 e 10 |
| Curso | Nome do curso | — |
| Email Institucional | Email institucional do candidato | Deve terminar com o domínio configurado (ex: `@al.insper.edu.br`) |
| Email Pessoal | Email pessoal | Qualquer email válido |
| Opção 1 … Opção N | Horários disponíveis em ordem de preferência | Uma coluna por opção |

Três escolhas são feitas antes do upload:

- **Preferência de horário** (ligada por padrão): se a planilha tiver colunas de opção, informe quantas são (N). A primeira opção tem maior prioridade no algoritmo. Se desligar, a planilha não precisa dessas colunas; os horários são digitados na etapa de parâmetros e todos os candidatos podem ir em qualquer um, com o mesmo custo.
- **Domínio do email institucional**: sufixo que todos os emails institucionais devem ter (ex: `@al.insper.edu.br`).
- **Avisos** (opcional): uma mensagem com variáveis e, se quiser, o SMTP e a URL de um webhook de WhatsApp. Nada é enviado nesta etapa. Você pode autorizar o envio ao concluir a alocação, ou enviar depois no resultado/CLI (`-avisar` / `enviar`).

A configuração de e-mail oferece Gmail/Google Workspace (endereço e senha de aplicativo), com o SMTP preenchido automaticamente, ou SMTP avançado. O mesmo formulário completo existe na entrada e no resultado. **Testar conexão** verifica autenticação sem enviar; **Ver prévia** mostra a mensagem preenchida.

O WhatsApp oferece **WhatsApp Business da Meta**: ID do número, token, modelo aprovado e variáveis na ordem desse modelo. Há um guia na tela com links para a conta. A Meta exige um modelo aprovado para esses avisos proativos; o texto livre do editor vale para e-mail e webhook. Um webhook de outro provedor continua como opção avançada. Telefones brasileiros são normalizados com código 55; matrículas nunca são tratadas como telefones.

A checkbox **Autorizar envio automático ao concluir a alocação** é um consentimento separado, desligado por padrão. Realocações mantêm sua própria confirmação. Os envios rodam em segundo plano, com histórico, totais de aceitação/falha e repetição somente das falhas. Mensagens aceitas pelo provedor não são repetidas. Aceitação não significa confirmação de entrega. Após um reinício que interrompa um envio, as pendências ficam visíveis para repetição explícita.

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

- [Go](https://golang.org/) 1.21+
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

Os mesmos passos existem na linha de comando. Sem argumentos, o programa sobe o site. Com um comando, ele trabalha numa pasta local (`alocacao-cli/` por padrão, arquivo `estado.json`):

```bash
go run . alocar -planilha Excels/base_exemplo.xlsx
go run . alocar -planilha planilha.xlsx -sem-preferencia -horarios "quarta 14-16,quinta 10-12"
go run . avisos -mensagem "Olá {{nome}}, sua entrevista é em {{data}}, no horário {{horario}}." -email -smtp-host smtp.exemplo.com -remetente org@exemplo.com
go run . realocar-dia -de "quarta 14-16" -para "sexta 14-16" -avisar
go run . realocar-mesa -mesa 1 -para "quinta 10-12"
go run . realocar-pessoa -pessoa "Ana" -horario "sexta 8-10" -avisar
go run . avaliar -pessoa "Ana" -resultado aprovado
go run . avaliar -listar
go run . enviar
go run . exportar -saida resultado.xlsx
go run . ajuda
```

`realocar-dia`, `realocar-mesa` e `realocar-pessoa` mudam o horário sem rodar o algoritmo de novo. `-avisar` e `enviar` disparam e-mail e WhatsApp. `avaliar` aceita `aprovado`/`passou` e `reprovado`/`não passou`. O Excel do `exportar` traz horário, e-mail, número e avaliação. O `estado.json` pode conter a senha do SMTP; a pasta `alocacao-cli/` não entra no git.

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
docker run -p 8080:8080 -v alocacao-dados:/app/alocacao-dados candidate-allocation
```

### Produção (Render)

O app está publicado no Render, com deploy da branch `main` pelo `Dockerfile` — um merge `main ← dev` publica a versão. O servidor lê quanta CPU o container tem (cota do cgroup) e se ajusta: no plano gratuito (0,1 CPU) cada alocação roda **uma** busca e só **uma** alocação roda por vez; as outras esperam na fila ("Aguardando outra alocação terminar..."). Se o usuário fecha a página, a alocação dele para. O servidor agora grava as sessões em SQLite no diretório `ALOCACAO_DADOS` (padrão `alocacao-dados/`). Para sobreviver a reinícios/redeploys do Render, esse diretório precisa estar em um disco persistente. O [plano gratuito do Render não suporta discos persistentes](https://render.com/docs/free); nele os arquivos continuam sujeitos à perda pela infraestrutura. Para usar o fluxo completo de entrevistas entre dias, configure um [disco persistente](https://render.com/docs/disks) e aponte `ALOCACAO_DADOS` para o caminho montado. Nenhum deploy ou mudança de plano é realizado pelo código.

Variáveis de ambiente opcionais (no painel do Render):

| Variável | Padrão | O que faz |
|---|---|---|
| `ALOCACAO_DADOS` | `alocacao-dados` | diretório dos bancos de sessão; montar em disco persistente em produção |
| `ALOCACAO_EXECUCOES` | uma por CPU, até 8 | buscas em paralelo por alocação |
| `ALOCACAO_ITERACOES` | 200000 | movimentos por busca (menos = mais rápido, pode piorar um pouco o resultado) |
| `ALOCACOES_SIMULTANEAS` | CPUs ÷ buscas (mínimo 1) | alocações ao mesmo tempo |
| `MAX_SESSOES` | 200 | sessões abertas ao mesmo tempo |
| `MAX_UPLOAD_MB` | 10 | tamanho máximo da planilha |

Referência medida com 1 núcleo: a planilha oficial (98 candidatos) leva ~0,5 s de CPU por busca — cerca de 6 s com 0,1 CPU — e chega ao mesmo resultado com 1 busca ou com 8. Com 300 candidatos, uma busca leva ~35 s com 0,1 CPU. O selo no canto da tela mostra branch e commit (no Render, via `RENDER_GIT_BRANCH`/`RENDER_GIT_COMMIT`).

---

## Estrutura do projeto

```
candidate_allocation/
├── main.go         -- entrypoint: servidor HTTP ou comandos da CLI
├── cli.go          -- alocar, avisar, realocar, avaliar e exportar pela linha de comando
├── handlers.go     -- handlers das rotas HTTP e router
├── app.go          -- SessionStore e lógica de sessão
├── alocate.go      -- algoritmo de alocação
├── preferencia.go  -- horários definidos pelo administrador quando a preferência está desligada
├── avisos.go       -- mensagem com variáveis, SMTP e webhook de WhatsApp
├── realocar.go     -- mudar dia, mesa ou pessoa e registrar a avaliação
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
├── Excels/         -- arquivo de exemplo para download
└── frontend/       -- app React/TypeScript (Vite + Tailwind)
    ├── e2e/                -- testes de interface (Playwright)
    ├── playwright.config.ts
    └── src/
        ├── main.tsx            -- roteamento e estado global
        ├── api.ts              -- cliente HTTP/SSE e gestão do sessionId
        ├── components/         -- UserCard, EditableCell, PainelResultado, OperacoesResultado (imprevisto, pessoa, avaliação, avisos), EditorCriterios
        └── pages/
            ├── Home.tsx            -- tela inicial e upload
            ├── MappingPage.tsx     -- mapeamento de colunas (reutilizado nas 3 etapas)
            ├── VerifyUsers.tsx     -- revisão de candidatos
            ├── UploadAvaliador.tsx
            ├── UploadRestricao.tsx
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
| `GET` | `/api/alocar?sessionId=` | Executa alocação via Server-Sent Events (streaming de progresso); aceita os mesmos parâmetros de `/api/capacidade`, mais `sem_preferencia` e `horarios_livres` |
| `GET` | `/api/resultado` | Última alocação, já com realocações e avaliações |
| `GET` | `/api/avisos` | Mensagem e canais configurados na sessão |
| `POST` | `/api/avisos` | Grava a mensagem e os canais (SMTP e WhatsApp) |
| `POST` | `/api/avisos/enviar` | Envia a mensagem para todo mundo alocado |
| `POST` | `/api/realocar` | Move um dia, uma mesa ou uma pessoa (`tipo`: `dia`, `mesa` ou `pessoa`) |
| `POST` | `/api/avaliar` | Marca quem passou e quem não passou |
| `GET` | `/api/export?sessionId=` | Download do resultado em `.xlsx` (inclui horário, e-mail, número e avaliação) |
| `GET` | `/api/exemplo` | Download do arquivo de exemplo |
| `GET` | `/api/versao` | Branch e commit que o servidor está rodando (exibidos no canto da tela) |
| `DELETE` | `/api/session` | Encerra e limpa a sessão atual |

Todas as rotas de sessão recebem o `sessionId` pelo header `X-Session-Id`.

## Agenda persistida e CLI conectada ao site

O servidor grava candidatos, parâmetros, configuração de canais, agenda atual, avaliações e histórico em um arquivo SQLite por alocação. Os arquivos novos são criados com permissão `0600`, e o diretório com `0700`. O acesso usa o identificador da sessão: trate-o como uma chave privada. O navegador guarda esse identificador e oferece **Retomar última alocação**. **Reiniciar** remove explicitamente os dados da sessão.

As operações usam sincronização por sessão e devolvem snapshots. Envios de rede usam uma cópia da configuração e da agenda; não bloqueiam realocações, leitura ou avaliação. A agenda atual é a fonte para a tela e o Excel; horários são normalizados após cada mudança manual, e qualidade/pontuação são recalculadas. Exceções manuais aparecem na lista de alertas.

Para a CLI trabalhar na **mesma alocação** do site, copie o comando mostrado na aba Avisos:

```bash
./server mesas -servidor http://localhost:8080 -sessao ID
./server sugestoes -servidor http://localhost:8080 -sessao ID -pessoa 12
./server realocar-pessoa -servidor http://localhost:8080 -sessao ID -pessoa 12 -mesa 500
./server realocar-dia -servidor http://localhost:8080 -sessao ID -de quarta -para sexta -avisar
./server avaliar -servidor http://localhost:8080 -sessao ID -id 12 -resultado passou
./server avisos -servidor http://localhost:8080 -sessao ID -config canais.json
./server avisos -servidor http://localhost:8080 -sessao ID -testar
./server avisos -servidor http://localhost:8080 -sessao ID -previa
./server historico -servidor http://localhost:8080 -sessao ID
./server repetir-falhas -servidor http://localhost:8080 -sessao ID -lote ID_DO_ENVIO
./server exportar -servidor http://localhost:8080 -sessao ID -saida resultado.xlsx
```

`mesas` e `resultado` exibem IDs reais, horários, pessoas e alertas em JSON. Para criar uma alocação pelo terminal e abri-la no site:

```bash
./server alocar -servidor http://localhost:8080 -planilha Excels/teste_oficial.xlsx
```

O comando devolve o link e o identificador. Sem `-servidor`, continua disponível o modo local com `-workspace`. Nesse modo, `avisos -config canais.json` aceita a configuração completa, incluindo o provedor Meta. A escrita do estado local é atômica. `enviar`, operações com `-avisar` e repetição de falhas retornam código não zero quando algum aviso falha; a realocação já aplicada permanece registrada, e a saída distingue sua conclusão das falhas de comunicação.

Um `canais.json` sem segredos reais pode partir de:

```json
{
  "ativo": true,
  "enviar_ao_alocar": false,
  "mensagem": "Olá {{nome}}, entrevista em {{data}} às {{horario}}, {{mesa}}.",
  "email": {"ativo": true, "provedor": "gmail", "host": "smtp.gmail.com", "porta": 587, "usuario": "org@exemplo.com", "remetente": "org@exemplo.com", "senha": ""},
  "whatsapp": {"ativo": true, "provedor": "meta", "phone_id": "", "token": "", "template": "aviso_entrevista", "idioma": "pt_BR", "variaveis_template": ["nome", "data", "horario", "mesa"]}
}
```

`GET /api/avisos` mascara senha/token e informa se há segredo salvo. Ao editar a mensagem, valores vazios preservam credenciais da mesma conta/provedor. Não versione configurações preenchidas com credenciais.

Rotas adicionais: `GET /api/realocar/sugestoes?pessoa=ID`, `GET /api/avisos/historico`, `POST /api/avisos/repetir` (`{"id":"ID_DO_ENVIO"}`), `POST /api/avisos/testar` e `POST /api/avisos/previa` (configuração dos canais). Realocações/envios devolvem `lote_id`; acompanhe o histórico para saber quando terminam.

Referências dos provedores: [modelo WhatsApp da Meta](https://whatsapp.github.io/WhatsApp-Nodejs-SDK/api-reference/messages/template/), [senhas de aplicativo Google](https://support.google.com/accounts/answer/185833). A conta Meta deve ter número e modelo aprovados; use um token permanente autorizado. A configuração não vincula automaticamente uma conta pessoal de WhatsApp.
