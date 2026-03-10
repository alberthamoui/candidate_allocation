import EntityVerificationView from "./EntityVerificationView";
import { SaveAvaliadoresFromMaps } from "../wailsjs/go/main/App";

interface ErrorItem {
	field: string;
	msg: string;
}

interface AvaliadorWrapper {
	erros: ErrorItem[];
	avaliador: any;
}

interface VerifyAvaliadoresPageProps {
	avaliadores: Record<number, AvaliadorWrapper>;
	duplicates: number[][];
	duplicateFields: string[];
}

export default function VerifyAvaliadoresPage({
	avaliadores,
	duplicates,
	duplicateFields,
}: VerifyAvaliadoresPageProps) {
	return (
		<EntityVerificationView
			title="Verificação de Avaliadores"
			subtitle="Revise e corrija os dados dos avaliadores. Clique em qualquer campo para editar."
			entities={avaliadores}
			entityKey="avaliador"
			duplicates={duplicates}
			duplicateFields={duplicateFields}
			onSave={SaveAvaliadoresFromMaps}
			nextRoute="/success" // Final step after verifying evaluators
		/>
	);
}
