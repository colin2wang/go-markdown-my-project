package symbol

// 各语言正则提取器（对齐设计文档 §7.2）。正则版允许漏报，用途是给 LLM 提供目录。

// Go: func / method
type goExtractor struct{ lineExtractor }

func newGoExtractor() Extractor {
	return &goExtractor{lineExtractor{
		langs: []string{"go"},
		rules: []lineRule{
			{re: mustCompile(`^func\s+\(([^)]+)\)\s*([A-Za-z_]\w*)`), kind: KindMethod, nameGroup: 2},
			{re: mustCompile(`^func\s+([A-Za-z_]\w*)`), kind: KindFunc},
			{re: mustCompile(`^type\s+([A-Za-z_]\w*)\s+struct\b`), kind: KindStruct},
			{re: mustCompile(`^type\s+([A-Za-z_]\w*)\s+interface\b`), kind: KindInterface},
		},
	}}
}

// Python: def / class（含 async def，带缩进）
type pythonExtractor struct{ lineExtractor }

func newPythonExtractor() Extractor {
	return &pythonExtractor{lineExtractor{
		langs: []string{"python", "py"},
		rules: []lineRule{
			{re: mustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_]\w*)`), kind: KindFunc},
			{re: mustCompile(`^\s*class\s+([A-Za-z_]\w*)`), kind: KindClass},
		},
	}}
}

// JavaScript / TypeScript: function / class / 方法简写
type jsExtractor struct{ lineExtractor }

func newJSExtractor() Extractor {
	return &jsExtractor{lineExtractor{
		langs: []string{"javascript", "js", "typescript", "ts"},
		rules: []lineRule{
			{re: mustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][\w$]*)`), kind: KindFunc},
			{re: mustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)`), kind: KindClass},
			{re: mustCompile(`^\s{2,}(?:public|private|protected|static|readonly|async|\s)*([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*\{`), kind: KindMethod},
			{re: mustCompile(`^\s*(?:export\s+)?(?:interface|type)\s+([A-Za-z_$][\w$]*)`), kind: KindInterface},
		},
	}}
}

// Java / C#: class / method
type javaExtractor struct{ lineExtractor }

func newJavaExtractor() Extractor {
	return &javaExtractor{lineExtractor{
		langs: []string{"java", "c#", "csharp"},
		rules: []lineRule{
			{re: mustCompile(`^\s*(?:(?:public|private|protected|static|final|abstract|sealed|synchronized|virtual|override)\s+)+class\s+(\w+)`), kind: KindClass},
			{re: mustCompile(`^\s*(?:public|private|protected)\s+[\w<>\[\],.\s]+?\s+(\w+)\s*\([^)]*\)\s*(?:throws\s+[\w,\s]+)?\{`), kind: KindMethod},
			{re: mustCompile(`^\s*(?:public|private|protected)\s+(?:static\s+)?(?:final\s+)?interface\s+(\w+)`), kind: KindInterface},
		},
	}}
}

// Rust: fn / struct / trait / enum / impl
type rustExtractor struct{ lineExtractor }

func newRustExtractor() Extractor {
	return &rustExtractor{lineExtractor{
		langs: []string{"rust", "rs"},
		rules: []lineRule{
			{re: mustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(?:const\s+)?(?:async\s+)?(?:unsafe\s+)?fn\s+([A-Za-z_]\w*)`), kind: KindFunc},
			{re: mustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?struct\s+([A-Za-z_]\w*)`), kind: KindStruct},
			{re: mustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?trait\s+([A-Za-z_]\w*)`), kind: KindTrait},
			{re: mustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?enum\s+([A-Za-z_]\w*)`), kind: KindClass},
		},
	}}
}

// C / C++: function / class / struct
type cExtractor struct{ lineExtractor }

func newCExtractor() Extractor {
	return &cExtractor{lineExtractor{
		langs: []string{"c", "c++", "cpp"},
		rules: []lineRule{
			{re: mustCompile(`^\s*(?:class|struct)\s+([A-Za-z_]\w*)`), kind: KindClass},
			{re: mustCompile(`^[A-Za-z_][\w:\s<>,*&~]*\s+\*?([A-Za-z_]\w*)\s*\([^;)]*\)\s*\{`), kind: KindFunc},
		},
	}}
}

// Shell: function
type shellExtractor struct{ lineExtractor }

func newShellExtractor() Extractor {
	return &shellExtractor{lineExtractor{
		langs: []string{"shell", "sh", "bash"},
		rules: []lineRule{
			{re: mustCompile(`^\s*(?:function\s+)?([A-Za-z_]\w*)\s*\(\)\s*\{`), kind: KindFunc},
		},
	}}
}

func init() {
	register(newGoExtractor())
	register(newPythonExtractor())
	register(newJSExtractor())
	register(newJavaExtractor())
	register(newRustExtractor())
	register(newCExtractor())
	register(newShellExtractor())
}
