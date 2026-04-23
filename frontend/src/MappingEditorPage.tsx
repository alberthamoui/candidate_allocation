import { HelpIcon } from "./components/Tooltip";
import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import {
	ExclamationTriangleIcon,
	InformationCircleIcon,
	PlusIcon,
	TrashIcon,
} from "@heroicons/react/24/outline";
import type { MappingDraft, MappingFieldInfo, MappingItem } from "./importTypes";
import {
	EmptyState,
	
	PrimaryButton,
	SectionCard,
	StickyActionBar,
} from "./workflowShell";

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

function cn(...values: Array<string | false | null | undefined>) {
	return values.filter(Boolean).join(" ");
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
					<span className="inline-flex items-center rounded-full border border-[rgba(178,122,68,0.18)] bg-[rgba(178,122,68,0.12)] px-3 py-1 text-xs font-semibold text-[var(--accent-strong)]">
						Extra
					</span>
					<span className="inline-flex items-center rounded-full border border-[rgba(18,48,71,0.14)] bg-white/80 px-3 py-1 text-xs font-semibold text-[var(--muted)]">
						Null se sem coluna
					</span>
				</div>
			);
		}

		const fieldInfo = fieldInfoMap[variable];
		if (!fieldInfo) {
			return (
				<span className="inline-flex items-center rounded-full border border-[var(--line)] bg-white/80 px-3 py-1 text-xs font-semibold text-[var(--muted)]">
					Disponivel
				</span>
			);
		}

		return (
			<div className="flex flex-wrap gap-2">
				<span
					className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-semibold ${
						fieldInfo.required
							? "border-[rgba(156,66,63,0.2)] bg-[rgba(156,66,63,0.12)] text-[var(--danger)]"
							: "border-[var(--line)] bg-white/80 text-[var(--muted)]"
					}`}
				>
					{fieldInfo.required ? "Obrigatorio" : "Opcional"}
				</span>
				{fieldInfo.unique && (
					<span className="inline-flex items-center rounded-full border border-[rgba(178,122,68,0.18)] bg-[rgba(178,122,68,0.12)] px-3 py-1 text-xs font-semibold text-[var(--accent-strong)]">
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
		<div className="space-y-6">
			{errorMsg && (
				<div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
					<motion.div
						initial={{ scale: 0.9, opacity: 0 }}
						animate={{ scale: 1, opacity: 1 }}
						className="executive-card executive-card-strong mx-4 max-w-md p-6"
					>
						<div className="flex items-center space-x-3 mb-4">
							<ExclamationTriangleIcon className="w-6 h-6 text-[var(--danger)]" />
							<h3 className="font-semibold text-[var(--text)]">
								Erro no mapeamento
							</h3>
						</div>
						<p className="mb-6 text-sm leading-6 text-[var(--muted)]">{errorMsg}</p>
						<PrimaryButton className="w-full justify-center" onClick={() => setErrorMsg(null)}>
							Entendido
						</PrimaryButton>
					</motion.div>
				</div>
			)}

			<SectionCard
				title={title}
			>
				<div className="grid gap-4 md:grid-cols-3">
					<div className="minimal-panel p-5">
						<div className="text-xs font-semibold text-gray-500 uppercase tracking-widest mb-2">
							Campos principais
						</div>
						<p className="text-sm text-gray-600">
							Mapeie primeiro os atributos estruturais. Eles sustentam identificação, regras e validações.
						</p>
					</div>
					<div className="minimal-panel p-5">
						<div className="flex items-center gap-2 text-xs font-semibold text-gray-500 uppercase tracking-widest mb-2">
							Campos extras
							<HelpIcon text="Extras preservam dados específicos de cada empresa. Se forem criados manualmente, precisam ter um nome válido antes de seguir." />
						</div>
						<p className="text-sm text-gray-600">
							Use extras para não perder informação útil que não faz parte do schema principal.
						</p>
					</div>
					<div className="minimal-panel p-5">
						<div className="flex items-center gap-2 text-xs font-semibold text-gray-500 uppercase tracking-widest mb-2">
							Colunas disponíveis
							<HelpIcon text="Essas colunas ainda não estão associadas a nenhum destino. Arraste-as para um campo principal ou extra quando fizer sentido." />
						</div>
						<p className="text-sm text-gray-600">
							O inventário restante permite revisar rapidamente o que ainda não foi aproveitado.
						</p>
					</div>
				</div>
			</SectionCard>

			<SectionCard
				title="Campos Principais"
			>
				<div className="space-y-4">
					{coreItems.map((item) => {
						const index = items.findIndex(
							(candidate) => candidate.clientId === item.clientId
						);

						return (
							<div
								key={item.clientId}
								className={cn(
									"grid gap-4 rounded-xl border px-5 py-5 transition-all md:grid-cols-[1.1fr_1fr_1.3fr]",
									dragOverIndex === index
										? "border-blue-300 bg-blue-50/50"
										: "border-gray-200 bg-white"
								)}
							>
								<div>
									<div className="text-xs font-medium uppercase tracking-wider text-gray-400 mb-1">
										Variavel
									</div>
									<div className="text-sm font-semibold capitalize text-gray-900">
										{item.variavel.replace(/_/g, " ")}
									</div>
								</div>
								<div>
									<div className="mb-1 text-xs font-medium uppercase tracking-wider text-gray-400">
										Regras
									</div>
									{renderBadges(item.variavel, false)}
								</div>
								<div>
									<div className="mb-1 text-xs font-medium uppercase tracking-wider text-gray-400">
										Coluna do arquivo
									</div>
									<div
										draggable
										onDragStart={(event) => onDragStart(event, index)}
										onDragOver={(event) => onDragOver(event, index)}
										onDragLeave={onDragLeave}
										onDrop={(event) => onDrop(event, index)}
										onDragEnd={onDragEnd}
										className="flex cursor-move items-center justify-between rounded-lg border border-gray-200 bg-gray-50 px-4 py-2.5 transition-all hover:border-blue-300 hover:bg-blue-50/30"
									>
										<span className="text-sm font-medium text-gray-700">
											{item.nomeColuna || "Clique e arraste uma coluna"}
										</span>
										<svg
											className="h-4 w-4 text-gray-400"
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
								</div>
							</div>
						);
					})}
				</div>
			</SectionCard>

			{allowExtraFields && (
				<SectionCard
					title="Campos Extras"
					aside={
						<PrimaryButton onClick={addExtraMapping} className="justify-center" data-testid="add-extra-button">
							<PlusIcon className="h-4 w-4" />
							Adicionar Extra
						</PrimaryButton>
					}
				>
					{extraItems.length === 0 ? (
						<EmptyState
							title="Nenhum campo extra configurado"
							description="Se a planilha trouxer contexto específico do cliente, adicione um campo extra e escolha se ele será ligado a uma coluna real ou seguirá como null quando estiver sem origem."
							action={
										<button
											onClick={addExtraMapping}
											className="minimal-btn-secondary"
										>
											<PlusIcon className="h-4 w-4" />
											Adicionar Extra
										</button>
							}
						/>
					) : (
						<div className="space-y-4">
							{extraItems.map((item) => {
								const index = items.findIndex(
									(candidate) => candidate.clientId === item.clientId
								);

								return (
									<div
										key={item.clientId}
										className={cn(
											"grid gap-4 rounded-[24px] border px-5 py-5 transition-all md:grid-cols-[1fr_1fr_1.1fr_auto]",
											dragOverIndex === index
												? "border-[rgba(178,122,68,0.4)] bg-[rgba(178,122,68,0.1)]"
												: "border-[var(--line)] bg-white/70"
										)}
									>
										<div>
											<div className="mb-2 flex items-center gap-2 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">
												Nome do campo
												<HelpIcon text="O nome é normalizado para gerar a chave final. Ele não pode colidir com campos principais nem com outro extra." />
											</div>
											<input
												type="text"
												value={item.variavel}
												onChange={(event) =>
													handleVariableChange(index, event.target.value)
												}
												onKeyDown={(event) => {
													event.stopPropagation();
												}}
												className="executive-input"
												placeholder="Nome do campo extra"
											/>
										</div>
										<div>
											<div className="mb-2 flex items-center gap-2 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">
												Regras
												<HelpIcon text="Quando um campo extra manual é mantido sem coluna, ele segue como null para preservar a estrutura sem inventar um valor." />
											</div>
											{renderBadges(item.variavel, true)}
										</div>
										<div>
											<div className="mb-2 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">
												Coluna do arquivo
											</div>
											<div
												draggable
												onDragStart={(event) => onDragStart(event, index)}
												onDragOver={(event) => onDragOver(event, index)}
												onDragLeave={onDragLeave}
												onDrop={(event) => onDrop(event, index)}
												onDragEnd={onDragEnd}
												className="flex cursor-move items-center justify-between rounded-[20px] border border-[rgba(18,48,71,0.16)] bg-[rgba(18,48,71,0.05)] px-4 py-3 transition-all hover:border-[rgba(178,122,68,0.36)] hover:bg-[rgba(178,122,68,0.08)]"
											>
												<span className="text-sm font-medium text-[var(--text)]">
													{item.nomeColuna || "Sem coluna mapeada"}
												</span>
												<svg
													className="h-5 w-5 text-[var(--accent-strong)]"
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
										</div>
										<div className="flex items-end justify-end">
											<button
												onClick={() => removeExtra(index)}
												className="rounded-full border border-[rgba(156,66,63,0.18)] bg-[rgba(156,66,63,0.1)] p-3 text-[var(--danger)] transition-colors hover:bg-[rgba(156,66,63,0.16)]"
												aria-label="Remover campo extra"
											>
												<TrashIcon className="h-5 w-5" />
											</button>
										</div>
									</div>
								);
							})}
						</div>
					)}
				</SectionCard>
			)}

			{availableItems.length > 0 && (
				<SectionCard
					title="Colunas Disponiveis"
				>
					<div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
						{availableItems.map((item) => {
							const index = items.findIndex(
								(candidate) => candidate.clientId === item.clientId
							);

							return (
								<div
									key={item.clientId}
									className={cn(
										"flex cursor-move items-center justify-between rounded-[22px] border px-4 py-4 transition-all",
										dragOverIndex === index
											? "border-[rgba(178,122,68,0.4)] bg-[rgba(178,122,68,0.1)]"
											: "border-[var(--line)] bg-white/70 hover:border-[rgba(178,122,68,0.28)] hover:bg-[rgba(178,122,68,0.08)]"
									)}
									draggable
									onDragStart={(event) => onDragStart(event, index)}
									onDragOver={(event) => onDragOver(event, index)}
									onDragLeave={onDragLeave}
									onDrop={(event) => onDrop(event, index)}
									onDragEnd={onDragEnd}
								>
									<span className="mr-3 truncate text-sm font-semibold text-[var(--text)]">
										{item.nomeColuna}
									</span>
									<svg
										className="h-4 w-4 shrink-0 text-[var(--accent-strong)]"
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
				</SectionCard>
			)}

			<StickyActionBar>
				<div>
					<div className="text-xs uppercase tracking-[0.18em] text-[var(--muted)]">
						Próxima etapa
					</div>
					<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
						Quando o mapeamento estiver consistente, avance para a revisão da entidade.
					</p>
				</div>
				<motion.div whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
					<PrimaryButton
						onClick={handleConfirm}
						className="px-8 py-4 text-base"
						data-testid="mapping-confirm-button"
					>
						{confirmLabel}
					</PrimaryButton>
				</motion.div>
			</StickyActionBar>
		</div>
	);
}
