import { useNavigate } from "react-router-dom";
import { useState, useEffect, useRef } from "react";
import { motion } from "framer-motion";
import {
	ArrowLeftIcon,
	ExclamationTriangleIcon,
	PlusIcon,
	XMarkIcon,
} from "@heroicons/react/24/outline";

interface MappingItem {
	nomeColuna: string;
	indice: number;
	variavel: string;
}
interface MappingPageProps {
	mapping: MappingItem[] | null;
	/** Function that receives the current mapping items and returns a promise with the result. */
	buildFn: (items: MappingItem[]) => Promise<any>;
	/** Called with the build result before navigation. May be async. */
	onSuccess: (result: any) => Promise<void> | void;
	/** Route to navigate to after onSuccess resolves. */
	nextRoute: string;
	/** Permite campos extras (colunas que não são campos fixos). */
	permitirExtras?: boolean;
}

// Campos extras vêm do backend com variavel = "extra:<nome>".
const PREFIXO_EXTRA = "extra:";
const ehExtra = (item: MappingItem) => item.variavel.startsWith(PREFIXO_EXTRA);
const nomeExtra = (item: MappingItem) => item.variavel.slice(PREFIXO_EXTRA.length);

// Mesma ideia da chaveExtra do backend: ignora caixa, acentos, espaços e pontuação.
const chaveNome = (nome: string) =>
	nome
		.normalize("NFD")
		.replace(/[̀-ͯ]/g, "")
		.toLowerCase()
		.replace(/[^a-z0-9]/g, "");

// Confere os nomes dos extras antes de enviar; devolve a mensagem de erro, se houver.
function validarExtras(items: MappingItem[]): string | null {
	const vistos = new Map<string, string>();
	for (const item of items.filter((i) => !ehExtra(i))) {
		vistos.set(chaveNome(item.variavel), item.variavel);
	}
	for (const item of items.filter(ehExtra)) {
		const nome = nomeExtra(item).trim();
		if (!chaveNome(nome)) return `O campo extra da coluna "${item.nomeColuna}" está sem nome.`;
		const outro = vistos.get(chaveNome(nome));
		if (outro) return `O campo extra "${nome}" tem o mesmo nome que "${outro}".`;
		vistos.set(chaveNome(nome), nome);
	}
	return null;
}

export default function MappingPage({
	mapping,
	buildFn,
	onSuccess,
	nextRoute,
	permitirExtras = false,
}: MappingPageProps) {
	const navigate = useNavigate();
	const [loading, setLoading] = useState(false);
	const [erro, setErro] = useState<string | null>(null);
	const dragActiveRef = useRef<boolean>(false);
	const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
	const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
	const [items, setItems] = useState<MappingItem[]>([]);
	// colunas que o usuário tirou dos extras; podem voltar
	const [ignoradas, setIgnoradas] = useState<{ nomeColuna: string; indice: number }[]>([]);

	useEffect(() => {
		setItems(mapping ?? []);
	}, []);
	// Listener global para autoscroll durante o drag
	useEffect(() => {
		const handleAutoScroll = (e: DragEvent) => {
			if (!dragActiveRef.current) return;
			const threshold = 50;
			const scrollSpeed = 4;
			if (e.clientY < threshold) {
				window.scrollBy(0, -scrollSpeed);
			} else if (e.clientY > window.innerHeight - threshold) {
				window.scrollBy(0, scrollSpeed);
			}
		};
		window.addEventListener("dragover", handleAutoScroll);
		return () => window.removeEventListener("dragover", handleAutoScroll);
	}, []);

	function onDragStart(e: React.DragEvent<HTMLDivElement>, index: number) {
		setDraggedIndex(index);
		dragActiveRef.current = true;

		const ghostElement = document.createElement("div");
		ghostElement.classList.add("ghost-element");
		ghostElement.textContent = items[index].nomeColuna || "sem coluna";
		ghostElement.style.width = "200px";
		ghostElement.style.padding = "10px";
		ghostElement.style.background = "rgba(59, 130, 246, 0.5)";
		ghostElement.style.borderRadius = "6px";
		ghostElement.style.color = "white";
		ghostElement.style.fontWeight = "bold";
		ghostElement.style.textAlign = "center";

		document.body.appendChild(ghostElement);
		e.dataTransfer.setDragImage(ghostElement, 100, 20);

		setTimeout(() => {
			document.body.removeChild(ghostElement);
		}, 0);
	}

	function onDragOver(e: React.DragEvent<HTMLDivElement>, index: number) {
		e.preventDefault();
		setDragOverIndex(index);
	}

	function onDragLeave(e: React.DragEvent<HTMLDivElement>) {
		if (e.currentTarget.contains(e.relatedTarget as Node)) return;
		setDragOverIndex(null);
	}

	function onDrop(e: React.DragEvent<HTMLDivElement>, dropIndex: number) {
		e.preventDefault();
		if (draggedIndex === null) return;
		const newMapping = items.map((i) => ({ ...i }));
		const temp = newMapping[draggedIndex].nomeColuna;
		newMapping[draggedIndex].nomeColuna = newMapping[dropIndex].nomeColuna;
		newMapping[dropIndex].nomeColuna = temp;

		const tempIndice = newMapping[draggedIndex].indice;
		newMapping[draggedIndex].indice = newMapping[dropIndex].indice;
		newMapping[dropIndex].indice = tempIndice;

		setItems(newMapping);
		setDraggedIndex(null);
		setDragOverIndex(null);
		dragActiveRef.current = false;
	}

	function onDragEnd() {
		setDraggedIndex(null);
		setDragOverIndex(null);
		dragActiveRef.current = false;
	}

	function renomearExtra(index: number, nome: string) {
		setItems((prev) => prev.map((item, i) => (i === index ? { ...item, variavel: PREFIXO_EXTRA + nome } : item)));
		setErro(null);
	}

	function removerExtra(index: number) {
		const item = items[index];
		if (item.nomeColuna) {
			setIgnoradas((prev) => [...prev, { nomeColuna: item.nomeColuna, indice: item.indice }]);
		}
		setItems((prev) => prev.filter((_, i) => i !== index));
		setErro(null);
	}

	function incluirComoExtra(col: { nomeColuna: string; indice: number }) {
		// nome livre: o da coluna, com sufixo se já existir
		const usados = new Set(items.map((i) => chaveNome(ehExtra(i) ? nomeExtra(i) : i.variavel)));
		let nome = col.nomeColuna.trim() || "extra";
		for (let n = 2; usados.has(chaveNome(nome)); n++) nome = `${col.nomeColuna.trim()} (${n})`;
		setItems((prev) => [...prev, { ...col, variavel: PREFIXO_EXTRA + nome }]);
		setIgnoradas((prev) => prev.filter((c) => c.indice !== col.indice));
	}

	async function onConfirm() {
		if (loading) return;
		// extra que ficou sem coluna (depois de arrastar) não entra
		const enviar = items
			.filter((i) => !ehExtra(i) || i.nomeColuna)
			.map((i) => (ehExtra(i) ? { ...i, variavel: PREFIXO_EXTRA + nomeExtra(i).trim() } : i));
		const msg = validarExtras(enviar);
		if (msg) {
			setErro(msg);
			return;
		}
		setLoading(true);
		setErro(null);
		try {
			const result = await buildFn(enviar);
			await onSuccess(result);
			navigate(nextRoute);
		} catch (err: any) {
			setErro(err?.message ?? String(err));
		} finally {
			setLoading(false);
		}
	}

	const temExtras = items.some(ehExtra);

	return (
		<div className="min-h-screen bg-gradient-to-b from-blue-50 to-gray-100 flex flex-col items-center py-12 px-4">
			<div className="max-w-4xl w-full bg-white rounded-xl shadow-lg p-8 mb-8">
				<h1 className="text-3xl font-bold mb-2 text-center text-gray-800">
					Reordenar Mapeamento
				</h1>
				<p className="mb-6 text-gray-600 text-center">
					Arraste as células da coluna direita para reordenar o
					mapeamento das variáveis
					{permitirExtras && (
						<>
							. Colunas que não são campos fixos entram como{" "}
							<span className="font-semibold text-purple-700">campos extras</span>: dá para
							renomear ou remover cada uma.
						</>
					)}
				</p>

				<div className="overflow-x-auto">
					<table className="w-full border-collapse">
						<thead>
							<tr className="bg-blue-600 text-white">
								<th className="px-6 py-3 text-left rounded-tl-lg">
									Variável
								</th>
								<th className="px-6 py-3 text-left rounded-tr-lg">
									Coluna do Arquivo
								</th>
							</tr>
						</thead>
						<tbody>
							{items.map((item, index) => {
								const extra = ehExtra(item);
								const primeiroExtra = extra && !ehExtra(items[index - 1] ?? item);
								return (
									<tr
										key={index}
										className={`border-b border-gray-200 transition-colors ${
											dragOverIndex === index
												? "bg-blue-100"
												: extra
													? "bg-purple-50/40"
													: ""
										} ${primeiroExtra ? "border-t-4 border-t-purple-100" : ""}`}
									>
										<td className="px-6 py-4 font-medium text-gray-700">
											{extra ? (
												<div className="flex items-center gap-2">
													<span className="text-xs font-semibold text-purple-700 bg-purple-100 px-2 py-0.5 rounded-full">
														extra
													</span>
													<input
														value={nomeExtra(item)}
														onChange={(e) => renomearExtra(index, e.target.value)}
														placeholder="nome do campo"
														aria-label="Nome do campo extra"
														className="flex-1 min-w-0 border border-gray-300 rounded-md px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-purple-300"
													/>
													<button
														onClick={() => removerExtra(index)}
														title="Não importar esta coluna"
														className="p-1 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-full"
													>
														<XMarkIcon className="w-4 h-4" />
													</button>
												</div>
											) : (
												item.variavel
											)}
										</td>
										<td className="px-6 py-4">
											<motion.div
												draggable
												onDragStart={(
													e: React.DragEvent<HTMLDivElement>
												) => onDragStart(e, index)}
												onDragOver={(
													e: React.DragEvent<HTMLDivElement>
												) => onDragOver(e, index)}
												onDragLeave={(
													e: React.DragEvent<HTMLDivElement>
												) => onDragLeave(e)}
												onDrop={(
													e: React.DragEvent<HTMLDivElement>
												) => onDrop(e, index)}
												onDragEnd={onDragEnd}
												whileHover={{ scale: 1.02 }}
												whileTap={{ scale: 0.98 }}
												className={`
                                                py-2 px-4
                                                cursor-move
                                                text-center
                                                bg-white
                                                rounded-lg
                                                border-2
                                                shadow-sm
                                                transition-all
                                                ${item.nomeColuna ? "" : "border-dashed"}
                                            `}
											>
												<div className="flex items-center justify-between">
													{item.nomeColuna ? (
														<span>{item.nomeColuna}</span>
													) : (
														<span className="text-gray-400 italic">sem coluna</span>
													)}
													<svg
														xmlns="http://www.w3.org/2000/svg"
														className="h-5 w-5 text-gray-400"
														fill="none"
														viewBox="0 0 24 24"
														stroke="currentColor"
													>
														<path
															strokeLinecap="round"
															strokeLinejoin="round"
															strokeWidth={2}
															d="M4 6h16M4 12h16M4 18h16"
														/>
													</svg>
												</div>
											</motion.div>
										</td>
									</tr>
								);
							})}
						</tbody>
					</table>
				</div>

				{permitirExtras && ignoradas.length > 0 && (
					<div className="mt-6">
						<div className="text-sm font-semibold text-gray-600 mb-2">
							Colunas não importadas
						</div>
						<div className="flex flex-wrap gap-2">
							{ignoradas.map((col) => (
								<button
									key={col.indice}
									onClick={() => incluirComoExtra(col)}
									title="Importar como campo extra"
									className="flex items-center gap-1 text-sm px-3 py-1 rounded-full border border-gray-300 text-gray-600 hover:border-purple-400 hover:text-purple-700 hover:bg-purple-50 transition-colors"
								>
									<PlusIcon className="w-4 h-4" />
									{col.nomeColuna}
								</button>
							))}
						</div>
					</div>
				)}

				{permitirExtras && temExtras && (
					<p className="mt-4 text-xs text-gray-500">
						Se arrastar a coluna de um campo extra para um campo fixo, o extra que ficar sem coluna é descartado.
					</p>
				)}

				{erro && (
					<div className="mt-6 flex items-start space-x-2 bg-red-50 border border-red-200 rounded-lg p-3">
						<ExclamationTriangleIcon className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
						<p className="text-sm text-red-700">{erro}</p>
					</div>
				)}
			</div>

			<div className="flex items-center space-x-4">
				<button
					onClick={() => navigate(-1)}
					className="flex items-center space-x-2 px-6 py-3 rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-100 font-medium transition-all"
				>
					<ArrowLeftIcon className="w-4 h-4" />
					<span>Voltar</span>
				</button>
				<motion.button
					onClick={onConfirm}
					disabled={loading}
					whileHover={{ scale: loading ? 1 : 1.05 }}
					whileTap={{ scale: loading ? 1 : 0.95 }}
					className={`px-8 py-3 rounded-lg shadow-md font-medium transition-all text-white ${
						loading
							? "bg-blue-400 cursor-not-allowed"
							: "bg-blue-600 hover:bg-blue-700"
					}`}
				>
					{loading ? "Processando..." : "Confirmar Mudanças"}
				</motion.button>
			</div>
		</div>
	);
}
