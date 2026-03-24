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
			className={`relative border-2 rounded-xl shadow-lg p-4 w-80 flex-shrink-0 transition-all duration-200 ${
				hasErrors
					? "border-red-300 bg-red-50 shadow-red-100"
					: "border-gray-200 bg-white hover:shadow-xl hover:border-blue-300"
			}`}
		>
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
						onClick={() => onDelete(userId)}
						className="p-1 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-full transition-colors"
						title="Deletar usuário"
					>
						<TrashIcon className="w-4 h-4" />
					</button>
				</div>
			</div>

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

			<div className="space-y-2">
				{coreEntries.map(([field, val]) => {
					const fieldError = errors.find((e) => e.field === field);
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
									{formatFieldLabel(field)}
								</span>
								{hasFieldError && (
									<ExclamationTriangleIcon className="w-3 h-3 text-red-500" />
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
						<span className="text-xs font-semibold uppercase tracking-wide text-gray-500">
							Campos extras
						</span>
						<button
							onClick={() => onAddExtraField(userId)}
							className="inline-flex items-center gap-1 rounded-lg bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 hover:bg-blue-100"
							title="Adicionar campo extra"
						>
							<PlusIcon className="h-3 w-3" />
							Adicionar
						</button>
					</div>

					{Object.entries(extras).length === 0 && (
						<div className="rounded-lg border border-dashed border-gray-300 bg-gray-50 px-3 py-2 text-xs text-gray-500">
							Nenhum campo extra adicionado.
						</div>
					)}

					{Object.entries(extras).map(([key, value]) => (
						<div
							key={key}
							className="rounded-lg border border-gray-200 bg-slate-50 p-2"
						>
							<div className="mb-2 flex items-center justify-between">
								<span className="text-xs font-semibold text-gray-600">
									Extra
								</span>
								<button
									onClick={() => onRemoveExtraField(userId, key)}
									className="p-1 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-full transition-colors"
									title="Remover campo extra"
								>
									<TrashIcon className="w-3 h-3" />
								</button>
							</div>

							<div className="space-y-2">
								<div>
									<div className="mb-1 text-[11px] font-semibold text-gray-600">
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
									<div className="mb-1 text-[11px] font-semibold text-gray-600">
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
				<div className="mt-4 pt-3 border-t border-gray-200">
					{extraBtn}
				</div>
			)}
		</motion.div>
	);
}
