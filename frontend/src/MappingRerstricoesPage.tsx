import { useLocation, useNavigate } from "react-router-dom";
import { useState, useEffect, useRef } from "react"; // adicionada a importação do useRef
import { motion } from "framer-motion";
import { BuildRestricoesWithMapping } from "../wailsjs/go/main/App";
interface MappingItem {
	nomeColuna: string;
	indice: number;
	variavel: string;
}
interface MappingPageProps {
	mapping: MappingItem[] | null;
	setRestricoes: (data: any) => void;
}
export default function MappingResticoesPage({
	mapping,
	setRestricoes,
}: MappingPageProps) {
	const navigate = useNavigate();
	const dragActiveRef = useRef<boolean>(false);
	const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
	const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
	const [items, setItems] = useState<MappingItem[]>([]);
	useEffect(() => {
		console.log("mapping : ", mapping);
		if (mapping) {
			setItems(mapping);
		} else {
			setItems([]);
		}
		console.log("items: ", items);
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

	// Inicia o drag armazenando o índice da linha
	function onDragStart(e: React.DragEvent<HTMLDivElement>, index: number) {
		setDraggedIndex(index);
		dragActiveRef.current = true;

		// Configura a imagem fantasma personalizada
		const ghostElement = document.createElement("div");
		ghostElement.classList.add("ghost-element");
		ghostElement.textContent = items[index].nomeColuna;
		ghostElement.style.width = "200px";
		ghostElement.style.padding = "10px";
		ghostElement.style.background = "rgba(59, 130, 246, 0.5)";
		ghostElement.style.borderRadius = "6px";
		ghostElement.style.color = "white";
		ghostElement.style.fontWeight = "bold";
		ghostElement.style.textAlign = "center";

		document.body.appendChild(ghostElement);
		e.dataTransfer.setDragImage(ghostElement, 100, 20);

		// Remove o elemento após um curto período
		setTimeout(() => {
			document.body.removeChild(ghostElement);
		}, 0);
	}

	function onDragOver(e: React.DragEvent<HTMLDivElement>, index: number) {
		e.preventDefault();
		setDragOverIndex(index);
	}

	// Melhore o evento onDragLeave para garantir que o estado é limpo
	function onDragLeave(e: React.DragEvent<HTMLDivElement>) {
		// Previne limpeza incorreta quando movendo entre elementos filhos
		if (e.currentTarget.contains(e.relatedTarget as Node)) return;
		setDragOverIndex(null);
	}

	// Ao soltar, troca o conteúdo da célula "nomeColuna" entre a linha arrastada e a linha alvo
	function onDrop(e: React.DragEvent<HTMLDivElement>, dropIndex: number) {
		e.preventDefault();
		if (draggedIndex === null) return;
		const newMapping = [...items];
		const temp = newMapping[draggedIndex].nomeColuna;
		newMapping[draggedIndex].nomeColuna = newMapping[dropIndex].nomeColuna;
		newMapping[dropIndex].nomeColuna = temp;

		// Também troca os índices
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

	async function onConfirm() {
		console.log(items, " : mapping");
		const restricoes = await BuildRestricoesWithMapping(items);
		setRestricoes(restricoes);
		navigate("/verify");
	}

	return (
		<div className="min-h-screen bg-gradient-to-b from-blue-50 to-gray-100 flex flex-col items-center py-12 px-4">
			<div className="max-w-4xl w-full bg-white rounded-xl shadow-lg p-8 mb-8">
				<h1 className="text-3xl font-bold mb-2 text-center text-gray-800">
					Reordenar Mapeamento Avaliadores
				</h1>
				<p className="mb-6 text-gray-600 text-center">
					Arraste as células da coluna direita para reordenar o
					mapeamento das variáveis
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
							{items.map((item, index) => (
								<tr
									key={index}
									className={`border-b border-gray-200 transition-colors ${
										dragOverIndex === index
											? "bg-blue-100"
											: ""
									}`}
								>
									<td className="px-6 py-4 font-medium text-gray-700">
										{item.variavel}
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
                                            `}
										>
											<div className="flex items-center justify-between">
												<span>{item.nomeColuna}</span>
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
							))}
						</tbody>
					</table>
				</div>
			</div>

			<motion.button
				onClick={onConfirm}
				whileHover={{ scale: 1.05 }}
				whileTap={{ scale: 0.95 }}
				className="bg-blue-600 hover:bg-blue-700 text-white px-8 py-3 rounded-lg shadow-md font-medium transition-all"
			>
				Confirmar Mudanças
			</motion.button>
		</div>
	);
}
