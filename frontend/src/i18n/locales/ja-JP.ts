import type { Messages } from './zh-CN';

const jaJP: Messages = {
  app: {
    brand: 'ProjectDocs',
    projects: 'プロジェクト管理',
    workbench: 'ワークベンチ',
    language: '表示言語',
    help: 'ヘルプ',
    helpConfig: '設定ファイルの説明',
    helpAbout: 'プログラムについて',
    helpConfigSample: 'プロジェクト設定ファイル',
  },
  help: {
    configTitle: '設定ファイルの説明',
    configBody: `プロジェクト設定は YAML ファイルで、プロジェクトディレクトリ（既定 config/projects）にプロジェクトごとに 1 ファイル保存されます：
- project_name：プロジェクト名（一覧カードのタイトル）
- project_path：プロジェクトソースのルートディレクトリ
- output_file：エクスポートする Markdown のファイル名
- markdown_lang：ドキュメント言語（zh_cn / en）
- files / directories：含めるファイルとディレクトリ（directories は再帰的に収集）
- exclude_directories / exclude_patterns：除外ディレクトリと除外パターン（例: *.log、RE2 正規表現）
- use_default_excludes：グローバル既定除外（config/exclude_defaults.yml）を統合するか（既定 true；node_modules、dist、画像、ビルド成果物などを手動で書く必要なし）
- skip_binary_files：内容によりバイナリファイルを自動スキップ（先頭 8KB に NUL があればバイナリ、既定 true；フォント等の文字化けエクスポートを防止）
- include_enabled / include_patterns：有効時は一致するファイルのみ読み込み
- max_file_size：1 ファイルのサイズ上限（バイト、0 = 無制限）
- export_mode：エクスポートモード（full / files / symbols / signatures）
- redaction：機密情報フィルタ（enabled、placeholder、strategy、custom_patterns、allowlist）
- split_tokens：このトークン数を超えると自動分割（0 = 分割しない）
- template：カスタムエクスポートテンプレート（export_mode がカスタムテンプレート時に使用）

UI で編集・保存するか、YAML を直接編集して「更新」をクリックしてください。`,
    aboutTitle: 'プログラムについて',
    aboutBody: `ProjectDocs はプロジェクトのソースを Markdown に集約し、LLM 分析に渡すためのツールです。

使用手順：
1. 「プロジェクト管理」ページで設定ディレクトリを選び、プロジェクト設定を新規作成・編集する；
2. ワークベンチに入り、含めるファイルにチェックを入れる；
3. エクスポートモードを選択（全コード / ファイル一覧 / シンボル一覧 / シグネチャ）。機密情報フィルタは任意；
4. プレビューで確認後、エクスポートを実行して Markdown を取得。

注意：機密情報フィルタは 2 段階（スキャンレポート → 確認 → エクスポート時に置換）で動作し、ログには件数とルール名のみ記録され、平文は書き込まれません。`,
    configSampleTitle: 'プロジェクト設定ファイル',
    configSampleBody: `プロジェクト設定ファイルは設定ディレクトリ（既定 config/projects）にあり、.yml ファイル 1 件が 1 プロジェクトに対応します。完全な例：

# プロジェクト名（一覧カードのタイトル）
project_name: My Project
# プロジェクトソースのルートディレクトリ（絶対パス）
project_path: F:/code/my-project
# エクスポートする Markdown のファイル名（.md/.markdown/.txt）
output_file: my_project.md
# ドキュメント言語：zh_cn / en
markdown_lang: zh_cn

# 含めるファイル（project_path 相対、省略可）
files: []
# 含めるディレクトリ（再帰的に収集）
directories:
  - src
  - go.mod

# 除外ディレクトリ（**/name をサポート）
exclude_directories:
  - "**/node_modules"
  - "**/dist"
# 除外パターン（RE2 正規表現、例: *.log）
exclude_patterns:
  - '*.log'
  - '**/*_test.go'

# グローバル既定除外（config/exclude_defaults.yml：node_modules、dist、画像、
# ビルド成果物など）を統合するか。既定 true、false で無効化
use_default_excludes: true
# 内容によりバイナリファイルを自動スキップ（先頭 8KB に NUL、例: フォント）；既定 true
skip_binary_files: true

# 有効時は include_patterns に一致するファイルのみ読み込み
include_enabled: false
include_patterns: []

# 1 ファイルのサイズ上限（バイト、0 = 無制限）
max_file_size: 0
# エクスポートモード：full / files / symbols / signatures
export_mode: full
# このトークン数を超えると自動分割（0 = 分割しない）
split_tokens: 0

# 機密情報フィルタ
redaction:
  enabled: false
  # 置換に使うプレースホルダ
  placeholder: '[REDACTED]'
  # 方針：placeholder（置換）/ drop_line（行ごと削除）
  strategy: placeholder
  # カスタム正規表現ルール（RE2）
  custom_patterns: []
  # 放行リスト、形式：相対パス:行:ルール名
  allowlist: []`,
  },
  common: {
    cancel: 'キャンセル',
    close: '閉じる',
    save: '保存',
    saving: '保存中…',
    delete: '削除',
    refresh: '更新',
    browse: '参照…',
    copy: 'コピー',
    back: '戻る',
    next: '次へ',
    scan: 'スキャン',
    scanning: 'スキャン中…',
    clear: 'クリア',
    on: '有効',
    off: '無効',
  },
  log: {
    title: 'ログ',
    autoscroll: '自動スクロール',
    clear: 'クリア',
    empty: '（ログはありません）',
  },
  projects: {
    dir: 'プロジェクトディレクトリ',
    refresh: '更新',
    open: '📂 開く',
    logOpenDir: '設定ディレクトリを開く：{dir}',
    create: '＋ 新規プロジェクト設定',
    empty: 'プロジェクト設定が見つかりません（ディレクトリを確認してください）',
    enter: 'ワークベンチへ ▸',
    edit: '✏ 編集',
    delete: '🗑 削除',
    notExported: '未エクスポート',
    lastExport: '最終エクスポート: {time}',
    language: '言語: {lang}',
    output: '→ {file}',
    deleteTitle: '削除の確認',
    deleteBody: '設定ファイルを削除します：{path}（プロジェクトのソースには影響しません）',
    logEdit: 'プロジェクト設定を編集中：{path}',
    logDelete: 'プロジェクト設定を削除：{path}',
    logCreate: '新規プロジェクトウィザードを開く',
    logWorkbench: 'ワークベンチへ移動：{path}',
    logSaved: 'プロジェクト設定を保存しました：{path}',
    logRefresh: 'プロジェクト一覧を更新：{dir}',
  },
  tree: {
    title: 'ファイル',
    search: '🔍 ファイルを検索…',
    selectAll: '全選択',
    selectNone: '全解除',
    selectInvert: '反転',
    codeOnly: 'コードのみ',
    matched: '{n} 件のファイルが一致',
    noFiles: 'ファイルがスキャンされていません',
    stats: '{count} ファイル · {size}',
    expand: '展開',
    collapse: '折りたたむ',
    logSelectAll: 'ファイルツリー：全選択（{n} ファイル）',
    logSelectNone: 'ファイルツリー：全解除',
    logInvert: 'ファイルツリー：反転（選択 {n}/{total}）',
    logCodeOnly: 'ファイルツリー：コードのみ（選択 {n}/{total}）',
  },
  wb: {
    back: '← 戻る',
    backList: '← プロジェクト一覧へ戻る',
    exportMode: 'エクスポートモード',
    modes: {
      full: { label: '全コード', desc: 'ファイルツリー + 全ファイルのソース' },
      files: { label: 'ファイル一覧', desc: 'ファイルツリーとパスのみ' },
      symbols: { label: 'シンボル一覧', desc: 'ファイル名 + メソッド/クラス名（ヒューリスティック）' },
      signatures: { label: 'シグネチャ', desc: 'ファイル名 + 関数の完全なシグネチャ' },
    },
    lineNumbers: '行番号を表示',
    sigLen: 'シグネチャの切り詰め長',
    redaction: '機密情報フィルタ',
    redactEnable: 'エクスポート時にパスワード/キーを除去',
    strategy: '方針',
    strategyPlaceholder: 'プレースホルダに置換',
    strategyDropLine: '行ごと削除',
    replaceWith: '置換後',
    scanSensitive: '🔍 機密情報をスキャン',
    noHit: '✓ 機密情報は見つかりませんでした',
    hits: '⚠ {n} 件検出',
    viewHits: '表示 →',
    unscanned: '○ 未スキャン',
    sizeControl: 'サイズ制御',
    overTokens: '上限',
    compressOutput: '空白を精簡して出力（行末空白の除去・空行の折りたたみ・インデント除去）',
    estParts: '推定 {n} 分割',
    previewTitle: '内容プレビュー',
    previewSummary: '現在のモード: {mode} · 機密フィルタ{state}',
    resultTitle: '✓ エクスポート完了',
    resultOutput: '出力ファイル',
    resultChars: '文字数',
    resultBytes: '出力サイズ',
    resultTokens: '≈トークン',
    resultDuration: '所要時間',
    resultRedaction: 'マスキング',
    hitsShort: '{n} 件',
    openDir: '📂 フォルダを開く',
    copyPath: '📋 パスをコピー',
    reExport: '↻ 再エクスポート',
    previewBtn: 'プレビュー',
    previewing: 'プレビュー中…',
    previewStats: '{chars} 文字 · ≈{tokens} トークン',
    previewTruncated: '内容が長いため切断されました',
    logPreviewOk: 'プレビュー完了：{chars} 文字（未書き込み）',
    logPreviewFail: 'プレビュー失敗: {err}',
    exportBtn: 'エクスポート実行 ▸',
    statusExporting: '⏳ エクスポート中…',
    statusReady: '● 準備完了',
    statusStats: '{count} ファイル · {size} · ≈{tokens} トークン',
    unscannedTitle: '機密情報を未スキャン',
    unscannedBody: '機密フィルタが有効ですが、まだスキャンしていません。先にスキャンすることを推奨します。',
    scanAndContinue: 'スキャンして続行',
    directExport: 'そのままエクスポート',
    redactOn: '有効',
    redactOff: '無効',
    logScan: 'プロジェクトをスキャン：{name}',
    logScanned: 'スキャン完了：{n} ファイル',
    logModeSwitch: 'エクスポートモードを変更：{from} → {to}',
    logSensitiveScan: '機密情報をスキャン：{n} ファイル、方針={strategy}',
    logHits: '{n} 件の機密情報を検出',
    logNoHit: '機密情報は見つかりませんでした',
    logSensitiveFail: '機密情報スキャン失敗：{err}',
    logExport: 'エクスポート開始：モード={mode}，マスキング={redact}，ファイル={n}',
    logExportOk: 'エクスポート成功：{paths}（{ms} ms）',
    logExportFail: 'エクスポート失敗：{err}',
    logOpenDir: '出力フォルダを開く：{path}',
    logCopyPath: '出力パスをコピーしました：{paths}',
    logCopyFail: 'コピー失敗：クリップボード権限がありません',
    outlineTitle: '# {name} のプロジェクトドキュメント',
    outlineTree: '├─ ## プロジェクトファイルツリー',
    outlineFiles: '├─ ## プロジェクトファイル ({n})',
    outlineFile: '│   ├─ ### ファイル: {path}',
    outlineMore: '│   └─ … 全 {n} ファイル',
    outlineSymbols: '├─ ## シンボル一覧 ({n})',
    outlineSignatures: '├─ ## シグネチャ ({n})',
  },
  sensitive: {
    title: '機密情報の検出 ({n})',
    tip: 'マスク済みプレビューのみ表示し、平文をディスクに書き込みません。エクスポート時は方針に従って置換します。',
    none: '検出なし',
    ok: '了解',
  },
  editor: {
    newTitle: '新規プロジェクト',
    editTitle: 'プロジェクト編集',
    restore: '既定値に戻す',
    save: '💾 保存',
    anchorBasic: '基本',
    anchorScope: '範囲',
    anchorExclude: '除外',
    anchorInclude: '包含',
    anchorExport: '出力',
    anchorRedaction: '機密',
    anchorPreview: '事前確認',
    sectionBasic: '基本情報',
    sectionScope: 'スキャン範囲',
    sectionExclude: '除外ルール',
    sectionInclude: '包含ルール',
    includeEnabled: '包含ルールを有効化',
    includePatterns: '包含パターン',
    sectionExport: 'エクスポート設定',
    sectionRedaction: '機密フィルタ',
    sectionPreview: '事前確認',
    hintScope: 'ディレクトリは再帰的に収集されます',
    projectName: 'プロジェクト名',
    projectPath: 'プロジェクトパス',
    outputFile: '出力ファイル',
    docLang: 'ドキュメント言語',
    langZh: '中国語',
    langEn: 'English',
    includeFiles: '含めるファイル',
    includeDirs: '含めるディレクトリ',
    excludeDirs: '除外ディレクトリ',
    excludePatterns: '除外パターン',
    useDefaultExcludes: 'グローバル既定除外を有効化',
    skipBinary: 'バイナリファイルを自動スキップ（内容判定、既定オン）',
    defaultExcludes: '既定除外（自動・読み取り専用）',
    defaultExcludesHint: '上のカスタムルールと自動的に統合されます。プロジェクト設定で use_default_excludes: false にすると無効化できます。',
    maxSize: 'サイズ上限',
    maxSizeSuffix: 'バイト（0 = 無制限）',
    exportMode: 'エクスポートモード',
    splitTokens: 'トークン分割',
    splitHint: '（0 = 分割しない）',
    redactEnable: 'エクスポート時にパスワード/キーを除去',
    strategy: '方針',
    replaceWith: '置換後',
    customPatterns: 'カスタム正規表現ルール',
    strategyDropLine: '行全体を削除',
    exportModeCustom: 'カスタムテンプレート',
    browseDisk: 'ディスクから選択…',
    multiPick: '連続追加',
    allSubdirs: 'すべてのサブフォルダを追加',
    noSubdirs: 'サブフォルダがありません: {dir}',
    pathChecking: '検証中…',
    pathValid: '● 有効なディレクトリ',
    pathNotFound: '● パスが存在しません',
    pathNotDir: '● ディレクトリではありません',
    runPreview: '▶ スキャンプレビュー',
    phFiles: 'Enter で追加（例: cargo.toml）',
    phDirs: 'Enter で追加（例: src）',
    phExcludeDirs: '**/name をサポート',
    phExcludePatterns: '*.log をサポート',
    phIncludePatterns: '*.java をサポート',
    phPatterns: '入力ごとに検証（例: api[_-]?key）',
    unsavedTitle: '未保存の変更があります',
    unsavedBody: '閉じると今回の編集内容は失われます。',
    discard: '変更を破棄',
    keepEditing: '編集を続ける',
    logOpenCreate: '新規プロジェクトエディタを開く',
    logOpenEdit: 'プロジェクト編集を開く：{path}',
    logRestore: '既定値に戻しました',
    logSaved: 'プロジェクト設定を保存しました：{path}',
    logSaveFail: '保存失敗：{err}',
    logPreview: 'スキャン事前確認を実行',
    yamlSyncing: '⟳ 同期中…',
    yamlErrors: '● {n} 件のエラー',
    yamlOk: '● 検証に合格',
    yamlLines: '{n} 行',
    yamlCopy: 'コピー',
    yamlWrap: '折り返し',
    yamlCopyOk: 'YAML をコピーしました（{n} 行）',
    yamlCopyFail: 'YAML のコピー失敗：クリップボード権限がありません',
    previewDone: '✓ スキャン完了：{count} ファイル · {size}',
    errNameRequired: 'プロジェクト名を入力してください',
    errNameTooLong: 'プロジェクト名は 100 文字以内です',
    errPathRequired: 'プロジェクトパスを入力してください',
    errOutputRequired: '出力ファイル名を入力してください',
    errOutputExt: '.md/.markdown/.txt のいずれかを指定してください',
    errMaxSize: '負の値は指定できません',
    errSplitTokens: '負の値は指定できません',
    errExcludeEmpty: '除外パターンを空にできません',
    errIncludeEmpty: '包含パターンを空にできません',
    errRegex: '正規表現が無効: {err}',
    tagDupIgnored: '重複 {n} 件を無視しました',
    dirsPicked: '{n} 件のディレクトリを追加しました',
  },
  wizard: {
    titleNew: '新規プロジェクトウィザード',
    titleEdit: 'プロジェクト編集',
    step1: '1 基本情報',
    step2: '2 範囲選択',
    step3: '3 確認',
    name: '名前 *',
    path: 'プロジェクトパス *',
    output: '出力ファイル *',
    lang: '表示言語',
    pathMissing: 'パスが存在しません',
    pathNotDir: 'パスはディレクトリではありません',
    langFilter: '言語',
    langAll: 'すべて',
    selectAll: '全選択（現在の絞り込み）',
    codeOnly: 'コードファイルのみ',
    rescan: '再スキャン',
    scanSummary: '全 {total} ファイル、{checked} 件選択中',
    noFiles: 'ファイルがスキャンされませんでした（パスと除外ルールを確認してください）',
    summaryName: '名前',
    summaryPath: 'プロジェクトパス',
    summaryOutput: '出力ファイル',
    summaryLang: '表示言語',
    summaryFiles: '含めるファイル',
    summaryDirs: '含めるディレクトリ',
    saveTo: '設定の保存先: {path}',
  },
};

export default jaJP;
