export type WorkflowStepKey =
	| "import"
	| "candidate"
	| "restriction"
	| "reviewer"
	| "checkpoint"
	| "config"
	| "run"
	| "result";

export interface HelpSection {
	title: string;
	body: string;
}

export interface PageMeta {
	path: string;
	step: WorkflowStepKey;
	kicker: string;
	title: string;
	description: string;
	panelTitle: string;
	panelSummary: string;
	helpSections: HelpSection[];
}

export interface WorkflowStep {
	key: WorkflowStepKey;
	label: string;
	caption: string;
	paths: string[];
}

export const WORKFLOW_STEPS: WorkflowStep[] = [
	{
		key: "import",
		label: "Importação",
		caption: "Entrada do arquivo e início do processo.",
		paths: ["/"],
	},
	{
		key: "candidate",
		label: "Candidatos",
		caption: "Mapeamento e revisão da base principal.",
		paths: ["/mapping", "/verify"],
	},
	{
		key: "restriction",
		label: "Restrições",
		caption: "Conferência das regras vindas da planilha.",
		paths: ["/mappingRestricoes", "/verifyRestricoes"],
	},
	{
		key: "reviewer",
		label: "Avaliadores",
		caption: "Preparação dos responsáveis pela avaliação.",
		paths: ["/mappingAvaliadores", "/verifyAvaliadores"],
	},
	{
		key: "checkpoint",
		label: "Checkpoint",
		caption: "Confirmação antes da etapa operacional.",
		paths: ["/success"],
	},
	{
		key: "config",
		label: "Configuração",
		caption: "Parâmetros e critérios da alocação.",
		paths: ["/allocation-config"],
	},
	{
		key: "run",
		label: "Processamento",
		caption: "Execução e cálculo da distribuição.",
		paths: ["/allocation-loading"],
	},
	{
		key: "result",
		label: "Resultado",
		caption: "Análise final da distribuição gerada.",
		paths: ["/allocation-result"],
	},
];

export const PAGE_META: Record<string, PageMeta> = {
	"/": {
		path: "/",
		step: "import",
		kicker: "Operação Inicial",
		title: "Candidate Allocator",
		description:
			"Importe a planilha e conduza um fluxo de alocação claro, rastreável e preparado para uso corporativo.",
		panelTitle: "Entrada controlada",
		panelSummary:
			"Esta etapa valida a origem do arquivo e inicia a leitura das três entidades do processo seletivo.",
		helpSections: [
			{
				title: "O que acontece aqui",
				body: "A planilha enviada alimenta candidatos, restrições e avaliadores. Depois do upload, o sistema abre a etapa de mapeamento de candidatos.",
			},
			{
				title: "Como usar",
				body: "Selecione o Excel principal e avance. O bloco técnico de saudação continua disponível apenas como utilitário do ambiente Wails.",
			},
			{
				title: "Impacto",
				body: "Escolher o arquivo correto evita retrabalho nas etapas seguintes e garante que o restante do wizard reflita a base real do processo seletivo.",
			},
		],
	},
	"/mapping": {
		path: "/mapping",
		step: "candidate",
		kicker: "Configuração de Dados",
		title: "Mapeamento de Candidatos",
		description:
			"Associe cada coluna do Excel aos campos do domínio e organize extras sem comprometer a revisão.",
		panelTitle: "Estrutura base dos candidatos",
		panelSummary:
			"Os campos principais sustentam a reconstrução dos dados. Os extras preservam informação relevante para cada empresa.",
		helpSections: [
			{
				title: "Campos principais",
				body: "São os atributos centrais do domínio. Badges indicam obrigatoriedade e participação na identificação única do candidato.",
			},
			{
				title: "Campos extras",
				body: "Use extras para dados específicos do cliente. Se um extra manual ficar sem coluna, ele segue como null apenas quando essa regra estiver explicitamente ativada.",
			},
			{
				title: "Como revisar",
				body: "Arraste colunas disponíveis para os destinos corretos e avance apenas quando a estrutura refletir a planilha final.",
			},
		],
	},
	"/verify": {
		path: "/verify",
		step: "candidate",
		kicker: "Curadoria de Dados",
		title: "Verificação de Usuários",
		description:
			"Revise o resultado do mapeamento, corrija dados inline e resolva duplicidades antes do salvamento.",
		panelTitle: "Ponto de controle",
		panelSummary:
			"Conflitos são tratados aqui para impedir persistência ambígua. O objetivo é sair desta página com uma base consistente.",
		helpSections: [
			{
				title: "Duplicados",
				body: "Os grupos destacados exigem decisão. Você pode aceitar um registro específico, aceitar todos quando não houver conflito real ou recusar o grupo inteiro.",
			},
			{
				title: "Edição inline",
				body: "Clique em qualquer campo para ajustar o valor sem sair do fluxo. Campos extras também podem ser adicionados ou renomeados.",
			},
			{
				title: "Impacto",
				body: "Nada é salvo enquanto existirem duplicidades pendentes. Resolver corretamente essa etapa evita erros nas entidades seguintes.",
			},
		],
	},
	"/mappingRestricoes": {
		path: "/mappingRestricoes",
		step: "restriction",
		kicker: "Configuração de Dados",
		title: "Mapeamento de Restricoes",
		description:
			"Organize a aba de restrições com foco somente nos campos centrais que o domínio aceita.",
		panelTitle: "Regras importadas",
		panelSummary:
			"Restrições não aceitam extras. O objetivo aqui é garantir correspondência correta entre colunas da planilha e o schema da aplicação.",
		helpSections: [
			{
				title: "Escopo da tela",
				body: "Somente os campos previstos pelo sistema devem ser mapeados. Colunas excedentes continuam visíveis para referência, mas não seguem para a persistência.",
			},
			{
				title: "Como decidir",
				body: "Use as colunas disponíveis como reserva de remapeamento. O ideal é sair desta etapa com os campos críticos plenamente associados.",
			},
		],
	},
	"/verifyRestricoes": {
		path: "/verifyRestricoes",
		step: "restriction",
		kicker: "Curadoria de Dados",
		title: "Verificação de Restrições",
		description:
			"Faça a conferência final das regras carregadas antes de avançar para os avaliadores.",
		panelTitle: "Conferência operacional",
		panelSummary:
			"Mesmo sem grupos de duplicidade, esta página funciona como uma revisão de segurança para o conjunto de restrições salvo.",
		helpSections: [
			{
				title: "O que validar",
				body: "Confirme nomes, regras e valores trazidos da planilha. Pequenos ajustes aqui evitam efeitos indevidos no algoritmo de alocação.",
			},
		],
	},
	"/mappingAvaliadores": {
		path: "/mappingAvaliadores",
		step: "reviewer",
		kicker: "Configuração de Dados",
		title: "Mapeamento de Avaliadores",
		description:
			"Associe a aba de avaliadores ao modelo do sistema e preserve dados adicionais quando forem úteis à operação.",
		panelTitle: "Base de avaliadores",
		panelSummary:
			"Esta etapa define quem participa da distribuição final e quais dados ficam disponíveis para checagem e auditoria.",
		helpSections: [
			{
				title: "Extras permitidos",
				body: "Avaliadores aceitam campos extras. Isso facilita adaptações por empresa sem exigir mudanças no backend.",
			},
			{
				title: "Badges",
				body: "Os badges indicam o peso estrutural de cada campo. Dê prioridade máxima aos campos obrigatórios e únicos.",
			},
		],
	},
	"/verifyAvaliadores": {
		path: "/verifyAvaliadores",
		step: "reviewer",
		kicker: "Curadoria de Dados",
		title: "Verificação de Avaliadores",
		description:
			"Revise os avaliadores, ajuste o que for necessário e conclua a preparação da base.",
		panelTitle: "Última revisão da importação",
		panelSummary:
			"Depois desta página a base importada estará consolidada e pronta para a etapa de configuração operacional.",
		helpSections: [
			{
				title: "Como finalizar",
				body: "Resolva duplicados, ajuste campos sensíveis e só então salve. Esta é a última trava antes da configuração da alocação.",
			},
		],
	},
	"/success": {
		path: "/success",
		step: "checkpoint",
		kicker: "Checkpoint",
		title: "Tudo Pronto!",
		description:
			"A base foi preparada com sucesso. A próxima etapa transforma essa base em uma estratégia de alocação configurável.",
		panelTitle: "Pronto para operar",
		panelSummary:
			"O sistema concluiu a importação e o salvamento das entidades. Agora o foco passa a ser decidir parâmetros e critérios da distribuição.",
		helpSections: [
			{
				title: "Próximo passo",
				body: "Siga para a configuração da alocação para definir capacidade dos grupos, avaliadores e regras desejáveis.",
			},
		],
	},
	"/allocation-config": {
		path: "/allocation-config",
		step: "config",
		kicker: "Orquestração",
		title: "Configurações de Alocação",
		description:
			"Defina capacidade, composição das mesas e critérios desejáveis com explicações claras de impacto.",
		panelTitle: "Decisões da distribuição",
		panelSummary:
			"Os parâmetros aqui moldam a montagem dos grupos e influenciam diretamente o espaço de soluções explorado pelo algoritmo.",
		helpSections: [
			{
				title: "Parâmetros base",
				body: "Quantidade de grupos, tamanho mínimo e máximo e avaliadores por grupo delimitam a capacidade operacional da rodada.",
			},
			{
				title: "Critérios adicionais",
				body: "Essas regras não invalidam uma solução, mas orientam o algoritmo para composições mais desejáveis dentro do contexto do RH.",
			},
			{
				title: "Impacto",
				body: "Valores mais restritivos reduzem o número de combinações possíveis. Valores mais flexíveis aumentam espaço de exploração, mas podem gerar resultados menos controlados.",
			},
		],
	},
	"/allocation-loading": {
		path: "/allocation-loading",
		step: "run",
		kicker: "Execução",
		title: "Processando Alocação",
		description:
			"O sistema está calculando a distribuição com base na configuração definida.",
		panelTitle: "Cálculo em andamento",
		panelSummary:
			"Esta etapa estima o espaço combinatório e executa a alocação. Não é necessário nenhuma ação manual enquanto o processamento acontece.",
		helpSections: [
			{
				title: "O que está acontecendo",
				body: "O algoritmo avalia as possibilidades válidas a partir da capacidade definida e então gera a proposta final de mesas e não alocados.",
			},
		],
	},
	"/allocation-result": {
		path: "/allocation-result",
		step: "result",
		kicker: "Leitura Executiva",
		title: "Resultados da Alocação",
		description:
			"Analise a distribuição formada, destaque perfis específicos e identifique rapidamente quem ficou fora.",
		panelTitle: "Painel final",
		panelSummary:
			"O resultado organiza as mesas por horário e concentra filtros, status e exceções em uma leitura executiva única.",
		helpSections: [
			{
				title: "Filtros",
				body: "Os destaques por semestre, curso e busca livre ajudam a revisar diversidade, equilíbrio e exceções sem alterar os dados originais.",
			},
			{
				title: "Não alocados",
				body: "Esse painel mostra quem ficou fora da distribuição final, facilitando análise de capacidade e ajustes futuros de configuração.",
			},
		],
	},
};

export function getPageMeta(pathname: string): PageMeta {
	return PAGE_META[pathname] ?? PAGE_META["/"];
}

export function getCurrentStepIndex(pathname: string): number {
	return WORKFLOW_STEPS.findIndex((step) => step.paths.includes(pathname));
}
