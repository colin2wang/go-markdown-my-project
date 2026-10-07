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
	    Enabled: boolean;
	    Placeholder: string;
	    Strategy: string;
	    Allowlist: string[];
	    CustomPatterns: string[];
	
	    static createFrom(source: any = {}) {
	        return new RedactionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = source["Enabled"];
	        this.Placeholder = source["Placeholder"];
	        this.Strategy = source["Strategy"];
	        this.Allowlist = source["Allowlist"];
	        this.CustomPatterns = source["CustomPatterns"];
	    }
	}
	export class ProjectConfig {
	    ProjectName: string;
	    ProjectPath: string;
	    OutputFile: string;
	    MarkdownLang: string;
	    Files: string[];
	    Directories: string[];
	    ExcludeDirectories: string[];
	    ExcludePatterns: string[];
	    MaxFileSize: number;
	    ExportMode: string;
	    Redaction: RedactionConfig;
	    SplitTokens: number;
	    Template: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProjectName = source["ProjectName"];
	        this.ProjectPath = source["ProjectPath"];
	        this.OutputFile = source["OutputFile"];
	        this.MarkdownLang = source["MarkdownLang"];
	        this.Files = source["Files"];
	        this.Directories = source["Directories"];
	        this.ExcludeDirectories = source["ExcludeDirectories"];
	        this.ExcludePatterns = source["ExcludePatterns"];
	        this.MaxFileSize = source["MaxFileSize"];
	        this.ExportMode = source["ExportMode"];
	        this.Redaction = this.convertValues(source["Redaction"], RedactionConfig);
	        this.SplitTokens = source["SplitTokens"];
	        this.Template = source["Template"];
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

