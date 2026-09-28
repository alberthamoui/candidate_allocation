# Atalhos do projeto. Rode `make` para ver a lista.
# No Windows, instale o make antes: scoop install make  (ou choco install make)
#
# Os comandos funcionam tanto no cmd do Windows quanto no sh (Linux/Mac/Git
# Bash): só `cd pasta && comando`, sem `./programa` nem rm/del.

# Versão fixa do linter, a última que roda com Go 1.24 (as mais novas pedem
# Go 1.25+). O `go run` baixa e guarda em cache na primeira vez.
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0

ifeq ($(OS),Windows_NT)
  EXE := .exe
endif
SERVIDOR := server$(EXE)

.DEFAULT_GOAL := ajuda
.PHONY: ajuda instalar frontend build rodar testar testar-go testar-interface lint formatar verificar limpar

ajuda:
	@echo Uso: make [tarefa]
	@echo   instalar          instala as dependencias do frontend e do Go
	@echo   rodar             compila o frontend e sobe o servidor em http://localhost:8080
	@echo   build             gera o executavel $(SERVIDOR) com o frontend embutido
	@echo   testar            roda os testes do Go e os de interface
	@echo   testar-go         so os testes do Go
	@echo   testar-interface  so os testes de interface (Playwright)
	@echo   lint              analisa o codigo Go com o golangci-lint
	@echo   formatar          formata o codigo Go
	@echo   verificar         lint + todos os testes, para rodar antes de abrir um PR
	@echo   limpar            apaga o executavel e os relatorios de teste

instalar:
	cd frontend && npm install
	go mod download

# o frontend compilado fica embutido no servidor Go (main.go)
frontend:
	cd frontend && npm run build

build: frontend
	go build -o $(SERVIDOR) .

rodar: frontend
	go run .

testar: testar-go testar-interface

testar-go:
	go test ./...

testar-interface:
	cd frontend && npm run test:e2e

lint:
	$(GOLANGCI_LINT) run ./...

formatar:
	$(GOLANGCI_LINT) fmt ./...

verificar: lint testar

limpar:
	node -e "for (const p of ['server', 'server.exe', 'frontend/playwright-report', 'frontend/test-results']) require('fs').rmSync(p, { recursive: true, force: true })"
