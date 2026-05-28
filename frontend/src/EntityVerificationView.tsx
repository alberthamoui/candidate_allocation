import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import {
	ExclamationTriangleIcon,
	CheckCircleIcon,
	XMarkIcon,
} from "@heroicons/react/24/outline";
import { UserCard, type MapUsuario, type UserExtras } from "./UserCard";
import {
	EmptyState,
	MetricPill,
	PrimaryButton,
	SecondaryButton,
	SectionCard,
	StatusBadge,
	StickyActionBar,
} from "./workflowShell";

interface EntityVerificationViewProps {
	title: string;
	subtitle: string;
	entities?: Record<number, any> | null;
	entityKey: string;
	duplicates: number[][];
	duplicateFields: string[];
	onSave: (entities: any[]) => Promise<void>;
	nextRoute: string;
	backRoute?: string;
	allowExtras?: boolean;
}

export default function EntityVerificationView({
	title,
	subtitle,
	entities,
	entityKey,
	duplicates,
	duplicateFields,
	onSave,
	nextRoute,
	backRoute,
	allowExtras = true,
}: EntityVerificationViewProps) {
	const navigate = useNavigate();
	const sourceEntities = entities ?? {};

	const cloneExtras = (extras: unknown): UserExtras => {
		if (!extras || typeof extras !== "object" || Array.isArray(extras)) {
			return {};
		}

		return Object.fromEntries(
			Object.entries(extras as Record<string, unknown>).map(([key, value]) => [
				key,
				value === null ? null : String(value ?? ""),
			])
		);
	};

	const cloneEntity = (entity: MapUsuario): MapUsuario => ({
		...entity,
		...(Array.isArray(entity.opcoes) ? { opcoes: [...entity.opcoes] } : {}),
		extras: cloneExtras(entity.extras),
	});

	const makeEditableCopy = () =>
		Object.fromEntries(
			Object.entries(sourceEntities).map(([id, wrapper]) => [
				id,
				cloneEntity(wrapper[entityKey]),
			])
		);

	const [editedEntities, setEditedEntities] = useState<Record<number, MapUsuario>>(
		makeEditableCopy()
	);
	const [dupGroups, setDupGroups] = useState<number[][]>(duplicates);
	const [acceptedIds, setAcceptedIds] = useState<Set<number>>(new Set());
	const [errorMsg, setErrorMsg] = useState<string | null>(null);

	const flattenDup = () => dupGroups.flat();
	const isDuplicate = (id: number) => flattenDup().includes(id);

	const coreKeysForEntity = (entity: MapUsuario) =>
		Object.keys(entity).filter((key) => key !== "extras" && key !== "id");

	const normalizeExtraKey = (raw: string) =>
		raw
			.normalize("NFD")
			.replace(/[\u0300-\u036f]/g, "")
			.trim()
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, "_")
			.replace(/^_+|_+$/g, "");

	const parseOpcoes = (value: string) =>
		value
			.split(",")
			.map((item) => item.trim())
			.filter(Boolean);

	const makeUniqueExtraKey = (entity: MapUsuario) => {
		const extras = entity.extras ?? {};
		const base = "novo_campo";
		if (!(base in extras)) {
			return base;
		}

		let counter = 2;
		for (;;) {
			const key = `${base}_${counter}`;
			if (!(key in extras)) {
				return key;
			}
			counter++;
		}
	};

	function handleCellChange(
		entityId: number,
		field: string,
		value: string | number
	) {
		setEditedEntities((prev) => {
			const entity = prev[entityId];
			if (!entity) {
				return prev;
			}

			const nextValue =
				field === "opcoes" ? parseOpcoes(String(value)) : value;

			return {
				...prev,
				[entityId]: { ...entity, [field]: nextValue },
			};
		});
	}

	function handleExtraValueChange(entityId: number, key: string, value: string) {
		setEditedEntities((prev) => {
			const entity = prev[entityId];
			if (!entity) {
				return prev;
			}

			return {
				...prev,
				[entityId]: {
					...entity,
					extras: {
						...(entity.extras ?? {}),
						[key]: value,
					},
				},
			};
		});
	}

	function handleExtraKeyChange(
		entityId: number,
		currentKey: string,
		nextKey: string
	) {
		setEditedEntities((prev) => {
			const entity = prev[entityId];
			if (!entity) {
				return prev;
			}

			const normalizedKey = normalizeExtraKey(nextKey);
			if (!normalizedKey) {
				setErrorMsg("A chave do campo extra não pode ficar vazia.");
				return prev;
			}
			if (coreKeysForEntity(entity).includes(normalizedKey)) {
				setErrorMsg(
					`A chave "${normalizedKey}" conflita com um campo principal.`
				);
				return prev;
			}

			const extras = { ...(entity.extras ?? {}) };
			if (normalizedKey !== currentKey && normalizedKey in extras) {
				setErrorMsg(
					`Já existe um campo extra com a chave "${normalizedKey}".`
				);
				return prev;
			}

			const currentValue = currentKey in extras ? extras[currentKey] : "";
			delete extras[currentKey];
			extras[normalizedKey] = currentValue;

			return {
				...prev,
				[entityId]: {
					...entity,
					extras,
				},
			};
		});
	}

	function addExtraField(entityId: number) {
		setEditedEntities((prev) => {
			const entity = prev[entityId];
			if (!entity) {
				return prev;
			}

			const nextKey = makeUniqueExtraKey(entity);
			return {
				...prev,
				[entityId]: {
					...entity,
					extras: {
						...(entity.extras ?? {}),
						[nextKey]: "",
					},
				},
			};
		});
	}

	function removeExtraField(entityId: number, key: string) {
		setEditedEntities((prev) => {
			const entity = prev[entityId];
			if (!entity) {
				return prev;
			}

			const extras = { ...(entity.extras ?? {}) };
			delete extras[key];

			return {
				...prev,
				[entityId]: {
					...entity,
					extras,
				},
			};
		});
	}

	const duplicateKeys = duplicateFields;
	const uniqueCount = Object.keys(editedEntities)
		.map(Number)
		.filter((id) => !isDuplicate(id)).length;

	function acceptOne(group: number[], idAccepted: number) {
		setAcceptedIds((s) => new Set(s).add(idAccepted));
		const others = group.filter((id) => id !== idAccepted);

		setEditedEntities((prev) => {
			const nxt = { ...prev };
			others.forEach((id) => delete nxt[id]);
			return nxt;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
	}

	function acceptAll(group: number[]) {
		const seen = new Map<string, number>();
		for (const id of group) {
			const ent = editedEntities[id];
			for (const k of duplicateKeys) {
				const v = ent[k];
				if (v && seen.has(`${k}_${v}`)) {
					setErrorMsg(
						`Não é possível aceitar todos: campo "${k}" duplicado entre IDs ${seen.get(
							`${k}_${v}`
						)} e ${id}`
					);
					return;
				}
				seen.set(`${k}_${v}`, id);
			}
		}
		setAcceptedIds((s) => {
			const n = new Set(s);
			group.forEach((id) => n.add(id));
			return n;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
	}

	function deleteEntity(entityId: number) {
		setEditedEntities((prev) => {
			const next = { ...prev };
			delete next[entityId];
			return next;
		});
		setAcceptedIds((prev) => {
			const next = new Set(prev);
			next.delete(entityId);
			return next;
		});
		setDupGroups((prev) =>
			prev
				.map((group) => group.filter((id) => id !== entityId))
				.filter((group) => group.length > 1)
		);
	}

	function rejectAll(group: number[]) {
		setEditedEntities((prev) => {
			const n = { ...prev };
			group.forEach((id) => delete n[id]);
			return n;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
		setAcceptedIds((s) => {
			const n = new Set(s);
			group.forEach((id) => n.delete(id));
			return n;
		});
	}

	const sanitizeExtras = (entity: MapUsuario) =>
		Object.fromEntries(
			Object.entries(entity.extras ?? {})
				.map(([key, value]) => [
					normalizeExtraKey(key),
					value === null ? null : String(value ?? ""),
				])
				.filter(([key]) => {
					if (!key) {
						return false;
					}
					return !coreKeysForEntity(entity).includes(key);
				})
		);

	async function saveAll() {
		if (dupGroups.length > 0) {
			setErrorMsg(
				"Não é possível salvar enquanto houver duplicados. Resolva todos os conflitos primeiro."
			);
			return;
		}

		const dataToSave = Object.values(editedEntities).map((ent) => ({
			...(allowExtras
				? {
						...ent,
						extras: sanitizeExtras(ent),
				  }
				: (() => {
						const { extras, ...rest } = ent;
						return rest;
				  })()),
		}));

		try {
			await onSave(dataToSave);
			navigate(nextRoute);
		} catch (err) {
			setErrorMsg("Erro ao salvar dados: " + (err as Error).message);
		}
	}

	const renderEntityCard = (entityId: number, extraBtn?: React.ReactNode) => {
		const entity = editedEntities[entityId];
		const errors = sourceEntities[entityId]?.erros ?? [];

		return (
			<UserCard
				userId={entityId}
				user={entity}
				errors={errors}
				onDelete={deleteEntity}
				onCellChange={handleCellChange}
				onExtraKeyChange={handleExtraKeyChange}
				onExtraValueChange={handleExtraValueChange}
				onAddExtraField={addExtraField}
				onRemoveExtraField={removeExtraField}
				extraBtn={extraBtn}
				allowExtras={allowExtras}
			/>
		);
	};

	return (
		<div className="space-y-6">
			{errorMsg && (
				<div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50" data-testid="verification-error-dialog">
					<motion.div
						initial={{ scale: 0.9, opacity: 0 }}
						animate={{ scale: 1, opacity: 1 }}
						className="executive-card executive-card-strong mx-4 max-w-md p-6"
					>
						<div className="flex items-center space-x-3 mb-4">
							<ExclamationTriangleIcon className="w-6 h-6 text-[var(--danger)]" />
							<h3 className="font-semibold text-[var(--text)]">
								Erro de Validação
							</h3>
						</div>
						<p className="mb-6 text-sm leading-6 text-[var(--muted)]">{errorMsg}</p>
						<PrimaryButton className="w-full justify-center" onClick={() => setErrorMsg(null)} data-testid="verification-error-close-button">
							Entendido
						</PrimaryButton>
					</motion.div>
				</div>
			)}

			<SectionCard
				title={title}
				description={subtitle}
				aside={
					<div className="flex flex-wrap gap-3">
						<StatusBadge tone={dupGroups.length > 0 ? "danger" : "success"}>
							{dupGroups.length} grupos duplicados
						</StatusBadge>
						<StatusBadge tone="accent">{uniqueCount} registros únicos</StatusBadge>
						{backRoute ? (
							<SecondaryButton onClick={() => navigate(backRoute)}>
								Voltar ao mapeamento
							</SecondaryButton>
						) : null}
					</div>
				}
			>
				<div className="grid gap-4 md:grid-cols-3">
					<MetricPill label="Conflitos em aberto" value={dupGroups.length} tone={dupGroups.length > 0 ? "danger" : "success"} />
					<MetricPill label="Registros prontos" value={uniqueCount} tone="accent" />
					<MetricPill label="Entidade" value={title.replace("Verificação de ", "")} />
				</div>
			</SectionCard>

			{dupGroups.length > 0 ? (
				<SectionCard
					title="Grupos Duplicados"
					description="Esses conjuntos exigem decisão explícita antes do salvamento. O sistema não persiste registros ambíguos."
				>
					<div className="space-y-6">
						{dupGroups.map((group, idx) => (
							<motion.div
								key={idx}
								initial={{ opacity: 0, y: 20 }}
								animate={{ opacity: 1, y: 0 }}
								transition={{ delay: idx * 0.08 }}
								className="overflow-hidden rounded-[28px] border border-[rgba(156,66,63,0.18)] bg-[rgba(156,66,63,0.08)]"
							>
								<div className="border-b border-[rgba(156,66,63,0.16)] px-6 py-5">
									<div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
										<div>
											<div className="text-xs uppercase tracking-[0.18em] text-[var(--danger)]">
												Ação necessária
											</div>
											<h2 className="mt-2 text-2xl text-[var(--text)]">Grupo Duplicado #{idx + 1}</h2>
											<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
												IDs conflitantes: {group.join(", ")}. Escolha um registro, aceite todos quando o conflito não for real ou remova o grupo inteiro.
											</p>
										</div>
										<div className="flex flex-wrap gap-3">
											<PrimaryButton
												className="bg-[linear-gradient(135deg,#256454_0%,#184c40_100%)]"
												onClick={() => acceptAll(group)}
											>
												<CheckCircleIcon className="h-5 w-5" />
												Aceitar Todos
											</PrimaryButton>
											<SecondaryButton onClick={() => rejectAll(group)}>
												<XMarkIcon className="h-5 w-5" />
												Recusar Todos
											</SecondaryButton>
										</div>
									</div>
								</div>
								<div className="p-6">
									<div className="flex flex-wrap justify-center gap-6">
										{group.map((id) =>
											renderEntityCard(
												id,
												<PrimaryButton
													className="w-full justify-center bg-[linear-gradient(135deg,#256454_0%,#184c40_100%)]"
													onClick={() => acceptOne(group, id)}
													data-testid="accept-duplicate-record-button"
												>
													<CheckCircleIcon className="h-5 w-5" />
													Aceitar Este
												</PrimaryButton>
											)
										)}
									</div>
								</div>
							</motion.div>
						))}
					</div>
				</SectionCard>
			) : (
				<SectionCard
					title="Conflitos resolvidos"
					description="Nenhum grupo duplicado está bloqueando o salvamento nesta etapa."
				>
					<EmptyState
						title="Base pronta para persistência"
						description="Todos os conflitos desta tela foram resolvidos. Revise os registros únicos abaixo e avance para a próxima entidade quando estiver satisfeito."
					/>
				</SectionCard>
			)}

			<SectionCard
				title="Registros Únicos"
				description="Os cartões abaixo representam os dados prontos ou já validados para esta entidade."
			>
				{uniqueCount === 0 ? (
					<EmptyState
						title="Nenhum registro único disponível"
						description="Enquanto houver apenas grupos conflitantes, esta área permanecerá vazia. Resolva os duplicados para liberar registros prontos."
					/>
				) : (
					<div className="flex flex-wrap justify-center gap-6">
						{Object.keys(editedEntities)
							.map(Number)
							.filter((id) => !isDuplicate(id))
							.map((id) => renderEntityCard(id))}
					</div>
				)}
			</SectionCard>

			<StickyActionBar>
				<div>
					<div className="text-xs uppercase tracking-[0.18em] text-[var(--muted)]">
						Estado do salvamento
					</div>
					<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
						{dupGroups.length > 0
							? "Resolva todos os grupos duplicados antes de persistir esta entidade."
							: "Sem conflitos pendentes. Os dados podem ser salvos e o fluxo seguirá para a próxima etapa."}
					</p>
				</div>
				<motion.div whileHover={{ scale: dupGroups.length === 0 ? 1.02 : 1 }} whileTap={{ scale: dupGroups.length === 0 ? 0.98 : 1 }}>
					<PrimaryButton
						onClick={saveAll}
						disabled={dupGroups.length > 0}
						className={`px-8 py-4 text-base ${dupGroups.length > 0 ? "cursor-not-allowed opacity-50" : ""}`}
						data-testid="verification-save-button"
					>
						<CheckCircleIcon className="h-5 w-5" />
						{dupGroups.length > 0
							? `Resolva ${dupGroups.length} grupo${
									dupGroups.length > 1 ? "s" : ""
							  } duplicado${dupGroups.length > 1 ? "s" : ""}`
							: "Salvar Dados"}
					</PrimaryButton>
				</motion.div>
			</StickyActionBar>
		</div>
	);
}
