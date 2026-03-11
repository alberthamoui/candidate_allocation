Toda vez que você alterar o workflow do códiog no import_cli.go que esta dentro do back.workflow você deve alterar o workflow no aplicativo mesmo então nas chamadas do front-end com o back-end. (necessario)

Sempre rode os testes depois para validar que você não quebrou nada. (necessário)

Sempre que você tiver alguma informação util para que o proximo agente não se perca na implementação do código altere o arquivo AGENTS.md e implemente essa alteração, ao escrever não precisa colocar as mudanças mas como funciona atualmente.(necessário)

Se você considerar que existe informações desnecessárias no AGENTS.md Você pode altera-las e removelas.(necessário)

Sempre que você fizer uma função crie um teste para essa função tbm

### Workflow de Importação e Verificação
O processo de importação segue uma sequência rigorosa para garantir a integridade dos dados:
1.  **Mapeamento Interativo (CLI/Frontend)**: O usuário associa colunas do Excel aos campos da struct alvo. Na CLI, o sistema sugere um mapeamento e permite a personalização manual.
2.  **Mapeamento de Candidatos**: Associação de colunas aos campos de `Candidato`. Campos não associados (ex: ao trocar colunas ou adicionar extras sem definir coluna) recebem o índice `-1` e são ignorados no processamento final.
3.  **Mapeamento de Avaliadores**: Associação de colunas aos campos de `Avaliador`. Assim como em candidatos, o índice `-1` indica campo não mapeado.
4.  **Mapeamento de Restrições**: Mapeamento de restrições entre candidatos e avaliadores.
5.  **Verificação Manual**: No frontend, interface para resolver duplicatas e corrigir erros. Na CLI, o processamento segue filtrando inválidos e duplicados automaticamente com base no mapeamento definido.
6.  **Sucesso**: Persistência final no banco de dados SQLite.

### Lógica de Mapeamento e Campos Extras
- **Similaridade**: A sugestão automática de mapeamento usa o algoritmo de Levenshtein com um threshold de **20%**. Colunas que não atingem esse nível de similaridade com campos principais são sugeridas como campos "Extras".
- **Remapeação Numérica**: A lógica de similaridade normaliza variações numéricas (ex: "1", "um", "primeira") para facilitar a associação automática de colunas como "1opcao" ou "primeira opcao".
- **Controle Total**: Apenas o que estiver explicitamente mapeado (seja como campo principal ou como "extra" definido pelo usuário) será importado. O preenchimento automático de colunas não mapeadas para o campo "Extras" foi desativado tanto no frontend quanto na CLI para evitar poluição de dados.
- **Edição**: Na interface de mapeamento, campos extras podem ser renomeados (alterando a chave no map `Extras`) ou removidos (ignorando a coluna).

### Deduplicação e Verificação
- **Genérica**: Implementada em `back/logic/logic.go` usando a tag `app:"duplicate"`. Valores nulos ou vazios são ignorados para evitar falsos positivos em campos opcionais.
- **Avaliadores**: Qualquer igualdade nos campos `Nome`, `Sigla` ou `Email` dispara um alerta de duplicidade.
- **Frontend**: Componente `EntityVerificationView.tsx` centraliza a lógica de resolução de conflitos e edição em tempo real. No mapeamento, campos extras sem coluna associada são removidos automaticamente.
