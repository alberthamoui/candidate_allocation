import React, { ButtonHTMLAttributes, ReactNode, useMemo, useState } from "react";
import { Link, Outlet, useLocation } from "react-router-dom";
import { getCurrentStepIndex, getPageMeta, WORKFLOW_STEPS } from "./workflowMeta";
import { HelpIcon } from "./components/Tooltip";
import { PageHelp } from "./components/PageHelp";

function cn(...values: Array<string | false | null | undefined>) {
	return values.filter(Boolean).join(" ");
}

export function PrimaryButton({ className, children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
	return <button className={cn("minimal-btn", className)} {...props}>{children}</button>;
}

export function SecondaryButton({ className, children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
	return <button className={cn("minimal-btn-secondary", className)} {...props}>{children}</button>;
}

export function StatusBadge({ children, tone = "neutral", className }: { children: ReactNode; tone?: "neutral" | "accent" | "success" | "danger"; className?: string }) {
	const toneClass =
		tone === "accent" ? "bg-[var(--accent)] text-white" :
		tone === "success" ? "bg-[var(--success)] text-white" :
		tone === "danger" ? "bg-[var(--danger)] text-white" :
		"bg-gray-100 text-gray-700 border border-gray-200";

	return (
		<span className={cn("inline-flex items-center px-2 py-0.5 text-[11px] font-medium rounded-full", toneClass, className)}>
			{children}
		</span>
	);
}

export function MetricPill({ label, value, tone = "neutral" }: { label: string; value: ReactNode; tone?: "neutral" | "accent" | "success" | "danger" }) {
	return (
		<div className="minimal-panel p-4 flex flex-col items-start gap-1">
			<div className="text-xs font-medium text-gray-500">{label}</div>
			<div className="text-xl font-semibold text-gray-900">{value}</div>
		</div>
	);
}

export function FieldLabel({ label, description, help }: { label: string; description?: string; help?: string }) {
	return (
		<div className="mb-2">
			<div className="flex items-center">
				<label className="minimal-label mr-2 mb-0">{label}</label>
				{(help || description) && <HelpIcon text={(help || description) as string} />}
			</div>
		</div>
	);
}

export function SectionCard({ title, description, children, aside, className }: { title?: string; description?: string; children: ReactNode; aside?: ReactNode; className?: string }) {
	return (
		<section className={cn("minimal-panel p-6 md:p-8 mb-6", className)}>
			{(title || description || aside) && (
				<div className="mb-6 flex flex-col md:flex-row md:items-end justify-between border-b border-gray-100 pb-4 gap-4">
					<div className="flex items-center">
						{title && <h2 className="text-lg font-semibold mr-2">{title}</h2>}
						{description && <HelpIcon text={description as string} />}
					</div>
					{aside}
				</div>
			)}
			{children}
		</section>
	);
}

export function EmptyState({ title, description, action }: { title: string; description: string; action?: ReactNode }) {
	return (
		<div className="minimal-panel p-12 text-center bg-gray-50 flex flex-col items-center justify-center min-h-[300px]">
			<h3 className="text-lg font-medium text-gray-900">{title}</h3>
			<p className="mt-2 text-sm text-gray-500 max-w-md mx-auto">{description}</p>
			{action && <div className="mt-6">{action}</div>}
		</div>
	);
}

export function StickyActionBar({ children, className }: { children: ReactNode; className?: string }) {
	return (
		<div className={cn("fixed bottom-[60px] left-0 right-0 z-40 flex justify-center px-4 pointer-events-none", className)}>
			<div className="minimal-panel shadow-[0_8px_30px_rgb(0,0,0,0.12)] flex flex-col gap-4 px-6 py-4 md:flex-row md:items-center md:justify-between bg-white/95 backdrop-blur-xl w-full max-w-[1200px] pointer-events-auto border border-gray-200">
				{children}
			</div>
		</div>
	);
}

function CompactHeader({ title, kicker, onHelpClick }: { title: string, kicker: string; onHelpClick: () => void }) {
	return (
		<header className="sticky top-0 z-40 border-b border-gray-100 bg-white/85 backdrop-blur-md">
			<div className="mx-auto flex w-full max-w-[1200px] items-center justify-between px-4 py-3 md:px-8">
				<div className="flex min-w-0 items-center gap-3">
					<StatusBadge tone="neutral">{kicker}</StatusBadge>
					<h1 className="truncate text-sm font-semibold text-gray-800">{title}</h1>
				</div>
				<div className="flex items-center gap-4">
					<button
						onClick={onHelpClick}
						className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-[var(--line)] bg-white text-base font-semibold leading-none text-gray-600 shadow-sm transition-all hover:bg-gray-50 hover:scale-105"
						aria-label="Ajuda da Página"
						data-testid="page-help-button"
					>
						<span className="leading-none select-none">?</span>
					</button>
					<Link to="/" className="text-xs font-medium text-gray-500 transition-colors hover:text-gray-900">Início</Link>
				</div>
			</div>
		</header>
	);
}

function CompactTimeline({ currentStepIndex }: { currentStepIndex: number }) {
	return (
		<div className="fixed bottom-0 left-0 right-0 z-50 border-t border-gray-100 bg-white/95 text-xs font-medium backdrop-blur-md">
			<div className="mx-auto flex h-11 w-full max-w-[1200px] overflow-hidden px-0 md:px-8">
				{WORKFLOW_STEPS.map((step, index) => {
					const active = index === currentStepIndex;
					const completed = currentStepIndex > index;
					return (
						<div
							key={step.key}
							className={cn(
								"flex-1 flex items-center justify-center border-r border-gray-100 last:border-r-0 transition-colors px-2 truncate",
								active ? "text-[var(--accent)] bg-blue-50/30 font-semibold" :
								completed ? "text-gray-800" :
								"text-gray-400"
							)}
						>
							<span className="hidden md:inline mr-1.5 opacity-50">{index + 1}.</span>
							<span className="truncate">{step.label}</span>
						</div>
					);
				})}
			</div>
		</div>
	);
}

export function WorkflowLayout() {
	const location = useLocation();
	const meta = useMemo(() => getPageMeta(location.pathname), [location.pathname]);
	const currentStepIndex = getCurrentStepIndex(location.pathname);
	const [isHelpOpen, setIsHelpOpen] = useState(false);

	return (
		<div className="min-h-screen bg-[var(--bg-subtle)] pb-[160px] flex flex-col">
			<CompactHeader title={meta.title} kicker={meta.kicker} onHelpClick={() => setIsHelpOpen(true)} />
			
			<main data-testid="workflow-main" className="flex-1 w-full max-w-[1200px] mx-auto px-4 py-8 md:px-8">
				<Outlet />
			</main>
			
			<PageHelp 
				isOpen={isHelpOpen}
				onClose={() => setIsHelpOpen(false)}
				title={meta.panelTitle} 
				description={meta.panelSummary}
				impacts={
					<ul className="list-disc pl-5 space-y-2">
						{meta.helpSections.map(s => <li key={s.title}><strong>{s.title}:</strong> {s.body}</li>)}
					</ul>
				}
			/>
			
			<CompactTimeline currentStepIndex={currentStepIndex} />
		</div>
	);
}
