import { ReactNode, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
	ArrowDownTrayIcon,
	ArrowLeftIcon,
	ArrowPathIcon,
	CheckCircleIcon,
	ExclamationTriangleIcon,
	XMarkIcon
} from "@heroicons/react/24/outline";
import {
	startAlocacao,
	downloadExcel,
	resetSession,
	getParametrosSalvos,
	getResultado,
	setSessionId,
	consumirNovaAlocacao,
	AlocacaoResponse
} from "../api";
import OperacoesAgenda from "../components/OperacoesAgenda";
import PainelResultado from "../components/PainelResultado";
import {
	Aviso,
	BarraAcoes,
	BotaoPerigo,
	BotaoPrimario,
	BotaoSecundario,
	CabecalhoPagina,
	Modal,
	Selo
} from "../components/ui";

function formatEta(ms: number): string {
	if (ms <= 0) return "";
	const s = Math.ceil(ms / 1000);
	if (s < 60) return `~${s}s restantes`;
	const m = Math.floor(s / 60);
	const rem = s % 60;
	return rem > 0 ? `~${m}min ${rem}s restantes` : `~${m}min restantes`;
}

export default function Resultado() {
	const navigate = useNavigate();
	const [result, setResult] = useState<AlocacaoResponse | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [exporting, setExporting] = useState(false);
	const [exportDone, setExportDone] = useState(false);
	const [confirmReset, setConfirmReset] = useState(false);
	const [resetting, setResetting] = useState(false);
	const [progressPct, setProgressPct] = useState(0);
	const [progressStep, setProgressStep] = useState("Iniciando...");
	const [eta, setEta] = useState<string>("");
	const progressRef = useRef(0);
	const startTimeRef = useRef<number | null>(null);

	useEffect(() => {
		const sid = new URLSearchParams(location.search).get("sessao");
		if (sid) setSessionId(sid);
		let es: EventSource | undefined;
		let cancelado = false;
		async function iniciar() {
			if (!consumirNovaAlocacao()) {
				try {
					const r = await getResultado();
					if (!cancelado) {
						setResult(r);
						setLoading(false);
					}
					return;
				} catch (e: any) {
					if (!e.message.includes("nenhuma alocação")) {
						if (!cancelado) {
							setError(e.message);
							setLoading(false);
						}
						return;
					}
				}
			}
			if (cancelado) return;
			es = startAlocacao(
				getParametrosSalvos(),
				(ev) => {
					if (ev.pct !== undefined && ev.pct > progressRef.current) {
						progressRef.current = ev.pct;
						setProgressPct(ev.pct);
					}
					if (ev.step) setProgressStep(ev.step);
					if (ev.tentativa && ev.total && ev.tentativa > 0) {
						if (startTimeRef.current === null) {
							startTimeRef.current = Date.now();
						} else {
							const elapsed = Date.now() - startTimeRef.current;
							const avgMs = elapsed / ev.tentativa;
							const remainingMs = avgMs * (ev.total - ev.tentativa);
							setEta(formatEta(remainingMs));
						}
					}
				},
				(result) => {
					setProgressPct(100);
					setProgressStep("Concluído!");
					setResult(result);
					setTimeout(() => setLoading(false), 400);
				},
				(msg) => {
					setError("Erro ao executar alocação: " + msg);
					setLoading(false);
				}
			);
		}
		void iniciar();
		return () => {
			cancelado = true;
			es?.close();
		};
	}, []);

	async function handleExport() {
		setExporting(true);
		try {
			await downloadExcel();
			setExportDone(true);
			setTimeout(() => setExportDone(false), 2500);
		} catch (err: any) {
			setError("Erro ao exportar: " + err.message);
		} finally {
			setExporting(false);
		}
	}

	async function handleReset() {
		setResetting(true);
		try {
			await resetSession();
			navigate("/");
		} catch (err: any) {
			setError("Erro ao reiniciar: " + err.message);
			setResetting(false);
		}
	}

	// ── Executando ───────────────────────────────────────────────────────────
	if (loading) {
		return (
			<div className="flex min-h-[60vh] items-center justify-center">
				<div className="painel w-full max-w-md space-y-5 p-8" role="status" aria-live="polite">
					<div className="space-y-1">
						<div className="rotulo-caps">Executando alocação</div>
						<p className="text-base font-medium">{progressStep}</p>
					</div>
					<div className="h-2 w-full overflow-hidden rounded-full bg-[var(--bg-hover)]">
						<div
							className="h-2 rounded-full bg-[var(--accent)] transition-[width] duration-300 ease-out"
							style={{ width: `${progressPct}%` }}
						/>
					</div>
					<div className="flex justify-between text-xs text-[var(--text-muted)] tabular-nums">
						<span>{progressPct}%</span>
						<span>{eta}</span>
					</div>
				</div>
			</div>
		);
	}

	// ── Falha ────────────────────────────────────────────────────────────────
	if (error && !result) {
		return (
			<div className="flex min-h-[60vh] items-center justify-center">
				<div className="painel w-full max-w-lg space-y-4 p-8">
					<h1 className="text-xl font-semibold">Falha na Alocação</h1>
					<Aviso tom="perigo" icone={<ExclamationTriangleIcon className="h-4 w-4" />}>
						{error}
					</Aviso>
					<div className="flex justify-end gap-2">
						<BotaoSecundario onClick={() => navigate(-1)}>
							<ArrowLeftIcon className="h-4 w-4" />
							Voltar
						</BotaoSecundario>
						<BotaoPrimario onClick={handleReset}>Recomeçar</BotaoPrimario>
					</div>
				</div>
			</div>
		);
	}

	if (!result) return null;

	const mesas = result.mesas ?? [];
	const naoAlocados = result.nao_alocados_info ?? [];

	return (
		<div className="space-y-6">
			{confirmReset && (
				<Modal
					titulo="Reiniciar aplicação?"
					onFechar={() => setConfirmReset(false)}
					acoes={
						<>
							<BotaoSecundario onClick={() => setConfirmReset(false)}>Cancelar</BotaoSecundario>
							<BotaoPerigo onClick={handleReset} disabled={resetting}>
								{resetting ? "Reiniciando..." : "Confirmar"}
							</BotaoPerigo>
						</>
					}
				>
					Todos os dados (candidatos, avaliadores, restrições) serão apagados. Esta ação não pode ser desfeita.
				</Modal>
			)}

			{error && (
				<AvisoFlutuante
					tom="perigo"
					icone={<ExclamationTriangleIcon className="h-4 w-4" />}
					onFechar={() => setError(null)}
				>
					{error}
				</AvisoFlutuante>
			)}
			{exportDone && (
				<AvisoFlutuante tom="sucesso" icone={<CheckCircleIcon className="h-4 w-4" />}>
					Arquivo exportado com sucesso!
				</AvisoFlutuante>
			)}

			<CabecalhoPagina
				titulo="Resultado da Alocação"
				descricao={`${mesas.length} mesa${mesas.length !== 1 ? "s" : ""} preenchida${mesas.length !== 1 ? "s" : ""}.`}
				acoes={
					<>
						{result.sem_preferencia && <Selo>Sem preferência de horário</Selo>}
						<Selo tom="sucesso" className="px-3 py-1 text-xs">
							<CheckCircleIcon className="h-3.5 w-3.5" />
							{result.total_alocados} alocados
						</Selo>
						{naoAlocados.length > 0 && (
							<Selo tom="perigo" className="px-3 py-1 text-xs">
								<ExclamationTriangleIcon className="h-3.5 w-3.5" />
								{naoAlocados.length} não alocados
							</Selo>
						)}
					</>
				}
			/>

			<OperacoesAgenda result={result} onResult={setResult}>
				<PainelResultado result={result} />
			</OperacoesAgenda>

			<BarraAcoes dica="Exporte as mesas para uma planilha Excel.">
				<BotaoSecundario onClick={() => navigate(-1)}>
					<ArrowLeftIcon className="h-4 w-4" />
					Voltar
				</BotaoSecundario>
				<BotaoPerigo onClick={() => setConfirmReset(true)}>
					<ArrowPathIcon className="h-4 w-4" />
					Reiniciar
				</BotaoPerigo>
				<BotaoPrimario onClick={handleExport} disabled={exporting}>
					<ArrowDownTrayIcon className="h-4 w-4" />
					{exporting ? "Exportando..." : "Exportar Excel"}
				</BotaoPrimario>
			</BarraAcoes>
		</div>
	);
}

// Aviso que flutua no topo da tela (erro depois do resultado, exportação concluída).
function AvisoFlutuante({
	tom,
	icone,
	children,
	onFechar
}: {
	tom: "perigo" | "sucesso";
	icone: ReactNode;
	children: ReactNode;
	onFechar?: () => void;
}) {
	return (
		<div
			className="fixed left-1/2 top-16 z-50 w-[min(520px,calc(100vw-2rem))] -translate-x-1/2 rounded-[var(--radius)] bg-white shadow-[var(--shadow-lg)]"
			role="status"
		>
			<Aviso tom={tom} icone={icone}>
				<div className="flex items-start justify-between gap-3">
					<span>{children}</span>
					{onFechar && (
						<button type="button" onClick={onFechar} aria-label="Fechar aviso" className="opacity-60 hover:opacity-100">
							<XMarkIcon className="h-4 w-4" />
						</button>
					)}
				</div>
			</Aviso>
		</div>
	);
}
