import { useLocation, useNavigate } from "react-router-dom";
import { useState, useEffect, useRef } from "react";
import { motion } from "framer-motion";
import { PlusIcon, TrashIcon } from "@heroicons/react/24/outline";
import { BuildUsuariosWithMapping } from "../wailsjs/go/main/App";

interface MappingItem {
	nomeColuna: string;
	indice: number;
	variavel: string;
}

interface MappingPageProps {
	mapping: MappingItem[] | null;
	setUsers: (data: any) => void;
	setDuplicatas: (data: any) => void;
	setDuplicateFields: (data: string[]) => void;
}

const CORE_CANDIDATO_FIELDS = [
	"timestamp",
	"nome",
	"cpf",
	"numero",
	"semestre",
	"curso",
	"email_secundario",
	"email_pessoal",
];

const isCoreField = (variable: string) =>
	CORE_CANDIDATO_FIELDS.includes(variable) || variable.startsWith("opcao ");

export default function MappingPage({
	mapping,
	setUsers,
	setDuplicatas,
	setDuplicateFields,
}: MappingPageProps) {
	const navigate = useNavigate();
	const dragActiveRef = useRef<boolean>(false);
	const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
	const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
	const [items, setItems] = useState<MappingItem[]>([]);

	useEffect(() => {
		if (mapping) {
			setItems(mapping);
		} else {
			setItems([]);
		}
	}, [mapping]);

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
		ghostElement.textContent = items[index].nomeColuna || "Nao Mapeado";
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
		const newMapping = [...items];
		const temp = newMapping[draggedIndex].nomeColuna;
		newMapping[draggedIndex].nomeColuna = newMapping[dropIndex].nomeColuna;
		newMapping[dropIndex].nomeColuna = temp;

		const tempIndice = newMapping[draggedIndex].indice;
		newMapping[draggedIndex].indice = newMapping[dropIndex].indice;
		newMapping[dropIndex].indice = tempIndice;

		// Remove extra fields that are now unmapped
		const filteredMapping = newMapping.filter(
			(item) => isCoreField(item.variavel) || item.nomeColuna !== ""
		);

		setItems(filteredMapping);
		setDraggedIndex(null);
		setDragOverIndex(null);
		dragActiveRef.current = false;
	}

	function onDragEnd() {
		setDraggedIndex(null);
		setDragOverIndex(null);
		dragActiveRef.current = false;
	}

	function handleVariableChange(index: number, newVariable: string) {
		const newItems = [...items];
		newItems[index].variavel = newVariable;
		setItems(newItems);
	}

	function removeMapping(index: number) {
		setItems(items.filter((_, i) => i !== index));
	}

	function addExtraMapping() {
		setItems([
			...items,
			{
				nomeColuna: "",
				indice: -1,
				variavel: "novo_campo_extra",
			},
		]);
	}

	async function onConfirm() {
		const { usuarios, duplicates, duplicateFields } =
			await BuildUsuariosWithMapping(items);
		setUsers(usuarios);
		setDuplicatas(duplicates);
		setDuplicateFields(duplicateFields ?? []);
		navigate("/mappingAvaliadores");
	}

	const coreItems = items.filter((it) => isCoreField(it.variavel));
	const extraItems = items.filter((it) => !isCoreField(it.variavel));

	return (
		<div className="min-h-screen bg-gradient-to-b from-blue-50 to-gray-100 flex flex-col items-center py-12 px-4">
			<div className="max-w-4xl w-full bg-white rounded-xl shadow-lg p-8 mb-8">
				<h1 className="text-3xl font-bold mb-2 text-center text-gray-800">
					Mapeamento de Candidatos
				</h1>
				<p className="mb-6 text-gray-600 text-center">
					Arraste as colunas do arquivo para as variáveis correspondentes.
					Você pode adicionar ou remover campos extras.
				</p>

				<div className="space-y-8">
					{/* Core Fields Section */}
					<section>
						<h2 className="text-lg font-semibold text-blue-800 mb-4 border-b pb-2">
							Campos Principais
						</h2>
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
										if (!isCoreField(item.variavel)) return null;
										return (
											<tr
												key={index}
												className={`border-b border-gray-200 transition-colors ${
													dragOverIndex === index
														? "bg-blue-100"
														: ""
												}`}
											>
												<td className="px-6 py-4 font-medium text-gray-700 capitalize">
													{item.variavel.replace(/_/g, " ")}
												</td>
												<td className="px-6 py-4">
													<div
														draggable
														onDragStart={(e) => onDragStart(e, index)}
														onDragOver={(e) => onDragOver(e, index)}
														onDragLeave={onDragLeave}
														onDrop={(e) => onDrop(e, index)}
														onDragEnd={onDragEnd}
														className="flex items-center justify-between py-2 px-4 cursor-move bg-blue-50 rounded-lg border-2 border-blue-200 shadow-sm hover:bg-blue-100 transition-all"
													>
														<span>{item.nomeColuna || "Clique e arraste uma coluna"}</span>
														<svg className="h-5 w-5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
															<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
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

					{/* Extra Fields Section */}
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
											Nome do Campo (no Banco)
										</th>
										<th className="px-6 py-3 text-left">
											Coluna do Arquivo
										</th>
										<th className="px-6 py-3 text-center rounded-tr-lg w-20">
											Ações
										</th>
									</tr>
								</thead>
								<tbody>
									{items.map((item, index) => {
										if (isCoreField(item.variavel)) return null;
										return (
											<tr
												key={index}
												className={`border-b border-gray-200 transition-colors ${
													dragOverIndex === index
														? "bg-purple-100"
														: ""
												}`}
											>
												<td className="px-6 py-4">
													<input
														type="text"
														value={item.variavel}
														onChange={(e) => handleVariableChange(index, e.target.value)}
														className="w-full border border-gray-300 rounded-md px-3 py-1 focus:ring-purple-500 focus:border-purple-500 text-sm"
													/>
												</td>
												<td className="px-6 py-4">
													<div
														draggable
														onDragStart={(e) => onDragStart(e, index)}
														onDragOver={(e) => onDragOver(e, index)}
														onDragLeave={onDragLeave}
														onDrop={(e) => onDrop(e, index)}
														onDragEnd={onDragEnd}
														className="flex items-center justify-between py-2 px-4 cursor-move bg-purple-50 rounded-lg border-2 border-purple-200 shadow-sm hover:bg-purple-100 transition-all"
													>
														<span>{item.nomeColuna || "Clique e arraste uma coluna"}</span>
														<svg className="h-5 w-5 text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
															<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
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
				</div>
			</div>

			<motion.button
				onClick={onConfirm}
				whileHover={{ scale: 1.05 }}
				whileTap={{ scale: 0.95 }}
				className="bg-blue-600 hover:bg-blue-700 text-white px-12 py-4 rounded-xl shadow-lg font-bold text-lg transition-all"
			>
				Confirmar e Seguir
			</motion.button>
		</div>
	);
}
