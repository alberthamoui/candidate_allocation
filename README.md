# Candidate Allocation

Aplicativo desktop para alocação de candidatos em mesas de entrevista, desenvolvido com Wails v2 (Go + React/TypeScript).

O sistema lê um arquivo Excel (.xlsx) contendo candidatos, avaliadores e restrições, permite revisar e corrigir os dados importados, executa o algoritmo de alocação e exporta o resultado.

## Requisitos

- [Go](https://golang.org/) 1.21 ou superior
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation) 
- [Node.js](https://nodejs.org/) 18 ou superior
- Compilador C (necessário para `go-sqlite3`) — no Windows, instale o [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)

## Instalação

1. Clone o repositório:

   ```bash
   git clone https://github.com/alberthamoui/candidate_allocation.git
   cd candidate_allocation
   ```

2. Instale as dependências Go e frontend:

   ```bash
   go mod tidy
   cd frontend && npm install && cd ..
   ```

## Formatação do arquivo Excel

O arquivo `.xlsx` deve ter exatamente **3 abas**, nesta ordem. Um arquivo de exemplo com dados fictícios está disponível em [`Execelteste/base_exemplo.xlsx`](Execelteste/base_exemplo.xlsx).

### Aba 1 — Candidatos

Uma linha por candidato. Os nomes de coluna não precisam ser exatos — o app sugere um mapeamento automático por posição, que pode ser ajustado na interface.

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

Dois parâmetros são configurados na tela inicial do app antes do upload:

- **Número de opções de horário** (N): quantas colunas de disponibilidade existem na planilha. A primeira opção tem maior prioridade no algoritmo de alocação.
- **Domínio do email institucional**: sufixo que todos os emails institucionais devem ter (ex: `@al.insper.edu.br`). Emails que não terminarem com esse domínio serão sinalizados como inválidos na etapa de revisão.

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

Exemplo: candidato `João Silva` com `NaoPosso = ABC, DEF` nunca será colocado em mesa com os avaliadores de sigla `ABC` ou `DEF`.



## Como rodar

**Modo desenvolvimento** (hot reload):

```bash
wails dev
```

**Build para distribuição** (gera executável em `build/bin/`):

```bash
wails build
```

## Fluxo de uso

O aplicativo guia o usuario em 3 etapas:

1. **Candidatos** — upload do arquivo `.xlsx`, mapeamento de colunas, revisao e correcao dos dados importados (resolucao de duplicatas, edicao de campos)
2. **Avaliadores** — leitura da aba "Avaliadores" do mesmo arquivo, mapeamento e confirmacao
3. **Restricoes** — leitura da aba "Restricoes", mapeamento e confirmacao

Apos as 3 etapas, o algoritmo de alocacao e executado automaticamente. O resultado mostra as mesas formadas e os candidatos nao alocados. E possivel exportar o resultado para `.xlsx` ou reiniciar o processo do zero.

## Estrutura principal

```
candidate_allocation/
├── app.go          -- backend: funcoes expostas ao frontend via Wails
├── alocate.go      -- algoritmo de alocacao
├── processa.go     -- parsing do arquivo Excel
├── setup.go        -- inicializacao do banco de dados SQLite
├── db/             -- funcoes auxiliares de banco
└── frontend/
    └── src/
        ├── main.tsx           -- roteamento e estado global
        ├── App.tsx            -- tela inicial e upload
        ├── MappingPage.tsx    -- tela de mapeamento de colunas (reutilizada nas 3 etapas)
        ├── VerifyUsers.tsx    -- revisao de candidatos
        ├── UploadAvaliador.tsx
        ├── UploadRestricao.tsx
        └── Resultado.tsx      -- resultado da alocacao e exportacao
```

## Banco de dados

O arquivo `insper.db` (SQLite) e criado automaticamente na primeira execucao. Tabelas: `pessoa`, `opcoes_horario`, `disponibilidade`, `avaliador`, `restricoes`.

O botao "Reiniciar" no resultado apaga todos os dados e permite comecar um novo processo de alocacao.
