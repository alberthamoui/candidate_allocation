import { useEffect, useState } from "react";
import { Link, Outlet, useLocation } from "react-router-dom";
import { XMarkIcon } from "@heroicons/react/24/outline";
import { ETAPAS, indiceEtapa, infoPagina } from "../fluxo";
import { BranchBadge } from "./BranchBadge";
import { cn, Selo } from "./ui";

// Moldura de todas as telas: barra fixa no topo (etapa, título, ajuda) e
// linha de etapas fixa no rodapé. Inspirada no workflowShell da mainjoao.
export default function Layout() {
	const { pathname } = useLocation();
	const info = infoPagina(pathname);
	const etapa = indiceEtapa(pathname);
	const [ajudaAberta, setAjudaAberta] = useState(false);

	useEffect(() => {
		window.scrollTo(0, 0);
		setAjudaAberta(false);
	}, [pathname]);

	useEffect(() => {
		const fechar = (e: KeyboardEvent) => e.key === "Escape" && setAjudaAberta(false);
		window.addEventListener("keydown", fechar);
		return () => window.removeEventListener("keydown", fechar);
	}, []);

	return (
		<div className="flex min-h-screen flex-col bg-[var(--bg-subtle)]">
			<header className="sticky top-0 z-40 border-b border-[var(--line)] bg-white/85 backdrop-blur-md">
				<div className="mx-auto flex w-full max-w-[1200px] items-center justify-between gap-3 px-4 py-3 md:px-8">
					<div className="flex min-w-0 items-center gap-3">
						<Link to="/" className="text-sm font-semibold tracking-tight whitespace-nowrap hover:text-[var(--accent)]">
							Candidate Allocator
						</Link>
						{etapa >= 0 && (
							<>
								<span className="text-[var(--text-light)]">/</span>
								<Selo>
									Etapa {etapa + 1} de {ETAPAS.length}
								</Selo>
								<span className="truncate text-sm text-[var(--text-muted)]" data-testid="titulo-pagina">
									{info.titulo}
								</span>
							</>
						)}
					</div>
					<div className="flex items-center gap-3">
						<BranchBadge />
						<button
							type="button"
							onClick={() => setAjudaAberta(true)}
							aria-label="Ajuda desta tela"
							className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-[var(--line-strong)] bg-white text-sm font-semibold text-[var(--text-muted)] shadow-sm transition-colors hover:bg-[var(--text)] hover:text-white"
						>
							?
						</button>
					</div>
				</div>
			</header>

			<main className="mx-auto w-full max-w-[1200px] flex-1 px-4 pb-20 pt-8 md:px-8">
				<Outlet />
			</main>

			<nav aria-label="Etapas" className="fixed inset-x-0 bottom-0 z-40 border-t border-[var(--line)] bg-white/95 backdrop-blur-md">
				<ol className="mx-auto flex h-12 w-full max-w-[1200px] md:px-8">
					{ETAPAS.map((e, i) => (
						<li
							key={e.chave}
							aria-current={i === etapa ? "step" : undefined}
							className={cn(
								"flex flex-1 items-center justify-center gap-1.5 truncate border-r border-[var(--line)] px-2 text-xs font-medium last:border-r-0",
								i === etapa
									? "bg-[var(--accent-soft)] font-semibold text-[var(--accent)]"
									: i < etapa
										? "text-[var(--text)]"
										: "text-[var(--text-light)]"
							)}
						>
							<span className="hidden opacity-60 md:inline">{i < etapa ? "✓" : `${i + 1}.`}</span>
							<span className="truncate">{e.rotulo}</span>
						</li>
					))}
				</ol>
			</nav>

			{ajudaAberta && (
				<div
					className="fixed inset-0 z-[60] flex items-center justify-center bg-black/20 p-4 backdrop-blur-sm"
					onClick={() => setAjudaAberta(false)}
				>
					<div
						role="dialog"
						aria-label={`Ajuda: ${info.titulo}`}
						className="painel max-h-[85vh] w-full max-w-2xl overflow-y-auto p-8 shadow-[var(--shadow-lg)]"
						onClick={(e) => e.stopPropagation()}
					>
						<div className="mb-5 flex items-start justify-between gap-4 border-b border-[var(--line)] pb-4">
							<div>
								<div className="rotulo-caps">Ajuda</div>
								<h2 className="mt-1 text-xl font-semibold">{info.titulo}</h2>
							</div>
							<button
								type="button"
								onClick={() => setAjudaAberta(false)}
								aria-label="Fechar ajuda"
								className="rounded-md p-1.5 text-[var(--text-muted)] hover:bg-[var(--bg-hover)]"
							>
								<XMarkIcon className="h-5 w-5" />
							</button>
						</div>
						<p className="text-base leading-7">{info.ajuda.resumo}</p>
						<dl className="mt-5 space-y-3 rounded-[var(--radius-lg)] bg-[var(--bg-subtle)] p-4 text-sm leading-6">
							{info.ajuda.topicos.map((t) => (
								<div key={t.titulo}>
									<dt className="font-semibold">{t.titulo}</dt>
									<dd className="text-[var(--text-muted)]">{t.texto}</dd>
								</div>
							))}
						</dl>
					</div>
				</div>
			)}
		</div>
	);
}
