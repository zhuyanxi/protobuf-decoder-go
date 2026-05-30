# Protobuf Decoder Wails 桌面版 Jira Story 拆分

日期：2026-05-30

## Epic: 无 Schema Protobuf Decoder 桌面版重写

### Epic Summary

将现有无 `.proto` schema 的 Protobuf 二进制解析工具重写为 Go + Wails 跨平台桌面应用。应用必须保留本地解码、无需 schema、快速查看字段结构和候选解释的核心体验，并覆盖 macOS、Windows、Linux。

### Business Value

- 用户可以在本地安全查看 Protobuf 二进制内容，无需上传敏感数据。
- 桌面应用支持文件打开、拖拽、导出、跨平台发布，比 Web 版本更适合日常排障。
- Go 后端实现 wire parser，提升可测试性、性能边界控制和 64-bit 数据安全性。

### Epic Scope

- Wails v2 + React + TypeScript 桌面工程。
- Go 后端 Protobuf wire-level decoder。
- hex、base64、文件输入。
- gRPC 5 字节前缀识别。
- varint length-delimited message stream 解析。
- 字段表格/树形展示、raw hex、byte range、leftover、错误提示。
- JSON/text 导出。
- macOS、Windows、Linux 构建与 smoke test。

### Epic Non-Goals

- 不实现 `.proto` schema 导入和 schema-aware 解码。
- 不恢复字段名、message 类型名、enum 名称、oneof、map 语义或默认值。
- 不保证候选解释等于真实业务类型。
- 不实现云端同步、远程解析或上传功能。

### Definition of Done

- 所有目标平台至少完成一次启动和解析 smoke test。
- README 示例、基础 wire type、gRPC header、delimited stream、非法输入都有测试覆盖。
- 所有解码逻辑本地执行，应用无解析相关网络请求。
- 10 MB 以内输入解析过程有明确限制和 loading 状态，窗口不应无响应。
- 64-bit / BigInt 候选值通过字符串返回给前端，不发生 JS number 精度丢失。

---

## Story 1: 初始化 Wails React TypeScript 桌面工程

### Jira Type

Story

### Summary

作为开发者，我需要初始化 Wails v2 + React + TypeScript 桌面工程，以便后续实现 Go 后端解析和 React 前端展示。

### User Story

作为开发者，
我希望拥有一个可运行、可构建、可调用 Go 方法的 Wails 桌面应用骨架，
以便在稳定工程基础上迁移 Protobuf Decoder 功能。

### Background

调研结论建议采用 Wails v2 稳定线，后端用 Go，前端用 React + TypeScript + Vite。M0 目标是跑通 `wails doctor`、`wails dev`、`wails build`、Go 方法绑定和原生文件选择框。

### Acceptance Criteria

- Given 本地已安装 Go、Node/NPM、Wails CLI，When 执行初始化命令，Then 生成 Wails React TypeScript 工程。
- Given 工程初始化完成，When 执行 `wails doctor`，Then 输出环境检查结果且无阻断性错误。
- Given 工程初始化完成，When 执行 `wails dev`，Then 桌面窗口可以启动并显示 React 页面。
- Given 工程初始化完成，When 执行 `wails build`，Then 生成本机平台可执行产物。
- Given React 前端调用 Go mock 方法，When 用户输入示例字符串，Then 前端能显示 Go 方法返回的 mock decode 结果。
- Given 用户点击打开文件入口，When 系统文件选择框弹出，Then Wails runtime dialog 能返回用户选择路径或取消状态。

### Implementation Tasks

- 使用 `wails init -n protobuf-decoder-desktop -t react-ts` 初始化工程。
- 保留 Go module path 与项目命名一致。
- 添加 `Decode` mock Wails 绑定方法，返回固定 `DecodeResult`。
- 添加 `OpenInputFile` mock 或真实文件选择入口，验证 runtime dialog 可用。
- 在前端添加最小输入框、按钮和结果区域，用于验证 Go binding。
- 记录本地开发命令和依赖要求。

### Technical Notes

- Wails dev 模式会生成 `wailsjs` 绑定模块，前端必须通过生成 binding 调用 Go 方法。
- macOS 15+ 建议 Go 1.23.3+；Windows 需关注 WebView2 runtime；Linux 需关注 WebKitGTK。

### Dependencies

- Go 1.21+。
- Node 15+ / NPM。
- Wails CLI。
- 平台 WebView 依赖。

### Estimate

3 Story Points

### Priority

High

---

## Story 2: 定义后端解码 API 与 JSON 数据契约

### Jira Type

Story

### Summary

作为前端开发者，我需要稳定的 Go 后端解码 API 和 JSON-friendly 数据结构，以便前端安全渲染字段、候选值、错误和 leftover。

### User Story

作为前端开发者，
我希望 Go 后端提供明确的 `DecodeRequest`、`DecodeResult`、`Part`、`ValueVariant` 数据结构，
以便前端不依赖二进制解析细节即可展示结果。

### Background

无 schema decoder 输出不能给出唯一业务语义，只能返回 wire-level 信息和候选解释。64-bit 数值如果传给 JS number 会丢精度，必须以字符串形式返回。

### Acceptance Criteria

- Given 前端传入 `DecodeRequest`，When 调用 `Decode(req)`，Then 后端返回结构化 `DecodeResult` 或明确 error。
- Given 解析结果包含 64-bit int、uint、double 原始位解释，When JSON 序列化，Then int64/uint64 候选值用 string 表达。
- Given 字段解析成功，When 返回 `Part`，Then 每个字段包含 `byteRange`、`index`、`fieldNumber`、`wireType`、`typeName`、`rawHex`、`value`。
- Given length-delimited 字段被识别为 nested protobuf，When 返回 `Part`，Then nested 字段放入 `children`，父字段仍保留 raw hex 和 byte range。
- Given 输入部分解析失败，When 返回结果，Then `Error` 描述错误位置和原因，`Leftover` 保留剩余 bytes hex。
- Given 前端只读取 JSON，When 渲染任意结果，Then 不需要重新解析 Protobuf 二进制。

### Implementation Tasks

- 定义 `DecodeRequest`，包含 `Input`、`InputEncoding`、`ParseDelimited`、`MaxDepth`、`MaxFields`、`MaxBytes`。
- 定义 `DecodeOptions`，复用文本输入和文件输入解析选项。
- 定义 `DecodeResult`，包含 `Parts`、`Leftover`、`Error`、`Warnings`、`InputSize`。
- 定义 `Part`，覆盖字段编号、wire type、类型名、字节范围、原始 hex、候选值和子节点。
- 定义 `ValueVariant`，包含候选类型、展示值、说明和可信度标记。
- 为所有 Wails 暴露结构添加 JSON tag。
- 添加 API contract 单元测试，验证 JSON 序列化字段名稳定。

### Technical Notes

- `byteRange` 使用 `[start, end)` 语义，end 为 exclusive，方便前端高亮 hex byte。
- `InputEncoding` 允许 `auto`、`hex`、`base64`。
- `MaxDepth`、`MaxFields`、`MaxBytes` 必须有后端默认值，不能只依赖前端。

### Dependencies

- Story 1 Wails 工程已初始化。

### Estimate

5 Story Points

### Priority

High

---

## Story 3: 实现输入规范化与文件读取

### Jira Type

Story

### Summary

作为用户，我需要粘贴 hex/base64 或选择本地文件作为输入，以便快速解析不同来源的 Protobuf 二进制数据。

### User Story

作为用户，
我希望应用支持文本粘贴、编码自动识别和本地文件读取，
以便不用额外转换即可解析 Protobuf 数据。

### Background

现有工具支持 hex、base64、文件上传。桌面版应保留这些输入方式，并保证所有输入都在本地进程内处理。

### Acceptance Criteria

- Given 用户选择 `hex` 输入编码，When 输入包含空格、换行或 `0x` 风格分隔，Then 后端规范化并解析有效 hex bytes。
- Given 用户选择 `base64` 输入编码，When 输入合法 base64，Then 后端解码为 bytes 并进入 parser。
- Given 用户选择 `auto` 输入编码，When 输入符合 hex 或 base64，Then 后端选择最可信编码并返回识别结果。
- Given 用户输入非法 hex，When 调用 decode，Then 返回清晰错误，不触发 panic。
- Given 用户输入非法 base64，When 调用 decode，Then 返回清晰错误，不触发 panic。
- Given 用户通过文件选择框选择文件，When 调用 `DecodeFile(path, options)`，Then 后端读取文件内容并解析。
- Given 文件不存在、无权限或超过 `MaxBytes`，When 调用 `DecodeFile`，Then 返回明确错误，不崩溃。
- Given 所有输入路径，When 解析完成，Then 原始内容不会发送到网络。

### Implementation Tasks

- 在 `internal/input` 或 `internal/decoder/input.go` 实现输入规范化。
- 实现 hex 清洗规则，支持空白字符和常见分隔符。
- 实现 base64 decode，支持标准 base64 和可选 raw base64 检测。
- 实现 `auto` 检测优先级和歧义处理规则。
- 实现 `DecodeFile(path string, options DecodeOptions)`，包含路径校验、大小限制、读取错误处理。
- 添加输入错误类型，包含编码类型、错误位置和错误原因。
- 添加输入规范化单元测试。

### Technical Notes

- `auto` 模式出现歧义时应返回 `Warnings`，提示用户可手动选择编码。
- 文件读取必须在 Go 后端完成，前端只传路径和选项。
- 大文件必须先检查大小，再读取到内存。

### Dependencies

- Story 2 后端 API 契约。

### Estimate

5 Story Points

### Priority

High

---

## Story 4: 实现 Protobuf wire reader 基础能力

### Jira Type

Story

### Summary

作为 decoder 维护者，我需要可靠的底层 reader 读取 varint、fixed32、fixed64 和 byte range，以便上层 parser 能准确消费 Protobuf wire format。

### User Story

作为 decoder 维护者，
我希望有一个边界安全的 `BufferReader`，
以便读取二进制字段时能准确记录 offset、处理截断和返回错误。

### Background

Protobuf tag 是 varint，字段内容按 wire type 消费。reader 是所有解析准确性的基础，必须防御 varint 超长、长度不足、offset 越界。

### Acceptance Criteria

- Given reader 读取 tag varint，When varint 长度小于等于 10 字节，Then 返回 `uint64` 值、开始 offset、结束 offset。
- Given reader 遇到超过 10 字节仍未结束的 varint，When 调用 read varint，Then 返回 varint overflow error。
- Given reader 读取 fixed32，When 剩余 bytes 足够，Then 按 little-endian 返回 4 字节原始值。
- Given reader 读取 fixed64，When 剩余 bytes 足够，Then 按 little-endian 返回 8 字节原始值。
- Given reader 读取 length-delimited payload，When length 超过剩余 bytes，Then 返回 truncated length-delimited error。
- Given 任意读取失败，When 返回错误，Then offset 不会错误前进到不可恢复状态。

### Implementation Tasks

- 实现 `BufferReader`，维护 `data`、`offset`、`limit`。
- 实现 `ReadVarint()`，限制最大 10 字节。
- 实现 `ReadFixed32()`、`ReadFixed64()`。
- 实现 `ReadBytes(length int)` 和 `Remaining()`。
- 实现 `Position()` 和 byte range 辅助方法。
- 定义 parser 错误类型，包含 offset、kind、message。
- 添加 reader 单元测试，覆盖正常、边界和截断场景。

### Technical Notes

- Protobuf varint 最大 10 字节，不能无限循环。
- fixed32/fixed64 只负责读取原始 little-endian bits，业务候选解释由 variants 层完成。

### Dependencies

- Story 2 后端 API 契约。

### Estimate

5 Story Points

### Priority

High

---

## Story 5: 实现 Protobuf wire parser 主流程

### Jira Type

Story

### Summary

作为用户，我需要在没有 `.proto` schema 的情况下解析普通 Protobuf message，以便查看字段编号、wire type、字节范围和原始值。

### User Story

作为用户，
我希望应用能解析 Protobuf wire format 的基础字段，
以便快速理解未知二进制 message 的结构。

### Background

即使没有 schema，Protobuf 每个字段仍包含 tag。tag 可拆分为 field number 和 wire type。parser 需要支持 wire type 0、1、2、5，并对未知 wire type 返回错误和 leftover。

### Acceptance Criteria

- Given 输入包含 wire type 0 字段，When 解析，Then 返回 VARINT 字段、field number、byte range、raw hex 和 varint 原始值。
- Given 输入包含 wire type 1 字段，When 解析，Then 返回 FIXED64 字段和 8 字节 little-endian 原始值。
- Given 输入包含 wire type 2 字段，When 解析，Then 返回 LENDELIM 字段、length、payload raw hex 和 byte range。
- Given 输入包含 wire type 5 字段，When 解析，Then 返回 FIXED32 字段和 4 字节 little-endian 原始值。
- Given 输入包含未知 wire type，When 解析，Then 返回已解析字段、错误位置、错误原因和 leftover。
- Given 输入被截断，When 解析，Then 返回已解析字段和截断错误，不崩溃。
- Given 字段数量超过 `MaxFields`，When 解析，Then 停止解析并返回 limit error。
- Given 输入大小超过 `MaxBytes`，When 解析，Then 拒绝解析并返回 size limit error。

### Implementation Tasks

- 实现 `DecodeBytes(data []byte, options DecodeOptions)` 主入口。
- 读取 tag varint 并计算 `fieldNumber = tag >> 3`、`wireType = tag & 0b111`。
- 校验 `fieldNumber > 0`。
- 分发处理 wire type 0、1、2、5。
- 对 wire type 3、4 和其他未知类型返回 unsupported wire type error。
- 记录每个字段完整 byte range 与 raw hex。
- 在错误情况下保留已解析字段和 leftover。
- 添加基础 parser 单元测试。

### Technical Notes

- 该工具是 wire-level inspector，不应宣称能恢复真实业务类型。
- parser 错误不应吞掉，应进入 `DecodeResult.Error`。

### Dependencies

- Story 4 wire reader。

### Estimate

8 Story Points

### Priority

High

---

## Story 6: 实现候选值解释策略

### Jira Type

Story

### Summary

作为用户，我需要查看同一个 wire value 的多种候选业务解释，以便在无 schema 场景下辅助判断字段含义。

### User Story

作为用户，
我希望每个字段展示合理的 candidate interpretations，
以便根据上下文判断字段可能是 int、uint、sint、float、string、bytes 或 nested message。

### Background

无 schema 解析不能确定真实业务类型。UI 必须展示候选解释，而不是把启发式结果包装成唯一结论。

### Acceptance Criteria

- Given VARINT 字段，When 展示候选值，Then 包含 unsigned、int32、int64、sint32、sint64、bool/enum hint。
- Given FIXED32 字段，When 展示候选值，Then 包含 uint32、int32、float32。
- Given FIXED64 字段，When 展示候选值，Then 包含 uint64、int64、double。
- Given LENDELIM 字段 payload 是合法 UTF-8，When 展示候选值，Then 包含 string candidate 和 bytes hex candidate。
- Given LENDELIM 字段 payload 不是合法 UTF-8，When 展示候选值，Then 至少包含 bytes hex candidate。
- Given 64-bit 整数候选值，When JSON 返回给前端，Then 使用 string，避免 JS 精度丢失。
- Given float/double 出现 NaN 或 Inf，When JSON 返回给前端，Then 使用可显示字符串，不产生非法 JSON 数值。

### Implementation Tasks

- 实现 varint unsigned 展示。
- 实现 two's complement int32/int64 展示。
- 实现 ZigZag sint32/sint64 解码。
- 实现 fixed32 uint32/int32/float32 候选解释。
- 实现 fixed64 uint64/int64/double 候选解释。
- 实现 UTF-8 string 检测和 bytes hex 展示。
- 为候选值添加 `candidate` 标识和说明。
- 添加 variants 单元测试，覆盖边界值、负数、NaN、Inf、大整数。

### Technical Notes

- ZigZag 公式：`(n >> 1) ^ -(n & 1)`，Go 中需谨慎处理 unsigned 到 signed 转换边界。
- 前端不应对候选数值再次做数学计算。

### Dependencies

- Story 5 wire parser 主流程。

### Estimate

5 Story Points

### Priority

High

---

## Story 7: 支持 nested protobuf 递归解析

### Jira Type

Story

### Summary

作为用户，我需要 length-delimited 字段在可能是嵌套 message 时显示子字段，以便快速查看嵌套结构。

### User Story

作为用户，
我希望 length-delimited 字段能尝试递归解析为 nested protobuf，
以便不用手动复制 payload 再单独解码。

### Background

length-delimited 可能是 string、bytes、packed repeated primitive 或 nested message。递归解析只能作为 candidate，且必须设置深度和字段数量限制。

### Acceptance Criteria

- Given LENDELIM payload 能完整解析为 protobuf 且无 leftover，When 解码，Then 父字段包含 nested protobuf candidate 和 `children`。
- Given LENDELIM payload 解析后有 leftover，When 解码，Then 不把它作为确定 nested protobuf，只保留解析尝试说明和 bytes/string candidate。
- Given LENDELIM payload 是空 bytes，When 解码，Then 不触发无意义递归循环。
- Given 嵌套深度达到 `MaxDepth`，When 继续遇到 LENDELIM，Then 停止递归并返回 depth limit warning。
- Given 伪造 payload 导致大量字段，When 递归解析，Then `MaxFields` 限制仍生效。
- Given nested 解析失败，When 返回父字段，Then 父字段 raw hex 和其他候选解释仍完整保留。

### Implementation Tasks

- 在 LENDELIM 处理逻辑中添加递归解析入口。
- 为递归传递当前 depth、共享或分层 field count limit。
- 判断 nested candidate 的完整性：必须无 leftover 且无 fatal error。
- 保留 string/bytes candidate，不被 nested candidate 覆盖。
- 为 depth limit 和 nested ambiguity 添加 warning。
- 添加 nested message golden tests。

### Technical Notes

- 任意 bytes 可能刚好符合 wire format，UI 必须显示 candidate，不得显示为确定事实。
- packed repeated primitive 可能被误判，需在说明中体现限制。

### Dependencies

- Story 5 wire parser 主流程。
- Story 6 候选值解释策略。

### Estimate

8 Story Points

### Priority

High

---

## Story 8: 支持 gRPC header 检测与跳过

### Jira Type

Story

### Summary

作为用户，我需要解析带 gRPC 5 字节前缀的 message，以便直接查看抓包或日志中的 gRPC payload。

### User Story

作为用户，
我希望 decoder 自动识别并跳过合法 gRPC message header，
以便解析真实 Protobuf message body。

### Background

现有逻辑在第 1 字节为 `0` 且后续 4 字节 big-endian message length 合法时，跳过 gRPC header。桌面版需要保持行为一致，并把跳过信息展示给用户。

### Acceptance Criteria

- Given 输入第 1 字节为 `0` 且剩余长度至少 5，When 后续 4 字节 big-endian length 等于剩余 message 长度，Then decoder 跳过 5 字节 header 并解析 body。
- Given header length 不合法，When 解码，Then 不跳过 header，按普通 protobuf 尝试解析或返回错误。
- Given 成功跳过 gRPC header，When 返回结果，Then 结果包含 header part 或 warning，说明 header byte range 和 message length。
- Given gRPC payload 截断，When 解码，Then 返回明确长度不足错误。
- Given compressed flag 不为 `0`，When 解码，Then 返回 unsupported compressed gRPC message error 或 warning，不误解析压缩内容。

### Implementation Tasks

- 实现 `DetectGRPCHeader(data []byte)`。
- 校验 compressed flag、message length 和 payload 长度。
- 在 decode 主流程前应用 header detection。
- 返回 header metadata，供前端展示。
- 添加 gRPC header 单元测试，覆盖合法、长度不符、截断、compressed flag。

### Technical Notes

- gRPC length 是 4 字节 big-endian。
- compressed flag 为 1 时需要压缩算法信息，无 schema decoder 不应盲目解压。

### Dependencies

- Story 5 wire parser 主流程。

### Estimate

3 Story Points

### Priority

Medium

---

## Story 9: 支持 varint length-delimited message stream

### Jira Type

Story

### Summary

作为用户，我需要解析由 varint length 前缀分隔的多条 Protobuf message，以便查看流式或批量编码数据。

### User Story

作为用户，
我希望开启 `parse delimited` 后，decoder 能按 varint length 拆分并解析多条 message，
以便查看 message stream 中每条记录。

### Background

现有实现支持可选 varint length-delimited message 流。桌面版应保留该选项，并把 delimiter 作为特殊 part 展示。

### Acceptance Criteria

- Given `ParseDelimited=false`，When 输入为普通 protobuf，Then decoder 按单条 message 解析。
- Given `ParseDelimited=true` 且输入包含合法 varint length 前缀，When 解码，Then decoder 先读取 message length，再解析该范围内的 message。
- Given stream 包含多条 message，When 解码，Then 每条 message 都被解析，并有 message index 或 delimiter part。
- Given delimiter length 超过剩余 bytes，When 解码，Then 返回 truncated delimited message error 和 leftover。
- Given 某条 message 内部解析失败，When 解码，Then 返回该 message 已解析字段、错误和后续 leftover，不静默跳过。
- Given delimiter varint 超过 10 字节，When 解码，Then 返回 varint overflow error。

### Implementation Tasks

- 在 decode options 中启用 `ParseDelimited`。
- 实现 stream 解析循环，读取 varint length 和 message payload。
- 为 delimiter 创建特殊 `Part`，包含 byte range、length、message index。
- 复用普通 message parser 解析每个 payload。
- 处理多 message 的 byte range offset 映射。
- 添加 delimited stream golden tests。

### Technical Notes

- delimiter part 不是真实 protobuf 字段，`typeName` 应明确标记为 `MessageDelimiter`。
- 子 message byte range 应映射回原始输入全局 offset。

### Dependencies

- Story 4 wire reader。
- Story 5 wire parser 主流程。

### Estimate

5 Story Points

### Priority

Medium

---

## Story 10: 建立 golden tests 保证与现有 JS 行为对齐

### Jira Type

Story

### Summary

作为维护者，我需要 golden tests 对齐迁移前后的主要行为，以便 Go decoder 替换 JS decoder 时不发生关键回归。

### User Story

作为维护者，
我希望用固定输入和预期输出验证 Go decoder，
以便重构和迁移过程中持续保证解析行为稳定。

### Background

调研要求覆盖 README 示例、varint、fixed32、fixed64、length-delimited、nested message、base64、hex、gRPC header、delimited stream、非法输入。

### Acceptance Criteria

- Given README 示例输入，When 运行 Go tests，Then 输出字段结构与 Web 版一致。
- Given varint/fixed32/fixed64/length-delimited 输入，When 运行 Go tests，Then wire type、byte range、raw hex 和候选值符合预期。
- Given nested message 输入，When 运行 Go tests，Then children 结构符合预期。
- Given hex/base64 输入，When 运行 Go tests，Then规范化 bytes 一致。
- Given gRPC header 输入，When 运行 Go tests，Then header 跳过行为符合现有逻辑。
- Given delimited stream 输入，When 运行 Go tests，Then message delimiter 和多 message 解析符合预期。
- Given 非法 wire type、截断 varint、长度不足输入，When 运行 Go tests，Then错误和 leftover 符合预期。

### Implementation Tasks

- 收集现有 Web 版 README 示例和典型输入。
- 为每类 wire type 添加最小测试样例。
- 添加 JSON golden fixture 或结构化断言。
- 添加错误场景测试。
- 将 tests 集成到 CI 命令。

### Technical Notes

- golden 输出应避免依赖 map iteration 顺序。
- 对浮点候选值使用稳定字符串格式。

### Dependencies

- Story 3 输入规范化。
- Story 5 wire parser 主流程。
- Story 6 候选值解释策略。
- Story 7 nested protobuf。
- Story 8 gRPC header。
- Story 9 delimited stream。

### Estimate

5 Story Points

### Priority

High

---

## Story 11: 实现前端输入与解码选项面板

### Jira Type

Story

### Summary

作为用户，我需要一个直接可用的输入和选项界面，以便粘贴数据、选择文件并控制解析方式。

### User Story

作为用户，
我希望首屏直接显示输入区、文件入口和解码选项，
以便快速开始解析 Protobuf 数据。

### Background

桌面版不应做营销页。首屏应是工具本身，包含输入、选项、结果区域。前端只负责用户体验，不承担核心解码。

### Acceptance Criteria

- Given 应用启动，When 首屏加载，Then 用户能看到输入区、编码选择、parse delimited 开关、限制项和 decode 按钮。
- Given 用户粘贴 hex/base64，When 点击 decode，Then 前端调用 Wails `Decode(req)` 并显示 loading 状态。
- Given 用户点击文件按钮，When 选择文件，Then 前端调用 `DecodeFile(path, options)`。
- Given 用户拖拽文件到窗口，When 文件可读，Then 应用解析该文件或显示确认提示。
- Given 用户修改 `MaxDepth`、`MaxFields`、`MaxBytes`，When 再次解析，Then 后端收到最新选项。
- Given 用户点击清空，When 操作完成，Then 输入、结果、错误和选择状态被清除。

### Implementation Tasks

- 创建输入 panel，支持多行 paste。
- 创建编码 segmented control：auto、hex、base64。
- 创建 parse delimited toggle。
- 创建 MaxDepth、MaxFields、MaxBytes 数字输入。
- 创建文件打开按钮并接入 Wails dialog。
- 创建拖拽文件处理。
- 创建 loading、empty、error 基础状态。

### Technical Notes

- 使用图标按钮和清晰 tooltip，避免大段说明文字占据工具界面。
- 限制项必须有合理默认值，防止用户不配置时触发高风险解析。

### Dependencies

- Story 1 Wails 工程。
- Story 2 后端 API 契约。
- Story 3 输入规范化与文件读取。

### Estimate

8 Story Points

### Priority

High

---

## Story 12: 实现结果树表、字段详情和 raw hex 联动

### Jira Type

Story

### Summary

作为用户，我需要用树表查看字段、候选解释和原始字节，以便快速定位未知 Protobuf message 的结构。

### User Story

作为用户，
我希望解析结果以树表展示，并能查看每个字段的 byte range、raw hex 和候选值，
以便高效分析二进制数据。

### Background

nested protobuf 需要展开/折叠。字段候选解释必须明确是 candidate。点击字段时应联动 raw hex 高亮。

### Acceptance Criteria

- Given 解码结果包含普通字段，When 前端渲染，Then 表格显示 field number、wire type、type name、byte range、candidate summary。
- Given 解码结果包含 children，When 用户点击展开，Then 展示 nested 字段树。
- Given nested 字段很多，When 默认渲染，Then nested protobuf 默认折叠，页面不被长结果淹没。
- Given 用户选择某个字段，When 字段有 byte range，Then raw hex preview 高亮对应 bytes。
- Given 字段有多个候选值，When 用户打开详情，Then 展示所有 candidate，且不把启发式猜测显示为唯一结论。
- Given 结果包含 leftover，When 渲染完成，Then leftover bytes 在辅助区域显示。
- Given 结果包含 error 或 warnings，When 渲染完成，Then 页面清晰显示错误位置、原因和 warning。

### Implementation Tasks

- 创建 result tree/table 组件。
- 创建 field detail panel。
- 创建 raw hex preview 组件，支持 byte range 高亮。
- 创建 leftover display。
- 创建 error/warning banner 或 panel。
- 实现展开/折叠状态管理。
- 添加前端组件测试或交互 smoke test。

### Technical Notes

- 表格应支持长 raw hex 截断和详情展开，避免布局被撑破。
- byte range 使用 `[start, end)` 语义，显示时可转成用户友好格式。

### Dependencies

- Story 2 后端 API 契约。
- Story 7 nested protobuf。
- Story 11 输入与选项面板。

### Estimate

8 Story Points

### Priority

High

---

## Story 13: 实现结果导出与复制

### Jira Type

Story

### Summary

作为用户，我需要复制或导出解析结果，以便把分析结果保存到工单、文档或排障记录中。

### User Story

作为用户，
我希望可以复制 JSON、导出 JSON 或文本报告，
以便复用解码结果并与团队共享。

### Background

调研建议后端提供 `ExportResult(result DecodeResult, format string)`，支持 JSON / text / copy。

### Acceptance Criteria

- Given 当前有解码结果，When 用户点击复制 JSON，Then 剪贴板获得 pretty JSON。
- Given 当前有解码结果，When 用户选择导出 JSON，Then 应用保存 JSON 文件到用户选择路径。
- Given 当前有解码结果，When 用户选择导出文本，Then 应用保存可读文本报告。
- Given 当前无结果，When 用户点击导出，Then 按钮不可用或显示明确提示。
- Given 导出路径不可写，When 保存失败，Then 显示错误，不丢失当前结果。
- Given 结果中包含 64-bit 候选值，When 导出 JSON，Then 仍以 string 保存。

### Implementation Tasks

- 实现 `ExportResult(result DecodeResult, format string)` 或前端 JSON stringify + Wails save dialog。
- 实现 JSON pretty formatter。
- 实现纯文本 formatter，包含字段编号、wire type、byte range、candidate、error、leftover。
- 接入剪贴板复制。
- 接入原生保存文件 dialog。
- 添加导出单元测试或 formatter 测试。

### Technical Notes

- 导出内容不得包含额外遥测或本机隐私路径，除非用户主动选择包含文件路径。
- JSON 格式应稳定，方便 diff。

### Dependencies

- Story 2 后端 API 契约。
- Story 12 结果展示。

### Estimate

3 Story Points

### Priority

Medium

---

## Story 14: 实现性能限制、loading 状态和大文件保护

### Jira Type

Story

### Summary

作为用户，我需要应用在解析大输入或恶意输入时保持可控，以便桌面窗口不会长时间无响应或崩溃。

### User Story

作为用户，
我希望 decoder 对大小、字段数、递归深度和耗时有清晰限制，
以便解析失败也能得到可理解反馈。

### Background

无 schema 递归解析可能误判 bytes 为 nested message，大文件也可能让 UI 卡顿。后端必须限制资源消耗，前端必须显示 loading 和限制提示。

### Acceptance Criteria

- Given 输入超过 `MaxBytes`，When 用户点击 decode，Then 后端拒绝解析并返回 size limit error。
- Given 字段数量超过 `MaxFields`，When 解析，Then 后端停止并返回 field limit warning/error。
- Given 递归深度超过 `MaxDepth`，When 解析 LENDELIM，Then 后端停止递归并返回 depth limit warning。
- Given 解析正在进行，When 前端等待结果，Then UI 显示 loading 状态且主要按钮避免重复提交。
- Given 10 MB 以内输入，When 解析，Then 过程可控，窗口不应长期无响应。
- Given 后端返回任何 limit error，When 前端展示，Then 用户能看到触发限制和建议调整项。

### Implementation Tasks

- 在后端 options 中设置默认 `MaxBytes`、`MaxFields`、`MaxDepth`。
- 在所有 parser 入口执行限制校验。
- 对递归解析共享限制计数。
- 在前端 decode 请求期间设置 loading 和 disabled 状态。
- 为大文件打开添加确认提示或限制说明。
- 添加性能边界测试。

### Technical Notes

- 限制必须以后端为准，前端校验只用于体验优化。
- 不应为了尝试完整解析而牺牲桌面窗口响应性。

### Dependencies

- Story 5 wire parser 主流程。
- Story 7 nested protobuf。
- Story 11 输入与选项面板。

### Estimate

5 Story Points

### Priority

High

---

## Story 15: 建立跨平台构建与 CI smoke test

### Jira Type

Story

### Summary

作为维护者，我需要 macOS、Windows、Linux 的构建和 smoke test，以便发布前发现平台依赖和 WebView 差异问题。

### User Story

作为维护者，
我希望 CI 在目标平台运行测试和构建，
以便每次发布都有可验证的桌面 artifact。

### Background

Wails 可跨平台，但桌面打包、签名和系统依赖验证更适合原生 OS runner 完成。Windows 关注 WebView2，Linux 关注 WebKitGTK，macOS 关注 `.app`、签名和 notarization。

### Acceptance Criteria

- Given PR 创建，When CI 运行，Then 执行 Go tests、前端 tests 和本机 Wails build smoke test。
- Given release tag 创建，When release workflow 运行，Then macOS、Windows、Linux runner 各自产生 artifact。
- Given Windows 构建，When 应用启动，Then WebView2 runtime 策略已验证或安装说明已生成。
- Given Linux 构建，When 应用启动，Then 目标发行版 WebKitGTK 依赖已记录。
- Given macOS 构建，When artifact 生成，Then `.app` 可本机启动，签名/公证状态有明确说明。
- Given 任一平台 build 失败，When 查看 CI 日志，Then 能定位失败阶段。

### Implementation Tasks

- 添加 PR CI workflow：Go tests、frontend tests、Wails build smoke。
- 添加 release workflow：三平台 native runner 构建 artifact。
- 缓存 Go 和 Node 依赖。
- 记录 Windows WebView2 策略。
- 记录 Linux WebKitGTK 依赖和测试发行版矩阵。
- 记录 macOS 签名和 notarization 后续步骤。

### Technical Notes

- 不依赖单机跨编译完成全部发布包。
- 签名和 notarization 可以作为独立发布阶段，但状态必须清晰。

### Dependencies

- Story 1 Wails 工程。
- Story 10 golden tests。
- Story 11-13 MVP 功能。

### Estimate

8 Story Points

### Priority

Medium

---

## Story 16: 补齐用户文档、安装说明和发布说明

### Jira Type

Story

### Summary

作为用户，我需要清晰的安装、使用和限制说明，以便正确理解无 schema 解码结果并完成桌面应用安装。

### User Story

作为用户，
我希望文档说明如何安装、输入数据、解读 candidate、处理错误和导出结果，
以便避免把启发式结果误认为真实 schema 语义。

### Background

无 schema Protobuf decoder 本质是 wire-level inspector。风险包括 nested/string/packed 误判、varint 类型歧义、field number 无法恢复字段名。文档必须明确这些限制。

### Acceptance Criteria

- Given 用户阅读 README，When 查看功能说明，Then 能理解应用支持 hex、base64、文件、gRPC header、delimited stream 和导出。
- Given 用户阅读限制说明，When 查看 candidate 解释，Then 能理解结果不是 schema-aware decoder 结论。
- Given Windows 用户安装，When 查看安装说明，Then 能理解 WebView2 runtime 要求和处理方式。
- Given Linux 用户安装，When 查看安装说明，Then 能看到 WebKitGTK 依赖和支持发行版范围。
- Given macOS 用户安装，When 查看安装说明，Then 能看到 `.app` 打开方式和签名/公证状态。
- Given 发布新版本，When 查看 release note，Then 能看到新增功能、修复、已知限制和 artifact 列表。

### Implementation Tasks

- 更新 README：功能、快速开始、输入格式、候选解释、隐私说明。
- 添加 troubleshooting：非法输入、leftover、截断、WebView 依赖。
- 添加 platform install 文档。
- 添加 release note 模板。
- 添加示例数据和预期结果截图。

### Technical Notes

- 文档需避免承诺恢复字段名或真实业务类型。
- 隐私说明应明确所有解码本地执行，无上传。

### Dependencies

- Story 12 结果展示。
- Story 13 导出。
- Story 15 跨平台构建。

### Estimate

3 Story Points

### Priority

Medium

---

## Suggested Release Milestones

### M0: 调研验证

- Story 1: 初始化 Wails React TypeScript 桌面工程。
- Story 2: 定义后端解码 API 与 JSON 数据契约。

### M1: Go 解码核心

- Story 3: 实现输入规范化与文件读取。
- Story 4: 实现 Protobuf wire reader 基础能力。
- Story 5: 实现 Protobuf wire parser 主流程。
- Story 6: 实现候选值解释策略。
- Story 7: 支持 nested protobuf 递归解析。
- Story 8: 支持 gRPC header 检测与跳过。
- Story 9: 支持 varint length-delimited message stream。
- Story 10: 建立 golden tests 保证与现有 JS 行为对齐。

### M2: 桌面 MVP

- Story 11: 实现前端输入与解码选项面板。
- Story 12: 实现结果树表、字段详情和 raw hex 联动。
- Story 13: 实现结果导出与复制。
- Story 14: 实现性能限制、loading 状态和大文件保护。

### M3: 发布准备

- Story 15: 建立跨平台构建与 CI smoke test。
- Story 16: 补齐用户文档、安装说明和发布说明。

## Story Dependency Map

```text
Story 1
  -> Story 2
      -> Story 3
      -> Story 4 -> Story 5 -> Story 6 -> Story 7
                              -> Story 8
                              -> Story 9
      -> Story 10
  -> Story 11 -> Story 12 -> Story 13
              -> Story 14
Story 10 + Story 11-13 -> Story 15 -> Story 16
```

## Initial Backlog Priority Order

1. Story 1: 初始化 Wails React TypeScript 桌面工程。
2. Story 2: 定义后端解码 API 与 JSON 数据契约。
3. Story 4: 实现 Protobuf wire reader 基础能力。
4. Story 5: 实现 Protobuf wire parser 主流程。
5. Story 6: 实现候选值解释策略。
6. Story 3: 实现输入规范化与文件读取。
7. Story 7: 支持 nested protobuf 递归解析。
8. Story 8: 支持 gRPC header 检测与跳过。
9. Story 9: 支持 varint length-delimited message stream。
10. Story 10: 建立 golden tests 保证与现有 JS 行为对齐。
11. Story 11: 实现前端输入与解码选项面板。
12. Story 12: 实现结果树表、字段详情和 raw hex 联动。
13. Story 14: 实现性能限制、loading 状态和大文件保护。
14. Story 13: 实现结果导出与复制。
15. Story 15: 建立跨平台构建与 CI smoke test。
16. Story 16: 补齐用户文档、安装说明和发布说明。
