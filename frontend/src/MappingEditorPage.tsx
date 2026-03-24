import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import {
	ExclamationTriangleIcon,
	PlusIcon,
	TrashIcon,
} from "@heroicons/react/24/outline";
import type { MappingDraft, MappingFieldInfo, MappingItem } from "./importTypes";

interface MappingEditorPageProps {
	title: string;
	description: string;
	mapping: MappingDraft[] | null;
	setMapping: (items: MappingDraft[]) => void;
	fieldInfos: MappingFieldInfo[];
	onConfirm: (items: MappingItem[]) => Promise<void>;
	confirmLabel: string;
	allowExtraFields?: boolean;
}

export default function MappingEditorPage({
	title,
	description,
	mapping,
	setMapping,
	fieldInfos,
	onConfirm,
	confirmLabel,
	allowExtraFields = true,
}: MappingEditorPageProps) {
	const dragActiveRef = useRef(false);
	const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
	const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
	const [items, setItems] = useState<MappingDraft[]>([]);
	const [errorMsg, setErrorMsg] = useState<string | null>(null);

	const fieldInfoMap = Object.fromEntries(
		fieldInfos.map((fieldInfo) => [fieldInfo.variavel, fieldInfo])
	);

	useEffect(() => {
		setItems(mapping ?? []);
	}, [mapping]);

	useEffect(() => {
		const handleAutoScroll = (event: DragEvent) => {
			if (!dragActiveRef.current) {
				return;
			}

			const threshold = 50;
			const scrollSpeed = 4;
			if (event.clientY < threshold) {
				window.scrollBy(0, -scrollSpeed);
			} else if (event.clientY > window.innerHeight - threshold) {
				window.scrollBy(0, scrollSpeed);
			}
		};

		window.addEventListener("dragover", handleAutoScroll);
		return () => window.removeEventListener("dragover", handleAutoScroll);
	}, []);

	const isCoreField = (variable: string) => variable in fieldInfoMap;

	const normalizeExtraKey = (raw: string) =>
		raw
			.normalize("NFD")
			.replace(/[\u0300-\u036f]/g, "")
			.trim()
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, "_")
			.replace(/^_+|_+$/g, "");

	const createClientId = () =>
		typeof crypto !== "undefined" && "randomUUID" in crypto
			? crypto.randomUUID()
			: `${Date.now()}-${Math.random()}`;

	function updateItems(nextItems: MappingDraft[]) {
		setItems(nextItems);
		setMapping(nextItems);
	}

	function onDragStart(event: React.DragEvent<HTMLDivElement>, index: number) {
		setDraggedIndex(index);
		dragActiveRef.current = true;

		const ghostElement = document.createElement("div");
		ghostElement.classList.add("ghost-element");
		ghostElement.textContent = items[index]?.nomeColuna || "Nao mapeado";
		ghostElement.style.width = "200px";
		ghostElement.style.padding = "10px";
		ghostElement.style.background = "rgba(37, 99, 235, 0.8)";
		ghostElement.style.borderRadius = "6px";
		ghostElement.style.color = "white";
		ghostElement.style.fontWeight = "bold";
		ghostElement.style.textAlign = "center";

		document.body.appendChild(ghostElement);
		event.dataTransfer.setDragImage(ghostElement, 100, 20);

		setTimeout(() => {
			document.body.removeChild(ghostElement);
		}, 0);
	}

	function onDragOver(event: React.DragEvent<HTMLDivElement>, index: number) {
		event.preventDefault();
		setDragOverIndex(index);
	}

	function onDragLeave(event: React.DragEvent<HTMLDivElement>) {
		if (event.currentTarget.contains(event.relatedTarget as Node)) {
			return;
		}
		setDragOverIndex(null);
	}

	function onDrop(event: React.DragEvent<HTMLDivElement>, dropIndex: number) {
		event.preventDefault();
		if (draggedIndex === null) {
			return;
		}

		const nextItems = [...items];
		const draggedItem = nextItems[draggedIndex];
		const targetItem = nextItems[dropIndex];

		nextItems[draggedIndex] = {
			...draggedItem,
			nomeColuna: targetItem.nomeColuna,
			indice: targetItem.indice,
		};
		nextItems[dropIndex] = {
			...targetItem,
			nomeColuna: draggedItem.nomeColuna,
			indice: draggedItem.indice,
		};

		updateItems(nextItems);
		setDraggedIndex(null);
		setDragOverIndex(null);
		dragActiveRef.current = false;
	}

	function onDragEnd() {
		setDraggedIndex(null);
		setDragOverIndex(null);
		dragActiveRef.current = false;
	}

	function handleVariableChange(index: number, nextVariable: string) {
		const nextItems = [...items];
		nextItems[index] = {
			...nextItems[index],
			variavel: nextVariable,
		};
		updateItems(nextItems);
	}

	function removeExtra(index: number) {
		const nextItems = [...items];
		const removedItem = nextItems[index];
		nextItems.splice(index, 1);

		if (removedItem.indice >= 0) {
			nextItems.push({
				nomeColuna: removedItem.nomeColuna,
				indice: removedItem.indice,
				variavel: "",
				includeWhenUnmapped: false,
				clientId: createClientId(),
				manualExtra: false,
			});
		}

		updateItems(nextItems);
	}

	function addExtraMapping() {
		updateItems([
			...items,
			{
				nomeColuna: "",
				indice: -1,
				variavel: "",
				includeWhenUnmapped: true,
				clientId: createClientId(),
				manualExtra: true,
			},
		]);
	}

	function buildPayload(): MappingItem[] | null {
		const normalizedExtras = new Map<string, string>();

		for (const item of items) {
			if (!item.manualExtra) {
				continue;
			}

			const rawName = item.variavel.trim();
			if (rawName === "") {
				setErrorMsg(
					"Todo campo extra criado manualmente precisa ter um nome antes de seguir."
				);
				return null;
			}

			const normalized = normalizeExtraKey(rawName);
			if (!normalized) {
				setErrorMsg("O nome do campo extra precisa gerar uma chave válida.");
				return null;
			}
			if (isCoreField(normalized)) {
				setErrorMsg(
					`O campo extra "${rawName}" conflita com um campo principal.`
				);
				return null;
			}
			if (normalizedExtras.has(normalized)) {
				setErrorMsg(
					`Os campos extras "${normalizedExtras.get(normalized)}" e "${rawName}" geram a mesma chave.`
				);
				return null;
			}

			normalizedExtras.set(normalized, rawName);
		}

		return items
			.filter((item) => {
				if (item.manualExtra) {
					return true;
				}
				if (item.variavel === "") {
					return false;
				}
				return item.indice >= 0;
			})
			.map((item) => ({
				nomeColuna: item.nomeColuna,
				indice: item.indice,
				variavel: item.manualExtra ? normalizeExtraKey(item.variavel) : item.variavel,
				includeWhenUnmapped: item.manualExtra && item.indice === -1,
			}));
	}

	async function handleConfirm() {
		const payload = buildPayload();
		if (!payload) {
			return;
		}
		await onConfirm(payload);
	}

	function renderBadges(variable: string, manualExtra: boolean) {
		if (manualExtra) {
			return (
				<div className="flex flex-wrap gap-2">
					<span className="inline-flex items-center rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600">
						Extra
					</span>
					<span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-700">
						Null se sem coluna
					</span>
				</div>
			);
		}

		const fieldInfo = fieldInfoMap[variable];
		if (!fieldInfo) {
			return (
				<span className="inline-flex items-center rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600">
					Disponivel
				</span>
			);
		}

		return (
			<div className="flex flex-wrap gap-2">
				<span
					className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${
						fieldInfo.required
							? "bg-red-100 text-red-700"
							: "bg-slate-100 text-slate-600"
					}`}
				>
					{fieldInfo.required ? "Obrigatorio" : "Opcional"}
				</span>
				{fieldInfo.unique && (
					<span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-700">
						Reconstrucao/Unico
					</span>
				)}
			</div>
		);
	}

	const coreItems = items.filter(
		(item) => !item.manualExtra && isCoreField(item.variavel)
	);
	const extraItems = items.filter((item) => item.manualExtra);
	const availableItems = items.filter(
		(item) => !item.manualExtra && item.variavel === "" && item.indice !== -1
	);

	return (
		<div className="min-h-screen bg-gradient-to-b from-blue-50 to-gray-100 flex flex-col items-center py-12 px-4">
			{errorMsg && (
				<div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
					<motion.div
						initial={{ scale: 0.9, opacity: 0 }}
						animate={{ scale: 1, opacity: 1 }}
						className="bg-white p-6 rounded-xl shadow-2xl max-w-md mx-4"
					>
						<div className="flex items-center space-x-3 mb-4">
							<ExclamationTriangleIcon className="w-6 h-6 text-red-500" />
							<h3 className="font-semibold text-gray-900">
								Erro no mapeamento
							</h3>
						</div>
						<p className="text-gray-700 mb-6">{errorMsg}</p>
						<button
							className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
							onClick={() => setErrorMsg(null)}
						>
							Entendido
						</button>
					</motion.div>
				</div>
			)}

			<div className="max-w-5xl w-full bg-white rounded-xl shadow-lg p-8 mb-8">
				<h1 className="text-3xl font-bold mb-2 text-center text-gray-800">
					{title}
				</h1>
				<p className="mb-2 text-gray-600 text-center">{description}</p>
				<p className="mb-6 text-sm text-gray-500 text-center">
					Os badges mostram se o campo e obrigatorio e se participa da
					reconstrucao por identificador unico.
				</p>

				<div className="space-y-8">
					<section>
						<h2 className="text-lg font-semibold text-blue-800 mb-4 border-b pb-2">
							Campos Principais
						</h2>
						<div className="overflow-x-auto">
							<table className="w-full border-collapse">
								<thead>
									<tr className="bg-blue-600 text-white">
										<th className="px-6 py-3 text-left rounded-tl-lg">
											Variavel
										</th>
										<th className="px-6 py-3 text-left">
											Regras
										</th>
										<th className="px-6 py-3 text-left rounded-tr-lg">
											Coluna do Arquivo
										</th>
									</tr>
								</thead>
								<tbody>
									{coreItems.map((item) => {
										const index = items.findIndex(
											(candidate) => candidate.clientId === item.clientId
										);

										return (
											<tr
												key={item.clientId}
												className={`border-b border-gray-200 transition-colors ${
													dragOverIndex === index ? "bg-blue-100" : ""
												}`}
											>
												<td className="px-6 py-4 font-medium text-gray-700 capitalize">
													{item.variavel.replace(/_/g, " ")}
												</td>
												<td className="px-6 py-4">
													{renderBadges(item.variavel, false)}
												</td>
												<td className="px-6 py-4">
													<div
														draggable
														onDragStart={(event) => onDragStart(event, index)}
														onDragOver={(event) => onDragOver(event, index)}
														onDragLeave={onDragLeave}
														onDrop={(event) => onDrop(event, index)}
														onDragEnd={onDragEnd}
														className="flex items-center justify-between py-2 px-4 cursor-move bg-blue-50 rounded-lg border-2 border-blue-200 shadow-sm hover:bg-blue-100 transition-all"
													>
														<span>{item.nomeColuna || "Clique e arraste uma coluna"}</span>
														<svg
															className="h-5 w-5 text-blue-400"
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
												</td>
											</tr>
										);
									})}
								</tbody>
							</table>
						</div>
					</section>

					{allowExtraFields && (
						<section>
							<div className="flex justify-between items-center mb-4 border-b pb-2">
								<h2 className="text-lg font-semibold text-purple-800">
									Campos Extras
								</h2>
								<button
									onClick={addExtraMapping}
									className="inline-flex items-center gap-1 rounded-lg bg-purple-100 px-3 py-1 text-sm font-medium text-purple-700 hover:bg-purple-200 transition-colors"
								>
									<PlusIcon className="h-4 w-4" />
									Adicionar Extra
								</button>
							</div>
							<div className="overflow-x-auto">
								<table className="w-full border-collapse">
									<thead>
										<tr className="bg-purple-600 text-white">
											<th className="px-6 py-3 text-left rounded-tl-lg">
												Nome do Campo
											</th>
											<th className="px-6 py-3 text-left">
												Regras
											</th>
											<th className="px-6 py-3 text-left">
												Coluna do Arquivo
											</th>
											<th className="px-6 py-3 text-center rounded-tr-lg w-20">
												Acoes
											</th>
										</tr>
									</thead>
									<tbody>
										{extraItems.map((item) => {
											const index = items.findIndex(
												(candidate) => candidate.clientId === item.clientId
											);

											return (
												<tr
													key={item.clientId}
													className={`border-b border-gray-200 transition-colors ${
														dragOverIndex === index ? "bg-purple-100" : ""
													}`}
												>
													<td className="px-6 py-4">
														<input
															type="text"
															value={item.variavel}
															onChange={(event) =>
																handleVariableChange(index, event.target.value)
															}
															onKeyDown={(event) => {
																event.stopPropagation();
															}}
															className="w-full border border-gray-300 rounded-md px-3 py-1 focus:ring-purple-500 focus:border-purple-500 text-sm"
															placeholder="Nome do campo extra"
														/>
													</td>
													<td className="px-6 py-4">
														{renderBadges(item.variavel, true)}
													</td>
													<td className="px-6 py-4">
														<div
															draggable
															onDragStart={(event) => onDragStart(event, index)}
															onDragOver={(event) => onDragOver(event, index)}
															onDragLeave={onDragLeave}
															onDrop={(event) => onDrop(event, index)}
															onDragEnd={onDragEnd}
															className="flex items-center justify-between py-2 px-4 cursor-move bg-purple-50 rounded-lg border-2 border-purple-200 shadow-sm hover:bg-purple-100 transition-all"
														>
															<span>{item.nomeColuna || "Sem coluna mapeada"}</span>
															<svg
																className="h-5 w-5 text-purple-400"
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
													</td>
													<td className="px-6 py-4 text-center">
														<button
															onClick={() => removeExtra(index)}
															className="p-2 text-red-500 hover:text-red-700 hover:bg-red-50 rounded-full transition-colors"
														>
															<TrashIcon className="h-5 w-5" />
														</button>
													</td>
												</tr>
											);
										})}
									</tbody>
								</table>
							</div>
						</section>
					)}

					{availableItems.length > 0 && (
						<section>
							<h2 className="text-lg font-semibold text-gray-600 mb-4 border-b pb-2">
								Colunas Disponiveis
							</h2>
							<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
								{availableItems.map((item) => {
									const index = items.findIndex(
										(candidate) => candidate.clientId === item.clientId
									);

									return (
										<div
											key={item.clientId}
											className={`p-4 rounded-lg border-2 transition-all cursor-move flex items-center justify-between ${
												dragOverIndex === index
													? "bg-gray-200 border-gray-400"
													: "bg-gray-50 border-gray-200 hover:bg-gray-100"
											}`}
											draggable
											onDragStart={(event) => onDragStart(event, index)}
											onDragOver={(event) => onDragOver(event, index)}
											onDragLeave={onDragLeave}
											onDrop={(event) => onDrop(event, index)}
											onDragEnd={onDragEnd}
										>
											<span className="text-sm font-medium text-gray-600 truncate mr-2">
												{item.nomeColuna}
											</span>
											<svg
												className="h-4 w-4 text-gray-400 shrink-0"
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
									);
								})}
							</div>
						</section>
					)}
				</div>
			</div>

			<motion.button
				onClick={handleConfirm}
				whileHover={{ scale: 1.05 }}
				whileTap={{ scale: 0.95 }}
				className="bg-blue-600 hover:bg-blue-700 text-white px-12 py-4 rounded-xl shadow-lg font-bold text-lg transition-all"
			>
				{confirmLabel}
			</motion.button>
		</div>
	);
}
