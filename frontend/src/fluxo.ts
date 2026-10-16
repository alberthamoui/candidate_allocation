// Etapas do fluxo e textos de cada tela (título no topo e ajuda do botão "?").
// Inspirado no workflowMeta da branch mainjoao.

export interface Etapa {
	chave: string;
	rotulo: string;
	rotas: string[];
}

export const ETAPAS: Etapa[] = [
	{ chave: "importacao", rotulo: "Importação", rotas: ["/"] },
	{ chave: "candidatos", rotulo: "Candidatos", rotas: ["/mapping", "/verify"] },
	{ chave: "avaliadores", rotulo: "Avaliadores", rotas: ["/upload-avaliador", "/mapping-avaliador"] },
	{ chave: "restricoes", rotulo: "Restrições", rotas: ["/upload-restricao", "/mapping-restricao"] },
	{ chave: "parametros", rotulo: "Parâmetros", rotas: ["/parametros"] },
	{ chave: "resultado", rotulo: "Resultado", rotas: ["/resultado"] },
];

export interface TopicoAjuda {
	titulo: string;
	texto: string;
}

export interface InfoPagina {
	titulo: string;
	ajuda: { resumo: string; topicos: TopicoAjuda[] };
}

const PAGINAS: Record<string, InfoPagina> = {
	"/": {
		titulo: "Importação da planilha",
		ajuda: {
			resumo: "Envie o arquivo .xlsx com três abas, nesta ordem: candidatos, avaliadores e restrições.",
			topicos: [
				{ titulo: "Opções de horário", texto: "Quantas colunas de horário (1ª opção, 2ª opção...) cada candidato preencheu. A 1ª opção tem prioridade na alocação." },
				{ titulo: "Domínio do email", texto: "Usado para validar o email institucional dos candidatos na revisão." },
				{ titulo: "Planilha de exemplo", texto: "Baixe o exemplo para ver o formato esperado das três abas." },
			],
		},
	},
	"/mapping": {
		titulo: "Mapeamento de candidatos",
		ajuda: {
			resumo: "Ligue cada campo do sistema a uma coluna da planilha. A sugestão é feita pelo nome das colunas.",
			topicos: [
				{ titulo: "Como ajustar", texto: "Arraste a coluna de um campo para outro: as duas trocam de lugar. Um campo sem par pelo nome fica \"sem coluna\"." },
				{ titulo: "Ao confirmar", texto: "Os candidatos são lidos com esse mapeamento e validados (CPF, telefone, semestre, emails) na tela seguinte." },
			],
		},
	},
	"/verify": {
		titulo: "Revisão de candidatos",
		ajuda: {
			resumo: "Confira os candidatos lidos da planilha, corrija dados e resolva duplicidades antes de salvar.",
			topicos: [
				{ titulo: "Duplicados", texto: "Candidatos com o mesmo CPF ou email aparecem agrupados. Aceite um, aceite todos (se não forem a mesma pessoa) ou recuse o grupo." },
				{ titulo: "Edição", texto: "Clique em qualquer campo para corrigir o valor. Campos com erro ficam em vermelho, mas não impedem o salvamento." },
				{ titulo: "Ao salvar", texto: "Nada é salvo enquanto houver duplicidades pendentes. Se algum candidato não puder ser gravado, nenhum é e o erro aparece aqui." },
			],
		},
	},
	"/upload-avaliador": {
		titulo: "Avaliadores",
		ajuda: {
			resumo: "Os avaliadores são lidos da segunda aba da mesma planilha.",
			topicos: [{ titulo: "Próximo passo", texto: "Processe a aba para conferir o mapeamento das colunas de nome, email e sigla." }],
		},
	},
	"/mapping-avaliador": {
		titulo: "Mapeamento de avaliadores",
		ajuda: {
			resumo: "Ligue nome, email e sigla às colunas da aba de avaliadores.",
			topicos: [
				{ titulo: "Sigla", texto: "É pela sigla que as restrições identificam o avaliador. Ela precisa ser única." },
				{ titulo: "Ao confirmar", texto: "Os avaliadores são salvos. Nome ou email repetidos impedem o salvamento, com aviso na tela." },
			],
		},
	},
	"/upload-restricao": {
		titulo: "Restrições",
		ajuda: {
			resumo: "As restrições são lidas da terceira aba da mesma planilha.",
			topicos: [{ titulo: "Próximo passo", texto: "Processe a aba para conferir o mapeamento das colunas." }],
		},
	},
	"/mapping-restricao": {
		titulo: "Mapeamento de restrições",
		ajuda: {
			resumo: "Cada linha diz quais avaliadores não podem (ou preferem não) avaliar um candidato.",
			topicos: [
				{ titulo: "Não posso", texto: "Siglas separadas por vírgula ou espaço. Esses avaliadores nunca ficam na mesa do candidato." },
				{ titulo: "Prefiro não", texto: "Evitado sempre que possível; cada caso que acontecer custa pontos na alocação." },
				{ titulo: "Dados desconhecidos", texto: "Candidatos ou siglas que não existem são ignorados." },
			],
		},
	},
	"/parametros": {
		titulo: "Parâmetros da alocação",
		ajuda: {
			resumo: "Defina como as mesas são formadas. A prévia mostra o que cabe antes de rodar.",
			topicos: [
				{ titulo: "Mesas e avaliadores", texto: "Cada avaliador fica em no máximo uma mesa por horário, então o número de mesas por horário também é limitado pela quantidade de avaliadores." },
				{ titulo: "Critérios adicionais", texto: "Regras opcionais sobre curso ou semestre. Pesam junto com as preferências de horário: com importância alta, alguém pode ir para uma opção de horário pior para cumprir a regra." },
				{ titulo: "Prévia", texto: "Mostra vagas, quantos candidatos cabem pelos horários que escolheram e avisos. Não roda a alocação." },
			],
		},
	},
	"/resultado": {
		titulo: "Resultado da alocação",
		ajuda: {
			resumo: "As mesas formadas, agrupadas por horário, e quem ficou sem mesa.",
			topicos: [
				{ titulo: "Qualidade", texto: "Clique num item (ex.: \"Na 2ª opção\") para destacar os candidatos dele nas mesas." },
				{ titulo: "Busca e filtros", texto: "Encontre candidatos por nome, email ou curso, avaliadores por nome ou sigla, e filtre por horário, curso e semestre." },
				{ titulo: "Detalhes", texto: "Clique numa pessoa para ver os horários que escolheu, restrições e, para avaliadores, as mesas em que está." },
				{ titulo: "Pontuação", texto: "Começa em 100 e perde pontos por opção de horário pior, avaliador \"prefiro não\", critérios não atendidos e candidato sem mesa." },
			],
		},
	},
};

export function infoPagina(rota: string): InfoPagina {
	return PAGINAS[rota] ?? PAGINAS["/"];
}

export function indiceEtapa(rota: string): number {
	return ETAPAS.findIndex((e) => e.rotas.includes(rota));
}
