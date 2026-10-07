package symbol

// 各语言正则提取器（对齐设计文档 §4）。正则版允许漏报，用途是给 LLM 提供目录。

// kwCommon 类 C 语言共用关键词排除词表（对齐设计文档 §4），防止 `if (...) {` 误报为 method。
var kwCommon = []string{"if", "for", "while", "switch", "catch", "return", "else", "do", "try", "function"}

// Go: func / method / type
func newGoExtractor() Extractor {
	return &lineExtractor{
		langs:        []string{"go"},
		lineComments: []string{"//"},
		blockOpen:    "/*", blockClose: "*/", hasBlock: true,
		sigEnd: '{',
		typeOpeners: []rule{
			{re: mustCompile(`^type\s+([A-Za-z_]\w*)\s+struct\b`), kind: KindStruct, nameGroup: 1, emit: true},
			{re: mustCompile(`^type\s+([A-Za-z_]\w*)\s+interface\b`), kind: KindInterface, nameGroup: 1, emit: true},
			{re: mustCompile(`^type\s+([A-Za-z_]\w*)\s+=`), kind: KindType, nameGroup: 1, emit: true},
		},
		funcRules: []rule{
			{re: mustCompile(`^func\s+\(([^)]*)\)\s*([A-Za-z_]\w*)`), kind: KindMethod, nameGroup: 2, parent: parentFromReceiver, receiverGroup: 1, emit: true},
			{re: mustCompile(`^func\s+([A-Za-z_]\w*)`), kind: KindFunc, nameGroup: 1, emit: true},
		},
		keywords: kwCommon,
	}
}

// Python: def / class（缩进驱动）
func newPythonExtractor() Extractor {
	return &lineExtractor{
		langs:        []string{"python", "py"},
		lineComments: []string{"#"},
		sigEnd:       ':',
		keepSigEnd:   true,
		indentBased:  true,
		typeOpeners: []rule{
			{re: mustCompile(`^class\s+([A-Za-z_]\w*)`), kind: KindClass, nameGroup: 1, emit: true},
		},
		funcRules: []rule{
			{re: mustCompile(`^(?:async\s+)?def\s+([A-Za-z_]\w*)`), kind: KindFunc, nameGroup: 1, emit: true},
		},
		keywords: kwCommon,
	}
}

// JavaScript / TypeScript: function / class / 方法 / 箭头函数
func newJSExtractor() Extractor {
	return &lineExtractor{
		langs:        []string{"javascript", "js", "typescript", "ts"},
		lineComments: []string{"//"},
		blockOpen:    "/*", blockClose: "*/", hasBlock: true,
		sigEnd: '{',
		typeOpeners: []rule{
			{re: mustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)`), kind: KindClass, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*(?:export\s+)?(?:interface|type)\s+([A-Za-z_$][\w$]*)`), kind: KindInterface, nameGroup: 1, emit: true},
		},
		funcRules: []rule{
			{re: mustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][\w$]*)`), kind: KindFunc, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s{2,}(?:public|private|protected|static|readonly|async|\s)*([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*\{`), kind: KindFunc, nameGroup: 1, parent: parentFromStack, emit: true},
			{re: mustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?\(`), kind: KindFunc, nameGroup: 1, emit: true},
		},
		keywords: kwCommon,
	}
}

// Java / C#: class / interface / enum / record / method
func newJavaExtractor() Extractor {
	mod := `(?:public|private|protected|static|final|abstract|sealed|synchronized|virtual|override|readonly|async)\s+`
	return &lineExtractor{
		langs:        []string{"java", "c#", "csharp"},
		lineComments: []string{"//"},
		blockOpen:    "/*", blockClose: "*/", hasBlock: true,
		sigEnd: '{',
		typeOpeners: []rule{
			{re: mustCompile(`^\s*(?:` + mod + `)*(?:class|record)\s+(\w+)`), kind: KindClass, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*(?:` + mod + `)*interface\s+(\w+)`), kind: KindInterface, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*(?:` + mod + `)*enum\s+(\w+)`), kind: KindEnum, nameGroup: 1, emit: true},
		},
		funcRules: []rule{
			{re: mustCompile(`^\s*(?:` + mod + `)*[\w<>\[\],.\s]+?\s+(\w+)\s*\([^;)]*\)\s*(?:throws\s+[\w,\s]+)?\{`), kind: KindFunc, nameGroup: 1, parent: parentFromStack, emit: true},
		},
		keywords: kwCommon,
	}
}

// Rust: fn / struct / trait / enum / impl
func newRustExtractor() Extractor {
	pub := `(?:pub(?:\([^)]*\))?\s+)?`
	return &lineExtractor{
		langs:        []string{"rust", "rs"},
		lineComments: []string{"//"},
		blockOpen:    "/*", blockClose: "*/", hasBlock: true,
		sigEnd: '{',
		typeOpeners: []rule{
			{re: mustCompile(`^\s*` + pub + `struct\s+([A-Za-z_]\w*)`), kind: KindStruct, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*` + pub + `trait\s+([A-Za-z_]\w*)`), kind: KindTrait, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*` + pub + `enum\s+([A-Za-z_]\w*)`), kind: KindEnum, nameGroup: 1, emit: true},
		},
		parentOpeners: []rule{
			{re: mustCompile(`^\s*` + pub + `impl(?:<[^>]*>)?\s+(?:\w+\s+for\s+)?([A-Za-z_]\w*)`), nameGroup: 1, emit: false},
		},
		funcRules: []rule{
			{re: mustCompile(`^\s*` + pub + `(?:const\s+)?(?:async\s+)?(?:unsafe\s+)?fn\s+([A-Za-z_]\w*)`), kind: KindFunc, nameGroup: 1, parent: parentFromStack, emit: true},
		},
		keywords: kwCommon,
	}
}

// C / C++: function / class / struct
func newCExtractor() Extractor {
	return &lineExtractor{
		langs:        []string{"c", "c++", "cpp"},
		lineComments: []string{"//"},
		blockOpen:    "/*", blockClose: "*/", hasBlock: true,
		sigEnd: '{',
		typeOpeners: []rule{
			{re: mustCompile(`^\s*typedef\s+struct[^{]*?(\w+)\s*;`), kind: KindStruct, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*(?:class|struct)\s+([A-Za-z_]\w*)`), kind: KindClass, nameGroup: 1, emit: true},
		},
		funcRules: []rule{
			{re: mustCompile(`^[\w\*\s]+?\s+(\*?\w+)\s*\([^;)]*\)\s*\{`), kind: KindFunc, nameGroup: 1, parent: parentFromStack, emit: true},
		},
		keywords: kwCommon,
	}
}

// Dart: class / mixin / extension / function
func newDartExtractor() Extractor {
	return &lineExtractor{
		langs:        []string{"dart"},
		lineComments: []string{"//"},
		blockOpen:    "/*", blockClose: "*/", hasBlock: true,
		sigEnd: '{',
		typeOpeners: []rule{
			{re: mustCompile(`^\s*(?:abstract\s+)?class\s+([A-Za-z_]\w*)`), kind: KindClass, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*(?:abstract\s+)?mixin\s+([A-Za-z_]\w*)`), kind: KindClass, nameGroup: 1, emit: true},
			{re: mustCompile(`^\s*extension\s+([A-Za-z_]\w*)`), kind: KindClass, nameGroup: 1, emit: true},
		},
		funcRules: []rule{
			{re: mustCompile(`^\s*(?:@override\s+)?(?:[\w<>,?\s]+?\s+)?([A-Za-z_]\w*)\s*\([^)]*\)\s*(?:async|sync\*)?\s*\{`), kind: KindFunc, nameGroup: 1, parent: parentFromStack, emit: true},
		},
		keywords: kwCommon,
	}
}

// Shell: function
func newShellExtractor() Extractor {
	return &lineExtractor{
		langs:        []string{"shell", "sh", "bash"},
		lineComments: []string{"#"},
		funcRules: []rule{
			{re: mustCompile(`^\s*(?:function\s+)?([A-Za-z_]\w*)\s*\(\)\s*\{`), kind: KindFunc, nameGroup: 1, emit: true},
		},
		keywords: kwCommon,
	}
}

func init() {
	register(newGoExtractor())
	register(newPythonExtractor())
	register(newJSExtractor())
	register(newJavaExtractor())
	register(newRustExtractor())
	register(newCExtractor())
	register(newDartExtractor())
	register(newShellExtractor())
}
