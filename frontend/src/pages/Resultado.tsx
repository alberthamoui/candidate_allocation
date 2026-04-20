import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { motion, AnimatePresence } from "framer-motion";
import {
	CheckCircleIcon,
	ExclamationTriangleIcon,
	UserGroupIcon,
	TableCellsIcon,
	ArrowDownTrayIcon,
	ArrowPathIcon,
	ArrowLeftIcon,
	XMarkIcon,
} from "@heroicons/react/24/outline";
import { startAlocacao, downloadExcel, resetSession, AlocacaoResponse } from "../api";

function formatEta(ms: number): string {
	if (ms <= 0) return "";
	const s = Math.ceil(ms / 1000);
	if (s < 60) return `~${s}s restantes`;
	const m = Math.floor(s / 60);
	const rem = s % 60;
	return rem > 0 ? `~${m}min ${rem}s restantes` : `~${m}min restantes`;
}

interface ResultadoProps {
	setAlocacaoResult: (data: AlocacaoResponse | null) => void;
}

export default function Resultado({ setAlocacaoResult }: ResultadoProps) {
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
		const es = startAlocacao(
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
				setAlocacaoResult(result);
				setTimeout(() => setLoading(false), 400);
			},
			(msg) => {
				setError("Erro ao executar alocação: " + msg);
				setLoading(false);
			}
		);
		return () => { es.close(); };
	}, []);

	function handleExport() {
		setExporting(true);
		try {
			downloadExcel();
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
			setAlocacaoResult(null);
			navigate("/");
		} catch (err: any) {
			setError("Erro ao reiniciar: " + err.message);
			setResetting(false);
		}
	}

	// ── Loading ──────────────────────────────────────────────────────────────
	if (loading) {
		return (
			<div className="min-h-screen bg-gradient-to-br from-blue-50 to-green-100 flex items-center justify-center">
				<div className="bg-white shadow-xl rounded-2xl p-10 w-full max-w-md space-y-6">
					<div className="text-center space-y-1">
						<h2 className="text-xl font-bold text-gray-800">Executando Alocação</h2>
						<p className="text-sm text-gray-500">{progressStep}</p>
					</div>

					{/* Barra de progresso */}
					<div className="w-full bg-gray-100 rounded-full h-4 overflow-hidden">
						<motion.div
							className="h-4 rounded-full bg-gradient-to-r from-blue-500 to-green-500"
							initial={{ width: "0%" }}
							animate={{ width: `${progressPct}%` }}
							transition={{ duration: 0.3, ease: "easeOut" }}
						/>
					</div>

					<div className="text-sm text-gray-500 text-center">
						{progressPct}%
						{eta && (
							<span className="ml-2 text-gray-400">&mdash; {eta}</span>
						)}
					</div>
				</div>
			</div>
		);
	}

	// ── Error ────────────────────────────────────────────────────────────────
	if (error && !result) {
		return (
			<div className="min-h-screen bg-gradient-to-br from-blue-50 to-green-100 flex items-center justify-center">
				<div className="bg-white shadow-xl rounded-2xl p-10 max-w-lg w-full space-y-4">
					<div className="flex items-center space-x-3">
						<ExclamationTriangleIcon className="w-8 h-8 text-red-500 flex-shrink-0" />
						<h1 className="text-xl font-bold text-red-700">Falha na Alocação</h1>
					</div>
					<p className="text-gray-700 text-sm bg-red-50 border border-red-200 rounded-lg p-4">{error}</p>
				</div>
			</div>
		);
	}

	if (!result) return null;

	const mesas = result.mesas ?? [];
	const naoAlocados = result.nao_alocados_info ?? [];

	return (
		<div className="min-h-screen bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50">

			{/* ── Confirm-reset modal ──────────────────────────────────────── */}
			<AnimatePresence>
				{confirmReset && (
					<div className="fixed inset-0 flex items-center justify-center bg-black/50 z-50">
						<motion.div
							initial={{ scale: 0.9, opacity: 0 }}
							animate={{ scale: 1, opacity: 1 }}
							exit={{ scale: 0.9, opacity: 0 }}
							className="bg-white rounded-2xl shadow-2xl p-8 max-w-sm mx-4 space-y-5"
						>
							<div className="flex items-center space-x-3">
								<ExclamationTriangleIcon className="w-6 h-6 text-orange-500" />
								<h3 className="font-bold text-gray-900 text-lg">Reiniciar aplicação?</h3>
							</div>
							<p className="text-gray-600 text-sm">
								Todos os dados (candidatos, avaliadores, restrições) serão apagados do banco de dados. Esta ação não pode ser desfeita.
							</p>
							<div className="flex space-x-3">
								<button
									onClick={() => setConfirmReset(false)}
									className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-xl hover:bg-gray-50 font-semibold transition-colors"
								>
									Cancelar
								</button>
								<button
									onClick={handleReset}
									disabled={resetting}
									className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-xl font-semibold transition-colors disabled:opacity-60"
								>
									{resetting ? "Reiniciando..." : "Confirmar"}
								</button>
							</div>
						</motion.div>
					</div>
				)}
			</AnimatePresence>

			{/* ── Inline error toast ───────────────────────────────────────── */}
			<AnimatePresence>
				{error && result && (
					<motion.div
						initial={{ opacity: 0, y: -20 }}
						animate={{ opacity: 1, y: 0 }}
						exit={{ opacity: 0, y: -20 }}
						className="fixed top-4 left-1/2 -translate-x-1/2 z-40 bg-red-600 text-white px-6 py-3 rounded-xl shadow-xl flex items-center space-x-3"
					>
						<ExclamationTriangleIcon className="w-5 h-5 flex-shrink-0" />
						<span className="text-sm font-medium">{error}</span>
						<button onClick={() => setError(null)}>
							<XMarkIcon className="w-4 h-4 ml-2" />
						</button>
					</motion.div>
				)}
			</AnimatePresence>

			{/* ── Export success toast ─────────────────────────────────────── */}
			<AnimatePresence>
				{exportDone && (
					<motion.div
						initial={{ opacity: 0, y: -20 }}
						animate={{ opacity: 1, y: 0 }}
						exit={{ opacity: 0, y: -20 }}
						className="fixed top-4 left-1/2 -translate-x-1/2 z-40 bg-green-600 text-white px-6 py-3 rounded-xl shadow-xl flex items-center space-x-3"
					>
						<CheckCircleIcon className="w-5 h-5" />
						<span className="text-sm font-medium">Arquivo exportado com sucesso!</span>
					</motion.div>
				)}
			</AnimatePresence>

			{/* ── Header ──────────────────────────────────────────────────── */}
			<div className="bg-white shadow-sm border-b">
				<div className="max-w-7xl mx-auto px-6 py-5">
					<div className="flex items-center justify-between flex-wrap gap-4">
						<div>
							<h1 className="text-3xl font-bold text-gray-900">Resultado da Alocação</h1>
							<p className="text-gray-500 mt-1">
								{mesas.length} mesa{mesas.length !== 1 ? "s" : ""} preenchida{mesas.length !== 1 ? "s" : ""}
							</p>
						</div>

						{/* Summary badges */}
						<div className="flex items-center gap-3 text-sm flex-wrap">
							<div className="flex items-center space-x-2 bg-green-100 text-green-800 px-4 py-2 rounded-lg font-semibold">
								<CheckCircleIcon className="w-4 h-4" />
								<span>{result.total_alocados} alocados</span>
							</div>
							{naoAlocados.length > 0 && (
								<div className="flex items-center space-x-2 bg-red-100 text-red-800 px-4 py-2 rounded-lg font-semibold">
									<ExclamationTriangleIcon className="w-4 h-4" />
									<span>{naoAlocados.length} não alocados</span>
								</div>
							)}

						</div>

						{/* Action buttons */}
						<div className="flex items-center gap-3 flex-wrap">
							<button
								onClick={() => navigate(-1)}
								className="flex items-center space-x-2 px-5 py-2.5 rounded-xl font-semibold text-sm border border-gray-300 text-gray-600 hover:bg-gray-100 transition-all"
							>
								<ArrowLeftIcon className="w-4 h-4" />
								<span>Voltar</span>
							</button>

							<motion.button
								onClick={handleExport}
								disabled={exporting}
								whileHover={{ scale: exporting ? 1 : 1.03 }}
								whileTap={{ scale: exporting ? 1 : 0.97 }}
								className={`flex items-center space-x-2 px-5 py-2.5 rounded-xl font-semibold text-sm shadow transition-all ${
									exporting
										? "bg-gray-300 text-gray-500 cursor-not-allowed"
										: "bg-green-600 hover:bg-green-700 text-white"
								}`}
							>
								<ArrowDownTrayIcon className="w-4 h-4" />
								<span>{exporting ? "Exportando..." : "Exportar Excel"}</span>
							</motion.button>

							<motion.button
								onClick={() => setConfirmReset(true)}
								whileHover={{ scale: 1.03 }}
								whileTap={{ scale: 0.97 }}
								className="flex items-center space-x-2 px-5 py-2.5 rounded-xl font-semibold text-sm shadow bg-red-100 hover:bg-red-200 text-red-700 transition-all"
							>
								<ArrowPathIcon className="w-4 h-4" />
								<span>Reiniciar</span>
							</motion.button>
						</div>
					</div>
				</div>
			</div>

			<div className="max-w-7xl mx-auto px-6 py-8 space-y-8">

				{/* ── Mesa list ─────────────────────────────────────────────── */}
				{mesas.length === 0 ? (
					<div className="bg-white rounded-2xl shadow p-10 text-center text-gray-500">
						Nenhuma mesa foi preenchida. Verifique se os dados foram salvos corretamente.
					</div>
				) : (
					<div className="space-y-5">
						{mesas.map((mesa, idx) => (
							<motion.div
								key={mesa.id}
								initial={{ opacity: 0, y: 16 }}
								animate={{ opacity: 1, y: 0 }}
								transition={{ delay: idx * 0.04 }}
								className="bg-white rounded-2xl shadow-lg border border-gray-100 overflow-hidden"
							>
								<div className="bg-gradient-to-r from-blue-600 to-blue-700 text-white px-6 py-4 flex items-center justify-between">
									<div className="flex items-center space-x-3">
										<TableCellsIcon className="w-5 h-5" />
										<span className="font-bold text-lg capitalize">{mesa.descricao}</span>
									</div>
									<span className="bg-white/20 px-3 py-1 rounded-full text-sm font-semibold">
										{mesa.candidatos.length} candidato{mesa.candidatos.length !== 1 ? "s" : ""}
									</span>
								</div>

								<div className="p-6 grid grid-cols-1 md:grid-cols-2 gap-6">
									<div>
										<div className="flex items-center space-x-2 mb-3">
											<UserGroupIcon className="w-4 h-4 text-blue-500" />
											<span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Candidatos</span>
										</div>
										<ul className="space-y-1.5">
											{mesa.candidatos.map((nome, i) => (
												<li key={i} className="text-sm text-gray-700 flex items-center space-x-2">
													<span className="w-5 h-5 bg-blue-100 text-blue-700 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0">
														{i + 1}
													</span>
													<span>{nome}</span>
												</li>
											))}
										</ul>
									</div>

									<div>
										<div className="flex items-center space-x-2 mb-3">
											<CheckCircleIcon className="w-4 h-4 text-green-500" />
											<span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Avaliadores</span>
										</div>
										<ul className="space-y-1.5">
											{mesa.avaliadores.length === 0 ? (
												<li className="text-sm text-gray-400 italic">Nenhum avaliador atribuído</li>
											) : (
												mesa.avaliadores.map((nome, i) => (
													<li key={i} className="text-sm text-gray-700 flex items-center space-x-2">
														<span className="w-5 h-5 bg-green-100 text-green-700 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0">
															{i + 1}
														</span>
														<span>{nome}</span>
													</li>
												))
											)}
										</ul>
									</div>
								</div>
							</motion.div>
						))}
					</div>
				)}

				{/* ── Não alocados ──────────────────────────────────────────── */}
				{naoAlocados.length > 0 && (
					<motion.div
						initial={{ opacity: 0, y: 16 }}
						animate={{ opacity: 1, y: 0 }}
						className="bg-white rounded-2xl shadow-lg border border-red-200 overflow-hidden"
					>
						<div className="bg-gradient-to-r from-red-500 to-red-600 text-white px-6 py-4 flex items-center justify-between">
							<div className="flex items-center space-x-3">
								<ExclamationTriangleIcon className="w-5 h-5" />
								<span className="font-bold text-lg">Candidatos Não Alocados</span>
							</div>
							<span className="bg-white/20 px-3 py-1 rounded-full text-sm font-semibold">
								{naoAlocados.length} candidato{naoAlocados.length !== 1 ? "s" : ""}
							</span>
						</div>

						<div className="overflow-x-auto">
							<table className="w-full text-sm">
								<thead className="bg-red-50 text-red-800">
									<tr>
										<th className="text-left px-6 py-3 font-semibold">#</th>
										<th className="text-left px-6 py-3 font-semibold">Nome</th>
										<th className="text-left px-6 py-3 font-semibold">Email Institucional</th>
										<th className="text-left px-6 py-3 font-semibold">Curso</th>
										<th className="text-left px-6 py-3 font-semibold">Semestre</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-red-100">
									{naoAlocados.map((p, i) => (
										<tr key={p.id} className="hover:bg-red-50 transition-colors">
											<td className="px-6 py-3 text-gray-400 font-mono">{i + 1}</td>
											<td className="px-6 py-3 font-medium text-gray-900">{p.nome}</td>
											<td className="px-6 py-3 text-gray-600">{p.email_insper}</td>
											<td className="px-6 py-3 text-gray-600">{p.curso}</td>
											<td className="px-6 py-3 text-gray-600">{p.semestre}º</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>
					</motion.div>
				)}
			</div>
		</div>
	);
}
