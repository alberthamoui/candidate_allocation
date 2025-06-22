import { useState } from "react";
import { motion } from "framer-motion";
import {
	PencilIcon,
	ExclamationTriangleIcon,
	CheckCircleIcon,
	XMarkIcon,
	TrashIcon,
} from "@heroicons/react/24/outline";

interface ErrorItem {
	field: string;
	msg: string;
}

interface Usuario {
	[key: string]: any;
}

interface UserWrapper {
	erros: ErrorItem[];
	usuario: Usuario;
}

interface VerifyUserPageProps {
	usuarios: Record<number, UserWrapper>;
	duplicates: number[][];
}

export default function VerifyUserPage({
	usuarios,
	duplicates,
}: VerifyUserPageProps) {
	const makeEditableCopy = () =>
		Object.fromEntries(
			Object.entries(usuarios).map(([id, u]) => [id, { ...u.usuario }])
		);

	const [editedUsers, setEditedUsers] = useState<Record<number, Usuario>>(
		makeEditableCopy()
	);
	const [dupGroups, setDupGroups] = useState<number[][]>(duplicates);
	const [acceptedIds, setAcceptedIds] = useState<Set<number>>(new Set());
	const [errorMsg, setErrorMsg] = useState<string | null>(null);

	// ... existing helper functions ...
	const getGroup = (id: number) =>
		duplicates.find((g) => g.includes(id)) || [];

	function handleCellChange(
		userId: number,
		field: string,
		value: string | number
	) {
		setEditedUsers((prev) => ({
			...prev,
			[userId]: { ...prev[userId], [field]: value },
		}));
	}
	const flattenDup = () => dupGroups.flat();
	const isDuplicate = (id: number) => flattenDup().includes(id);

	// ... existing action functions ...
	function acceptOne(group: number[], idAccepted: number) {
		setAcceptedIds((s) => new Set(s).add(idAccepted));
		const others = group.filter((id) => id !== idAccepted);

		setEditedUsers((prev) => {
			const nxt = { ...prev };
			others.forEach((id) => delete nxt[id]);
			return nxt;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
	}

	function acceptAll(group: number[]) {
		const keys = ["cpf", "emailpessoal", "emailinsper"];
		const seen = new Map<string, number>();
		for (const id of group) {
			const usr = editedUsers[id];
			for (const k of keys) {
				const v = usr[k];
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
	function deleteUser(userId: number) {
		setEditedUsers((prev) => {
			const next = { ...prev };
			delete next[userId];
			return next;
		});
		setAcceptedIds((prev) => {
			const next = new Set(prev);
			next.delete(userId);
			return next;
		});
		// Remove from duplicate groups if exists
		setDupGroups((prev) =>
			prev
				.map((group) => group.filter((id) => id !== userId))
				.filter((group) => group.length > 1)
		);
	}

	function rejectAll(group: number[]) {
		const toRemove = new Set(group);
		setEditedUsers((prev) => {
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

	function saveCandidates() {
		console.log("salvei");
		// Here you can add the actual save logic
	}

	const renderUserCard = (userId: number, extraBtn?: React.ReactNode) => {
		const user = editedUsers[userId];
		const errors = usuarios[userId]?.erros ?? [];
		const hasErrors = errors.length > 0;

		return (
			<motion.div
				key={userId}
				initial={{ opacity: 0, y: 20 }}
				animate={{ opacity: 1, y: 0 }}
				className={`relative border-2 rounded-xl shadow-lg p-4 w-80 flex-shrink-0 transition-all duration-200 ${
					hasErrors
						? "border-red-300 bg-red-50 shadow-red-100"
						: "border-gray-200 bg-white hover:shadow-xl hover:border-blue-300"
				}`}
			>
				{/* Header with ID, error indicator, and delete button */}
				<div className="flex items-center justify-between mb-3">
					<div className="flex items-center space-x-2">
						<span className="text-sm font-bold text-gray-600">
							ID: {userId}
						</span>
						{hasErrors && (
							<div className="flex items-center space-x-1 bg-red-100 px-2 py-1 rounded-full">
								<ExclamationTriangleIcon className="w-4 h-4 text-red-600" />
								<span className="text-xs font-semibold text-red-600">
									{errors.length} erro
									{errors.length > 1 ? "s" : ""}
								</span>
							</div>
						)}
					</div>
					<div className="flex items-center space-x-2">
						<div className="flex items-center space-x-1 text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded-full">
							<PencilIcon className="w-3 h-3" />
							<span>Editável</span>
						</div>
						<button
							onClick={() => deleteUser(userId)}
							className="p-1 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-full transition-colors"
							title="Deletar usuário"
						>
							<TrashIcon className="w-4 h-4" />
						</button>
					</div>
				</div>

				{/* Error summary - Show errors at the top */}
				{hasErrors && (
					<div className="mb-3 p-2 bg-red-100 border border-red-200 rounded-lg">
						<div className="text-xs font-semibold text-red-700 mb-1">
							Erros encontrados:
						</div>
						<div className="space-y-1">
							{errors.map((error, idx) => (
								<div key={idx} className="text-xs text-red-600">
									<span className="font-medium">
										{error.field}:
									</span>{" "}
									{error.msg}
								</div>
							))}
						</div>
					</div>
				)}

				{/* User fields - Reduced spacing */}
				<div className="space-y-2">
					{Object.entries(user).map(([field, val]) => {
						const fieldError = errors.find(
							(e) => e.field === field
						);
						const hasFieldError = !!fieldError;

						return (
							<div
								key={field}
								className={`p-2 rounded-lg border transition-all duration-200 ${
									hasFieldError
										? "border-red-300 bg-red-50"
										: "border-gray-200 bg-gray-50 hover:bg-gray-100"
								}`}
							>
								<div className="flex items-center justify-between mb-1">
									<span className="text-xs font-semibold text-gray-700 capitalize">
										{field
											.replace(/([A-Z])/g, " $1")
											.replace(/^./, (str) =>
												str.toUpperCase()
											)}
									</span>
									{hasFieldError && (
										<ExclamationTriangleIcon className="w-3 h-3 text-red-500" />
									)}
								</div>

								<EditableCell
									value={val}
									onChange={(v) =>
										handleCellChange(userId, field, v)
									}
									hasError={hasFieldError}
								/>
							</div>
						);
					})}
				</div>

				{/* Action button */}
				{extraBtn && (
					<div className="mt-4 pt-3 border-t border-gray-200">
						{extraBtn}
					</div>
				)}
			</motion.div>
		);
	};

	return (
		<div className="min-h-screen bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50">
			{/* Error Modal */}
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

			{/* Header */}
			<div className="bg-white shadow-sm border-b">
				<div className="max-w-7xl mx-auto px-6 py-6">
					<div className="flex items-center justify-between">
						<div>
							<h1 className="text-3xl font-bold text-gray-900">
								Verificação de Usuários
							</h1>
							<p className="text-gray-600 mt-1">
								Revise e corrija os dados importados. Clique em
								qualquer campo para editar.
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
										Object.keys(editedUsers).filter(
											(id) => !isDuplicate(Number(id))
										).length
									}{" "}
									únicos
								</span>
							</div>
						</div>
					</div>
				</div>
			</div>

			<div className="max-w-7xl mx-auto px-6 py-8 space-y-12">
				{/* Duplicate Groups */}
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
								{/* Group header */}
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

								{/* User cards */}
								<div className="p-8">
									<div className="flex flex-wrap gap-6 justify-center">
										{group.map((id) =>
											renderUserCard(
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
													Aceitar Este Usuário
												</motion.button>
											)
										)}
									</div>
								</div>
							</motion.div>
						))}
					</div>
				)}

				{/* Non-duplicate users */}
				<div className="space-y-6">
					<h2 className="text-2xl font-bold text-gray-900 flex items-center space-x-3">
						<CheckCircleIcon className="w-8 h-8 text-green-500" />
						<span>Usuários Únicos</span>
					</h2>

					<div className="bg-white rounded-2xl shadow-lg border border-gray-200 overflow-hidden">
						<div className="bg-gradient-to-r from-blue-600 to-blue-700 text-white px-8 py-6">
							<h3 className="text-xl font-semibold">
								Dados Validados
							</h3>
							<p className="text-blue-100 mt-1">
								{
									Object.keys(editedUsers).filter(
										(id) => !isDuplicate(Number(id))
									).length
								}{" "}
								usuários sem conflitos
							</p>
						</div>

						<div className="p-8">
							<div className="flex flex-wrap gap-6 justify-center">
								{Object.keys(editedUsers)
									.map(Number)
									.filter((id) => !isDuplicate(id))
									.map((id) => renderUserCard(id))}
							</div>
						</div>
					</div>
				</div>
				{/* Save button at the bottom */}
				<div className="flex justify-center pt-8">
					<motion.button
						whileHover={{ scale: 1.02 }}
						whileTap={{ scale: 0.98 }}
						onClick={saveCandidates}
						className="bg-gradient-to-r from-green-600 to-green-700 text-white px-12 py-4 rounded-xl font-bold text-lg shadow-xl hover:shadow-2xl transition-all"
					>
						<CheckCircleIcon className="w-6 h-6 inline mr-3" />
						Salvar Candidatos
					</motion.button>
				</div>
			</div>
		</div>
	);
}

/* Enhanced EditableCell Component */
interface EditableCellProps {
	value: string | number;
	onChange: (v: string) => void;
	hasError?: boolean;
}
/* Enhanced EditableCell Component - Reduced padding */
function EditableCell({
	value,
	onChange,
	hasError = false,
}: EditableCellProps) {
	const [editing, setEditing] = useState(false);
	const [temp, setTemp] = useState(String(value));

	function commit() {
		onChange(temp);
		setEditing(false);
	}

	if (editing) {
		return (
			<div className="relative">
				<input
					className={`w-full border-2 rounded-lg px-2 py-1 text-sm focus:outline-none focus:ring-2 transition-all ${
						hasError
							? "border-red-300 focus:border-red-500 focus:ring-red-200"
							: "border-blue-300 focus:border-blue-500 focus:ring-blue-200"
					}`}
					value={temp}
					onChange={(e) => setTemp(e.target.value)}
					onBlur={commit}
					onKeyDown={(e) => e.key === "Enter" && commit()}
					autoFocus
				/>
			</div>
		);
	}

	return (
		<div
			className={`group cursor-pointer p-1 rounded-lg border-2 border-dashed transition-all hover:border-solid ${
				hasError
					? "border-red-300 hover:border-red-400 hover:bg-red-50"
					: "border-gray-300 hover:border-blue-400 hover:bg-blue-50"
			}`}
			onClick={() => setEditing(true)}
		>
			<div className="flex items-center justify-between">
				<span
					className={`text-sm ${
						value === "" ? "text-gray-400 italic" : "text-gray-800"
					}`}
				>
					{value === "" ? "Clique para adicionar..." : String(value)}
				</span>
				<PencilIcon className="w-3 h-3 text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity" />
			</div>
		</div>
	);
}
