// 简体中文语言包（作为其他语言的类型基准，新增键请先在此定义）
const zhCN = {
  app: {
    brand: 'ProjectDocs',
    projects: '项目管理',
    workbench: '工作台',
    language: '界面语言',
    help: '帮助',
    helpConfig: '配置文件说明',
    helpAbout: '程序说明',
    helpConfigSample: '项目配置文件说明',
  },
  help: {
    configTitle: '配置文件说明',
    configBody: `项目配置为 YAML 文件，保存在项目目录（默认 config/projects），每个项目一个文件：
- project_name：项目名称（列表卡片标题）
- project_path：项目源码根目录
- output_file：导出的 Markdown 文件名
- markdown_lang：文档语言（zh_cn / en）
- files / directories：包含的文件与目录（directories 递归收集）
- exclude_directories / exclude_patterns：排除目录与排除规则（如 *.log，RE2 正则）
- use_default_excludes：是否合并全局默认排除（config/exclude_defaults.yml，默认 true；含 node_modules、dist、图片、编译产物等，无需手动排除）
- skip_binary_files：按内容自动跳过二进制文件（前 8KB 含 NUL 即判定，默认 true；避免字体等文件导出成乱码）
- include_enabled / include_patterns：激活后仅加载匹配的文件
- max_file_size：单文件大小上限（字节，0 = 不限制）
- export_mode：导出模式（full / files / symbols / signatures）
- redaction：敏感信息过滤（enabled、placeholder、strategy、custom_patterns、allowlist）
- split_tokens：超过该 token 数自动分片（0 = 不分片）
- template：自定义导出模板（export_mode 为自定义模板时使用）

可在界面中编辑并保存，也可直接修改 YAML 后点击"刷新"。`,
    aboutTitle: '程序说明',
    aboutBody: `ProjectDocs 用于把项目源码聚合为 Markdown 文档，交给 LLM 分析。

使用流程：
1. 在"项目管理"页选择配置目录，新建或编辑项目配置；
2. 进入工作台，勾选要包含的文件；
3. 选择导出模式（完整代码 / 文件名清单 / 符号目录 / 签名导出），可开启敏感信息过滤；
4. 预览确认后执行导出，得到 Markdown 文件。

注意：敏感信息过滤采用两阶段流程（扫描报告 → 确认 → 导出替换），日志中只记录数量与规则名，不会落盘明文。`,
    configSampleTitle: '项目配置文件说明',
    configSampleBody: `项目配置文件位于配置目录（默认 config/projects），每个 .yml 文件对应一个项目。完整示例：

# 项目名称（列表卡片标题）
project_name: My Project
# 项目源码根目录（绝对路径）
project_path: F:/code/my-project
# 导出的 Markdown 文件名（.md/.markdown/.txt）
output_file: my_project.md
# 文档语言：zh_cn / en
markdown_lang: zh_cn

# 包含的文件（相对 project_path，可省略）
files: []
# 包含的目录（递归收集）
directories:
  - src
  - go.mod

# 排除的目录（支持 **/name）
exclude_directories:
  - "**/node_modules"
  - "**/dist"
# 排除规则（RE2 正则，如 *.log）
exclude_patterns:
  - '*.log'
  - '**/*_test.go'

# 是否合并全局默认排除（config/exclude_defaults.yml：node_modules、dist、图片、
# 编译产物等常见排除项，自动与上方规则合并去重）；默认 true，设为 false 可整体关闭
use_default_excludes: true
# 按内容自动跳过二进制文件（前 8KB 含 NUL，如字体文件）；默认 true
skip_binary_files: true

# 激活后仅加载匹配 include_patterns 的文件
include_enabled: false
include_patterns: []

# 单文件大小上限（字节，0 = 不限制）
max_file_size: 0
# 导出模式：full / files / symbols / signatures
export_mode: full
# 超过该 token 数自动分片（0 = 不分片）
split_tokens: 0

# 敏感信息过滤
redaction:
  enabled: false
  # 替换用的占位符
  placeholder: '[REDACTED]'
  # 策略：placeholder（占位符替换）/ drop_line（删除整行）
  strategy: placeholder
  # 自定义正则规则（RE2）
  custom_patterns: []
  # 放行清单，格式：相对路径:行号:规则名
  allowlist: []`,
  },
  common: {
    cancel: '取消',
    close: '关闭',
    save: '保存',
    saving: '保存中…',
    delete: '删除',
    refresh: '刷新',
    browse: '浏览…',
    copy: '复制',
    back: '上一步',
    next: '下一步',
    scan: '扫描',
    scanning: '扫描中…',
    clear: '清空',
    on: '已开启',
    off: '未开启',
  },
  log: {
    title: '日志',
    autoscroll: '自动滚动',
    clear: '清空',
    empty: '（暂无日志）',
  },
  projects: {
    dir: '项目目录',
    refresh: '刷新',
    open: '📂 打开',
    logOpenDir: '打开配置目录：{dir}',
    create: '＋ 新建项目配置',
    empty: '未找到项目配置（检查目录是否正确）',
    enter: '进入工作台 ▸',
    edit: '✏ 编辑',
    delete: '🗑 删除',
    notExported: '尚未导出',
    lastExport: '上次导出: {time}',
    language: '语言: {lang}',
    output: '→ {file}',
    deleteTitle: '确认删除',
    deleteBody: '将删除配置文件：{path}（不影响项目源码）',
    logEdit: '编辑项目配置：{path}',
    logDelete: '删除项目配置：{path}',
    logCreate: '打开新建项目向导',
    logWorkbench: '进入工作台：{path}',
    logSaved: '项目配置已保存：{path}',
    logRefresh: '刷新项目列表：{dir}',
  },
  tree: {
    title: '文件',
    search: '🔍 搜索文件…',
    selectAll: '全选',
    selectNone: '全不选',
    selectInvert: '反选',
    codeOnly: '仅代码',
    matched: '匹配 {n} 个文件',
    noFiles: '未扫描到文件',
    stats: '{count} 文件 · {size}',
    expand: '展开',
    collapse: '收起',
    logSelectAll: '文件树：全选（{n} 个文件）',
    logSelectNone: '文件树：全不选',
    logInvert: '文件树：反选（选中 {n}/{total}）',
    logCodeOnly: '文件树：仅代码（选中 {n}/{total}）',
  },
  wb: {
    back: '← 返回',
    backList: '← 返回项目列表',
    exportMode: '导出模式',
    modes: {
      full: { label: '完整代码', desc: '文件树 + 全部文件代码' },
      files: { label: '文件名清单', desc: '仅文件树与路径列表' },
      symbols: { label: '符号目录', desc: '文件名 + 方法/类名（启发式解析）' },
      signatures: { label: '签名导出', desc: '文件名 + 函数完整签名' },
    },
    lineNumbers: '显示行号',
    sigLen: '签名截断长度',
    redaction: '敏感信息过滤',
    redactEnable: '导出时自动剔除密码/密钥',
    strategy: '策略',
    strategyPlaceholder: '占位符替换',
    strategyDropLine: '删除所在行',
    replaceWith: '替换为',
    scanSensitive: '🔍 扫描敏感信息',
    noHit: '✓ 未发现敏感信息',
    hits: '⚠ 发现 {n} 处',
    viewHits: '查看 →',
    unscanned: '○ 尚未扫描',
    sizeControl: '大小控制',
    overTokens: '超过',
    compressOutput: '精简空白输出（裁行尾空白、折叠空行、去缩进，保持语法）',
    estParts: '预计切成 {n} 片',
    previewTitle: '内容预览',
    previewSummary: '当前模式: {mode} · 敏感过滤{state}',
    resultTitle: '✓ 导出成功',
    resultOutput: '输出文件',
    resultChars: '字符数',
    resultBytes: '输出大小',
    resultTokens: '≈Token',
    resultDuration: '耗时',
    resultRedaction: '脱敏',
    hitsShort: '{n} 处',
    openDir: '📂 打开目录',
    copyPath: '📋 复制路径',
    reExport: '↻ 重新导出',
    previewBtn: '预览',
    previewing: '预览中…',
    previewStats: '字符 {chars} · ≈{tokens} tokens',
    previewTruncated: '内容过长已截断',
    logPreviewOk: '预览完成：{chars} 字符（未写盘）',
    logPreviewFail: '预览失败: {err}',
    exportBtn: '执行导出 ▸',
    statusExporting: '⏳ 正在导出…',
    statusReady: '● 就绪',
    statusStats: '{count} 文件 · {size} · ≈{tokens} tokens',
    unscannedTitle: '尚未扫描敏感信息',
    unscannedBody: '已开启敏感过滤但还未扫描，建议先扫描确认命中情况。',
    scanAndContinue: '扫描并继续',
    directExport: '直接导出',
    redactOn: '开',
    redactOff: '关',
    logScan: '扫描项目：{name}',
    logScanned: '扫描完成：共 {n} 个文件',
    logModeSwitch: '切换导出模式：{from} → {to}',
    logSensitiveScan: '扫描敏感信息：{n} 个文件，策略={strategy}',
    logHits: '发现 {n} 处敏感信息',
    logNoHit: '未发现敏感信息',
    logSensitiveFail: '敏感信息扫描失败：{err}',
    logExport: '触发导出：模式={mode}，脱敏={redact}，文件={n}',
    logExportOk: '导出成功：{paths}（{ms} ms）',
    logExportFail: '导出失败：{err}',
    logOpenDir: '打开输出目录：{path}',
    logCopyPath: '已复制输出路径：{paths}',
    logCopyFail: '复制路径失败：剪贴板权限不可用',
    outlineTitle: '# 项目文档 for {name}',
    outlineTree: '├─ ## 项目文件树',
    outlineFiles: '├─ ## 项目文件 ({n})',
    outlineFile: '│   ├─ ### 文件: {path}',
    outlineMore: '│   └─ … 共 {n} 个文件',
    outlineSymbols: '├─ ## 符号目录 ({n})',
    outlineSignatures: '├─ ## 签名导出 ({n})',
  },
  sensitive: {
    title: '敏感信息命中 ({n})',
    tip: '命中仅显示掩码预览，绝不落盘明文。导出时按策略替换。',
    none: '无命中',
    ok: '知道了',
  },
  editor: {
    newTitle: '新建项目',
    editTitle: '编辑项目',
    restore: '恢复默认',
    save: '💾 保存',
    anchorBasic: '基本',
    anchorScope: '范围',
    anchorExclude: '排除',
    anchorInclude: '包含',
    anchorExport: '导出',
    anchorRedaction: '敏感',
    anchorPreview: '预检',
    sectionBasic: '基本信息',
    sectionScope: '扫描范围',
    sectionExclude: '排除规则',
    sectionInclude: '包含规则',
    includeEnabled: '激活包含规则',
    includePatterns: '包含模式',
    sectionExport: '导出设置',
    sectionRedaction: '敏感过滤',
    sectionPreview: '预检',
    hintScope: '目录将递归收集全部文件',
    projectName: '项目名称',
    projectPath: '项目路径',
    outputFile: '输出文件',
    docLang: '文档语言',
    langZh: '中文',
    langEn: 'English',
    includeFiles: '包含文件',
    includeDirs: '包含目录',
    excludeDirs: '排除目录',
    excludePatterns: '排除规则',
    useDefaultExcludes: '启用全局默认排除',
    skipBinary: '自动跳过二进制文件（按内容检测，默认开启）',
    defaultExcludes: '默认排除（自动生效，只读）',
    defaultExcludesHint: '以上规则自动与上方自定义规则合并；写入项目配置 use_default_excludes: false 可整体关闭。',
    maxSize: '大小上限',
    maxSizeSuffix: '字节（0 = 不限制）',
    exportMode: '导出模式',
    splitTokens: 'Token 分片',
    splitHint: '（0 = 不分片）',
    redactEnable: '导出时剔除密码/密钥',
    strategy: '策略',
    replaceWith: '替换为',
    customPatterns: '自定义正则规则',
    strategyDropLine: '删除整行',
    exportModeCustom: '自定义模板',
    browseDisk: '从磁盘选…',
    multiPick: '连续添加',
    allSubdirs: '添加所有子文件夹',
    noSubdirs: '该目录下没有子文件夹：{dir}',
    pathChecking: '校验中…',
    pathValid: '● 目录有效',
    pathNotFound: '● 路径不存在',
    pathNotDir: '● 不是目录',
    runPreview: '▶ 扫描预览',
    phFiles: '回车添加，如 cargo.toml',
    phDirs: '回车添加，如 src',
    phExcludeDirs: '支持 **/name',
    phExcludePatterns: '支持 *.log',
    phIncludePatterns: '支持 *.java',
    phPatterns: '每条即时校验，如 api[_-]?key',
    unsavedTitle: '有未保存的修改',
    unsavedBody: '关闭将丢失本次编辑内容。',
    discard: '放弃修改',
    keepEditing: '继续编辑',
    logOpenCreate: '打开新建项目编辑器',
    logOpenEdit: '打开编辑项目：{path}',
    logRestore: '已恢复默认值',
    logSaved: '项目配置已保存：{path}',
    logSaveFail: '保存失败：{err}',
    logPreview: '执行扫描预检',
    yamlSyncing: '⟳ 同步中…',
    yamlErrors: '● {n} 个错误',
    yamlOk: '● 校验通过',
    yamlLines: '{n} 行',
    yamlCopy: '复制',
    yamlWrap: '换行',
    yamlCopyOk: '已复制 YAML（{n} 行）',
    yamlCopyFail: '复制 YAML 失败：剪贴板权限不可用',
    previewDone: '✓ 扫描完成：{count} 文件 · {size}',
    errNameRequired: '请输入项目名称',
    errNameTooLong: '项目名称不能超过 100 字符',
    errPathRequired: '请输入项目路径',
    errOutputRequired: '请输入输出文件名',
    errOutputExt: '必须为 .md/.markdown/.txt 文档格式',
    errMaxSize: '不能为负',
    errSplitTokens: '不能为负',
    errExcludeEmpty: '排除规则不能为空',
    errIncludeEmpty: '包含规则不能为空',
    errRegex: '正则无效: {err}',
    tagDupIgnored: '重复项已忽略 {n} 条',
    dirsPicked: '已添加 {n} 个目录',
  },
  wizard: {
    titleNew: '新建项目向导',
    titleEdit: '编辑项目',
    step1: '1 基本信息',
    step2: '2 选择范围',
    step3: '3 确认',
    name: '名称 *',
    path: '项目路径 *',
    output: '输出文件 *',
    lang: '界面语言',
    pathMissing: '路径不存在',
    pathNotDir: '该路径不是目录',
    langFilter: '语言',
    langAll: '全部',
    selectAll: '全选（当前过滤）',
    codeOnly: '仅代码文件',
    rescan: '重新扫描',
    scanSummary: '共 {total} 个文件，已勾选 {checked} 个',
    noFiles: '未扫描到文件（检查项目路径与排除规则）',
    summaryName: '名称',
    summaryPath: '项目路径',
    summaryOutput: '输出文件',
    summaryLang: '界面语言',
    summaryFiles: '包含文件',
    summaryDirs: '包含目录',
    saveTo: '配置将保存到: {path}',
  },
};

export type Messages = typeof zhCN;

export default zhCN;
