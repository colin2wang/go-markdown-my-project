export namespace app {
	
	export class ExcludeDefaultsResult {
	    directories: string[];
	    files: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExcludeDefaultsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.directories = source["directories"];
	        this.files = source["files"];
	    }
	}
	export class ExportOptions {
	    mode: string;
	    splitTokens: number;
	    compress: boolean;
	    fileOverrides: string[];
	    includeLineNumbers?: boolean;
	    maxSignatureLen: number;
	    budgetTokens: number;
	    stripComments: boolean;
	    dropMarkedComments: boolean;
	    maxLineLen: number;
	    smartFullTokens: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.splitTokens = source["splitTokens"];
	        this.compress = source["compress"];
	        this.fileOverrides = source["fileOverrides"];
	        this.includeLineNumbers = source["includeLineNumbers"];
	        this.maxSignatureLen = source["maxSignatureLen"];
	        this.budgetTokens = source["budgetTokens"];
	        this.stripComments = source["stripComments"];
	        this.dropMarkedComments = source["dropMarkedComments"];
	        this.maxLineLen = source["maxLineLen"];
	        this.smartFullTokens = source["smartFullTokens"];
	    }
	}
	export class ExportResult {
	    outputPaths: string[];
	    totalChars: number;
	    tokenCount: number;
	    outputBytes: number;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputPaths = source["outputPaths"];
	        this.totalChars = source["totalChars"];
	        this.tokenCount = source["tokenCount"];
	        this.outputBytes = source["outputBytes"];
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
	export class PathCheck {
	    exists: boolean;
	    isDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PathCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.isDir = source["isDir"];
	    }
	}
	export class PreviewResult {
	    content: string;
	    totalChars: number;
	    tokenCount: number;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.totalChars = source["totalChars"];
	        this.tokenCount = source["tokenCount"];
	        this.truncated = source["truncated"];
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
	    estTokens: number;
	    byLang: Record<string, number>;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileCount = source["fileCount"];
	        this.totalBytes = source["totalBytes"];
	        this.estTokens = source["estTokens"];
	        this.byLang = source["byLang"];
	        this.warnings = source["warnings"];
	    }
	}
	export class SensitiveHit {
	    file: string;
	    line: number;
	    rule: string;
	    masked: string;
	    value?: string;
	
	    static createFrom(source: any = {}) {
	        return new SensitiveHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.line = source["line"];
	        this.rule = source["rule"];
	        this.masked = source["masked"];
	        this.value = source["value"];
	    }
	}
	export class SuggestResult {
	    config: config.ProjectConfig;
	    report: infer.Report;
	
	    static createFrom(source: any = {}) {
	        return new SuggestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], config.ProjectConfig);
	        this.report = this.convertValues(source["report"], infer.Report);
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

export namespace config {
	
	export class GlobalSettings {
	    outputEncoding: string;
	    interfaceLanguage: string;
	
	    static createFrom(source: any = {}) {
	        return new GlobalSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputEncoding = source["outputEncoding"];
	        this.interfaceLanguage = source["interfaceLanguage"];
	    }
	}
	export class RedactionConfig {
	    enabled: boolean;
	    placeholder?: string;
	    strategy?: string;
	    rule_profile?: string;
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
	        this.rule_profile = source["rule_profile"];
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
	    include_enabled?: boolean;
	    include_patterns?: string[];
	    export_mode?: string;
	    redaction?: RedactionConfig;
	    split_tokens?: number;
	    template?: string;
	    use_default_excludes?: boolean;
	    skip_binary_files?: boolean;
	    write_bom?: boolean;
	
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
	        this.include_enabled = source["include_enabled"];
	        this.include_patterns = source["include_patterns"];
	        this.export_mode = source["export_mode"];
	        this.redaction = this.convertValues(source["redaction"], RedactionConfig);
	        this.split_tokens = source["split_tokens"];
	        this.template = source["template"];
	        this.use_default_excludes = source["use_default_excludes"];
	        this.skip_binary_files = source["skip_binary_files"];
	        this.write_bom = source["write_bom"];
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

export namespace infer {
	
	export class Report {
	    fileCount: number;
	    scanned: number;
	    totalBytes: number;
	    mainLang: string;
	    byLang: Record<string, number>;
	    excludeDirs: string[];
	    excludePats: string[];
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileCount = source["fileCount"];
	        this.scanned = source["scanned"];
	        this.totalBytes = source["totalBytes"];
	        this.mainLang = source["mainLang"];
	        this.byLang = source["byLang"];
	        this.excludeDirs = source["excludeDirs"];
	        this.excludePats = source["excludePats"];
	        this.notes = source["notes"];
	    }
	}

}

