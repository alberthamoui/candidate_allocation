# candidate_allocation

# Candidate Allocation (receiveexcel)

Este projeto lê um arquivo Excel (`.xlsx`) e interativamente mapeia cada coluna para um dos campos de `Usuario`, gerando um JSON de saída.

## Pré-requisitos

- Go 1.24 ou superior  
- (Opcional) compilador C para permitir o uso de `github.com/mattn/go-sqlite3`  
- Arquivo Excel com ao menos um cabeçalho (primeira linha)

## Instalação

1. Clone o repositório:

   ```bash
   git clone https://github.com/alberthamoui/candidate_allocation.git
   cd candidate_allocation
   ```

2. Baixe as dependências e limpe o módulo:

   ```bash
   go mod tidy
   ```

## Como rodar

Modo desktop com Wails:

```bash
wails dev
```

Modo CLI, sem comentar o bootstrap do Wails:

```bash
go run . cli -file caminho/para/arquivo.xlsx
```

Se quiser mudar a quantidade de opcoes de horario esperadas na aba de candidatos:

```bash
go run . cli -file caminho/para/arquivo.xlsx -opcoes 5
```

Para rodar o Linter se roda assim que vai rodar o linter do go e o que eu fiz para o meu codigo

```bash
 make lint
```

## About

Wails template which includes: Vite, React, TS, TailwindCSS out of the box.

Build with `Wails CLI v2.0.0`.

To use this [template](https://wails.io/docs/community/templates):

```shell
wails init -n "Your Project Name" -t https://github.com/hotafrika/wails-vite-react-ts-tailwind-template
cd frontend/src
npm install
```

[Here](scripts) you can find useful scripts for building on different platforms and Wails CLI installation.

## UI Testing and Verification

This project includes a fully isolated, headless UI testing environment built with Playwright to verify the application's frontend.

### For Humans: How to run tests

The tests target the development server. Make sure you have the Wails dev server running:

1. Start your application in dev mode:

   ```bash
   wails dev
   ```

2. In a new terminal, run the tests:

   ```bash
   cd tools/ui-testing
   npm install # (first time only)
   ./runner.sh
   ```

### For the AI Agent: How to use this for verification

The testing infrastructure is built to be "agent-friendly" and completely headless. When building UI features, the agent should:

1. Ensure the app is running in the background (`wails dev &`).
2. Navigate to `tools/ui-testing` and run `./runner.sh`.
3. If tests fail, read the generated JSON report at `tools/ui-testing/test-results/report.json` to understand why.
4. If a visual layout needs verification, tests can be configured to take screenshots (e.g., `await page.screenshot({ path: 'ui-state.png' });`), which will be placed in the `test-results/` folder for analysis.

# Permitir que o opencode acesse a ui

va no arquivo `opencode.josn` e coloque tools  `true` e permission `allow` se não tiver assim ele não vai ter a tool de vizualizar a ui.

## Live Development

To run in live development mode, run `wails dev` in the project directory. In another terminal, go into the `frontend`
directory and run `npm run dev`. The frontend dev server will run on <http://localhost:34115>. Connect to this in your
browser and connect to your application.

## Building

To build a redistributable, production mode package, use `wails build`.

## JJ

O jj é top, gostei disso de branchless flow
como jj é legal, realmente legal, muito legal mesmo
Vou mandar isso por uma branch para ver como funciona
mais informações mesmo no readme
---

Bom trabalho e bons alocamentos! 🚀
