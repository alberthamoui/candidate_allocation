import EntityVerificationView from "./EntityVerificationView";
import { SaveRestricoesFromMaps } from "../wailsjs/go/main/App";

interface VerifyRestricoesPageProps {
	restricoes: any[] | null;
}

export default function VerifyRestricoesPage({
	restricoes,
}: VerifyRestricoesPageProps) {
	const entities = Object.fromEntries(
		(restricoes ?? []).map((restricao, index) => [
			index + 1,
			{
				erros: [],
				restricao,
			},
		])
	);

	return (
		<EntityVerificationView
			title="Verificação de Restrições"
			subtitle="Revise os dados construidos para a aba de restricoes antes do salvamento."
			entities={entities}
			entityKey="restricao"
			duplicates={[]}
			duplicateFields={[]}
			onSave={SaveRestricoesFromMaps}
			nextRoute="/mappingAvaliadores"
			backRoute="/mappingRestricoes"
			allowExtras={false}
		/>
	);
}
