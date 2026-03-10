import EntityVerificationView from "./EntityVerificationView";
import {
	SaveUsuariosFromMaps,
	SaveRestricoesFromMaps,
} from "../wailsjs/go/main/App";

interface ErrorItem {
	field: string;
	msg: string;
}

interface UserWrapper {
	erros: ErrorItem[];
	usuario: any;
}

interface VerifyUserPageProps {
	usuarios: Record<number, UserWrapper>;
	restricoes: any;
	duplicates: number[][];
	duplicateFields: string[];
}

export default function VerifyUserPage({
	usuarios,
	restricoes,
	duplicates,
	duplicateFields,
}: VerifyUserPageProps) {
	return (
		<EntityVerificationView
			title="Verificação de Usuários"
			subtitle="Revise e corrija os dados importados. Clique em qualquer campo para editar."
			entities={usuarios}
			entityKey="usuario"
			duplicates={duplicates}
			duplicateFields={duplicateFields}
			onSave={SaveUsuariosFromMaps}
			nextRoute="/verifyAvaliadores"
			extraData={restricoes}
			onSaveExtra={SaveRestricoesFromMaps}
		/>
	);
}
