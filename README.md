# Candidate Allocation

Sistema web para alocação de candidatos em mesas de entrevista, desenvolvido em Go (servidor HTTP) + React/TypeScript (frontend embarcado).

**Acesso online:** [https://candidate-allocation.onrender.com/](https://candidate-allocation.onrender.com/)

---

## Como funciona

O sistema guia o usuário em 5 etapas:

1. **Upload** — envia o arquivo `.xlsx` com candidatos, avaliadores e restrições e configura parâmetros iniciais
2. **Candidatos** — mapeamento de colunas, revisão e correção dos dados (duplicatas, campos inválidos)
3. **Avaliadores** — mapeamento e confirmação da aba de avaliadores
4. **Restrições** — mapeamento e confirmação da aba de restrições
5. **Parâmetros** — mesas por horário, avaliadores por mesa e mínimo/máximo de candidatos por mesa, com uma prévia da capacidade (mesas que cabem com os avaliadores, vagas, quantos candidatos cabem pelos horários que escolheram e avisos)

Ao confirmar os parâmetros, o algoritmo de alocação é executado. O resultado mostra as mesas formadas e os candidatos não alocados. É possível voltar e ajustar os parâmetros, exportar o resultado para `.xlsx` ou reiniciar do zero.

### Algoritmo de alocação

Cada candidato vai para uma mesa de um dos horários que escolheu. Por padrão, cada horário tem até 5 mesas (limitado por `avaliadores / avaliadores por mesa`, pois cada avaliador fica em uma mesa por horário), cada mesa tem 5 avaliadores distintos e só é formada com 5 a 8 candidatos. Esses quatro valores são editáveis na etapa de parâmetros. A pontuação (em [alocate.go](alocate.go)) penaliza:

| Situação | Pontos |
|---|---|
| Alocado na 1ª / 2ª / 3ª / 4ª / 5ª opção | 0 / −1 / −3 / −5 / −7 (−2 a cada opção seguinte) |
| Avaliador "prefiro não" na mesa | −5 por avaliador |
| Candidato sem mesa | −1000 |
| Avaliador "não posso" na mesa | proibido |

A busca é um **simulated annealing** sobre a divisão dos candidatos em mesas (mover um candidato de mesa ou trocar dois candidatos). Os avaliadores não são sorteados: para cada divisão, a melhor escolha de avaliadores de um horário é um problema de atribuição, resolvido de forma exata pelo **algoritmo húngaro** a cada movimento. Mesas incompletas e conflitos "não posso" recebem penalidades que começam brandas e endurecem ao longo da busca, para que ela consiga montar mesas aos poucos. Rodam 8 buscas independentes em paralelo (sementes fixas, então a mesma planilha gera o mesmo resultado) e fica a melhor.

Um limite inferior exato (fluxo de custo mínimo sobre a escolha de horários) encerra a busca antes quando a solução é comprovadamente ótima.

---

## Formatação do arquivo Excel

O arquivo `.xlsx` deve ter exatamente **3 abas**, nesta ordem. Um arquivo de exemplo pode ser baixado diretamente na tela inicial do app (botão "Baixar exemplo").

### Aba 1 — Candidatos

Uma linha por candidato. Os nomes e a ordem das colunas não precisam ser exatos — o app sugere o mapeamento pelo nome de cada coluna (ignorando maiúsculas, acentos e pontuação, e entendendo variações como "Primeira Opção" ou "E-mail institucional"), ajustável na interface. Campos sem correspondência pelo nome recebem as colunas que sobraram, na ordem da planilha.

| Campo | Descrição | Validação |
|---|---|---|
| Timestamp | Data/hora do preenchimento (opcional) | — |
| Nome | Nome completo | — |
| CPF | Apenas dígitos, sem pontuação | 11 dígitos numéricos |
| Número | RA ou número de matrícula | 9 dígitos numéricos |
| Semestre | Semestre atual do candidato | Valor entre 1 e 10 |
| Curso | Nome do curso | — |
| Email Institucional | Email institucional do candidato | Deve terminar com o domínio configurado (ex: `@al.insper.edu.br`) |
| Email Pessoal | Email pessoal | Qualquer email válido |
| Opção 1 … Opção N | Horários disponíveis em ordem de preferência | Uma coluna por opção |

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

- [Go](https://golang.org/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- Compilador C (necessário para `go-sqlite3`) — no Windows, instale o [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)

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

### Docker

```bash
docker build -t candidate-allocation .
docker run -p 8080:8080 candidate-allocation
```

---

## Estrutura do projeto

```
candidate_allocation/
├── main.go         -- entrypoint: servidor HTTP, roteamento
├── handlers.go     -- handlers das rotas HTTP e router
├── app.go          -- SessionStore e lógica de sessão
├── alocate.go      -- algoritmo de alocação
├── capacidade.go   -- prévia de capacidade para os parâmetros da alocação
├── versao.go       -- branch e commit exibidos no canto da tela
├── processa.go     -- parsing do arquivo Excel
├── mapping.go      -- lógica de mapeamento de colunas
├── sugestao_mapping.go -- sugestão de mapeamento pelo nome das colunas
├── export.go       -- geração do Excel de resultado
├── models.go       -- structs de dados
├── setup.go        -- inicialização do banco SQLite
├── db/             -- funções auxiliares de banco
├── Dockerfile      -- build multi-stage (Node → Go → Alpine)
├── Excels/         -- arquivo de exemplo para download
└── frontend/       -- app React/TypeScript (Vite + Tailwind)
    └── src/
        ├── main.tsx            -- roteamento e estado global
        ├── api.ts              -- cliente HTTP/SSE e gestão do sessionId
        ├── components/         -- UserCard, EditableCell
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
| `GET` | `/api/alocar?sessionId=` | Executa alocação via Server-Sent Events (streaming de progresso); aceita os mesmos parâmetros de `/api/capacidade` |
| `GET` | `/api/export?sessionId=` | Download do resultado em `.xlsx` |
| `GET` | `/api/exemplo` | Download do arquivo de exemplo |
| `GET` | `/api/versao` | Branch e commit que o servidor está rodando (exibidos no canto da tela) |
| `DELETE` | `/api/session` | Encerra e limpa a sessão atual |

Todas as rotas de sessão recebem o `sessionId` pelo header `X-Session-Id`.
