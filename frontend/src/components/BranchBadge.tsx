import { useEffect, useState } from "react";
import { getVersao, VersaoInfo } from "../api";

// Selo no canto da tela com a branch e o commit que o servidor está rodando.
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
			className={`fixed bottom-3 right-3 z-50 flex items-center gap-1.5 rounded-full border px-3 py-1 font-mono text-xs shadow-sm ${
				versao.desatualizado
					? "border-red-300 bg-red-50 text-red-700"
					: "border-gray-200 bg-white/90 text-gray-600"
			}`}
		>
			<span className="font-semibold">{versao.branch || "sem branch"}</span>
			{versao.commit && <span className="text-gray-400">{versao.commit}</span>}
			{versao.modificado && <span className="text-amber-600">*</span>}
			{versao.desatualizado && <span>⚠ recompile</span>}
		</div>
	);
}
