---
id: documentation-update
title: Atualização da documentação
summary: Comando, estado e contrato usados pelo Codex para sincronizar documentos com changes do Jujutsu.
status: active
sources:
  - scripts/update-docs
  - scripts/update-docs.test.sh
  - docs/documentation-state.yaml
  - docs/prompts/documentation-updater.md
  - docs/manifest.yaml
---

# Atualização automática da documentação

## Comando

Depois de concluir mudanças de código, execute uma vez na raiz:

```bash
make docs-update
```

Isso chama `scripts/update-docs`, encontra tudo que ainda não foi coberto e inicia uma sessão não interativa e efêmera com:

```bash
codex exec --ephemeral --sandbox workspace-write
```

O Codex reutiliza a autenticação local. A execução não depende do contexto de uma conversa anterior e não persiste uma sessão para retomada.

Para apenas examinar o intervalo:

```bash
make docs-update-dry-run
```

## Componentes

| Arquivo | Responsabilidade |
|---|---|
| `scripts/update-docs` | calcula intervalo, isola change, executa Codex, valida escopo e avança estado |
| `docs/documentation-state.yaml` | guarda o último estado de código analisado |
| `docs/prompts/documentation-updater.md` | prompt versionado da instância do Codex |
| `docs/manifest.yaml` | roteia fontes, testes e consumidores para documentos |
| `scripts/update-docs.test.sh` | testa o orquestrador com executáveis falsos, sem consumir Codex |

## Contrato editorial

Esta seção é a fonte das regras de escrita da documentação. Ela é carregada pelo prompt somente quando `make docs-update` executa; não precisa ser duplicada no `AGENTS.md`.

### Objetivo e público

A documentação deve funcionar para dois públicos ao mesmo tempo:

- LLMs precisam localizar rapidamente fatos, fontes, dependências, invariantes e efeitos de uma mudança;
- pessoas precisam entender o sistema sem reconstruir o comportamento lendo todo o código.

Cada documento deve ser autocontido dentro de seu assunto, mas deve usar links para evitar repetir conteúdo mantido por outro documento. Código e testes são evidência; documentação explica o comportamento e orienta navegação.

### Estilo de escrita

- Escreva em português brasileiro, salvo nomes de símbolos, APIs e termos consolidados do código.
- Use linguagem direta, voz ativa e tempo presente para descrever o estado atual.
- Comece seções pelo resultado, regra ou responsabilidade principal; detalhe o mecanismo depois.
- Prefira parágrafos curtos. Use listas para conjuntos e tabelas para mappings, campos, estados ou comparações repetidas.
- Use sempre o mesmo termo para o mesmo conceito. Quando código e negócio usam nomes diferentes, explique a equivalência no glossário.
- Cite paths relativos ao repositório entre crases, como `back/logic/workflow_definition.go`, e símbolos exatos, como `WorkflowDefinition`.
- Use blocos de código com linguagem identificada e comandos que possam ser copiados diretamente.
- Explique siglas e termos pouco óbvios na primeira ocorrência ou crie uma entrada em `docs/reference/glossary.md`.
- Use links Markdown relativos para navegar entre documentos relacionados.
- Seja específico: escreva condições, entradas, saídas, estados, defaults, erros e efeitos observáveis quando forem relevantes.
- Evite linguagem promocional, adjetivos vagos, introduções genéricas e frases que não ajudam uma decisão ou implementação.
- Evite referências temporais como “agora”, “recentemente” e “novo”. Documentos canônicos descrevem o presente; ADRs preservam história.

### Regras para consumo por LLMs

- Um título deve nomear exatamente o assunto pesquisável; não use títulos criativos.
- Cada documento deve ter escopo coeso. Não crie um arquivo por função, mas também não misture domínios sem relação.
- Declare explicitamente fontes de verdade, produtores, consumidores, invariantes, persistência e efeitos colaterais.
- Use nomes reais de tipos, funções, eventos, rotas, tabelas e arquivos para permitir busca textual no repositório.
- Não dependa de contexto de conversa, conhecimento tribal ou ordem de leitura implícita.
- Diferencie comportamento oficial, caminho legado, artefato gerado e limitação conhecida.
- Quando uma afirmação não puder ser comprovada, use `needs-review` e descreva exatamente qual evidência falta.
- Preserve IDs estáveis de frontmatter e relações do manifesto; eles são chaves de roteamento, não rótulos decorativos.
- Mantenha fatos importantes em texto e tabelas Markdown. Não dependa apenas de imagens ou diagramas.
- Não copie grandes trechos de código. Resuma a regra e aponte para a fonte canônica.

### Frontmatter obrigatório

Todo Markdown canônico em `docs/` começa com YAML:

```yaml
---
id: identificador-estavel
title: Título humano e pesquisável
summary: Uma frase que informa escopo e utilidade do documento.
status: active
---
```

Regras dos campos:

| Campo | Regra |
|---|---|
| `id` | único, estável e igual ao `id` correspondente em `docs/manifest.yaml` |
| `title` | curto, descritivo e coerente com o primeiro `#` |
| `summary` | uma frase concreta; não repetir apenas o título |
| `status` | `active`, `generated`, `accepted`, `superseded` ou `needs-review` |
| `sources` | opcional no Markdown; lista apenas fontes canônicas principais, sem tentar substituir o manifesto |

Não altere um `id` por ajuste de redação. Se o assunto for substituído conceitualmente, atualize links e manifesto de forma explícita.

### Estrutura por tipo de documento

#### Índices e READMEs

Devem orientar navegação, não repetir a documentação inteira. Inclua:

1. propósito do conjunto;
2. ordem de leitura recomendada, quando houver;
3. links agrupados por responsabilidade;
4. indicação da fonte de verdade ou do manifesto aplicável.

#### Arquitetura

Use para responsabilidades e relações entre partes estáveis do sistema. Inclua, quando aplicável:

1. escopo e responsabilidade do sistema ou componente;
2. componentes e fronteiras;
3. fluxo de dados, controle ou estado;
4. fontes de verdade e invariantes;
5. dependências e consumidores;
6. persistência e duração do estado;
7. falhas relevantes e links para limitações conhecidas.

Não transforme arquitetura em inventário de todos os arquivos; use o mapa de código para isso.

#### Funcionalidades

Descreva comportamento observável de ponta a ponta. Inclua:

1. objetivo para o usuário ou consumidor;
2. entradas e pré-condições;
3. sequência principal;
4. validações e regras de domínio;
5. saídas e efeitos colaterais;
6. estado e persistência;
7. erros, casos vazios e caminhos alternativos;
8. contratos, telas, eventos ou serviços envolvidos;
9. testes que comprovam os cenários importantes;
10. limitações conhecidas relacionadas.

#### Contratos

Use para interfaces consumidas por mais de uma camada. Inclua:

1. fonte canônica;
2. tipos, campos, operações ou eventos expostos;
3. invariantes, obrigatoriedade e defaults;
4. produtores e consumidores;
5. serialização, persistência ou compatibilidade, quando existirem;
6. comportamento de erro;
7. procedimento seguro de alteração e testes afetados.

Defaults devem apontar para a fonte de verdade. Podem ser apresentados para leitura, mas não descritos como uma segunda configuração editável.

#### Operação e desenvolvimento

Escreva como procedimento verificável. Inclua:

1. objetivo;
2. pré-requisitos;
3. comando copiável;
4. sequência e arquivos envolvidos;
5. resultado esperado;
6. falhas comuns e recuperação segura;
7. validação posterior;
8. artefatos gerados e o que não deve ser versionado.

Não assuma ferramentas instaladas sem listá-las.

#### ADRs

ADRs registram decisões, não o estado completo da implementação. Use:

1. título numerado `ADR-NNNN`;
2. status;
3. contexto e forças que exigiram decisão;
4. decisão tomada;
5. consequências positivas e negativas;
6. alternativas consideradas;
7. `supersedes` ou `superseded-by`, quando aplicável.

Um ADR aceito é histórico e não deve ser reescrito para acompanhar código novo. Crie outro ADR para substituir a decisão.

#### Referências e inventários gerados

- Glossário: termo, significado no projeto e equivalências importantes.
- Mapa de código: path ou símbolo e responsabilidade, sem explicar toda a implementação.
- Inventários: cobertura mecânica e rastreável; mantenha `status: generated`.
- Limitações conhecidas: comportamento atual, causa comprovada, impacto e evidência. Não apresente uma solução futura como decidida.

### Conteúdo que não deve entrar

- hipóteses apresentadas como fatos;
- planos futuros sem decisão registrada;
- changelog commit a commit em documentos que descrevem o estado atual;
- detalhes transitórios que não afetam contrato, operação, manutenção ou entendimento;
- caminhos absolutos, números de linha e dados específicos da máquina de quem escreveu;
- segredos, tokens, dados pessoais ou conteúdo sensível de fixtures;
- cópias extensas de código, arquivos gerados ou relatórios temporários;
- regras duplicadas que possam divergir da fonte canônica;
- instruções obsoletas mantidas apenas para evitar remover texto.

Quando o código deixa de oferecer um comportamento, remova ou reescreva a afirmação correspondente. Preserve história somente em ADRs ou quando ela for necessária para migração e compatibilidade.

### Checklist de completude

Antes de concluir uma atualização, confirme:

- o documento descreve o comportamento presente no hash alvo;
- título, resumo, headings e termos permitem localizar o assunto por busca;
- fontes de verdade e consumidores estão explícitos;
- entradas, saídas, estado, erros e limites relevantes foram cobertos;
- afirmações importantes têm evidência em código ou testes;
- links relativos, paths e nomes de símbolos existem;
- não há contradição ou duplicação com documentos relacionados;
- limitações descobertas foram registradas;
- o manifesto cobre fontes, testes e consumidores novos;
- conteúdo obsoleto foi removido, não apenas acrescido de ressalvas;
- uma pessoa consegue entender o fluxo e uma LLM consegue localizar onde alterá-lo.

## Marcador do Jujutsu

O estado guarda dois identificadores:

- `change_id`: identidade estável do change coberto, usada como âncora do próximo intervalo;
- `commit_id`: hash exato do conteúdo de código que foi entregue ao Codex naquela execução, usado para auditoria.

Não é possível gravar de forma confiável o hash final dentro do próprio commit: alterar o arquivo que contém o hash também altera o hash do commit. Para evitar essa autorreferência, a documentação é escrita em um change filho e o marcador aponta para o change pai de código.

O arquivo de estado só é atualizado depois que `codex exec` termina com sucesso, permanece no mesmo change e não altera arquivos fora do escopo documental.

## Fluxo

1. lê a base em `docs/documentation-state.yaml`;
2. escolhe como alvo o change atual, ou seu pai quando `@` já está vazio;
3. confirma que a base continua ancestral do alvo;
4. calcula commits e arquivos entre base e alvo;
5. ignora documentação e artefatos gerados para decidir se há trabalho;
6. se o código está no change atual, executa `jj new` para reservar um filho documental;
7. envia hashes, commits, paths e o prompt versionado para `codex exec` pela entrada padrão;
8. confirma que apenas `docs/**`, `README.md`, `Docs.md` ou `AGENTS.md` foram modificados;
9. grava o novo marcador e descreve o change como `docs: update through <hash>`.

Se o change atual já contém somente documentação, ele é reutilizado. Isso permite repetir o comando depois de uma falha do Codex sem criar uma cadeia de children vazios.

## Trabalho da instância do Codex

O agente deve ler o manifesto e as fontes reais, inspecionar todos os commits do intervalo e atualizar apenas documentos afetados. Mensagens de commit não são evidência suficiente. Código e testes determinam o comportamento documentado.

Regras importantes:

- preservar IDs e títulos salvo mudança conceitual;
- usar paths relativos e símbolos, não números de linha;
- não duplicar defaults que pertencem a contratos do backend;
- criar outro ADR em vez de reescrever um ADR aceito;
- registrar divergências comprovadas em `known-limitations.md`;
- marcar `needs-review` quando a evidência for insuficiente;
- adicionar ao manifesto qualquer responsabilidade nova sem cobertura;
- não editar `documentation-state.yaml` nem executar comandos mutáveis do `jj`.

## Falhas e segurança

- Falha do Codex: o marcador não avança. Corrija o problema e execute o mesmo comando novamente; o change documental atual será reutilizado.
- Arquivo de código alterado pelo Codex: o comando falha e não avança o marcador. Revise o diff manualmente.
- Base não ancestral: o comando para antes de editar. Isso normalmente indica rebase ou abandono do change registrado; escolha conscientemente uma nova base antes de alterar o estado.
- Nenhum arquivo relevante: termina sem abrir Codex e sem criar change.
- Codex ausente ou sem autenticação: termina antes de modificar o estado.

O comando usa `workspace-write`; não usa `danger-full-access` nem aumenta limites do Jujutsu.

Dentro do sandbox do Codex, leituras do histórico usam `jj --ignore-working-copy`. O wrapper já cria o snapshot necessário antes de iniciar o agente e faz a validação mutável depois que ele termina.

## Validação e manutenção

Execute os testes do orquestrador com:

```bash
make test-docs-update
```

Ao alterar script, estado, prompt ou manifesto, mantenha os quatro sincronizados. O estado é versionado e deve avançar somente pelo script depois de uma execução bem-sucedida; não o atualize antecipadamente para esconder commits ainda não analisados.
