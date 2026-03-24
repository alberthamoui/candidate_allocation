import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { PlusIcon, TrashIcon } from "@heroicons/react/24/outline";
import type { MappingFieldInfo, MappingItem } from "./importTypes";

interface MappingEditorPageProps {
	title: string;
	description: string;
	mapping: MappingItem[] | null;
	setMapping: (items: MappingItem[]) => void;
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
	const [items, setItems] = useState<MappingItem[]>([]);

	const fieldInfoMap = Object.fromEntries(
		fieldInfos.map((fieldInfo) => [fieldInfo.variavel, fieldInfo])
	);

	const isCoreField = (variable: string) => variable in fieldInfoMap;

	useEffect(() => {
		const normalizedItems = (mapping ?? []).map((item) => {
			if (allowExtraFields || isCoreField(item.variavel) || item.variavel === "") {
				return item;
			}
			return { ...item, variavel: "" };
		});
		setItems(normalizedItems);
	}, [allowExtraFields, fieldInfos, mapping]);

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

	function updateItems(nextItems: MappingItem[]) {
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

	function removeMapping(index: number) {
		const nextItems = [...items];
		if (nextItems[index].indice !== -1) {
			nextItems[index] = {
				...nextItems[index],
				variavel: "",
			};
			updateItems(nextItems);
			return;
		}

		updateItems(nextItems.filter((_, itemIndex) => itemIndex !== index));
	}

	function addExtraMapping() {
		updateItems([
			...items,
			{
				nomeColuna: "",
				indice: -1,
				variavel: "novo_campo_extra",
			},
		]);
	}

	async function handleConfirm() {
		const filteredItems = items.filter((item) => {
			if (item.indice === -1 || item.variavel === "") {
				return false;
			}
			if (!allowExtraFields && !isCoreField(item.variavel)) {
				return false;
			}
			return true;
		});

		updateItems(items);
		await onConfirm(filteredItems);
	}

	function renderBadges(variable: string) {
		const fieldInfo = fieldInfoMap[variable];
		if (!fieldInfo) {
			return (
				<span className="inline-flex items-center rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600">
					Extra
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

	const unmappedItems = items.filter(
		(item) => !isCoreField(item.variavel) && item.variavel === "" && item.indice !== -1
	);

	return (
		<div className="min-h-screen bg-gradient-to-b from-blue-50 to-gray-100 flex flex-col items-center py-12 px-4">
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
									{items.map((item, index) => {
										if (!isCoreField(item.variavel)) {
											return null;
										}

										return (
											<tr
												key={`${item.variavel}-${index}`}
												className={`border-b border-gray-200 transition-colors ${
													dragOverIndex === index ? "bg-blue-100" : ""
												}`}
											>
												<td className="px-6 py-4 font-medium text-gray-700 capitalize">
													{item.variavel.replace(/_/g, " ")}
												</td>
												<td className="px-6 py-4">
													{renderBadges(item.variavel)}
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
										{items.map((item, index) => {
											if (isCoreField(item.variavel) || item.variavel === "") {
												return null;
											}

											return (
												<tr
													key={`${item.variavel}-${index}`}
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
															className="w-full border border-gray-300 rounded-md px-3 py-1 focus:ring-purple-500 focus:border-purple-500 text-sm"
														/>
													</td>
													<td className="px-6 py-4">{renderBadges(item.variavel)}</td>
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
															<span>{item.nomeColuna || "Clique e arraste uma coluna"}</span>
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
															onClick={() => removeMapping(index)}
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

					{unmappedItems.length > 0 && (
						<section>
							<h2 className="text-lg font-semibold text-gray-600 mb-4 border-b pb-2">
								Colunas Disponiveis
							</h2>
							<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
								{items.map((item, index) => {
									if (isCoreField(item.variavel) || item.variavel !== "" || item.indice === -1) {
										return null;
									}

									return (
										<div
											key={`${item.nomeColuna}-${index}`}
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
