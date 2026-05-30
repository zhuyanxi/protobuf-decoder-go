# Protobuf Decoder Wails 桌面版技术调研方案

日期：2026-05-30

## 1. 背景

我想在没有 `.proto` schema 的情况下解析 Protobuf 二进制数据，并把结果展示成表格。现有实现特点：

- 输入支持 hex、base64、文件上传。
- 解码完全本地执行，不上传数据。
- 支持普通 Protobuf message、gRPC 5 字节前缀、可选 varint length-delimited message 流。
- 输出字段编号、wire type、字节范围、候选内容解释。

目标：用 Go + Wails 重写为跨平台桌面应用，覆盖 macOS、Windows、Linux，并保留“无需 schema、本地解码、快速查看”的核心体验。

## 2. 无 schema 反序列化原理

### 2.1 Protobuf wire format 够用信息

Protobuf 二进制流不是完全自描述格式，但每个字段都会带 tag。tag 是 varint，结构如下：

```text
tag = field_number << 3 | wire_type
field_number = tag >> 3
wire_type = tag & 0b111
```

所以即使没有 `.proto`，仍可知道：

- 字段编号：`field_number`。
- 底层编码类型：`wire_type`。
- 当前字段占用多少字节。

但无法可靠知道：字段名、业务类型、message 类型名、enum 名称、oneof、map、默认值、required/optional 语义、packed repeated 语义、真实字符串还是 bytes、真实 int/uint/sint/fixed/float 语义。

### 2.2 当前解析流程

现有 `decodeProto(buffer, parseDelimited)` 逻辑可概括为：

1. 创建顺序 reader，维护 `offset`。
2. 尝试跳过 gRPC header：若第 1 字节为 `0` 且剩余长度至少 5 字节，则读取后续 4 字节 big-endian message length；如果长度合法，跳过 header。
3. 如果开启 `parseDelimited`，且当前 offset 到达上一条消息结尾，则先读取 varint message length，并把它作为特殊 `Message delimiter` 输出。
4. 循环读取字段：
   - 读 tag varint。
   - 低 3 位作为 wire type。
   - 高位右移 3 位作为 field number。
   - 按 wire type 消费后续字节。
5. 支持 wire type：
   - `0 VARINT`：再读一个 varint。
   - `1 FIXED64`：读 8 字节 little-endian 原始值。
   - `2 LENDELIM`：先读 varint length，再读 length 字节。
   - `5 FIXED32`：读 4 字节 little-endian 原始值。
6. 遇到未知 wire type、长度不够、varint 越界时停止，返回已解析字段和剩余 bytes。

### 2.3 候选解释策略

因为没有 schema，同一个原始值可能对应多个业务类型。当前 UI 用“候选解释”而非“唯一结论”：

- `VARINT`：展示 unsigned、不同 bit 宽度下的 two's complement signed 值、ZigZag `sint` 值。
- `FIXED32`：展示 int32、uint32、float32 候选值。
- `FIXED64`：展示 int64、uint64、double 候选值。
- `LENDELIM`：先尝试把内容递归解析成嵌套 Protobuf；若能完整解析且无剩余字节，则展示为 nested protobuf；否则尝试 UTF-8 string；失败则展示 bytes hex。

这类工具本质是 wire-level inspector，不是 schema-aware decoder。它能帮助人快速定位字段结构和内容，但不能保证字段真实业务含义。

### 2.4 典型误判来源

- 任意 bytes 可能刚好符合 Protobuf wire format，被误判为嵌套 message。
- packed repeated primitive 也是 length-delimited，可能被误判为 bytes 或 nested message。
- `bool`、`enum`、`int32`、`uint32`、`int64` 都可能表现为 varint。
- `sint32/sint64` 使用 ZigZag，需要 schema 才知道是否应采用 ZigZag 解释。
- `string` 与 `bytes` 同属 length-delimited，只能用 UTF-8 有效性启发式猜测。
- field number 只能给出数字，无法恢复字段名。

## 3. Wails 可行性结论

Wails 适合此项目，原因：

- 技术匹配：Wails 用 Go 写后端，用 Web 技术写 UI。现有 React/Vite 前端可迁移成本低。
- 桌面能力：支持 macOS、Windows、Linux；可用原生窗口、菜单、文件对话框。
- 体积优势：Wails 不打包 Chromium，复用系统 WebView，通常比 Electron 更轻。
- Go 解码适配：Protobuf wire parser 用 Go 实现直接、可测试、可处理大整数和字节流。
- 本地隐私：所有解码在本机进程内完成，符合当前 Web 版“本地解析”特性。

建议方案：采用 Wails v2 稳定线，React + TypeScript 前端，Go 后端实现解码核心，前端只做输入、状态、结果展示和交互。

## 4. 目标架构

```text
Wails App
├── Go backend
│   ├── app/service: Wails 绑定方法
│   ├── decoder: Protobuf wire parser
│   ├── input: hex/base64/file 读取与规范化
│   └── export: JSON / text / copy 支持
└── React frontend
    ├── input panel
    ├── decode options
    ├── result tree/table
    ├── byte range / hex preview
    └── error + leftover display
```

### 4.1 Go 后端职责

后端提供稳定 API：

```go
type DecodeRequest struct {
    Input          string `json:"input"`
    InputEncoding  string `json:"inputEncoding"` // auto, hex, base64
    ParseDelimited bool   `json:"parseDelimited"`
    MaxDepth       int    `json:"maxDepth"`
    MaxFields      int    `json:"maxFields"`
}

type DecodeResult struct {
    Parts    []Part `json:"parts"`
    Leftover string `json:"leftover"`
    Error    string `json:"error,omitempty"`
}

type Part struct {
    ByteRange [2]int         `json:"byteRange"`
    Index     int            `json:"index"`
    WireType  int            `json:"wireType"`
    TypeName  string         `json:"typeName"`
    RawHex    string         `json:"rawHex"`
    Value     []ValueVariant `json:"value"`
    Children  []Part         `json:"children,omitempty"`
}
```

绑定方法建议：

- `Decode(req DecodeRequest) (DecodeResult, error)`：解析文本输入。
- `DecodeFile(path string, options DecodeOptions) (DecodeResult, error)`：读取本地文件并解析。
- `OpenInputFile() (string, error)`：用 Wails runtime 打开原生文件选择框。
- `ExportResult(result DecodeResult, format string) (string, error)`：保存 JSON 或文本。

### 4.2 React 前端职责

前端负责用户体验，不再承担核心解码：

- 输入区：hex/base64 paste、文件选择、拖拽文件。
- 选项区：auto/hex/base64、parse delimited、最大递归深度、最大字段数。
- 结果区：树表混合展示，支持展开/折叠 nested protobuf。
- 辅助区：byte range、raw hex、leftover bytes、错误提示。
- 操作区：复制 JSON、导出结果、清空、加载示例。

### 4.3 数据流

```text
用户输入/文件
  -> React 收集请求
  -> Wails generated binding 调 Go Decode/DecodeFile
  -> Go parser 返回结构化 JSON
  -> React 渲染树表
```

## 5. Go 解码器设计

### 5.1 包结构

```text
internal/decoder/
├── decoder.go       // Decode 主流程
├── reader.go        // BufferReader, varint, fixed32, fixed64
├── variants.go      // 候选解释
├── input.go         // hex/base64 auto parse
├── grpc.go          // gRPC header 检测
└── decoder_test.go
```

### 5.2 关键实现点

- 使用 `uint64` 承载 varint 原始值；Protobuf varint 最大 10 字节。
- ZigZag 解码实现：`(n >> 1) ^ -(n & 1)`，注意 Go unsigned 到 signed 转换边界。
- fixed32/fixed64 使用 `encoding/binary.LittleEndian`。
- float 使用 `math.Float32frombits`、`math.Float64frombits`。
- length-delimited 递归解析必须设置 `MaxDepth`，避免深层伪消息导致栈过深。
- 设置 `MaxFields` / `MaxBytes`，避免大文件或恶意输入卡 UI。
- 不要把解析错误直接丢弃；返回错误位置、原因、leftover，帮助用户定位坏数据。

### 5.3 与现有 JS 行为对齐

需建立 golden tests，保证迁移前后主要输入输出一致：

- README 示例数据。
- varint、fixed32、fixed64、length-delimited、nested message。
- base64 与 hex 输入。
- gRPC header 跳过。
- delimited message 流。
- 非法 wire type、截断 varint、长度不足。

## 6. Wails 工程方案

### 6.1 初始化方式

推荐新建 Wails 工程，再迁移 UI：

```bash
wails init -n protobuf-decoder-desktop -t react-ts
```

再迁移当前 React 组件，逐步改为 TypeScript。

### 6.2 开发命令

```bash
wails doctor
wails dev
wails build
```

Wails dev 模式会启动桌面 app、监听 Go 文件变更、支持前端热加载，并生成 `wailsjs` 绑定模块。生产构建会把前端 assets embed 到 Go binary。

### 6.3 平台依赖

Wails v2.12 文档要求：

- Go 1.21+；macOS 15+ 建议 Go 1.23.3+。
- Node / NPM，Node 15+。
- Windows 10/11 需要 WebView2 runtime。
- macOS 支持 Intel 与 Apple Silicon。
- Linux 支持 AMD64/ARM64，但需验证目标发行版 WebKitGTK 依赖。

### 6.4 构建与发布

目标产物：

- macOS：`.app`，后续可加 dmg、签名、notarization。
- Windows：`.exe`，可选 NSIS installer，配置 WebView2 strategy。
- Linux：二进制包；后续调研 AppImage/deb/rpm。

建议 CI：

- 每个平台用对应 runner 原生构建。
- PR 跑 Go tests + frontend tests + Wails build smoke test。
- Release tag 触发三平台构建与 artifact 上传。

跨平台构建虽有 `wails build -platform` 支持，但桌面打包、签名、系统依赖验证更适合原生 OS runner 完成。

## 7. UI/UX 改造建议

现有 Web UI 够简单，但桌面版可增强这些场景：

- 原生打开文件：菜单和按钮都可触发 `OpenFileDialog`。
- 拖拽文件：桌面用户更自然。
- 大文件提示：超过阈值时先显示大小和确认解析。
- 树形结果：nested protobuf 默认折叠，避免长结果淹没页面。
- 原始字节联动：点击字段时高亮对应 hex byte range。
- 结果导出：JSON、纯文本、pretty hex。
- 最近文件：本地保存最近打开路径，注意不要保存内容。

UI 保留“本地解码，无网络上传”提示，但不要做成营销页。首屏直接给输入和结果区域。

## 8. 风险与对策

| 风险 | 影响 | 对策 |
| --- | --- | --- |
| 无 schema 误判 nested/string/packed | 用户误解结果 | UI 明确显示“candidate”，保留 raw hex，展示 leftover/error |
| 大输入卡顿 | 桌面窗口无响应 | Go 后端设置大小、深度、字段数限制；前端显示 loading |
| int64 精度丢失 | JS number 不安全 | Go 返回 string，前端不转 number |
| Wails WebView 差异 | UI 在不同平台表现不一 | 三平台截图/手动 smoke test |
| Windows WebView2 缺失 | 用户无法启动 | 构建时选择 WebView2 安装策略，文档说明 |
| Linux 依赖差异 | 安装失败 | 先限定主流发行版测试矩阵，再扩展包格式 |
| macOS 签名与 notarization | 分发受阻 | Release 阶段单独处理 Apple Developer ID、notarytool |

## 9. 里程碑

### M0：调研验证

- 初始化 Wails React TypeScript 模板。
- 跑通 `wails doctor`、`wails dev`、`wails build`。
- 验证 Go 方法绑定到 React。
- 验证原生文件选择框。

交付物：空壳桌面 app + 一个 `Decode("0a...")` mock 调用。

### M1：Go 解码核心

- 实现 Go wire parser。
- 覆盖现有 JS 测试用例。
- 输出结构化 `DecodeResult`。
- 与 README 示例对齐。

交付物：Go decoder package + 单元测试。

### M2：桌面 MVP

- 接入真实 `Decode`。
- 实现输入、文件打开、parse delimited 选项。
- 实现树表展示、raw hex、leftover、错误提示。
- 支持 JSON 导出。

交付物：可日常使用的桌面版。

### M3：发布准备

- 三平台构建。
- Windows WebView2 策略验证。
- macOS `.app` 签名/公证调研。
- Linux 依赖与包格式调研。
- 补齐 README、release note、用户安装说明。

交付物：三平台可下载 artifact。

## 10. 验收标准

- 输入 README 示例，桌面版字段结构与 Web 版一致。
- hex、base64、文件输入都可用。
- gRPC header 与 delimited message 流解析行为与现有逻辑一致。
- 非法输入能显示错误和 leftover，不崩溃。
- 10 MB 以内输入解析过程可控，UI 有 loading 状态。
- macOS、Windows、Linux 至少完成一次启动和解码 smoke test。
- 所有解码逻辑本地执行，无网络请求。

## 11. 推荐技术决策

- Wails 版本：优先 Wails v2 稳定版。
- 前端：React + TypeScript + Vite。
- 后端：Go 实现核心 parser，不复用 JS decoder。
- 数据格式：Go 返回 JSON-friendly struct；所有 64-bit / BigInt 候选值用 string。
- 测试：Go decoder golden tests 为主，前端只测渲染和交互。
- 发布：每个平台原生 CI 构建；不要依赖单机跨编译完成所有发布包。

## 12. 参考资料

- Wails Introduction: https://wails.io/docs/introduction/
- Wails Installation: https://wails.io/docs/gettingstarted/installation/
- Wails How does it work: https://wails.io/docs/howdoesitwork/
- Wails CLI: https://wails.io/docs/reference/cli/
- Wails Runtime Dialog: https://wails.io/docs/reference/runtime/dialog/
- Protobuf Encoding Guide: https://protobuf.dev/programming-guides/encoding/