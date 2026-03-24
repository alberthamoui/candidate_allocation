import { useEffect, useState } from "react";
import { PencilIcon } from "@heroicons/react/24/outline";
/* Enhanced EditableCell Component */
interface EditableCellProps {
	value: string | number | null;
	onChange: (v: string) => void;
	hasError?: boolean;
}
/* Enhanced EditableCell Component - Reduced padding */
export function EditableCell({
	value,
	onChange,
	hasError = false,
}: EditableCellProps) {
	const [editing, setEditing] = useState(false);
	const [temp, setTemp] = useState(String(value ?? ""));

	useEffect(() => {
		if (!editing) {
			setTemp(String(value ?? ""));
		}
	}, [editing, value]);

	function commit() {
		onChange(temp);
		setEditing(false);
	}

	function startEditing() {
		setTemp(String(value ?? ""));
		setEditing(true);
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
					onKeyDown={(e) => {
						e.stopPropagation();
						if (e.key === "Enter") {
							commit();
						}
						if (e.key === "Escape") {
							setTemp(String(value ?? ""));
							setEditing(false);
						}
					}}
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
			onClick={startEditing}
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
