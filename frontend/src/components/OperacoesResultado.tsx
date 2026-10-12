import { ReactNode, useEffect, useState } from "react";
import {
	AlocacaoResponse,
	AvisoGerado,
	ConfigAvisos,
	avaliar,
	configAvisosVazia,
	enviarAvisos,
	getAvisos,
	realocar,
	salvarAvisos,
} from "../api";

type Aba = "avisos" | "imprevisto" | "pessoa" | "avaliacao";

const ABAS: { id: Aba; rotulo: string }[] = [
	{ id: "imprevisto", rotulo: "Imprevisto" },
	{ id: "pessoa", rotulo: "Uma pessoa" },
	{ id: "avaliacao", rotulo: "Avaliação" },
	{ id: "avisos", rotulo: "Avisos" },
];

export default function OperacoesResultado({
	result,
	onResultado,
}: {
	result: AlocacaoResponse;
	onResultado: (r: AlocacaoResponse) => void;
}) {
	const [aba, setAba] = useState<Aba>("imprevisto");
	const [cfg, setCfg] = useState<ConfigAvisos>(configAvisosVazia());
	const [de, setDe] = useState("");
	const [paraDia, setParaDia] = useState("");
	const [mesaId, setMesaId] = useState<number>(result.mesas[0]?.id ?? 0);
	const [paraMesa, setParaMesa] = useState("");
	const [pessoaId, setPessoaId] = useState<number>(0);
	const [horarioPessoa, setHorarioPessoa] = useState("");
	const [avisar, setAvisar] = useState(true);
	const [resumo, setResumo] = useState("");
	const [avisos, setAvisos] = useState<AvisoGerado[]>([]);
	const [erro, setErro] = useState("");
	const [ocupado, setOcupado] = useState(false);

	const mesas = result.mesas ?? [];
	const dias = Array.from(new Set(mesas.map((m) => m.dia_nome)));
	const pessoas = [
		...mesas.flatMap((m) => m.candidatos.map((c) => ({ ...c, onde: m.dia_nome }))),
		...(result.nao_alocados_info ?? []).map((c) => ({ ...c, onde: "sem mesa" })),
	];

	useEffect(() => {
		getAvisos().then(setCfg).catch(() => {});
	}, []);

	useEffect(() => {
		if ((!de || !dias.includes(de)) && dias[0]) setDe(dias[0]);
		if (!mesas.some((m) => m.id === mesaId) && mesas[0]) setMesaId(mesas[0].id);
		if (!pessoas.some((p) => p.id === pessoaId) && pessoas[0]) setPessoaId(pessoas[0].id);
	}, [result]);

	async function executar(acao: () => Promise<{ resumo: string; avisos: AvisoGerado[]; resultado?: AlocacaoResponse }>) {
		setOcupado(true);
		setErro("");
		setResumo("");
		setAvisos([]);
		try {
			const op = await acao();
			setResumo(op.resumo);
			setAvisos(op.avisos ?? []);
			if (op.resultado) onResultado(op.resultado);
		} catch (err: any) {
			setErro(err.message ?? String(err));
		} finally {
			setOcupado(false);
		}
	}

	function confirmarEnvio(frase: string) {
		return window.confirm(frase);
	}

	return (
		<section className="bg-white rounded-2xl shadow-lg border border-gray-100 p-6 space-y-4">
			<p className="text-xs text-gray-400">
				A lista de mesas e o Excel acompanham estas mudanças. A pontuação e o relatório de qualidade continuam os da alocação original.
			</p>
			<div className="flex flex-wrap gap-2">
				{ABAS.map((item) => (
					<button
						key={item.id}
						type="button"
						onClick={() => setAba(item.id)}
						className={`px-4 py-2 rounded-full text-sm font-semibold ${
							aba === item.id ? "bg-blue-600 text-white" : "bg-gray-100 text-gray-700 hover:bg-gray-200"
						}`}
					>
						{item.rotulo}
					</button>
				))}
			</div>

			{aba === "imprevisto" && (
				<div className="space-y-4">
					<p className="text-sm text-gray-600">
						Se o horário furou, mova o dia inteiro ou uma mesa. Com o aviso marcado, o e-mail e o WhatsApp saem para todo mundo afetado — só depois que você confirmar.
					</p>
					<div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
						<Rotulo texto="Dia atual">
							<select className={campo} value={de} onChange={(e) => setDe(e.target.value)}>
								{dias.map((d) => (
									<option key={d} value={d}>{d}</option>
								))}
							</select>
						</Rotulo>
						<Rotulo texto="Novo dia">
							<input className={campo} value={paraDia} onChange={(e) => setParaDia(e.target.value)} placeholder="sexta 14-16" />
						</Rotulo>
						<div className="flex items-end">
							<button
								type="button"
								disabled={ocupado}
								onClick={() => {
									if (!confirmarEnvio(avisar
										? `Mover todo mundo de "${de}" para "${paraDia}" e enviar e-mail e WhatsApp?`
										: `Mover todo mundo de "${de}" para "${paraDia}"?`)) return;
									executar(() => realocar({ tipo: "dia", de, para: paraDia, avisar }));
								}}
								className={botao}
							>
								Mover o dia
							</button>
						</div>
					</div>
					<div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
						<Rotulo texto="Mesa">
							<select className={campo} value={mesaId} onChange={(e) => setMesaId(Number(e.target.value))}>
								{mesas.map((m) => (
									<option key={m.id} value={m.id}>{m.descricao}</option>
								))}
							</select>
						</Rotulo>
						<Rotulo texto="Destino da mesa">
							<input className={campo} value={paraMesa} onChange={(e) => setParaMesa(e.target.value)} placeholder="quinta 10-12" />
						</Rotulo>
						<div className="flex items-end">
							<button
								type="button"
								disabled={ocupado}
								onClick={() => {
									if (!confirmarEnvio(avisar
										? `Mover esta mesa para "${paraMesa}" e avisar todo mundo dela?`
										: `Mover esta mesa para "${paraMesa}"?`)) return;
									executar(() => realocar({ tipo: "mesa", mesa_id: mesaId, para: paraMesa, avisar }));
								}}
								className={botao}
							>
								Mover a mesa
							</button>
						</div>
					</div>
					<label className="flex items-center gap-2 text-sm text-gray-700">
						<input type="checkbox" checked={avisar} onChange={(e) => setAvisar(e.target.checked)} />
						Avisar todo mundo por e-mail e WhatsApp
					</label>
				</div>
			)}

			{aba === "pessoa" && (
				<div className="space-y-3">
					<p className="text-sm text-gray-600">
						Quando alguém pede outro horário, escolha a pessoa e o horário novo. Se o horário ainda não existe, o sistema abre uma mesa nele.
					</p>
					<div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
						<Rotulo texto="Pessoa">
							<select className={campo} value={pessoaId} onChange={(e) => setPessoaId(Number(e.target.value))}>
								{pessoas.map((p) => (
									<option key={p.id} value={p.id}>{p.nome} — {p.onde}</option>
								))}
							</select>
						</Rotulo>
						<Rotulo texto="Novo horário">
							<input className={campo} value={horarioPessoa} onChange={(e) => setHorarioPessoa(e.target.value)} placeholder="sexta 8-10" list="horarios-conhecidos" />
							<datalist id="horarios-conhecidos">
								{dias.map((d) => <option key={d} value={d} />)}
							</datalist>
						</Rotulo>
						<div className="flex items-end">
							<button
								type="button"
								disabled={ocupado}
								onClick={() => {
									const nome = pessoas.find((p) => p.id === pessoaId)?.nome ?? "esta pessoa";
									if (avisar && !confirmarEnvio(`Mover ${nome} para "${horarioPessoa}" e enviar o aviso?`)) return;
									executar(() => realocar({ tipo: "pessoa", pessoa: String(pessoaId), horario: horarioPessoa, avisar }));
								}}
								className={botao}
							>
								Trocar horário
							</button>
						</div>
					</div>
					<label className="flex items-center gap-2 text-sm text-gray-700">
						<input type="checkbox" checked={avisar} onChange={(e) => setAvisar(e.target.checked)} />
						Avisar esta pessoa
					</label>
				</div>
			)}

			{aba === "avaliacao" && (
				<div className="space-y-3">
					<p className="text-sm text-gray-600">Marque quem passou e quem não passou. A avaliação entra no Excel.</p>
					<ul className="divide-y divide-gray-100 border border-gray-100 rounded-xl">
						{pessoas.map((p) => (
							<li key={p.id} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2">
								<span className="text-sm text-gray-800">
									{p.nome} <span className="text-gray-400">· {p.onde}</span>
								</span>
								<span className="flex gap-1">
									<BotaoNota ativo={p.avaliacao === "aprovado"} cor="green" disabled={ocupado} onClick={() => executar(() => avaliar([{ id: p.id, resultado: "aprovado" }]))}>
										Passou
									</BotaoNota>
									<BotaoNota ativo={p.avaliacao === "reprovado"} cor="red" disabled={ocupado} onClick={() => executar(() => avaliar([{ id: p.id, resultado: "reprovado" }]))}>
										Não passou
									</BotaoNota>
									{p.avaliacao && (
										<BotaoNota ativo={false} cor="gray" disabled={ocupado} onClick={() => executar(() => avaliar([{ id: p.id, resultado: "pendente" }]))}>
											Limpar
										</BotaoNota>
									)}
								</span>
							</li>
						))}
					</ul>
				</div>
			)}

			{aba === "avisos" && (
				<div className="space-y-3">
					<p className="text-sm text-gray-600">
						A mensagem vale para todo mundo. Use {"{{nome}}"}, {"{{data}}"}, {"{{horario}}"} e {"{{mesa}}"}.
					</p>
					<textarea
						value={cfg.mensagem}
						onChange={(e) => setCfg({ ...cfg, mensagem: e.target.value })}
						rows={3}
						className={campo}
					/>
					<label className="flex items-center gap-2 text-sm">
						<input type="checkbox" checked={cfg.ativo} onChange={(e) => setCfg({ ...cfg, ativo: e.target.checked })} />
						Avisos ligados
					</label>
					<div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
						<div className="space-y-2">
							<label className="flex items-center gap-2 text-sm font-medium">
								<input type="checkbox" checked={cfg.email.ativo} onChange={(e) => setCfg({ ...cfg, email: { ...cfg.email, ativo: e.target.checked } })} />
								E-mail
							</label>
							<input className={campo} placeholder="servidor SMTP" value={cfg.email.host} onChange={(e) => setCfg({ ...cfg, email: { ...cfg.email, host: e.target.value } })} />
							<input className={campo} placeholder="remetente" value={cfg.email.remetente} onChange={(e) => setCfg({ ...cfg, email: { ...cfg.email, remetente: e.target.value } })} />
							<input className={campo} type="password" placeholder="senha" value={cfg.email.senha} onChange={(e) => setCfg({ ...cfg, email: { ...cfg.email, senha: e.target.value } })} />
						</div>
						<div className="space-y-2">
							<label className="flex items-center gap-2 text-sm font-medium">
								<input type="checkbox" checked={cfg.whatsapp.ativo} onChange={(e) => setCfg({ ...cfg, whatsapp: { ...cfg.whatsapp, ativo: e.target.checked } })} />
								WhatsApp
							</label>
							<input className={campo} placeholder="URL do webhook" value={cfg.whatsapp.url} onChange={(e) => setCfg({ ...cfg, whatsapp: { ...cfg.whatsapp, url: e.target.value } })} />
							<input className={campo} placeholder="token (opcional)" value={cfg.whatsapp.token} onChange={(e) => setCfg({ ...cfg, whatsapp: { ...cfg.whatsapp, token: e.target.value } })} />
						</div>
					</div>
					<div className="flex flex-wrap gap-2">
						<button type="button" className={botao} disabled={ocupado} onClick={() => executar(async () => { await salvarAvisos(cfg); return { resumo: "Configuração de avisos salva.", avisos: [] }; })}>
							Salvar mensagem
						</button>
						<button
							type="button"
							className={botao}
							disabled={ocupado}
							onClick={() => {
								if (!confirmarEnvio("Enviar e-mail e WhatsApp para todos os candidatos alocados?")) return;
								executar(async () => {
									await salvarAvisos(cfg);
									return enviarAvisos();
								});
							}}
						>
							Enviar para todos
						</button>
					</div>
				</div>
			)}

			{erro && <p className="text-sm text-red-700 bg-red-50 border border-red-200 rounded-lg p-3">{erro}</p>}
			{resumo && <p className="text-sm text-green-800 bg-green-50 border border-green-200 rounded-lg p-3">{resumo}</p>}
			{avisos.length > 0 && (
				<ul className="text-sm space-y-2">
					{avisos.map((a, i) => (
						<li key={i} className="border border-gray-100 rounded-lg p-3">
							<div className="font-medium text-gray-800">
								{a.nome} · {a.canal} {a.destino && <span className="text-gray-500">{a.destino}</span>}
								<span className={a.enviado ? "text-green-700" : "text-amber-700"}>
									{" "}· {a.enviado ? "enviado" : a.erro || "preparado"}
								</span>
							</div>
							<p className="text-gray-600 mt-1">{a.mensagem}</p>
						</li>
					))}
				</ul>
			)}
		</section>
	);
}

const campo = "w-full border border-gray-300 rounded-lg px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-300";
const botao = "px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold disabled:opacity-50";

function Rotulo({ texto, children }: { texto: string; children: ReactNode }) {
	return (
		<label className="block text-xs font-semibold text-gray-500 uppercase tracking-wide">
			{texto}
			<div className="mt-1 normal-case tracking-normal font-normal">{children}</div>
		</label>
	);
}

function BotaoNota({
	ativo, cor, disabled, onClick, children,
}: {
	ativo: boolean;
	cor: "green" | "red" | "gray";
	disabled: boolean;
	onClick: () => void;
	children: ReactNode;
}) {
	const cores = {
		green: ativo ? "bg-green-600 text-white" : "bg-green-50 text-green-800 hover:bg-green-100",
		red: ativo ? "bg-red-600 text-white" : "bg-red-50 text-red-800 hover:bg-red-100",
		gray: "bg-gray-100 text-gray-600 hover:bg-gray-200",
	};
	return (
		<button type="button" disabled={disabled} onClick={onClick} className={`px-3 py-1 rounded-full text-xs font-semibold ${cores[cor]}`}>
			{children}
		</button>
	);
}
