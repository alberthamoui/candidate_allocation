import { motion } from "framer-motion";
import {
	PencilIcon,
	ExclamationTriangleIcon,
	TrashIcon,
} from "@heroicons/react/24/outline";
import { EditableCell } from "./EditableCell";

interface ErrorItem {
	field: string;
	msg: string;
}
interface MapUsuario {
	[key: string]: string | number;
}
interface UserCardProps {
	userId: number;
	user: MapUsuario;
	errors: ErrorItem[];
	onDelete: (userId: number) => void;
	onCellChange: (userId: number, field: string, value: string) => void;
	extraBtn?: React.ReactNode;
}

export function UserCard({
	userId,
	user,
	errors,
	onDelete,
	onCellChange,
	extraBtn,
}: UserCardProps) {
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
						onClick={() => onDelete(userId)}
						className="p-1 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-full transition-colors"
						title="Deletar usuário"
					>
						<TrashIcon className="w-4 h-4" />
					</button>
				</div>
			</div>

			{/* Error summary */}
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

			{/* User fields */}
			<div className="space-y-2">
				{Object.entries(user).map(([field, val]) => {
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
								onChange={(v) => onCellChange(userId, field, v)}
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
}
