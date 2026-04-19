# Candidate Allocation

Sistema web para alocação de candidatos em mesas de entrevista, desenvolvido em Go (servidor HTTP) + React/TypeScript (frontend embarcado).

**Acesso online:** [https://candidate-allocation.onrender.com/](https://candidate-allocation.onrender.com/)

---

## Como funciona

O sistema guia o usuário em 4 etapas:

1. **Upload** — envia o arquivo `.xlsx` com candidatos, avaliadores e restrições e configura parâmetros iniciais
2. **Candidatos** — mapeamento de colunas, revisão e correção dos dados (duplicatas, campos inválidos)
3. **Avaliadores** — mapeamento e confirmação da aba de avaliadores
4. **Restrições** — mapeamento e confirmação da aba de restrições

Após as 4 etapas, o algoritmo de alocação é executado automaticamente. O resultado mostra as mesas formadas e os candidatos não alocados. É possível exportar o resultado para `.xlsx` ou reiniciar do zero.

---

## Formatação do arquivo Excel

O arquivo `.xlsx` deve ter exatamente **3 abas**, nesta ordem. Um arquivo de exemplo pode ser baixado diretamente na tela inicial do app (botão "Baixar exemplo").

### Aba 1 — Candidatos

Uma linha por candidato. Os nomes de coluna não precisam ser exatos — o app sugere mapeamento automático por posição, ajustável na interface.

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
├── processa.go     -- parsing do arquivo Excel
├── mapping.go      -- lógica de mapeamento de colunas
├── export.go       -- geração do Excel de resultado
├── models.go       -- structs de dados
├── setup.go        -- inicialização do banco SQLite
├── db/             -- funções auxiliares de banco
├── Dockerfile      -- build multi-stage (Node → Go → Alpine)
├── Excels/         -- arquivo de exemplo para download
└── frontend/       -- app React/TypeScript (Vite + Tailwind)
    └── src/
        ├── main.tsx            -- roteamento e estado global
        ├── App.tsx             -- tela inicial e upload
        ├── MappingPage.tsx     -- mapeamento de colunas (reutilizado nas 3 etapas)
        ├── VerifyUsers.tsx     -- revisão de candidatos
        ├── UploadAvaliador.tsx
        ├── UploadRestricao.tsx
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
| `GET` | `/api/alocar?sessionId=` | Executa alocação via Server-Sent Events (streaming de progresso) |
| `GET` | `/api/export?sessionId=` | Download do resultado em `.xlsx` |
| `GET` | `/api/exemplo` | Download do arquivo de exemplo |
| `DELETE` | `/api/session` | Encerra e limpa a sessão atual |

Todas as rotas de sessão recebem o `sessionId` pelo header `X-Session-Id`.
