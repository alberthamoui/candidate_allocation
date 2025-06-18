export namespace main {
	
	export class Usuario {
	    timestamp: string;
	    nome: string;
	    cpf: string;
	    numero: string;
	    semestre: string;
	    curso: string;
	    email_insper: string;
	    email_pessoal: string;
	    opcoes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Usuario(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.nome = source["nome"];
	        this.cpf = source["cpf"];
	        this.numero = source["numero"];
	        this.semestre = source["semestre"];
	        this.curso = source["curso"];
	        this.email_insper = source["email_insper"];
	        this.email_pessoal = source["email_pessoal"];
	        this.opcoes = source["opcoes"];
	    }
	}

}

