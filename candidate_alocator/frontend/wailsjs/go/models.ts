export namespace main {
	
	export class MappingItem {
	    nomeColuna: string;
	    indice: number;
	    variavel: string;
	
	    static createFrom(source: any = {}) {
	        return new MappingItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomeColuna = source["nomeColuna"];
	        this.indice = source["indice"];
	        this.variavel = source["variavel"];
	    }
	}

}

