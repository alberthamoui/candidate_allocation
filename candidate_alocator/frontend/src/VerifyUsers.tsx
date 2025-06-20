import { useState } from "react";
import { motion } from "framer-motion";

interface ErrorItem {
	field: number;
	msg: string;
}

interface Usuario {
	[key: string]: any; // permite edição dinâmica de todos os campos
}

interface UserWrapper {
	erros: ErrorItem[];
	usuario: Usuario;
}

interface VerifyUserPageProps {
	usuarios: Record<number, UserWrapper>; // { "1": {…}, "2": {…} }
	duplicates: number[][]; // [[1,2,3], [4,5]]
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
	const [dupGroups, setDupGroups] = useState<number[][]>(duplicates); // mutável
	const [acceptedIds, setAcceptedIds] = useState<Set<number>>(new Set());
	const [errorMsg, setErrorMsg] = useState<string | null>(null);
	/* --------------------------------------------------------------------- */
	// Helpers --------------------------------------------------------------

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
	/* --------------------------------------------------------------------- */
	// Resolução de duplicatas ----------------------------------------------
	/* --------------------- AÇÕES DOS BOTÕES --------------------- */

	function acceptOne(group: number[], idAccepted: number) {
		// marca o escolhido como aceito
		setAcceptedIds((s) => new Set(s).add(idAccepted));
		// rejeita todos os demais do grupo
		const others = group.filter((id) => id !== idAccepted);

		setEditedUsers((prev) => {
			const nxt = { ...prev };
			others.forEach((id) => delete nxt[id]);
			return nxt;
		});
		// remove o grupo inteiro
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
		// aceita todos
		setAcceptedIds((s) => {
			const n = new Set(s);
			group.forEach((id) => n.add(id));
			return n;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
	}

	function rejectAll(group: number[]) {
		// remove do estado e exclui grupo
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

	const renderUserCard = (userId: number, extraBtn?: React.ReactNode) => {
		const user = editedUsers[userId];
		const errors = usuarios[userId]?.erros ?? [];

		return (
			<div
				key={userId}
				className="border rounded-lg shadow p-4 w-64 flex-shrink-0 bg-white"
			>
				<div className="space-y-1">
					{Object.entries(user).map(([field, val]) => (
						<div key={field} className="text-sm">
							<span className="font-semibold">{field}: </span>
							<EditableCell
								value={val}
								onChange={(v) =>
									handleCellChange(userId, field, v)
								}
							/>
						</div>
					))}
				</div>

				{/* erros */}
				{errors.length > 0 && (
					<div className="mt-2 text-xs text-red-600 space-y-0.5">
						{errors.map((e, idx) => (
							<p key={idx}>
								<strong>{e.field}</strong>: {e.msg}
							</p>
						))}
					</div>
				)}

				{/* botão extra (Aceitar este) */}
				{extraBtn && <div className="mt-3">{extraBtn}</div>}
			</div>
		);
	};

	/* ---------------------- JSX FINAL ---------------------- */
	return (
		<div className="min-h-screen bg-gradient-to-b from-blue-50 to-gray-100 flex flex-col items-center py-12 px-4">
			<>
				{/* pop-up de erro ao aceitar todos */}
				{errorMsg && (
					<div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50">
						<div className="bg-white p-4 rounded shadow-lg">
							<p className="mb-4 text-red-600">{errorMsg}</p>
							<button
								className="px-3 py-1 bg-blue-600 text-white rounded"
								onClick={() => setErrorMsg(null)}
							>
								Fechar
							</button>
						</div>
					</div>
				)}
				{/* ... restante do JSX ... */}
			</>
			<div className="max-w-7xl w-full space-y-10">
				{/* ---------- GRUPOS DUPLICADOS ---------- */}
				{dupGroups.map((group, idx) => (
					<div
						key={idx}
						className="border-2 border-red-400 rounded-lg shadow-lg bg-white"
					>
						{/* cabeçalho */}
						<div className="flex justify-between items-center bg-red-400 text-white px-6 py-3 rounded-t-lg">
							<h2 className="font-semibold">
								Grupo de Duplicados {idx + 1} (IDs:{" "}
								{group.join(", ")})
							</h2>
							<div className="space-x-2">
								<motion.button
									whileHover={{ scale: 1.05 }}
									whileTap={{ scale: 0.95 }}
									className="bg-green-600 px-3 py-1 rounded"
									onClick={() => acceptAll(group)}
								>
									Aceitar Todos
								</motion.button>
								<motion.button
									whileHover={{ scale: 1.05 }}
									whileTap={{ scale: 0.95 }}
									className="bg-gray-200 text-gray-800 px-3 py-1 rounded"
									onClick={() => rejectAll(group)}
								>
									Recusar Todos
								</motion.button>
							</div>
						</div>

						{/* cartões dos usuários duplicados */}
						<div className="flex flex-wrap gap-4 p-4">
							{group.map((id) =>
								renderUserCard(
									id,
									<motion.button
										whileHover={{ scale: 1.05 }}
										whileTap={{ scale: 0.95 }}
										className="bg-green-600 text-white w-full py-1 rounded"
										onClick={() => acceptOne(group, id)}
									>
										Aceitar este
									</motion.button>
								)
							)}
						</div>
					</div>
				))}

				{/* ---------- USUÁRIOS SEM DUPLICATAS ---------- */}
				<div className="border-2 border-blue-400 rounded-lg shadow-lg bg-white">
					<div className="bg-blue-600 text-white px-6 py-3 rounded-t-lg">
						<h2 className="font-semibold">
							Usuários sem Duplicatas
						</h2>
					</div>

					<div className="p-4 flex flex-wrap gap-4">
						{Object.keys(editedUsers)
							.map(Number)
							.filter((id) => !isDuplicate(id))
							.map((id) => renderUserCard(id))}
					</div>
				</div>
			</div>
		</div>
	);
}

/* ---------------------- Componente EditableCell ------------------------ */
interface EditableCellProps {
	value: string | number;
	onChange: (v: string) => void;
}

function EditableCell({ value, onChange }: EditableCellProps) {
	const [editing, setEditing] = useState(false);
	const [temp, setTemp] = useState(String(value));

	function commit() {
		onChange(temp);
		setEditing(false);
	}

	if (editing) {
		return (
			<input
				className="w-full border border-blue-300 rounded px-1 py-0.5"
				value={temp}
				onChange={(e) => setTemp(e.target.value)}
				onBlur={commit}
				onKeyDown={(e) => e.key === "Enter" && commit()}
				autoFocus
			/>
		);
	}

	return (
		<div
			className="cursor-pointer"
			onClick={() => setEditing(true)}
			title="Clique para editar"
		>
			{value === "" ? (
				<span className="text-gray-400">— vazio —</span>
			) : (
				value
			)}
		</div>
	);
}
