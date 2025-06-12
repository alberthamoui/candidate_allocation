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

No diretório do projeto, execute:

```bash
go run receiveexel.go -file path/para/arquivo.xlsx
```

Exemplo:

```bash
go run receiveexel.go -file Base.xlsx
```

O programa irá:

1. Perguntar quantas opções de alocação existem (nOpcoes).  
2. Listar os tipos válidos: timestamp, nome, cpf, numero, semestre, curso, email_insper, email_pessoal, opcao, none
3. Para cada coluna do seu Excel, solicitar que você escolha um dos tipos acima.  
4. Se você escolher `opcao`, perguntar também qual índice (1…nOpcoes).  
5. Ao final, imprimir na tela um JSON com todos os usuários mapeados.

## Known issue / Próxima melhoria

Hoje, **se você atribuir o mesmo campo a mais de uma coluna**, o programa não avisa que já há um mapeamento pré-existente.  
**O comportamento desejado** seria:

1. Detectar que aquele tipo de campo já foi atribuído a outra coluna.  
2. Perguntar se você quer **substituir** o mapeamento anterior.  
3. Caso sim, perguntar para qual outro campo deseja remapear a coluna antiga.  
4. Atualizar o mapeamento de forma consistente.

> **Exemplo de fluxo ideal**  
>
> - Você atribui `nome` à coluna 2.  
> - Mais adiante, tenta atribuir `nome` à coluna 5.  
> - O sistema diz: “O campo `nome` já está usado na coluna 2. Deseja substituir? (s/N)”  
>   - Se “s”: pergunta “Para qual tipo deseja remapear a coluna 2?” e então prossegue.  
>   - Se “N”: mantém o mapeamento original e solicita outro tipo para a coluna 5.

Você pode implementar essa lógica dentro do loop de perguntas em `ParseExcelInteractive`, armazenando um mapa `field -> colunaIndex` e, ao detectar conflito, reaproveitar o laço interativo para resolver a substituição.

---

Bom trabalho e bons alocamentos! 🚀
