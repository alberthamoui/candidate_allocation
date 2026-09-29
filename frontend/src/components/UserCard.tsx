import { ReactNode } from "react";
import { ExclamationTriangleIcon, TrashIcon } from "@heroicons/react/24/outline";
import { EditableCell } from "./EditableCell";
import { cn, Selo } from "./ui";
import { nomeCampo } from "../nomesCampos";

interface ErrorItem {
	field: string;
	msg: string;
}
interface UserCardProps {
	userId: number;
	user: Record<string, any>;
	errors: ErrorItem[];
	onDelete: (userId: number) => void;
	onCellChange: (userId: number, field: string, value: string) => void;
	extraBtn?: ReactNode;
	destaque?: boolean; // ex.: faz parte de um grupo de duplicados
}

// Cartão de um candidato na revisão: campos editáveis, erros e opções de horário.
export function UserCard({ userId, user, errors, onDelete, onCellChange, extraBtn, destaque }: UserCardProps) {
	const temErro = errors.length > 0;
	const opcoes: string[] = Array.isArray(user.opcoes) ? user.opcoes.filter(Boolean) : [];
	const campos = Object.entries(user).filter(([campo]) => campo !== "opcoes" && campo !== "nome");

	return (
		<div
			data-testid="cartao-candidato"
			className={cn(
				"painel flex flex-col p-4",
				temErro && "border-[rgba(212,32,32,0.3)]",
				destaque && "border-[rgba(245,166,35,0.5)] shadow-[var(--shadow-md)]"
			)}
		>
			<div className="flex items-start justify-between gap-2">
				<div className="min-w-0 flex-1">
					<div className="font-mono text-[11px] text-[var(--text-light)]">#{userId}</div>
					<div className="-mx-2 font-medium">
						<EditableCell value={user.nome ?? ""} onChange={(v) => onCellChange(userId, "nome", v)} hasError={errors.some((e) => e.field === "nome")} />
					</div>
				</div>
				<div className="flex items-center gap-1">
					{temErro && (
						<Selo tom="perigo">
							<ExclamationTriangleIcon className="h-3 w-3" />
							{errors.length} erro{errors.length > 1 ? "s" : ""}
						</Selo>
					)}
					<button
						type="button"
						onClick={() => onDelete(userId)}
						className="rounded-md p-1 text-[var(--text-light)] transition-colors hover:bg-[var(--danger-soft)] hover:text-[var(--danger)]"
						title="Remover candidato"
						aria-label={`Remover candidato ${userId}`}
					>
						<TrashIcon className="h-4 w-4" />
					</button>
				</div>
			</div>

			{temErro && (
				<ul className="mt-2 space-y-0.5 rounded-[var(--radius)] bg-[var(--danger-soft)] px-2.5 py-2 text-xs text-[#a51818]">
					{errors.map((e, i) => (
						<li key={i}>
							<b>{nomeCampo(e.field)}:</b> {e.msg}
						</li>
					))}
				</ul>
			)}

			<dl className="mt-3 space-y-1">
				{campos.map(([campo, valor]) => (
					<div key={campo} className="grid grid-cols-[96px_1fr] items-center gap-2">
						<dt className="truncate text-xs text-[var(--text-muted)]" title={nomeCampo(campo)}>
							{nomeCampo(campo)}
						</dt>
						<dd className="min-w-0">
							<EditableCell value={valor ?? ""} onChange={(v) => onCellChange(userId, campo, v)} hasError={errors.some((e) => e.field === campo)} />
						</dd>
					</div>
				))}
			</dl>

			{opcoes.length > 0 && (
				<div className="mt-3 border-t border-[var(--line)] pt-2">
					<div className="rotulo-caps mb-1">Horários escolhidos</div>
					<ol className="space-y-0.5 text-xs text-[var(--text-muted)]">
						{opcoes.map((o, i) => (
							<li key={i} className="capitalize">
								<span className="text-[var(--text-light)]">{i + 1}.</span> {o}
							</li>
						))}
					</ol>
				</div>
			)}

			{extraBtn && <div className="mt-4 border-t border-[var(--line)] pt-3">{extraBtn}</div>}
		</div>
	);
}
