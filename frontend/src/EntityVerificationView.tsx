import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import {
	ExclamationTriangleIcon,
	CheckCircleIcon,
	XMarkIcon,
} from "@heroicons/react/24/outline";
import { UserCard, type MapUsuario, type UserExtras } from "./UserCard";

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
		opcoes: Array.isArray(entity.opcoes) ? [...entity.opcoes] : undefined,
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
		<div className="min-h-screen bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50">
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
								Erro de Validação
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

			<div className="bg-white shadow-sm border-b">
				<div className="max-w-7xl mx-auto px-6 py-6">
					<div className="flex items-center justify-between">
						<div>
							<h1 className="text-3xl font-bold text-gray-900">
								{title}
							</h1>
							<p className="text-gray-600 mt-1">
								{subtitle}
							</p>
						</div>
						<div className="flex items-center space-x-4 text-sm">
							<div className="flex items-center space-x-2 bg-red-100 px-3 py-2 rounded-lg">
								<div className="w-3 h-3 bg-red-500 rounded-full"></div>
								<span>
									{dupGroups.length} grupos duplicados
								</span>
							</div>
							<div className="flex items-center space-x-2 bg-blue-100 px-3 py-2 rounded-lg">
								<div className="w-3 h-3 bg-blue-500 rounded-full"></div>
								<span>
									{
										Object.keys(editedEntities).filter(
											(id) => !isDuplicate(Number(id))
										).length
									}{" "}
									únicos
								</span>
							</div>
							{backRoute && (
								<button
									onClick={() => navigate(backRoute)}
									className="rounded-lg border border-gray-300 px-4 py-2 font-medium text-gray-700 transition-colors hover:bg-gray-50"
								>
									Voltar ao mapeamento
								</button>
							)}
						</div>
					</div>
				</div>
			</div>

			<div className="max-w-7xl mx-auto px-6 py-8 space-y-12">
				{dupGroups.length > 0 && (
					<div className="space-y-8">
						<h2 className="text-2xl font-bold text-gray-900 flex items-center space-x-3">
							<ExclamationTriangleIcon className="w-8 h-8 text-red-500" />
							<span>Grupos Duplicados - Ação Necessária</span>
						</h2>

						{dupGroups.map((group, idx) => (
							<motion.div
								key={idx}
								initial={{ opacity: 0, y: 20 }}
								animate={{ opacity: 1, y: 0 }}
								transition={{ delay: idx * 0.1 }}
								className="border-2 border-red-300 rounded-2xl shadow-xl bg-white overflow-hidden"
							>
								<div className="bg-gradient-to-r from-red-500 to-red-600 text-white px-8 py-6">
									<div className="flex justify-between items-center">
										<div>
											<h3 className="text-xl font-bold">
												Grupo Duplicado #{idx + 1}
											</h3>
											<p className="text-red-100 mt-1">
												IDs conflitantes:{" "}
												{group.join(", ")} • Escolha uma
												ação
											</p>
										</div>
										<div className="flex space-x-3">
											<motion.button
												whileHover={{ scale: 1.02 }}
												whileTap={{ scale: 0.98 }}
												className="bg-green-600 hover:bg-green-700 px-6 py-3 rounded-xl font-semibold transition-colors shadow-lg"
												onClick={() => acceptAll(group)}
											>
												<CheckCircleIcon className="w-5 h-5 inline mr-2" />
												Aceitar Todos
											</motion.button>
											<motion.button
												whileHover={{ scale: 1.02 }}
												whileTap={{ scale: 0.98 }}
												className="bg-gray-600 hover:bg-gray-700 px-6 py-3 rounded-xl font-semibold transition-colors shadow-lg"
												onClick={() => rejectAll(group)}
											>
												<XMarkIcon className="w-5 h-5 inline mr-2" />
												Recusar Todos
											</motion.button>
										</div>
									</div>
								</div>

								<div className="p-8">
									<div className="flex flex-wrap gap-6 justify-center">
										{group.map((id) =>
											renderEntityCard(
												id,
												<motion.button
													whileHover={{ scale: 1.02 }}
													whileTap={{ scale: 0.98 }}
													className="w-full bg-gradient-to-r from-green-600 to-green-700 text-white py-3 rounded-xl font-semibold shadow-lg hover:shadow-xl transition-all"
													onClick={() =>
														acceptOne(group, id)
													}
												>
													<CheckCircleIcon className="w-5 h-5 inline mr-2" />
													Aceitar Este
												</motion.button>
											)
										)}
									</div>
								</div>
							</motion.div>
						))}
					</div>
				)}

				<div className="space-y-6">
					<h2 className="text-2xl font-bold text-gray-900 flex items-center space-x-3">
						<CheckCircleIcon className="w-8 h-8 text-green-500" />
						<span>Registros Únicos</span>
					</h2>

					<div className="bg-white rounded-2xl shadow-lg border border-gray-200 overflow-hidden">
						<div className="bg-gradient-to-r from-blue-600 to-blue-700 text-white px-8 py-6">
							<h3 className="text-xl font-semibold">
								Dados Validados
							</h3>
							<p className="text-blue-100 mt-1">
								{
									Object.keys(editedEntities).filter(
										(id) => !isDuplicate(Number(id))
									).length
								}{" "}
								registros sem conflitos
							</p>
						</div>

						<div className="p-8">
							<div className="flex flex-wrap gap-6 justify-center">
								{Object.keys(editedEntities)
									.map(Number)
									.filter((id) => !isDuplicate(id))
									.map((id) => renderEntityCard(id))}
							</div>
						</div>
					</div>
				</div>
				<div className="flex justify-center pt-8">
					<motion.button
						whileHover={{
							scale: dupGroups.length === 0 ? 1.02 : 1,
						}}
						whileTap={{ scale: dupGroups.length === 0 ? 0.98 : 1 }}
						onClick={saveAll}
						disabled={dupGroups.length > 0}
						className={`px-12 py-4 rounded-xl font-bold text-lg shadow-xl transition-all ${
							dupGroups.length > 0
								? "bg-gray-400 text-gray-200 cursor-not-allowed"
								: "bg-gradient-to-r from-green-600 to-green-700 text-white hover:shadow-2xl cursor-pointer"
						}`}
					>
						<CheckCircleIcon className="w-6 h-6 inline mr-3" />
						{dupGroups.length > 0
							? `Resolva ${dupGroups.length} grupo${
									dupGroups.length > 1 ? "s" : ""
							  } duplicado${dupGroups.length > 1 ? "s" : ""}`
							: "Salvar Dados"}
					</motion.button>
				</div>
			</div>
		</div>
	);
}
