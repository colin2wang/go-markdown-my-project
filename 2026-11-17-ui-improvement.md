先提一句：你这条消息开头出现的「摩根纳斯达克100指数(QDII)」看起来是误粘贴的内容，与本项目无关，我就不展开了——如有需要可以单独再问。
下面基于本轮可见的代码（`App.vue` 顶栏/弹窗/toast 结构、组件清单、前几轮定稿的智能省流参数区）给出 UI 建议，按区域组织。
## 一、全局设计系统
CSS 变量体系（`--bg-1`/`--fg-1`/`--accent`/`--line`/`--r-sm`）底子不错，建议补齐：
- **间距刻度**：统一为 4/8/12/16/24px，避免各组件随手写 margin。目前顶栏 `gap: 20px`、面板 padding 各处不一致。
- **层级（elevation）规范**：modal、dropdown、toast、drawer 四类浮层用两档阴影即可，避免越做越重。
- **明暗主题**：变量已就位，加个 `data-theme` 切换成本很低。可作为全局设置项（需要后端 `GlobalSettings` 加字段，属于可选项）。
- **动效克制**：所有 Transition 包一层 `@media (prefers-reduced-motion: reduce)` 降级。
## 二、顶栏（App.vue 可见部分）
1. **三个 `help-btn` 语义混淆**：语言、设置、帮助共用同一按钮样式。建议视觉分组：语言选择器独立（已是这样），「设置」改为纯齿轮图标按钮，「帮助」保留文字+chevron，中间加一条 1px 分隔线。
2. **设置=modal、帮助=dropdown 交互不一致**：可接受，但帮助里的「配置说明/示例/关于」点击后跳到 modal，路径较长。建议「关于」直接做成 dropdown 内的小面板或页脚项。
3. **面包屑**：`crumb-name` 有 `title` 提示和 `crumb-text` 截断，很好。建议补一个细节——项目名超长时用**中间省略**（`text-overflow` 只截尾部，路径类名称截中间更可读）。
4. **键盘导航**：dropdown 已有 `role="menu"`/`menuitemradio`，缺方向键上下移动和 Esc 关闭。补一个通用 composable `useDropdown(keys)` 一次解决两处。
```ts
// 伪码：两个 dropdown 的 Esc 互斥已有（langOpen/helpOpen 手动互斥），
// 建议封装成 useExclusiveDropdown('lang' | 'help')，避免到处手写 `helpOpen = false`
```
## 三、工作台（重点区域）
1. **智能省流参数区**（上轮定稿的 checkbox + 三个数字框）：
    - 用 `<fieldset>` + legend 包成「智能省流参数」分组，与导出模式卡片视觉分离；
    - 三个数字框加「恢复默认」链接（一键重置为 0/true）；
    - 在每个 hint 旁展示**当前生效值**，如 `生效 1200`（`smartFullTokens || 1200` 计算属性），用户不用心算「0 是多少」；
    - disabled 态除了 `opacity:.5`，补 `cursor: not-allowed` + `aria-disabled`。
2. **模式卡片**：用 `role="radiogroup"`/`aria-checked` 实现真单选语义，方向键切换；desc 保持一句话，长说明进 tooltip 或折叠的「详情」。
3. **导出前预估**：`estimate.ts` 已存在——在预览/导出按钮旁常驻显示 `预计 ≈N tokens`，随选项变化实时更新，这是这类工具最有价值的反馈。
4. **导出中反馈**：`RunExport` 是同步长任务，期间按钮置灰 + spinner + 已耗时秒数；完成后出结果卡片：
    - 输出路径列表每项带「复制路径」和「在文件管理器中显示」（后端 `OpenPath` 已有，直接接上）；
    - 显示 chars/tokens/耗时/过滤命中数（`ExportResult` 字段齐全）。
5. **分片选项互斥提示**：后端逻辑是 `SplitParts > 1` 优先于 `SplitTokens`，UI 上两者同时填写时给一行提示「已按份数优先」，避免用户疑惑。
6. **预览截断横幅**：`PreviewResult.Truncated` 目前大概率没被消费——在预览区顶部加「预览已截断至前 10 万字符，完整内容以导出文件为准」。
## 四、日志面板与 Toast
- **LogPanel**（slog 全量转发到前端）做成 VS Code 终端式底部面板：
    - level 过滤 chips（debug/info/warn/error）+ 关键字搜索 + 自动滚动开关 + 复制全部 + 清屏（仅前端清）；
    - 顶栏出现 error 时给一个小红点徽标，点击展开面板。
- **Toast**：当前单实例 + `:key="seq"` 替换。建议：
    - error 类时长加长（如 8s）且可手动关闭；
    - 加图标区分 info/success/error，error 用 `--danger` 色；
    - 连续导出日志类信息别走 toast，走 LogPanel，避免刷屏。
## 五、弹窗与表单
1. **校验错误映射**：`ValidateConfig` 返回 `FieldError{field, message}`，ProjectEditorDialog 里把 `field`（`project_name`/`project_path`…）映射到对应输入框下方红字并聚焦第一个错误项，而不是首条错误塞进一个全局 error string。
2. **模态焦点管理**：现在只有 `@click.self` 关闭，补 Esc 关闭 + 打开时焦点陷阱 + 关闭后焦点还原到触发按钮。
3. **YamlPreview 双栏联动**：项目编辑器左侧表单、右侧 YAML 预览，当前编辑字段的对应 YAML 行高亮（至少滚动跟随），这是 `SerializeYAML` 接口现成能支撑的高价值细节。
4. **数字输入**：所有数值框统一 `min="0"` + 失焦时负数回弹 + `step` 合理（token 类 step=100）。
## 六、文案与 i18n
- 日语文本普遍偏长，**避免给按钮/标签定死宽度**，用 `min-width` + 自适应；三种语言各过一遍换行表现。
- 「0 = 默认 1200」这类约定统一成一种措辞模板（zh/en/ja 三份对齐），别一处写「0=默认」一处写「留空使用默认值」。
## 优先级小结
| 级别 | 事项 |
|---|---|
| P0 | 导出中/完成的状态反馈（spinner + 结果卡 + OpenPath）；FieldError 表单映射；预览截断横幅 |
| P1 | LogPanel 过滤/徽标；智能省流参数 fieldset + 生效值展示 + 恢复默认；dropdown 键盘导航 |
| P2 | 明暗主题；YAML 联动高亮；中间省略号；reduced-motion；toast 图标分级 |
WorkbenchView 的完整代码在这份文档里被截断了，以上第 2、3 节涉及它的建议请对照实际结构取舍；如果你贴出该文件或全局 `style.css`，我可以给出更具体的选择器级修改。
