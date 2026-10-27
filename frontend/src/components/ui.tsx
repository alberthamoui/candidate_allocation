import { ButtonHTMLAttributes, ReactNode, useState, useRef, useEffect } from "react";

// Peças visuais compartilhadas pelas telas (inspiradas no workflowShell da
// branch mainjoao). As classes base ficam em index.css.

export function cn(...valores: Array<string | false | null | undefined>) {
	return valores.filter(Boolean).join(" ");
}

type Botao = ButtonHTMLAttributes<HTMLButtonElement>;

export function BotaoPrimario({ className, type = "button", ...props }: Botao) {
	return <button type={type} className={cn("btn-primario", className)} {...props} />;
}

export function BotaoSecundario({ className, type = "button", ...props }: Botao) {
	return <button type={type} className={cn("btn-secundario", className)} {...props} />;
}

export function BotaoPerigo({ className, type = "button", ...props }: Botao) {
	return <button type={type} className={cn("btn-perigo", className)} {...props} />;
}

export type Tom = "neutro" | "destaque" | "sucesso" | "atencao" | "perigo";

const TOM_SELO: Record<Tom, string> = {
	neutro: "bg-gray-100 text-gray-700 border border-gray-200",
	destaque: "bg-[var(--accent-soft)] text-[var(--accent)] border border-[rgba(94,106,210,0.2)]",
	sucesso: "bg-[var(--success-soft)] text-[var(--success)] border border-[rgba(31,138,59,0.2)]",
	atencao: "bg-[var(--warning-soft)] text-[var(--warning)] border border-[rgba(245,166,35,0.3)]",
	perigo: "bg-[var(--danger-soft)] text-[var(--danger)] border border-[rgba(212,32,32,0.2)]",
};

export function Selo({ children, tom = "neutro", className }: { children: ReactNode; tom?: Tom; className?: string }) {
	return (
		<span className={cn("inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium rounded-full whitespace-nowrap", TOM_SELO[tom], className)}>
			{children}
		</span>
	);
}

const TOM_METRICA: Record<Tom, string> = {
	neutro: "text-gray-900",
	destaque: "text-[var(--accent)]",
	sucesso: "text-[var(--success)]",
	atencao: "text-[var(--warning)]",
	perigo: "text-[var(--danger)]",
};

/** Número de destaque com rótulo (ex.: "98 · candidatos"). */
export function Metrica({
	rotulo,
	valor,
	detalhe,
	tom = "neutro",
	ajuda,
	testId,
}: {
	rotulo: string;
	valor: ReactNode;
	detalhe?: ReactNode;
	tom?: Tom;
	ajuda?: string;
	testId?: string;
}) {
	return (
		<div className="painel p-4 flex flex-col gap-0.5" data-testid={testId}>
			<div className="flex items-center gap-1.5 text-xs font-medium text-[var(--text-muted)]">
				{rotulo}
				{ajuda && <IconeAjuda texto={ajuda} />}
			</div>
			<div className={cn("text-2xl font-semibold tabular-nums", TOM_METRICA[tom])}>{valor}</div>
			{detalhe && <div className="text-xs text-[var(--text-muted)]">{detalhe}</div>}
		</div>
	);
}

/** Bloco com título, ajuda opcional e ações à direita. */
export function Secao({
	titulo,
	ajuda,
	subtitulo,
	acoes,
	children,
	className,
	rotulo,
}: {
	titulo?: ReactNode;
	ajuda?: string;
	subtitulo?: ReactNode;
	acoes?: ReactNode;
	children: ReactNode;
	className?: string;
	rotulo?: string; // aria-label
}) {
	return (
		<section className={cn("painel p-5 md:p-6", className)} aria-label={rotulo}>
			{(titulo || acoes) && (
				<div className="mb-5 flex flex-col gap-3 border-b border-[var(--line)] pb-4 md:flex-row md:items-center md:justify-between">
					<div>
						<div className="flex items-center gap-2">
							{titulo && <h2 className="text-base font-semibold">{titulo}</h2>}
							{ajuda && <IconeAjuda texto={ajuda} />}
						</div>
						{subtitulo && <p className="mt-1 text-sm text-[var(--text-muted)]">{subtitulo}</p>}
					</div>
					{acoes && <div className="flex flex-wrap items-center gap-2">{acoes}</div>}
				</div>
			)}
			{children}
		</section>
	);
}

/** Cabeçalho da página: título grande e descrição curta. */
export function CabecalhoPagina({ titulo, descricao, acoes }: { titulo: string; descricao?: ReactNode; acoes?: ReactNode }) {
	return (
		<div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
			<div className="space-y-1">
				<h1 className="text-2xl font-semibold">{titulo}</h1>
				{descricao && <p className="text-sm leading-6 text-[var(--text-muted)] max-w-3xl">{descricao}</p>}
			</div>
			{acoes && <div className="flex flex-wrap items-center gap-2">{acoes}</div>}
		</div>
	);
}

/** Mensagem de estado vazio. */
export function EstadoVazio({ titulo, descricao, acao }: { titulo: string; descricao?: ReactNode; acao?: ReactNode }) {
	return (
		<div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--line-strong)] bg-[var(--bg-subtle)] px-6 py-10 text-center">
			<h3 className="text-base font-medium">{titulo}</h3>
			{descricao && <p className="mx-auto mt-1 max-w-md text-sm text-[var(--text-muted)]">{descricao}</p>}
			{acao && <div className="mt-5">{acao}</div>}
		</div>
	);
}

const TOM_AVISO: Record<"info" | "atencao" | "perigo" | "sucesso", string> = {
	info: "border-[rgba(94,106,210,0.2)] bg-[var(--accent-soft)] text-[#3b449a]",
	atencao: "border-[rgba(245,166,35,0.3)] bg-[var(--warning-soft)] text-[#8a5a12]",
	perigo: "border-[rgba(212,32,32,0.2)] bg-[var(--danger-soft)] text-[#a51818]",
	sucesso: "border-[rgba(31,138,59,0.2)] bg-[var(--success-soft)] text-[#17692d]",
};

/** Caixa de aviso em linha. */
export function Aviso({
	tom = "info",
	icone,
	children,
	className,
}: {
	tom?: keyof typeof TOM_AVISO;
	icone?: ReactNode;
	children: ReactNode;
	className?: string;
}) {
	return (
		<div className={cn("flex items-start gap-2 rounded-[var(--radius)] border px-3 py-2.5 text-sm", TOM_AVISO[tom], className)}>
			{icone && <span className="mt-0.5 flex-shrink-0">{icone}</span>}
			<div className="min-w-0 flex-1">{children}</div>
		</div>
	);
}

/** Barra fixa acima da linha de etapas, com a ação principal da tela. */
export function BarraAcoes({ children, dica }: { children: ReactNode; dica?: ReactNode }) {
	return (
		<>
			{/* espaço para o conteúdo não ficar atrás da barra */}
			<div className="h-20" aria-hidden />
			<div className="pointer-events-none fixed inset-x-0 bottom-12 z-30 flex justify-center px-4">
				<div className="painel pointer-events-auto flex w-full max-w-[1200px] flex-col gap-3 bg-white/95 px-5 py-3 shadow-[var(--shadow-lg)] backdrop-blur md:flex-row md:items-center md:justify-between">
					<div className="hidden text-sm text-[var(--text-muted)] md:block">{dica}</div>
					<div className="flex flex-wrap items-center gap-2">{children}</div>
				</div>
			</div>
		</>
	);
}

/** Dica ao passar o mouse. */
export function Dica({ conteudo, children }: { conteudo: ReactNode; children: ReactNode }) {
	const [visivel, setVisivel] = useState(false);
	return (
		<span
			className="relative inline-flex items-center"
			onMouseEnter={() => setVisivel(true)}
			onMouseLeave={() => setVisivel(false)}
			onFocus={() => setVisivel(true)}
			onBlur={() => setVisivel(false)}
		>
			{children}
			{visivel && (
				<span
					role="tooltip"
					className="painel absolute bottom-full left-1/2 z-50 mb-2 w-max max-w-xs -translate-x-1/2 p-3 text-left text-xs font-normal leading-relaxed text-[var(--text)] shadow-[var(--shadow-lg)]"
				>
					{conteudo}
				</span>
			)}
		</span>
	);
}

export function IconeAjuda({ texto }: { texto: string }) {
	return (
		<Dica conteudo={texto}>
			<span className="icone-ajuda" tabIndex={0} aria-label={texto}>
				?
			</span>
		</Dica>
	);
}

/** Janela sobre a tela (confirmações, erros). */
export function Modal({
  titulo,
  children,
  acoes,
  onFechar,
}: {
  titulo: ReactNode;
  children: ReactNode;
  acoes?: ReactNode;
  onFechar?: () => void;
}) {
  const dialog = useRef<HTMLDivElement>(null);
  const fechar = useRef(onFechar);
  fechar.current = onFechar;
  useEffect(() => {
    const anterior = document.activeElement as HTMLElement | null;
    const el = dialog.current;
    el?.focus();
    function tecla(e: KeyboardEvent) {
      if (e.key === "Escape") {
        e.stopPropagation();
        fechar.current?.();
      }
      if (e.key === "Tab" && el) {
        const elementos = Array.from(
          el.querySelectorAll<HTMLElement>(
            'button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),a[href],[tabindex="0"]',
          ),
        ).filter((x) => x.offsetParent !== null);
        const primeiro = elementos[0],
          ultimo = elementos[elementos.length - 1];
        if (!primeiro) {
          e.preventDefault();
          return;
        }
        if (
          e.shiftKey &&
          (document.activeElement === primeiro || document.activeElement === el)
        ) {
          e.preventDefault();
          ultimo.focus();
        } else if (
          !e.shiftKey &&
          (document.activeElement === ultimo || document.activeElement === el)
        ) {
          e.preventDefault();
          primeiro.focus();
        }
      }
    }
    el?.addEventListener("keydown", tecla);
    return () => {
      el?.removeEventListener("keydown", tecla);
      anterior?.focus();
    };
  }, []);
  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/25 p-4 backdrop-blur-sm"
      onClick={onFechar}
    >
      <div
        ref={dialog}
        tabIndex={-1}
        role="dialog"
        aria-modal
        className="painel w-full max-w-md max-h-[calc(100dvh-2rem)] overflow-y-auto p-6 shadow-[var(--shadow-lg)]"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold">{titulo}</h3>
        <div className="mt-2 text-sm leading-6 text-[var(--text-muted)]">
          {children}
        </div>
        {acoes && <div className="mt-6 flex justify-end gap-2">{acoes}</div>}
      </div>
    </div>
  );
}
