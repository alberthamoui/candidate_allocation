# Documentação Técnica - Candidate Allocation

## Visão Geral
O sistema permite a importação de candidatos, avaliadores e restrições a partir de arquivos Excel, realizando o mapeamento flexível entre as colunas do arquivo e os campos do banco de dados.

## Fluxo de Mapeamento (Frontend)

O mapeamento é realizado em três etapas principais: Candidatos, Avaliadores e Restrições. Em cada etapa, a interface apresenta três seções:

1.  **Campos Principais:** Campos obrigatórios ou padrão do sistema (ex: Nome, CPF, Email).
2.  **Campos Extras:** Colunas do Excel que o usuário deseja importar com um nome personalizado. Elas são armazenadas em um campo `extras` (JSON) no banco de dados.
3.  **Colunas Disponíveis (Não Mapeadas):** Colunas presentes no arquivo Excel que ainda não foram atribuídas a nenhuma variável.

### Regras de Manipulação
-   **Arrastar e Soltar:** Permite mover colunas entre variáveis ou para a seção de colunas disponíveis.
-   **Remoção de Mapeamento:** Ao "remover" um mapeamento de um campo extra, a coluna não desaparece; ela é movida de volta para a seção "Colunas Disponíveis" (a variável é limpa).
-   **Persistência:** Apenas colunas com uma variável atribuída são enviadas ao backend para processamento. Colunas na seção "Disponíveis" são ignoradas durante a importação.

## Lógica de Backend

### Processamento de Mapping
O backend recebe uma lista de `MappingItem`:
-   `Indice`: O índice da coluna no Excel (0-based).
-   `Variavel`: O nome do campo no banco ou a chave no mapa de `extras`.
-   `NomeColuna`: Nome original da coluna (usado para referência).

### Comportamento de Importação
-   Se `Indice` for negativo ou a `Variavel` for vazia, o item é ignorado.
-   Campos que não existem na struct principal do modelo (ex: `Candidato`) são automaticamente movidos para o mapa `Extras`.
-   A validação de duplicatas (CPF, Email) ocorre após o processamento do mapeamento.

## Tecnologias Utilizadas
-   **Frontend:** React, Tailwind CSS, Framer Motion (animações de drag-and-drop), Wails (ponte com backend).
-   **Backend:** Go, Excelize (leitura de Excel), SQLite (banco de dados).

## Como adicionar novos campos
Para adicionar campos fixos, altere as structs em `back/type/types.go` e atualize as constantes de campos core no frontend (`MappingPage.tsx`). Para campos dinâmicos, o sistema de `extras` já lida com a persistência automaticamente.
