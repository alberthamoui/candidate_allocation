# Documentação técnica

Este arquivo existe para compatibilidade com referências antigas. A documentação canônica foi dividida por responsabilidade em [`docs/README.md`](docs/README.md).

## Resumo do sistema

O Candidate Allocation importa três abas de uma planilha — candidatos, avaliadores e restrições —, sugere mapeamentos, permite revisão e persistência em SQLite, coleta parâmetros de alocação e executa um solver exato. O backend Go é a fonte dos contratos; o Wails expõe esses contratos ao frontend React; a CLI reutiliza as mesmas camadas de lógica e alocação.

Use os seguintes pontos de entrada:

- arquitetura: [`docs/architecture/system-overview.md`](docs/architecture/system-overview.md)
- funcionalidades: [`docs/features/`](docs/features/)
- contratos: [`docs/contracts/`](docs/contracts/)
- desenvolvimento e testes: [`docs/operations/`](docs/operations/)
- mapa código → documentação: [`docs/manifest.yaml`](docs/manifest.yaml)
- limitações atuais: [`docs/reference/known-limitations.md`](docs/reference/known-limitations.md)
