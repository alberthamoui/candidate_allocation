export namespace logic {
	
	export class ErrorEntry {
	    field: number;
	    msg: string;
	
	    static createFrom(source: any = {}) {
	        return new ErrorEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.msg = source["msg"];
	    }
	}
	export class AvaliadorValidationResult {
	    erros: ErrorEntry[];
	    avaliador: types.Avaliador;
	
	    static createFrom(source: any = {}) {
	        return new AvaliadorValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.erros = this.convertValues(source["erros"], ErrorEntry);
	        this.avaliador = this.convertValues(source["avaliador"], types.Avaliador);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AvaliadoresResponse {
	    avaliadores: Record<number, AvaliadorValidationResult>;
	    duplicates: number[][];
	    duplicateFields: string[];
	
	    static createFrom(source: any = {}) {
	        return new AvaliadoresResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avaliadores = this.convertValues(source["avaliadores"], AvaliadorValidationResult, true);
	        this.duplicates = source["duplicates"];
	        this.duplicateFields = source["duplicateFields"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ValidationResult {
	    erros: ErrorEntry[];
	    usuario: types.Candidato;
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.erros = this.convertValues(source["erros"], ErrorEntry);
	        this.usuario = this.convertValues(source["usuario"], types.Candidato);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UsuariosResponse {
	    usuarios: Record<number, ValidationResult>;
	    duplicates: number[][];
	    duplicateFields: string[];
	
	    static createFrom(source: any = {}) {
	        return new UsuariosResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.usuarios = this.convertValues(source["usuarios"], ValidationResult, true);
	        this.duplicates = source["duplicates"];
	        this.duplicateFields = source["duplicateFields"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class UIAvaliador {
	    id: number;
	    nome: string;
	    sigla: string;
	    email: string;
	    naoPosso: string[];
	    prefiroNao: string[];
	    extras: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new UIAvaliador(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nome = source["nome"];
	        this.sigla = source["sigla"];
	        this.email = source["email"];
	        this.naoPosso = source["naoPosso"];
	        this.prefiroNao = source["prefiroNao"];
	        this.extras = source["extras"];
	    }
	}
	export class UICandidate {
	    id: number;
	    nome: string;
	    semestre: number;
	    curso: string;
	    emailPessoal: string;
	    emailSecundario: string;
	    opcoes: string[];
	    naoPosso: string[];
	    prefiroNao: string[];
	    extras: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new UICandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nome = source["nome"];
	        this.semestre = source["semestre"];
	        this.curso = source["curso"];
	        this.emailPessoal = source["emailPessoal"];
	        this.emailSecundario = source["emailSecundario"];
	        this.opcoes = source["opcoes"];
	        this.naoPosso = source["naoPosso"];
	        this.prefiroNao = source["prefiroNao"];
	        this.extras = source["extras"];
	    }
	}
	export class UIMesa {
	    id: number;
	    horario: string;
	    descricao: string;
	    candidatos: UICandidate[];
	    avaliadores: UIAvaliador[];
	
	    static createFrom(source: any = {}) {
	        return new UIMesa(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.horario = source["horario"];
	        this.descricao = source["descricao"];
	        this.candidatos = this.convertValues(source["candidatos"], UICandidate);
	        this.avaliadores = this.convertValues(source["avaliadores"], UIAvaliador);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UIAllocationResult {
	    status: string;
	    solverStatus: string;
	    mesas: UIMesa[];
	    naoAlocados: UICandidate[];
	    score: types.SoftScoreBreakdown;
	    quality: types.AllocationQualityReport;
	    hardViolations: types.HardConstraintViolation[];
	    rejectionReason: string;
	    metrics: types.SolverMetrics;
	    debugNotes: string[];
	
	    static createFrom(source: any = {}) {
	        return new UIAllocationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.solverStatus = source["solverStatus"];
	        this.mesas = this.convertValues(source["mesas"], UIMesa);
	        this.naoAlocados = this.convertValues(source["naoAlocados"], UICandidate);
	        this.score = this.convertValues(source["score"], types.SoftScoreBreakdown);
	        this.quality = this.convertValues(source["quality"], types.AllocationQualityReport);
	        this.hardViolations = this.convertValues(source["hardViolations"], types.HardConstraintViolation);
	        this.rejectionReason = source["rejectionReason"];
	        this.metrics = this.convertValues(source["metrics"], types.SolverMetrics);
	        this.debugNotes = source["debugNotes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace types {
	
	export class Avaliador {
	    id: number;
	    nome: string;
	    email: string;
	    sigla: string;
	    extras: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Avaliador(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nome = source["nome"];
	        this.email = source["email"];
	        this.sigla = source["sigla"];
	        this.extras = source["extras"];
	    }
	}
	export class NullableString {
	
	
	    static createFrom(source: any = {}) {
	        return new NullableString(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Candidato {
	    timestamp: string;
	    nome: string;
	    cpf: string;
	    numero: string;
	    semestre: string;
	    curso: string;
	    email_secundario: string;
	    email_pessoal: string;
	    opcoes: string[];
	    extras: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Candidato(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.nome = source["nome"];
	        this.cpf = source["cpf"];
	        this.numero = source["numero"];
	        this.semestre = source["semestre"];
	        this.curso = source["curso"];
	        this.email_secundario = source["email_secundario"];
	        this.email_pessoal = source["email_pessoal"];
	        this.opcoes = source["opcoes"];
	        this.extras = source["extras"];
	    }
	}
	export class NaoAlocados {
	    candidatos: Candidato[];
	    avaliadores: Avaliador[];
	
	    static createFrom(source: any = {}) {
	        return new NaoAlocados(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidatos = this.convertValues(source["candidatos"], Candidato);
	        this.avaliadores = this.convertValues(source["avaliadores"], Avaliador);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Horario {
	    ID: number;
	    Descricao: string;
	    Candidatos: number[];
	    Avaliadores: number[];
	
	    static createFrom(source: any = {}) {
	        return new Horario(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Descricao = source["Descricao"];
	        this.Candidatos = source["Candidatos"];
	        this.Avaliadores = source["Avaliadores"];
	    }
	}
	export class Allocation {
	    horarios: Horario[];
	    naoAlocados: NaoAlocados;
	
	    static createFrom(source: any = {}) {
	        return new Allocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.horarios = this.convertValues(source["horarios"], Horario);
	        this.naoAlocados = this.convertValues(source["naoAlocados"], NaoAlocados);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AllocationExecutionResult {
	    status: string;
	    allocation?: Allocation;
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new AllocationExecutionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.allocation = this.convertValues(source["allocation"], Allocation);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ValidationMessage {
	    level: string;
	    code: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class SoftCriterionDiagnostic {
	    originalCriterion: SoftCriterion;
	    normalizedCriterion: SoftCriterion;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new SoftCriterionDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalCriterion = this.convertValues(source["originalCriterion"], SoftCriterion);
	        this.normalizedCriterion = this.convertValues(source["normalizedCriterion"], SoftCriterion);
	        this.summary = source["summary"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PreferenceMappingDiagnostic {
	    detectedValue: UniqueValueDetection;
	    originalMapping: PreferenceScheduleMapping;
	    normalizedMapping: PreferenceScheduleMapping;
	
	    static createFrom(source: any = {}) {
	        return new PreferenceMappingDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detectedValue = this.convertValues(source["detectedValue"], UniqueValueDetection);
	        this.originalMapping = this.convertValues(source["originalMapping"], PreferenceScheduleMapping);
	        this.normalizedMapping = this.convertValues(source["normalizedMapping"], PreferenceScheduleMapping);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UniqueValueDetection {
	    valorOriginal: string;
	    valorNormalizado: string;
	    ocorrencias: number;
	
	    static createFrom(source: any = {}) {
	        return new UniqueValueDetection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valorOriginal = source["valorOriginal"];
	        this.valorNormalizado = source["valorNormalizado"];
	        this.ocorrencias = source["ocorrencias"];
	    }
	}
	export class AllocationDiagnostics {
	    detectedPreferences: UniqueValueDetection[];
	    originalMappings: PreferenceScheduleMapping[];
	    normalizedMappings: PreferenceScheduleMapping[];
	    originalParams: AllocationParams;
	    normalizedParams: AllocationParams;
	    preferenceMappings: PreferenceMappingDiagnostic[];
	    softCriteria: SoftCriterionDiagnostic[];
	    validationMessages: ValidationMessage[];
	    hasErrors: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AllocationDiagnostics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detectedPreferences = this.convertValues(source["detectedPreferences"], UniqueValueDetection);
	        this.originalMappings = this.convertValues(source["originalMappings"], PreferenceScheduleMapping);
	        this.normalizedMappings = this.convertValues(source["normalizedMappings"], PreferenceScheduleMapping);
	        this.originalParams = this.convertValues(source["originalParams"], AllocationParams);
	        this.normalizedParams = this.convertValues(source["normalizedParams"], AllocationParams);
	        this.preferenceMappings = this.convertValues(source["preferenceMappings"], PreferenceMappingDiagnostic);
	        this.softCriteria = this.convertValues(source["softCriteria"], SoftCriterionDiagnostic);
	        this.validationMessages = this.convertValues(source["validationMessages"], ValidationMessage);
	        this.hasErrors = source["hasErrors"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SoftCriterion {
	    type: string;
	    columnKey: string;
	    selectedValues: string[];
	    threshold: number;
	
	    static createFrom(source: any = {}) {
	        return new SoftCriterion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.columnKey = source["columnKey"];
	        this.selectedValues = source["selectedValues"];
	        this.threshold = source["threshold"];
	    }
	}
	export class AllocationParams {
	    gruposPorHorario: number;
	    minPessoasPorGrupo: number;
	    maxPessoasPorGrupo: number;
	    avaliadoresPorGrupo: number;
	    softCriteria: SoftCriterion[];
	
	    static createFrom(source: any = {}) {
	        return new AllocationParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gruposPorHorario = source["gruposPorHorario"];
	        this.minPessoasPorGrupo = source["minPessoasPorGrupo"];
	        this.maxPessoasPorGrupo = source["maxPessoasPorGrupo"];
	        this.avaliadoresPorGrupo = source["avaliadoresPorGrupo"];
	        this.softCriteria = this.convertValues(source["softCriteria"], SoftCriterion);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PreferenceScheduleMapping {
	    valorPreferencia: string;
	    dia: string;
	    hora: string;
	
	    static createFrom(source: any = {}) {
	        return new PreferenceScheduleMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valorPreferencia = source["valorPreferencia"];
	        this.dia = source["dia"];
	        this.hora = source["hora"];
	    }
	}
	export class NormalizedAllocationInput {
	    preferenceMappings: PreferenceScheduleMapping[];
	    params: AllocationParams;
	
	    static createFrom(source: any = {}) {
	        return new NormalizedAllocationInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preferenceMappings = this.convertValues(source["preferenceMappings"], PreferenceScheduleMapping);
	        this.params = this.convertValues(source["params"], AllocationParams);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HumanSummary {
	    detectedPreferences: string[];
	    mappedPreferences: string[];
	    allocationParameters: string[];
	    softCriteria: string[];
	    normalizedValues: string[];
	    validationObservations: string[];
	
	    static createFrom(source: any = {}) {
	        return new HumanSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detectedPreferences = source["detectedPreferences"];
	        this.mappedPreferences = source["mappedPreferences"];
	        this.allocationParameters = source["allocationParameters"];
	        this.softCriteria = source["softCriteria"];
	        this.normalizedValues = source["normalizedValues"];
	        this.validationObservations = source["validationObservations"];
	    }
	}
	export class AllocationConfiguration {
	    summary: HumanSummary;
	    normalized: NormalizedAllocationInput;
	    diagnostics: AllocationDiagnostics;
	    result: AllocationExecutionResult;
	
	    static createFrom(source: any = {}) {
	        return new AllocationConfiguration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = this.convertValues(source["summary"], HumanSummary);
	        this.normalized = this.convertValues(source["normalized"], NormalizedAllocationInput);
	        this.diagnostics = this.convertValues(source["diagnostics"], AllocationDiagnostics);
	        this.result = this.convertValues(source["result"], AllocationExecutionResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class AllocationQualityCharacteristic {
	    code: string;
	    label: string;
	    description: string;
	    value: number;
	    valueLabel: string;
	    penalty: number;
	    tone: string;
	    candidateIds: number[];
	    evaluatorIds: number[];
	    groupIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new AllocationQualityCharacteristic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.value = source["value"];
	        this.valueLabel = source["valueLabel"];
	        this.penalty = source["penalty"];
	        this.tone = source["tone"];
	        this.candidateIds = source["candidateIds"];
	        this.evaluatorIds = source["evaluatorIds"];
	        this.groupIds = source["groupIds"];
	    }
	}
	export class AllocationQualityReport {
	    characteristics: AllocationQualityCharacteristic[];
	
	    static createFrom(source: any = {}) {
	        return new AllocationQualityReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.characteristics = this.convertValues(source["characteristics"], AllocationQualityCharacteristic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class BaseOptimizationPolicy {
	    preferencePenaltyByRank: number[];
	    avoidEvaluatorPenalty: number;
	
	    static createFrom(source: any = {}) {
	        return new BaseOptimizationPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preferencePenaltyByRank = source["preferencePenaltyByRank"];
	        this.avoidEvaluatorPenalty = source["avoidEvaluatorPenalty"];
	    }
	}
	export class CandidateCriterionColumn {
	    key: string;
	    label: string;
	    isExtra: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CandidateCriterionColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.isExtra = source["isExtra"];
	    }
	}
	
	export class HardConstraintViolation {
	    code: string;
	    message: string;
	    candidateId: number;
	    groupId: number;
	    evaluatorId: number;
	
	    static createFrom(source: any = {}) {
	        return new HardConstraintViolation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.candidateId = source["candidateId"];
	        this.groupId = source["groupId"];
	        this.evaluatorId = source["evaluatorId"];
	    }
	}
	
	
	export class MappingFieldInfo {
	    variavel: string;
	    required: boolean;
	    unique: boolean;
	    duplicate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MappingFieldInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.variavel = source["variavel"];
	        this.required = source["required"];
	        this.unique = source["unique"];
	        this.duplicate = source["duplicate"];
	    }
	}
	export class MappingItem {
	    nomeColuna: string;
	    indice: number;
	    variavel: string;
	    includeWhenUnmapped: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MappingItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomeColuna = source["nomeColuna"];
	        this.indice = source["indice"];
	        this.variavel = source["variavel"];
	        this.includeWhenUnmapped = source["includeWhenUnmapped"];
	    }
	}
	
	
	
	
	export class Restricao {
	    candidato: string;
	    naoPosso: string;
	    prefiroNao: string;
	
	    static createFrom(source: any = {}) {
	        return new Restricao(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidato = source["candidato"];
	        this.naoPosso = source["naoPosso"];
	        this.prefiroNao = source["prefiroNao"];
	    }
	}
	
	
	export class SoftCriterionOption {
	    type: string;
	    label: string;
	    description: string;
	    requiresThreshold: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SoftCriterionOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.requiresThreshold = source["requiresThreshold"];
	    }
	}
	export class SoftScoreComponent {
	    code: string;
	    penalty: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SoftScoreComponent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.penalty = source["penalty"];
	        this.message = source["message"];
	    }
	}
	export class SoftScoreBreakdown {
	    totalPenalty: number;
	    components: SoftScoreComponent[];
	
	    static createFrom(source: any = {}) {
	        return new SoftScoreBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalPenalty = source["totalPenalty"];
	        this.components = this.convertValues(source["components"], SoftScoreComponent);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SolverMetrics {
	    nodesVisited: number;
	    completeStates: number;
	    nodesPrunedByHard: number;
	    nodesPrunedByBound: number;
	    nodesPrunedByFlow: number;
	    branchesSkippedBySymmetry: number;
	    bestUpdates: number;
	    parallelTasks: number;
	
	    static createFrom(source: any = {}) {
	        return new SolverMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodesVisited = source["nodesVisited"];
	        this.completeStates = source["completeStates"];
	        this.nodesPrunedByHard = source["nodesPrunedByHard"];
	        this.nodesPrunedByBound = source["nodesPrunedByBound"];
	        this.nodesPrunedByFlow = source["nodesPrunedByFlow"];
	        this.branchesSkippedBySymmetry = source["branchesSkippedBySymmetry"];
	        this.bestUpdates = source["bestUpdates"];
	        this.parallelTasks = source["parallelTasks"];
	    }
	}
	
	
	export class WorkflowStep {
	    key: string;
	    label: string;
	    required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkflowStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.required = source["required"];
	    }
	}
	export class WorkflowDefinition {
	    steps: WorkflowStep[];
	    defaultAllocationParams: AllocationParams;
	    softCriterionOptions: SoftCriterionOption[];
	    baseOptimization: BaseOptimizationPolicy;
	
	    static createFrom(source: any = {}) {
	        return new WorkflowDefinition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steps = this.convertValues(source["steps"], WorkflowStep);
	        this.defaultAllocationParams = this.convertValues(source["defaultAllocationParams"], AllocationParams);
	        this.softCriterionOptions = this.convertValues(source["softCriterionOptions"], SoftCriterionOption);
	        this.baseOptimization = this.convertValues(source["baseOptimization"], BaseOptimizationPolicy);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

