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
					className={`executive-input px-3 py-2 text-sm ${
						hasError
							? "border-[rgba(156,66,63,0.3)] bg-[rgba(156,66,63,0.06)]"
							: "border-[rgba(178,122,68,0.24)] bg-white"
					}`}
					data-testid="editable-cell-input"
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
			data-testid="editable-cell-display"
			className={`group cursor-pointer rounded-[14px] border border-dashed p-2 transition-all ${
				hasError
					? "border-[rgba(156,66,63,0.28)] hover:bg-[rgba(156,66,63,0.08)]"
					: "border-[var(--line-strong)] hover:border-[rgba(178,122,68,0.34)] hover:bg-[rgba(178,122,68,0.08)]"
			}`}
			onClick={startEditing}
		>
			<div className="flex items-center justify-between">
				<span
					className={`text-sm ${
						value === "" ? "italic text-[var(--muted)]" : "text-[var(--text)]"
					}`}
				>
					{value === "" ? "Clique para adicionar..." : String(value)}
				</span>
				<PencilIcon className="w-3 h-3 text-[var(--accent-strong)] opacity-0 transition-opacity group-hover:opacity-100" />
			</div>
		</div>
	);
}
