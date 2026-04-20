# Recomendações e regras para o código

Toda vez que você alterar o workflow do códiog no import_cli.go que esta dentro do back.workflow você deve alterar o workflow no aplicativo mesmo então nas chamadas do front-end com o back-end. (necessario)

Sempre rode os testes depois para validar que você não quebrou nada. (necessário)
Sempre rode o linter depois para validar se não existe algum erro (necessário)

Sempre que você tiver alguma informação util para que o proximo agente não se perca na implementação do código altere o arquivo Docs.md nele faça uma documentação high level de como funciona a solução .

Se você considerar que existe informações desnecessárias no AGENTS.md ou no Docs.md Você pode altera-las e removelas.(necessário)

Sempre que você fizer uma função crie um teste para essa função
