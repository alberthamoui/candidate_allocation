import { UserGroupIcon } from "@heroicons/react/24/outline";
import { suggestMappingAvaliador } from "../api";
import EtapaAba from "../components/EtapaAba";

interface UploadAvaliadorProps {
	setMappingAvaliador: (data: any) => void;
}

export default function UploadAvaliador({ setMappingAvaliador }: UploadAvaliadorProps) {
	return (
		<EtapaAba
			icone={<UserGroupIcon className="h-6 w-6" />}
			titulo="Avaliadores"
			descricao={
				<>
					Os candidatos foram salvos. Agora os avaliadores serão lidos da <b>segunda aba</b> da mesma planilha.
				</>
			}
			campos={["Nome", "Email", "Sigla"]}
			rotuloBotao="Processar Avaliadores"
			processar={async () => setMappingAvaliador(await suggestMappingAvaliador())}
			proximaRota="/mapping-avaliador"
			erroPrefixo="Erro ao processar avaliadores. Confira se o arquivo tem a aba de avaliadores."
		/>
	);
}
