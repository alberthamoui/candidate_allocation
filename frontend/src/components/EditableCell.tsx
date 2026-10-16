import { useState } from "react";
import { PencilIcon } from "@heroicons/react/24/outline";
import { cn } from "./ui";

interface EditableCellProps {
	value: string | number;
	onChange: (v: string) => void;
	hasError?: boolean;
}

// Valor que vira campo de texto ao clicar; confirma com Enter ou ao sair.
export function EditableCell({ value, onChange, hasError = false }: EditableCellProps) {
	const [editing, setEditing] = useState(false);
	const [temp, setTemp] = useState(String(value));

	function commit() {
		onChange(temp);
		setEditing(false);
	}

	if (editing) {
		return (
			<input
				className={cn("campo py-1", hasError && "border-[var(--danger)]")}
				value={temp}
				onChange={(e) => setTemp(e.target.value)}
				onBlur={commit}
				onKeyDown={(e) => {
					if (e.key === "Enter") commit();
					if (e.key === "Escape") {
						setTemp(String(value));
						setEditing(false);
					}
				}}
				autoFocus
			/>
		);
	}

	const vazio = value === "" || value === null || value === undefined;
	return (
		<button
			type="button"
			onClick={() => {
				setTemp(String(value ?? ""));
				setEditing(true);
			}}
			className={cn(
				"group flex w-full items-center justify-between gap-2 rounded-[var(--radius)] px-2 py-1 text-left text-sm transition-colors",
				hasError ? "bg-[var(--danger-soft)] text-[#a51818] hover:bg-[rgba(224,36,36,0.12)]" : "hover:bg-[var(--bg-hover)]"
			)}
		>
			<span className={cn("min-w-0 truncate", vazio && "italic text-[var(--text-light)]")}>{vazio ? "vazio" : String(value)}</span>
			<PencilIcon className="h-3 w-3 flex-shrink-0 text-[var(--text-light)] opacity-0 transition-opacity group-hover:opacity-100" />
		</button>
	);
}
