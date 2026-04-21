import { motion } from "framer-motion";
import {
	ArrowPathIcon,
	CheckCircleIcon,
	Cog6ToothIcon,
} from "@heroicons/react/24/outline";
import { useNavigate } from "react-router-dom";
import {
	MetricPill,
	PrimaryButton,
	SectionCard,
	SecondaryButton,
	StatusBadge,
} from "./workflowShell";

export default function SuccessPage() {
	const navigate = useNavigate();

	return (
		<div className="space-y-6">
			<SectionCard
				title="Base consolidada"
				description="Candidatos, restrições e avaliadores foram salvos. O sistema está pronto para a definição operacional da alocação."
				aside={<StatusBadge tone="success">Importação concluída</StatusBadge>}
			>
				<div className="grid gap-4 md:grid-cols-3">
					<MetricPill label="Etapa atual" value="Checkpoint" tone="success" />
					<MetricPill label="Próxima ação" value="Configurar alocação" tone="accent" />
					<MetricPill label="Estado" value="Pronto para operar" />
				</div>
			</SectionCard>

			<div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
				<motion.div initial={{ opacity: 0, y: 18 }} animate={{ opacity: 1, y: 0 }}>
					<SectionCard
						title="Tudo Pronto!"
						description="A preparação da base terminou com sucesso. A partir daqui você define capacidade, critérios desejáveis e executa a distribuição."
					>
						<div className="rounded-[28px] border border-[rgba(37,100,84,0.18)] bg-[var(--success-soft)] p-6">
							<div className="flex items-start gap-4">
								<div className="flex h-16 w-16 items-center justify-center rounded-full bg-white/80 text-[var(--success)]">
									<CheckCircleIcon className="h-9 w-9" />
								</div>
								<div>
									<h2 className="text-3xl text-[var(--text)]">Checkpoint institucional concluído</h2>
									<p className="mt-3 max-w-2xl text-sm leading-7 text-[var(--muted)]">
										Os dados principais já estão persistidos e disponíveis para a próxima fase do software. Se a importação estiver correta, siga para a configuração. Se precisar reiniciar o processo com outra planilha, faça isso agora para evitar retrabalho.
									</p>
								</div>
							</div>
						</div>
					</SectionCard>
				</motion.div>

				<motion.div initial={{ opacity: 0, y: 18 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.08 }}>
					<SectionCard
						title="Próximos passos"
						description="Duas rotas possíveis dependendo do momento da operação."
					>
						<div className="space-y-4">
							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-5">
								<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
									Seguir com a rodada
								</div>
								<p className="mt-3 text-sm leading-6 text-[var(--muted)]">
									Abra a configuração para definir capacidade de grupos, avaliadores por mesa e critérios soft.
								</p>
								<div className="mt-4">
									<PrimaryButton onClick={() => navigate("/allocation-config")} className="w-full justify-center">
										<Cog6ToothIcon className="h-5 w-5" />
										Configurar Alocação
									</PrimaryButton>
								</div>
							</div>

							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-5">
								<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
									Reiniciar processo
								</div>
								<p className="mt-3 text-sm leading-6 text-[var(--muted)]">
									Use esta opção quando a planilha de origem precisar ser trocada ou o fluxo tiver que ser refeito do início.
								</p>
								<div className="mt-4">
									<SecondaryButton onClick={() => navigate("/")} className="w-full justify-center">
										<ArrowPathIcon className="h-5 w-5" />
										Nova Importação
									</SecondaryButton>
								</div>
							</div>
						</div>
					</SectionCard>
				</motion.div>
			</div>
		</div>
	);
}
