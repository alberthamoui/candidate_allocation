---
id: debug-validation
title: Validação nativa, canais e desempenho em debug
status: active
sources: [debug_cli.go, debug_cli_release.go, back/diagnostics, tools/native-validation]
tests: [debug_cli_test.go, debug_cli_release_test.go, back/diagnostics/debug_test.go, back/diagnostics/release_test.go]
---

# Validação e desempenho

## Disponibilidade

As amostras de desempenho e a CLI de medição só são compiladas com a tag `debug` ou `dev`. A compilação release usa funções sem coleta e rejeita `debug performance`; variáveis de ambiente não podem ativá-la. `wails dev` usa a tag `dev`, e `wails build -debug` usa `debug`. Esta instrumentação não altera o relatório de qualidade do solver nem acrescenta métricas à interface de operação.

```bash
go test ./...
go test -race -tags debug ./...
go run -tags debug . debug performance -sizes 100,1000,5000 -runs 3 -solver-budget 2s -case-timeout 30s
go run -tags debug . debug performance -sizes 100,1000,5000 -runs 3 -scenario criteria -solver-budget 2s -case-timeout 30s
```

O comando emite JSON com ambiente, cenário, volume, repetição, status, quantidade alocada e amostras de parsing, persistência, leitura, solver, montagem da agenda, recuperação e histórico. O cenário `preferences` usa duas preferências por candidato entre cinco horários; `criteria` adiciona distribuição equilibrada de três cursos. São dados sintéticos, gerados em XLSX, sem credenciais ou envios. O banco de importação, a agenda e os registros de comunicação ficam em diretórios temporários exclusivos e são removidos, inclusive após timeout.

Cada cenário executa em outro processo. `solver-budget` controla o cancelamento cooperativo da busca; a preparação inicial por fluxo pode ultrapassá-lo. `case-timeout` limita o teste inteiro e encerra o processo caso necessário. Nesse caso o resultado é `timeout`, sem uma alocação concluída; somente as amostras completadas antes da busca podem estar disponíveis. O limite do benchmark não foi aplicado à busca normal do produto.

A duração usa relógio monotônico. `allocatedBytes` é a diferença de `TotalAlloc` do processo durante a etapa, e `heapBytes` é o heap ao terminar a etapa, **não o pico de memória**. Na aplicação, atividades concorrentes podem participar dessas contagens; no benchmark, cada execução é isolada. A retenção em memória é limitada às últimas 128 amostras. Não se coletam nomes, contatos, mensagens, planilhas ou credenciais.

## Aplicativo nativo Linux

O servidor desta revisão usa Ubuntu Server 26.04 e não possui GTK/WebKit/pkg-config instalados nem sudo sem senha. Para testar o desktop sem alterar o host, execute, após o build do frontend:

```bash
cd frontend && npm ci && npm run build && cd ..
tools/native-validation/run.sh
```

O script usa Docker com Go 1.25.9 e Debian Bookworm, GTK 3/WebKit 4.1 e uma tela virtual Xvfb. Compila os binários release/debug, confirma que o release rejeita a medição e abre o aplicativo nativo com dados separados. As screenshots registram a janela, a importação de um XLSX de teste pela WebView e o layout em largura reduzida. O diretório de saída é informado ao terminar; `CANDIDATE_NATIVE_OUTPUT` permite escolher um diretório persistente.

Essa evidência valida o binário Linux no contêiner e a integração GTK/WebKit. Não representa uma instalação gráfica diretamente no Ubuntu Server, nem uma versão web de produção. O preview com os mesmos métodos App é usado pelos testes Playwright; o browser colaborativo T3 do host não inicia por falta de bibliotecas.

## Canais

Nesta execução, conexões TLS com `accounts.google.com:443`, `gmail.googleapis.com:443`, `graph.facebook.com:443` e `smtp.gmail.com:465` funcionaram. Não foram encontradas contas habilitadas no armazenamento local inspecionado. Portanto autenticação com contas reais, consentimento OAuth e entrega real continuam **não validados**. Conectividade não é prova de autenticação ou de entrega.

Os testes automatizados cobrem SMTP local, OAuth/refresh e MIME do Gmail por transporte de teste, modelo aprovado da Meta e prevenção de repetição de envios ambíguos. Para homologar uma conta, conecte-a no painel, execute **Salvar e testar conexão** e autorize um destinatário de teste. Confirme o recebimento na caixa/telefone e confira cada envio no histórico. Nunca cole senhas ou tokens no chat, no Git ou em relatórios.

## Resultados da execução

Rodada de 25/10/2026 em Linux amd64, Go 1.27.3, oito CPUs visíveis, servidor compartilhado com outras tarefas. Três execuções isoladas por volume/cenário, prazo cooperativo de busca de 2 s e limite total de 30 s. Valores abaixo são medianas em milissegundos; o total inclui geração da planilha e todas as etapas.

| Cenário | Candidatos | Parsing XLSX (ms) | Persistência (ms) | Total (ms) | Status das três execuções |
|---|---:|---:|---:|---:|---|
| preferences | 100 | 18.6 | 9.4 | 94.5 | optimal |
| preferences | 1000 | 138.5 | 66.4 | 3128.8 | feasible |
| preferences | 5000 | 604.1 | 355.4 | 30128.4 | timeout |
| criteria | 100 | 11.3 | 6.2 | 2062.9 | feasible |
| criteria | 1000 | 123.1 | 65.5 | 3268.1 | feasible |
| criteria | 5000 | 519.7 | 301.8 | 30114.6 | timeout |

Nos cenários concluídos, todos os candidatos foram alocados; `feasible` significa solução viável sem prova de optimalidade dentro da janela. Em 5.000 candidatos, o parsing e a gravação terminaram, mas a preparação/busca excedeu o limite total em todas as repetições. As contagens de alocação de casos interrompidos não devem ser usadas. Esses resultados identificam o solver como próximo gargalo; não demonstram suporte operacional a 5.000 candidatos nem uma melhoria de velocidade do solver neste PR.

Validações adicionais: testes Go com detector de concorrência em debug e release; lint normal/debug; build TypeScript/Vite; 26 testes Playwright; builds nativos release/debug e inspeção das screenshots da janela GTK/WebKit. O teste de erro de gravação confirma que mensagens recebidas como string continuam legíveis e que a revisão permite correção.
