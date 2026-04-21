import { Fragment, type ButtonHTMLAttributes, type ReactNode, useMemo, useState } from "react";
import { Link, Outlet, useLocation } from "react-router-dom";
import { AnimatePresence, motion } from "framer-motion";
import {
	ArrowTopRightOnSquareIcon,
	CheckCircleIcon,
	QuestionMarkCircleIcon,
	SparklesIcon,
} from "@heroicons/react/24/outline";
import { getCurrentStepIndex, getPageMeta, WORKFLOW_STEPS } from "./workflowMeta";

function cn(...values: Array<string | false | null | undefined>) {
	return values.filter(Boolean).join(" ");
}

export function PrimaryButton({
	className,
	children,
	...props
}: ButtonHTMLAttributes<HTMLButtonElement>) {
	return (
		<button className={cn("executive-button", className)} {...props}>
			{children}
		</button>
	);
}

export function SecondaryButton({
	className,
	children,
	...props
}: ButtonHTMLAttributes<HTMLButtonElement>) {
	return (
		<button className={cn("executive-button-secondary", className)} {...props}>
			{children}
		</button>
	);
}

export function StatusBadge({
	children,
	tone = "neutral",
	className,
}: {
	children: ReactNode;
	tone?: "neutral" | "accent" | "success" | "danger";
	className?: string;
}) {
	const toneClass =
		tone === "accent"
			? "border-[rgba(178,122,68,0.28)] bg-[rgba(178,122,68,0.12)] text-[var(--accent-strong)]"
			: tone === "success"
				? "border-[rgba(37,100,84,0.22)] bg-[var(--success-soft)] text-[var(--success)]"
				: tone === "danger"
					? "border-[rgba(156,66,63,0.22)] bg-[var(--danger-soft)] text-[var(--danger)]"
					: "border-[var(--line)] bg-white/70 text-[var(--muted)]";

	return (
		<span className={cn("inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold", toneClass, className)}>
			{children}
		</span>
	);
}

export function MetricPill({
	label,
	value,
	tone = "neutral",
}: {
	label: string;
	value: ReactNode;
	tone?: "neutral" | "accent" | "success" | "danger";
}) {
	return (
		<div
			className={cn(
				"rounded-[18px] border px-4 py-3",
				tone === "accent" && "border-[rgba(178,122,68,0.22)] bg-[rgba(178,122,68,0.1)]",
				tone === "success" && "border-[rgba(37,100,84,0.22)] bg-[var(--success-soft)]",
				tone === "danger" && "border-[rgba(156,66,63,0.22)] bg-[var(--danger-soft)]",
				tone === "neutral" && "border-[var(--line)] bg-white/70"
			)}
		>
			<div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">
				{label}
			</div>
			<div className="mt-1 text-lg font-semibold text-[var(--text)]">{value}</div>
		</div>
	);
}

export function HelpHint({
	label,
	content,
}: {
	label: string;
	content: string;
}) {
	return (
		<span className="group relative inline-flex items-center">
			<button type="button" className="executive-hint" aria-label={label}>
				?
			</button>
			<span className="pointer-events-none absolute left-1/2 top-full z-20 mt-2 hidden w-64 -translate-x-1/2 rounded-2xl border border-[var(--line)] bg-[rgba(19,42,61,0.96)] px-4 py-3 text-left text-xs leading-5 text-[#f8efe2] shadow-2xl group-hover:block group-focus-within:block">
				<strong className="mb-1 block font-semibold text-white">{label}</strong>
				{content}
			</span>
		</span>
	);
}

export function FieldLabel({
	label,
	description,
	help,
}: {
	label: string;
	description?: string;
	help?: string;
}) {
	return (
		<div className="mb-2 flex items-start justify-between gap-3">
			<div>
				<div className="flex items-center gap-2">
					<label className="text-sm font-semibold text-[var(--text)]">{label}</label>
					{help ? <HelpHint label={label} content={help} /> : null}
				</div>
				{description ? (
					<p className="mt-1 text-xs leading-5 text-[var(--muted)]">{description}</p>
				) : null}
			</div>
		</div>
	);
}

export function SectionCard({
	title,
	description,
	children,
	aside,
	className,
}: {
	title?: string;
	description?: string;
	children: ReactNode;
	aside?: ReactNode;
	className?: string;
}) {
	return (
		<section className={cn("executive-card executive-card-strong p-6 md:p-8", className)}>
			{title || description || aside ? (
				<div className="mb-6 flex flex-col gap-4 border-b border-[var(--line)] pb-5 md:flex-row md:items-end md:justify-between">
					<div>
						{title ? <h2 className="text-2xl text-[var(--text)]">{title}</h2> : null}
						{description ? (
							<p className="mt-2 max-w-3xl text-sm leading-6 text-[var(--muted)]">
								{description}
							</p>
						) : null}
					</div>
					{aside}
				</div>
			) : null}
			{children}
		</section>
	);
}

export function EmptyState({
	title,
	description,
	action,
}: {
	title: string;
	description: string;
	action?: ReactNode;
}) {
	return (
		<div className="rounded-[24px] border border-dashed border-[var(--line-strong)] bg-white/60 px-6 py-10 text-center">
			<div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-[rgba(178,122,68,0.1)] text-[var(--accent-strong)]">
				<SparklesIcon className="h-6 w-6" />
			</div>
			<h3 className="mt-4 text-xl text-[var(--text)]">{title}</h3>
			<p className="mx-auto mt-2 max-w-xl text-sm leading-6 text-[var(--muted)]">
				{description}
			</p>
			{action ? <div className="mt-5">{action}</div> : null}
		</div>
	);
}

export function StickyActionBar({
	children,
	className,
}: {
	children: ReactNode;
	className?: string;
}) {
	return (
		<div className={cn("sticky bottom-5 z-20 mt-8", className)}>
			<div className="executive-card executive-card-strong flex flex-col gap-4 px-5 py-4 md:flex-row md:items-center md:justify-between">
				{children}
			</div>
		</div>
	);
}

function HelpDrawer({
	open,
	onClose,
}: {
	open: boolean;
	onClose: () => void;
}) {
	const location = useLocation();
	const meta = getPageMeta(location.pathname);

	return (
		<AnimatePresence>
			{open ? (
				<Fragment>
					<motion.button
						type="button"
						initial={{ opacity: 0 }}
						animate={{ opacity: 1 }}
						exit={{ opacity: 0 }}
						onClick={onClose}
						className="fixed inset-0 z-40 bg-[rgba(10,23,34,0.36)] backdrop-blur-sm"
						aria-label="Fechar ajuda"
					/>
					<motion.aside
						initial={{ x: 440, opacity: 0 }}
						animate={{ x: 0, opacity: 1 }}
						exit={{ x: 440, opacity: 0 }}
						transition={{ type: "spring", damping: 28, stiffness: 260 }}
						className="fixed right-4 top-4 z-50 h-[calc(100vh-2rem)] w-full max-w-[420px] overflow-hidden rounded-[32px] border border-white/10 executive-panel-dark"
					>
						<div className="flex h-full flex-col">
							<div className="border-b border-white/10 px-6 py-6">
								<div className="flex items-start justify-between gap-4">
									<div>
										<div className="executive-pill border-white/10 bg-white/5 text-[#f2debf]">
											Ajuda desta página
										</div>
										<h2 className="mt-4 text-3xl text-white">{meta.panelTitle}</h2>
										<p className="mt-3 text-sm leading-6 text-[#d8c9b5]">
											{meta.panelSummary}
										</p>
									</div>
									<button
										type="button"
										onClick={onClose}
										className="rounded-full border border-white/10 px-3 py-2 text-sm text-[#f8efe2] transition-colors hover:bg-white/10"
									>
										Fechar
									</button>
								</div>
							</div>
							<div className="flex-1 space-y-5 overflow-y-auto px-6 py-6">
								{meta.helpSections.map((section) => (
									<div
										key={section.title}
										className="rounded-[24px] border border-white/10 bg-white/5 px-5 py-5"
									>
										<h3 className="text-xl text-white">{section.title}</h3>
										<p className="mt-2 text-sm leading-6 text-[#e6dbcc]">
											{section.body}
										</p>
									</div>
								))}
							</div>
						</div>
					</motion.aside>
				</Fragment>
			) : null}
		</AnimatePresence>
	);
}

export function WorkflowLayout() {
	const location = useLocation();
	const meta = useMemo(() => getPageMeta(location.pathname), [location.pathname]);
	const [helpOpen, setHelpOpen] = useState(false);
	const currentStepIndex = getCurrentStepIndex(location.pathname);

	return (
		<div className="min-h-screen">
			<div className="mx-auto flex min-h-screen max-w-[1600px] flex-col gap-6 px-4 py-4 xl:flex-row xl:gap-8 xl:px-6 xl:py-6">
				<aside className="executive-panel-dark executive-gridline relative overflow-hidden px-6 py-6 xl:sticky xl:top-6 xl:h-[calc(100vh-3rem)] xl:w-[300px] xl:flex-shrink-0">
					<div className="relative z-10 flex h-full flex-col">
						<div>
							<div className="executive-pill border-white/10 bg-white/5 text-[#f0ddc2]">
								RH Desktop Suite
							</div>
							<h2 className="mt-5 text-3xl text-white">Candidate Allocator</h2>
							<p className="mt-3 text-sm leading-6 text-[#d8c9b5]">
								Um fluxo claro para importar, revisar e alocar candidatos com controle operacional.
							</p>
						</div>

						<div className="mt-8 space-y-4">
							{WORKFLOW_STEPS.map((step, index) => {
								const active = index === currentStepIndex;
								const completed = currentStepIndex > index;

								return (
									<div
										key={step.key}
										className={cn(
											"rounded-[24px] border px-4 py-4 transition-all",
											active
												? "border-[rgba(178,122,68,0.34)] bg-[rgba(178,122,68,0.14)]"
												: completed
													? "border-white/10 bg-white/8"
													: "border-white/8 bg-transparent"
										)}
									>
										<div className="flex items-start gap-3">
											<div
												className={cn(
													"mt-1 flex h-8 w-8 items-center justify-center rounded-full border text-xs font-semibold",
													active
														? "border-[rgba(247,224,188,0.35)] bg-[rgba(247,224,188,0.14)] text-[#f5e5ca]"
														: completed
															? "border-[rgba(255,255,255,0.14)] bg-[rgba(255,255,255,0.08)] text-white"
															: "border-white/10 text-[#d3c3af]"
												)}
											>
												{completed ? <CheckCircleIcon className="h-4 w-4" /> : index + 1}
											</div>
											<div className="min-w-0">
												<div className="text-sm font-semibold text-[#fbf3e7]">
													{step.label}
												</div>
												<div className="mt-1 text-xs leading-5 text-[#d3c3af]">
													{step.caption}
												</div>
											</div>
										</div>
									</div>
								);
							})}
						</div>

						<div className="mt-auto rounded-[24px] border border-white/10 bg-white/5 px-4 py-5">
							<div className="text-xs uppercase tracking-[0.22em] text-[#d8c9b5]">
								Leitura da etapa
							</div>
							<p className="mt-3 text-sm leading-6 text-[#f8efe2]">{meta.panelSummary}</p>
						</div>
					</div>
				</aside>

				<div className="min-w-0 flex-1">
					<div className="executive-card executive-card-strong relative overflow-hidden px-6 py-6 md:px-8 md:py-8">
						<div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-[rgba(178,122,68,0.5)] to-transparent" />
						<div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
							<div className="max-w-4xl">
								<div className="executive-pill">{meta.kicker}</div>
								<h1 className="mt-5 text-4xl text-[var(--text)] md:text-5xl">{meta.title}</h1>
								<p className="mt-4 max-w-3xl text-sm leading-7 text-[var(--muted)] md:text-base">
									{meta.description}
								</p>
							</div>
							<div className="flex flex-wrap items-center gap-3">
								<Link to="/" className="executive-button-secondary">
									Visão inicial
									<ArrowTopRightOnSquareIcon className="h-4 w-4" />
								</Link>
								<button
									type="button"
									onClick={() => setHelpOpen(true)}
									className="executive-button"
									data-testid="page-help-button"
								>
									<QuestionMarkCircleIcon className="h-5 w-5" />
									Ajuda da página
								</button>
							</div>
						</div>
					</div>

					<main className="pb-12 pt-6">
						<Outlet />
					</main>
				</div>
			</div>
			<HelpDrawer open={helpOpen} onClose={() => setHelpOpen(false)} />
		</div>
	);
}
