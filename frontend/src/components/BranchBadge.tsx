import { useEffect, useState } from "react";
import { getVersao, VersaoInfo } from "../api";

// Selo (no topo da tela) com a branch e o commit que o servidor está rodando.
export function BranchBadge() {
	const [versao, setVersao] = useState<VersaoInfo | null>(null);

	useEffect(() => {
		getVersao().then(setVersao).catch(() => setVersao(null));
	}, []);

	if (!versao || (!versao.branch && !versao.commit)) return null;

	const avisos = [
		versao.modificado && "compilado com alterações não commitadas",
		versao.desatualizado && "binário desatualizado: o repositório está em outro commit, recompile",
	].filter(Boolean);

	return (
		<div
			title={avisos.length ? avisos.join("\n") : "Branch e commit do servidor"}
			data-testid="selo-versao"
			className={`hidden sm:flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 font-mono text-[11px] ${
				versao.desatualizado
					? "border-red-300 bg-red-50 text-red-700"
					: "border-[var(--line)] bg-[var(--bg-subtle)] text-[var(--text-muted)]"
			}`}
		>
			<span className="font-semibold">{versao.branch || "sem branch"}</span>
			{versao.commit && <span className="text-gray-400">{versao.commit}</span>}
			{versao.modificado && <span className="text-amber-600">*</span>}
			{versao.desatualizado && <span>⚠ recompile</span>}
		</div>
	);
}
