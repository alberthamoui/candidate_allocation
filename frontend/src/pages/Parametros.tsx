import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import {
	AdjustmentsHorizontalIcon,
	ArrowLeftIcon,
	ExclamationTriangleIcon,
	CheckCircleIcon,
} from "@heroicons/react/24/outline";
import {
	getCapacidade,
	getParametrosSalvos,
	salvarParametros,
	CapacidadeResponse,
	CriterioAlocacao,
	ParametrosAlocacao,
} from "../api";
import EditorCriterios from "../components/EditorCriterios";

type Campo = Exclude<keyof ParametrosAlocacao, "criterios">; // campos numéricos

const CAMPOS: { campo: Campo; label: string; dica: string }[] = [
	{ campo: "mesas_por_horario", label: "Mesas por horário", dica: "Máximo de mesas em cada horário" },
	{ campo: "avaliadores_por_mesa", label: "Avaliadores por mesa", dica: "Cada avaliador fica em uma mesa por horário" },
	{ campo: "min_pessoas_por_mesa", label: "Mínimo de candidatos por mesa", dica: "Mesa com menos que isso não é formada" },
	{ campo: "max_pessoas_por_mesa", label: "Máximo de candidatos por mesa", dica: "" },
];

// Texto digitado → parâmetros; null se algum campo estiver vazio ou não for inteiro.
function paraParametros(valores: Record<Campo, string>, criterios: CriterioAlocacao[]): ParametrosAlocacao | null {
	const p = { criterios } as ParametrosAlocacao;
	for (const { campo } of CAMPOS) {
		const n = Number(valores[campo]);
		if (valores[campo].trim() === "" || !Number.isInteger(n)) return null;
		p[campo] = n;
	}
	return p;
}

function paraTexto(p: ParametrosAlocacao): Record<Campo, string> {
	return Object.fromEntries(CAMPOS.map(({ campo }) => [campo, String(p[campo])])) as Record<Campo, string>;
}

export default function Parametros() {
	const navigate = useNavigate();
	const [valores, setValores] = useState<Record<Campo, string> | null>(null);
	const [criterios, setCriterios] = useState<CriterioAlocacao[]>([]);
	const [padrao, setPadrao] = useState<ParametrosAlocacao | null>(null);
	const [previa, setPrevia] = useState<CapacidadeResponse | null>(null);
	const [erro, setErro] = useState<string | null>(null);
	const [carregando, setCarregando] = useState(true);
	const requisicao = useRef(0);

	// Primeira carga: padrões do backend e, se houver, os últimos parâmetros usados
	useEffect(() => {
		(async () => {
			try {
				const inicial = await getCapacidade(null);
				setPadrao(inicial.parametros);
				const salvos = getParametrosSalvos();
				setValores(paraTexto(salvos ?? inicial.parametros));
				setCriterios(salvos?.criterios ?? []);
				if (!salvos) setPrevia(inicial);
			} catch (err: any) {
				setErro(err.message);
				setCarregando(false);
			}
		})();
	}, []);

	// Recalcula a prévia a cada mudança (com um pequeno atraso enquanto digita)
	useEffect(() => {
		if (!valores) return;
		const params = paraParametros(valores, criterios);
		if (!params) {
			setErro("Preencha todos os campos com números inteiros.");
			return;
		}
		const id = ++requisicao.current;
		setCarregando(true);
		const t = setTimeout(async () => {
			try {
				const r = await getCapacidade(params);
				if (id !== requisicao.current) return;
				setPrevia(r);
				setErro(null);
			} catch (err: any) {
				if (id !== requisicao.current) return;
				setErro(err.message);
			} finally {
				if (id === requisicao.current) setCarregando(false);
			}
		}, 250);
		return () => clearTimeout(t);
	}, [valores, criterios]);

	const params = valores ? paraParametros(valores, criterios) : null;
	const podeRodar = !!params && !erro && !carregando && !!previa && previa.mesas_por_horario > 0;

	function rodar() {
		if (!params || !podeRodar) return;
		salvarParametros(params);
		navigate("/resultado");
	}

	const todosCabem = previa && previa.max_alocaveis >= previa.candidatos;

	return (
		<div className="min-h-screen bg-gradient-to-br from-blue-50 to-green-100 flex flex-col items-center py-10 px-4">
			<div className="bg-white shadow-xl rounded-2xl p-8 w-full max-w-3xl space-y-8">
				{/* Step indicator */}
				<div className="flex items-center justify-center space-x-2 text-sm text-gray-500 flex-wrap gap-y-2">
					{[1, 2, 3].map((n) => (
						<span key={n} className="flex items-center space-x-2">
							<span className="bg-green-100 text-green-700 font-semibold px-3 py-1 rounded-full">
								Passo {n} ✓
							</span>
							<span className="text-gray-300">→</span>
						</span>
					))}
					<span className="bg-blue-600 text-white font-semibold px-3 py-1 rounded-full">Passo 4</span>
				</div>

				<div className="text-center space-y-2">
					<AdjustmentsHorizontalIcon className="w-14 h-14 text-blue-500 mx-auto" />
					<h1 className="text-3xl font-bold text-blue-700">Parâmetros da Alocação</h1>
					<p className="text-gray-500 text-sm">
						Defina como as mesas são formadas. A prévia abaixo mostra o que cabe com esses valores.
					</p>
				</div>

				{/* Campos */}
				{valores && (
					<div className="space-y-3">
						<div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
							{CAMPOS.map(({ campo, label, dica }) => (
								<label key={campo} className="block">
									<span className="text-sm font-semibold text-gray-700">{label}</span>
									<input
										type="number"
										min={1}
										value={valores[campo]}
										onChange={(e) => setValores({ ...valores, [campo]: e.target.value })}
										className="mt-1 w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400"
									/>
									{dica && <span className="text-xs text-gray-400">{dica}</span>}
								</label>
							))}
						</div>
						{padrao && CAMPOS.some(({ campo }) => valores[campo] !== String(padrao[campo])) && (
							<button
								onClick={() => setValores(paraTexto(padrao))}
								className="text-sm text-blue-600 hover:underline"
							>
								Restaurar valores padrão
							</button>
						)}
					</div>
				)}

				{valores && (
					<EditorCriterios
						criterios={criterios}
						onChange={setCriterios}
						valoresColunas={previa?.valores_colunas ?? {}}
					/>
				)}

				{erro && (
					<div className="flex items-start space-x-2 bg-red-50 border border-red-200 rounded-lg p-3">
						<ExclamationTriangleIcon className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
						<p className="text-sm text-red-700">{erro}</p>
					</div>
				)}

				{/* Prévia */}
				{previa && (
					<div className={`space-y-4 transition-opacity ${carregando || erro ? "opacity-50" : ""}`}>
						<div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
							<Numero
								valor={previa.mesas_por_horario}
								label="mesas por horário"
								detalhe={
									previa.mesas_por_horario < previa.parametros.mesas_por_horario
										? `de ${previa.parametros.mesas_por_horario} pedidas`
										: undefined
								}
								alerta={previa.mesas_por_horario < previa.parametros.mesas_por_horario}
							/>
							<Numero valor={previa.capacidade_total} label="vagas no total" detalhe={`${previa.capacidade_por_horario} por horário`} />
							<Numero valor={previa.candidatos} label="candidatos" detalhe={`${previa.avaliadores} avaliadores`} />
							<Numero
								valor={previa.max_alocaveis}
								label="cabem no máximo"
								detalhe="pelos horários escolhidos"
								alerta={!todosCabem}
								ok={!!todosCabem}
							/>
						</div>

						{previa.avisos.length > 0 ? (
							<ul className="space-y-2">
								{previa.avisos.map((a, i) => (
									<li key={i} className="flex items-start space-x-2 bg-amber-50 border border-amber-200 rounded-lg p-3 text-sm text-amber-800">
										<ExclamationTriangleIcon className="w-5 h-5 flex-shrink-0" />
										<span>{a}</span>
									</li>
								))}
							</ul>
						) : (
							<div className="flex items-center space-x-2 bg-green-50 border border-green-200 rounded-lg p-3 text-sm text-green-800">
								<CheckCircleIcon className="w-5 h-5 flex-shrink-0" />
								<span>Há mesas e vagas para todos os candidatos nos horários que escolheram.</span>
							</div>
						)}

						<div className="overflow-x-auto border border-gray-200 rounded-xl">
							<table className="w-full text-sm">
								<thead className="bg-gray-50 text-gray-600">
									<tr>
										<th className="text-left px-4 py-2 font-semibold">Horário</th>
										<th className="text-right px-4 py-2 font-semibold">1ª opção</th>
										<th className="text-right px-4 py-2 font-semibold">Escolheram (qualquer opção)</th>
										<th className="text-right px-4 py-2 font-semibold">Vagas</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-100">
									{previa.horarios.map((h) => (
										<tr key={h.descricao}>
											<td className="px-4 py-2 capitalize text-gray-800">{h.descricao}</td>
											<td
												className={`px-4 py-2 text-right ${h.primeira_opcao > previa.capacidade_por_horario ? "text-amber-700 font-semibold" : "text-gray-700"}`}
												title={h.primeira_opcao > previa.capacidade_por_horario ? "Mais gente quer este horário como 1ª opção do que há vagas" : undefined}
											>
												{h.primeira_opcao}
											</td>
											<td className="px-4 py-2 text-right text-gray-700">{h.interessados}</td>
											<td className="px-4 py-2 text-right text-gray-700">{previa.capacidade_por_horario}</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>
					</div>
				)}

				{/* Ações */}
				<div className="flex flex-col-reverse sm:flex-row gap-3">
					<button
						onClick={() => navigate(-1)}
						className="flex items-center justify-center space-x-2 sm:w-40 py-3 rounded-xl border border-gray-300 text-gray-600 hover:bg-gray-50 font-medium transition-all text-sm"
					>
						<ArrowLeftIcon className="w-4 h-4" />
						<span>Voltar</span>
					</button>
					<motion.button
						onClick={rodar}
						disabled={!podeRodar}
						whileHover={{ scale: podeRodar ? 1.02 : 1 }}
						whileTap={{ scale: podeRodar ? 0.98 : 1 }}
						className={`flex-1 py-3 rounded-xl font-semibold text-white shadow-lg transition-all ${
							podeRodar ? "bg-blue-600 hover:bg-blue-700" : "bg-blue-300 cursor-not-allowed"
						}`}
					>
						Rodar alocação
					</motion.button>
				</div>
			</div>
		</div>
	);
}

function Numero({
	valor,
	label,
	detalhe,
	alerta,
	ok,
}: {
	valor: number;
	label: string;
	detalhe?: string;
	alerta?: boolean;
	ok?: boolean;
}) {
	const cor = alerta
		? "border-amber-200 bg-amber-50 text-amber-800"
		: ok
			? "border-green-200 bg-green-50 text-green-800"
			: "border-gray-200 bg-gray-50 text-gray-800";
	return (
		<div className={`border rounded-xl p-3 ${cor}`}>
			<div className="text-2xl font-bold">{valor}</div>
			<div className="text-xs font-semibold">{label}</div>
			{detalhe && <div className="text-xs opacity-75">{detalhe}</div>}
		</div>
	);
}
