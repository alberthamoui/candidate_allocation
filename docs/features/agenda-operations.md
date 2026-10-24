---
id: agenda-operations
title: Operações da agenda e comunicação
status: active
sources: [agenda_app.go, channels_app.go, google_app.go, agenda_cli.go, agenda_export.go, back/operations/store.go, frontend/src/AgendaActions.tsx, frontend/src/ChannelsPanel.tsx]
---

# Operações da agenda e comunicação

A base de desenvolvimento e destino de pull requests é `mainjoao`. A implementação preserva o solver exato e usa os contratos de `back/type` e `back/logic/workflow_definition.go` para regras e otimização.

O resultado final é salvo como uma agenda em `operations.db`; importação e solver continuam usando o banco de domínio. Uma nova importação válida limpa apenas os dados de domínio. Avaliações e mudanças manuais gravam a agenda; consultas e exportação leem a mesma fonte, inclusive pela CLI. A versão do snapshot invalida caches quando outra instância/CLI altera o arquivo.

O cache de snapshots guarda até oito itens por instância, expira após quinze minutos e recupera dados do SQLite sob demanda respeitando o limite. O `App` também descarrega seu resultado inativo. Cada operação de armazenamento abre e fecha sua própria conexão; o histórico não é carregado no cache. Um item retirado da memória permanece no disco.

As ações ficam junto ao candidato e à mesa. O administrador pode avaliar, consultar destinos e realocar livremente; o backend recalcula os alertas, os critérios e o relatório de qualidade usando o contrato do solver. Os indicadores filtram pendências de avaliação, comunicação, alocação e conflitos. Enquanto a busca está em andamento, alterações são bloqueadas para impedir que um resultado posterior sobrescreva o trabalho manual.

O painel de canais separa conexão, mensagem e histórico. Gmail pode usar senha de aplicativo com SMTP preenchido automaticamente. Google OAuth usa PKCE, estado aleatório, callback apenas em loopback e credenciais de cliente Desktop configuradas por ambiente. O refresh token fica no armazenamento privado; a API da UI recebe apenas o estado da conexão. WhatsApp Business usa ID do número, token e modelo aprovado; webhooks HTTPS são uma alternativa avançada. O texto livre e o modelo da Meta são apresentados como formatos distintos.

Os envios guardam um lote e um registro por destinatário/canal. Primeiro uma mensagem passa de `pending` para `sending` por uma atualização condicional; a chamada de rede ocorre fora das travas da agenda; seu resultado atualiza somente o registro daquela mensagem. Estados finais são `accepted` e `failed`. Após interrupção, `sending` vira `unknown`: o provedor pode ter aceitado e o operador precisa conferir. Mensagens ainda `pending`, já autorizadas anteriormente, podem continuar no próximo startup. Falhas confirmadas são repetidas apenas uma vez por lote de origem, com contatos e agenda atuais; sucessos e estados desconhecidos não entram nessa repetição.

O histórico consulta primeiro a página de lotes e agrega somente seus registros. Os resumos não contêm corpos nem listas de destinatários. Detalhes são paginados separadamente. O frontend consulta ao abrir o histórico, ao pedir atualização e quando um envio termina, enquanto o painel está visível; não usa intervalo periódico. A CLI espera os próprios envios e retorna falha se houver pendência ou rejeição. Uma falha de comunicação depois de realocar preserva a mudança e é informada separadamente.

Validação: testes de cache, recuperação, paginação, concorrência de claim, repetição, estados ambíguos, operações manuais, Excel e contratos SMTP/Meta/Google. As telas têm testes em `tools/ui-testing/tests/07-agenda.spec.ts`. O modo `preview` usa métodos reais do App em hosts sem GTK/WebKit e não é uma implantação de produção. Entrega por contas reais exige credenciais e autorização dos provedores.
