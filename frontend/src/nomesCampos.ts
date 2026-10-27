// Nomes legíveis dos campos do sistema, usados no mapeamento e na revisão.
const NOMES: Record<string, string> = {
	timestamp: "Data de envio",
	nome: "Nome",
	cpf: "CPF (opcional)",
	numero: "Matrícula / RA",
	telefone: "Telefone / WhatsApp (opcional)",
	semestre: "Semestre",
	curso: "Curso",
	email_insper: "Email institucional",
	email_pessoal: "Email pessoal",
	email: "Email",
	sigla: "Sigla",
	candidato: "Candidato",
	naoPosso: "Não posso",
	prefiroNao: "Prefiro não",
	opcoes: "Horários escolhidos",
};

export function nomeCampo(variavel: string): string {
	const opcao = variavel.match(/^opcao (\d+)$/);
	if (opcao) return `${opcao[1]}ª opção de horário`;
	return NOMES[variavel] ?? variavel;
}
