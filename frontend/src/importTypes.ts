export interface MappingItem {
	nomeColuna: string;
	indice: number;
	variavel: string;
}

export interface MappingFieldInfo {
	variavel: string;
	required: boolean;
	unique: boolean;
	duplicate: boolean;
}
