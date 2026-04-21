import { motion } from "framer-motion";
import {
	PencilIcon,
	ExclamationTriangleIcon,
	TrashIcon,
	PlusIcon,
} from "@heroicons/react/24/outline";
import { EditableCell } from "./EditableCell";

interface ErrorItem {
	field: string;
	msg: string;
}

export interface UserExtras {
	[key: string]: string | null;
}

export interface MapUsuario {
	[key: string]: string | number | string[] | UserExtras | null | undefined;
	opcoes?: string[];
	extras?: UserExtras;
}

interface UserCardProps {
	userId: number;
	user: MapUsuario;
	errors: ErrorItem[];
	onDelete: (userId: number) => void;
	onCellChange: (userId: number, field: string, value: string) => void;
	onExtraKeyChange: (
		userId: number,
		currentKey: string,
		nextKey: string
	) => void;
	onExtraValueChange: (userId: number, key: string, value: string) => void;
	onAddExtraField: (userId: number) => void;
	onRemoveExtraField: (userId: number, key: string) => void;
	extraBtn?: React.ReactNode;
	allowExtras?: boolean;
}

function formatFieldLabel(field: string) {
	return field
		.replace(/_/g, " ")
		.replace(/([A-Z])/g, " $1")
		.replace(/\s+/g, " ")
		.trim()
		.replace(/^./, (str) => str.toUpperCase());
}

function displayValue(value: MapUsuario[string]) {
	if (Array.isArray(value)) {
		return value.join(", ");
	}
	if (typeof value === "number" || typeof value === "string") {
		return value;
	}
	if (value === null) {
		return "";
	}
	return "";
}

export function UserCard({
	userId,
	user,
	errors,
	onDelete,
	onCellChange,
	onExtraKeyChange,
	onExtraValueChange,
	onAddExtraField,
	onRemoveExtraField,
	extraBtn,
	allowExtras = true,
}: UserCardProps) {
	const hasErrors = errors.length > 0;
	const extras = user.extras ?? {};
	const coreEntries = Object.entries(user).filter(
		([field, value]) => field !== "extras" && value !== undefined
	);

	return (
		<motion.div
			key={userId}
			initial={{ opacity: 0, y: 20 }}
			animate={{ opacity: 1, y: 0 }}
			className={`relative w-80 flex-shrink-0 rounded-[28px] border p-5 shadow-[0_20px_36px_rgba(24,35,45,0.12)] transition-all duration-200 ${
				hasErrors
					? "border-[rgba(156,66,63,0.22)] bg-[rgba(156,66,63,0.08)]"
					: "border-[var(--line)] bg-white/78 hover:border-[rgba(178,122,68,0.24)] hover:shadow-[0_24px_42px_rgba(24,35,45,0.14)]"
			}`}
		>
			<div className="flex items-center justify-between mb-3">
				<div className="flex items-center space-x-2">
					<span className="text-sm font-bold text-[var(--muted)]">
						ID: {userId}
					</span>
					{hasErrors && (
						<div className="flex items-center space-x-1 rounded-full border border-[rgba(156,66,63,0.18)] bg-[rgba(156,66,63,0.12)] px-2 py-1">
							<ExclamationTriangleIcon className="w-4 h-4 text-[var(--danger)]" />
							<span className="text-xs font-semibold text-[var(--danger)]">
								{errors.length} erro
								{errors.length > 1 ? "s" : ""}
							</span>
						</div>
					)}
				</div>
				<div className="flex items-center space-x-2">
					<div className="flex items-center space-x-1 rounded-full border border-[var(--line)] bg-white/70 px-2 py-1 text-xs text-[var(--muted)]">
						<PencilIcon className="w-3 h-3" />
						<span>Editável</span>
					</div>
					<button
						onClick={() => onDelete(userId)}
						className="rounded-full p-1 text-[var(--danger)] transition-colors hover:bg-[rgba(156,66,63,0.12)]"
						title="Deletar usuário"
					>
						<TrashIcon className="w-4 h-4" />
					</button>
				</div>
			</div>

			{hasErrors && (
				<div className="mb-3 rounded-[18px] border border-[rgba(156,66,63,0.18)] bg-[rgba(156,66,63,0.1)] p-3">
					<div className="mb-1 text-xs font-semibold text-[var(--danger)]">
						Erros encontrados:
					</div>
					<div className="space-y-1">
						{errors.map((error, idx) => (
							<div key={idx} className="text-xs text-[var(--danger)]">
								<span className="font-medium">
									{error.field}:
								</span>{" "}
								{error.msg}
							</div>
						))}
					</div>
				</div>
			)}

			<div className="space-y-2">
				{coreEntries.map(([field, val]) => {
					const fieldError = errors.find((e) => e.field === field);
					const hasFieldError = !!fieldError;

					return (
						<div
							key={field}
							className={`rounded-[18px] border p-3 transition-all duration-200 ${
								hasFieldError
									? "border-[rgba(156,66,63,0.2)] bg-[rgba(156,66,63,0.08)]"
									: "border-[var(--line)] bg-[rgba(255,252,247,0.8)] hover:bg-white/80"
							}`}
						>
							<div className="flex items-center justify-between mb-1">
								<span className="text-xs font-semibold capitalize text-[var(--muted)]">
									{formatFieldLabel(field)}
								</span>
								{hasFieldError && (
									<ExclamationTriangleIcon className="w-3 h-3 text-[var(--danger)]" />
								)}
							</div>

							<EditableCell
								value={displayValue(val)}
								onChange={(v) => onCellChange(userId, field, v)}
								hasError={hasFieldError}
							/>
						</div>
					);
				})}
			</div>

			{allowExtras && (
				<div className="mt-4 space-y-2">
					<div className="flex items-center justify-between">
						<span className="text-xs font-semibold uppercase tracking-wide text-[var(--muted)]">
							Campos extras
						</span>
						<button
							onClick={() => onAddExtraField(userId)}
							className="inline-flex items-center gap-1 rounded-full border border-[var(--line)] bg-white/70 px-3 py-1 text-xs font-medium text-[var(--text)] hover:bg-white"
							title="Adicionar campo extra"
						>
							<PlusIcon className="h-3 w-3" />
							Adicionar
						</button>
					</div>

					{Object.entries(extras).length === 0 && (
						<div className="rounded-[18px] border border-dashed border-[var(--line-strong)] bg-white/60 px-3 py-3 text-xs text-[var(--muted)]">
							Nenhum campo extra adicionado.
						</div>
					)}

					{Object.entries(extras).map(([key, value]) => (
						<div
							key={key}
							className="rounded-[18px] border border-[var(--line)] bg-[rgba(255,252,247,0.82)] p-3"
						>
							<div className="mb-2 flex items-center justify-between">
								<span className="text-xs font-semibold text-[var(--muted)]">
									Extra
								</span>
								<button
									onClick={() => onRemoveExtraField(userId, key)}
									className="rounded-full p-1 text-[var(--danger)] transition-colors hover:bg-[rgba(156,66,63,0.12)]"
									title="Remover campo extra"
								>
									<TrashIcon className="w-3 h-3" />
								</button>
							</div>

							<div className="space-y-2">
								<div>
									<div className="mb-1 text-[11px] font-semibold text-[var(--muted)]">
										Chave
									</div>
									<EditableCell
										value={key}
										onChange={(nextKey) =>
											onExtraKeyChange(userId, key, nextKey)
										}
									/>
								</div>
								<div>
									<div className="mb-1 text-[11px] font-semibold text-[var(--muted)]">
										Valor
									</div>
									<EditableCell
										value={value}
										onChange={(nextValue) =>
											onExtraValueChange(userId, key, nextValue)
										}
									/>
								</div>
							</div>
						</div>
					))}
				</div>
			)}

			{extraBtn && (
				<div className="mt-4 border-t border-[var(--line)] pt-3">
					{extraBtn}
				</div>
			)}
		</motion.div>
	);
}
