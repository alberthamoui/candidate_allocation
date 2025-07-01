import { useState } from "react";
import { motion } from "framer-motion";
import {
	PencilIcon,
	ExclamationTriangleIcon,
	CheckCircleIcon,
	XMarkIcon,
	TrashIcon,
} from "@heroicons/react/24/outline";
import { UserCard } from "./UserCard";

import { SaveUsuariosFromMaps } from "../wailsjs/go/main/App";

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
		// Check if there are still duplicates
		if (dupGroups.length > 0) {
			setErrorMsg(
				"Não é possível salvar enquanto houver usuários duplicados. Resolva todos os conflitos primeiro."
			);
			return;
		}

		// Convert editedUsers to the format expected by backend
		const usuariosParaSalvar = Object.entries(editedUsers).map(
			([id, user]) => ({
				timestamp: user.timestamp || "",
				nome: user.nome || "",
				cpf: user.cpf || "",
				numero: user.numero || "",
				semestre: user.semestre || "",
				curso: user.curso || "",
				email_insper: user.emailinsper || "",
				email_pessoal: user.emailpessoal || "",
				opcoes: user.opcoes || [],
			})
		);

		console.log("Usuários para salvar:", usuariosParaSalvar);
		SaveUsuariosFromMaps(usuariosParaSalvar);
	}

	const renderUserCard = (userId: number, extraBtn?: React.ReactNode) => {
		const user = editedUsers[userId];
		const errors = usuarios[userId]?.erros ?? [];

		return (
			<UserCard
				userId={userId}
				user={user}
				errors={errors}
				onDelete={deleteUser}
				onCellChange={handleCellChange}
				extraBtn={extraBtn}
			/>
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
						whileHover={{
							scale: dupGroups.length === 0 ? 1.02 : 1,
						}}
						whileTap={{ scale: dupGroups.length === 0 ? 0.98 : 1 }}
						onClick={saveCandidates}
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
							: "Salvar Candidatos"}
					</motion.button>
				</div>
			</div>
		</div>
	);
}
