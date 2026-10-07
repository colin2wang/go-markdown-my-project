export namespace app {
	
	export class ExportOptions {
	    mode: string;
	    redact: boolean;
	    splitTokens: number;
	    fileOverrides: string[];
	    includeLineNumbers?: boolean;
	    maxSignatureLen: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.redact = source["redact"];
	        this.splitTokens = source["splitTokens"];
	        this.fileOverrides = source["fileOverrides"];
	        this.includeLineNumbers = source["includeLineNumbers"];
	        this.maxSignatureLen = source["maxSignatureLen"];
	    }
	}
	export class ExportResult {
	    outputPaths: string[];
	    totalChars: number;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputPaths = source["outputPaths"];
	        this.totalChars = source["totalChars"];
	        this.durationMs = source["durationMs"];
	    }
	}
	export class FieldError {
	    field: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new FieldError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.message = source["message"];
	    }
	}
	export class FileInfo {
	    path: string;
	    language: string;
	    sizeBytes: number;
	    lines: number;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.language = source["language"];
	        this.sizeBytes = source["sizeBytes"];
	        this.lines = source["lines"];
	    }
	}
	export class ProjectSummary {
	    configPath: string;
	    name: string;
	    path: string;
	    outputFile: string;
	    exportMode: string;
	    markdownLang: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configPath = source["configPath"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.outputFile = source["outputFile"];
	        this.exportMode = source["exportMode"];
	        this.markdownLang = source["markdownLang"];
	    }
	}
	export class ScanPreview {
	    fileCount: number;
	    totalBytes: number;
	    byLang: Record<string, number>;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileCount = source["fileCount"];
	        this.totalBytes = source["totalBytes"];
	        this.byLang = source["byLang"];
	        this.warnings = source["warnings"];
	    }
	}
	export class SensitiveHit {
	    file: string;
	    line: number;
	    rule: string;
	    masked: string;
	
	    static createFrom(source: any = {}) {
	        return new SensitiveHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.line = source["line"];
	        this.rule = source["rule"];
	        this.masked = source["masked"];
	    }
	}

}

export namespace config {
	
	export class RedactionConfig {
	    enabled: boolean;
	    placeholder?: string;
	    strategy?: string;
	    allowlist?: string[];
	    custom_patterns?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RedactionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.placeholder = source["placeholder"];
	        this.strategy = source["strategy"];
	        this.allowlist = source["allowlist"];
	        this.custom_patterns = source["custom_patterns"];
	    }
	}
	export class ProjectConfig {
	    project_name: string;
	    project_path: string;
	    output_file: string;
	    markdown_lang?: string;
	    files?: string[];
	    directories?: string[];
	    exclude_directories?: string[];
	    exclude_patterns?: string[];
	    max_file_size?: number;
	    export_mode?: string;
	    redaction?: RedactionConfig;
	    split_tokens?: number;
	    template?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project_name = source["project_name"];
	        this.project_path = source["project_path"];
	        this.output_file = source["output_file"];
	        this.markdown_lang = source["markdown_lang"];
	        this.files = source["files"];
	        this.directories = source["directories"];
	        this.exclude_directories = source["exclude_directories"];
	        this.exclude_patterns = source["exclude_patterns"];
	        this.max_file_size = source["max_file_size"];
	        this.export_mode = source["export_mode"];
	        this.redaction = this.convertValues(source["redaction"], RedactionConfig);
	        this.split_tokens = source["split_tokens"];
	        this.template = source["template"];
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

