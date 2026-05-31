# Protobuf Decoder Desktop

基于 Go、Wails、React 和 TypeScript 构建的无 Schema Protobuf Wire 解析桌面工具。

应用完全本地运行：粘贴的载荷、拖入的文件、选择的文件、解码启发式判断以及导出结果都只在本机处理。应用中不存在任何与解码相关的网络上传路径。

## 功能说明

- 在没有 `.proto` Schema 的情况下解析未知 Protobuf 载荷。
- 支持 `auto`、`hex`、`base64` 以及本地二进制文件输入。
- 检测并跳过有效的 gRPC 5 字节消息头。
- 可选解析以 varint 长度前缀分隔的消息流。
- 展示字段树、字节范围、原始十六进制、嵌套候选、警告、剩余字节以及解码错误。
- 支持将结果复制或导出为格式化 JSON 或文本报告。

## 不包含的能力

- 不支持导入 `.proto`。
- 不支持 Schema 感知的字段名、消息名、枚举名、oneof、map 或默认值恢复。
- 候选值属于启发式推断，不保证代表真实业务语义。
- 不提供云同步、远程解析或上传能力。

## 截图

结果工作区示例：

![Result workspace](docs/screenshots/result-workspace.png)

## 快速开始

前置要求：

- Go 1.21+
- Node.js 20+
- Wails CLI v2

安装依赖：

```sh
npm --prefix frontend install
```

检查本地环境：

```sh
wails doctor
```

以开发模式运行桌面应用：

```sh
wails dev
```

为当前操作系统构建生产制品：

```sh
wails build
```

运行项目测试：

```sh
go test ./...
npm --prefix frontend test
```

## 输入格式

支持的输入来源：

- 在 UI 中粘贴文本载荷。
- 使用原生文件选择器。
- 通过桌面拖放文件触发解码。

支持的文本编码：

- `auto`：通过启发式判断选择 hex 或 base64；若存在歧义则返回警告。
- `hex`：忽略空白以及常见分隔符，例如 `,`、`:`、`-`、`_` 和 `0x` 前缀。
- `base64`：在合法情况下接受标准与 raw/url 变体。

## 解码限制

后端强制默认值：

- `MaxDepth = 4`
- `MaxFields = 256`
- `MaxBytes = 10 MiB`

前端可以提示更安全的取值，并在大文件解码前发出警告，但后端仍是唯一真实约束来源。

达到限制时：

- `MaxBytes`：解码器快速失败，并返回大小限制错误。
- `MaxFields`：顶层、嵌套和 delimited message 共享全局已解码字段预算；预算耗尽时解码器停止，并返回剩余字节。
- `MaxDepth`：嵌套解码停止，父字段仍保留字节信息及其他候选解释。

## 候选解释说明

这个工具是 Wire 级检查器。一个字段可能同时存在多个合理视图。

- `VARINT`：无符号、有符号、ZigZag、bool/enum 提示。
- `FIXED32`：`uint32`、`int32`、`float32`。
- `FIXED64`：`uint64`、`int64`、`double`。
- `LENDELIM`：UTF-8 字符串候选、原始字节候选，以及可选的嵌套 protobuf 候选。

重要阅读规则：

- 排在最前的候选只是最强启发式，不代表 Schema 真相。
- 只有当载荷可被完整解析且没有 leftover 时，才会接受嵌套 protobuf 候选。
- `leftover` 和 warnings 很重要。它们通常解释为何嵌套猜测被拒绝，或解码为何中途停止。

## 示例载荷

简单 UTF-8 字符串字段：

```text
0a03666f6f
```

预期高层结果：

- 字段 `#1`
- Wire type `2`
- 类型 `LENDELIM`
- 候选 `string.utf8 = foo`
- 候选 `bytes.hex = 666f6f`

分隔流示例：

```text
020801021002
```

预期高层结果：

- 每条消息对应一个 `MessageDelimiter` part。
- 每条消息内部字段按全局字节范围展示。

gRPC 头示例：

```text
00000000020801
```

预期高层结果：

- 首先显示 `GRPC_HEADER` part。
- 消息体字段从字节偏移 `5` 之后开始解析。

## 安装说明

各平台安装与运行说明见 [docs/platform-install.md](docs/platform-install.md)。

## 故障排查

面向用户的故障排查说明见 [docs/troubleshooting.md](docs/troubleshooting.md)。

## 发布说明

发布说明模板见 [.github/release-template.md](.github/release-template.md)。

CI/发布运行器说明见 [.github/platform-release-notes.md](.github/platform-release-notes.md)。

## 隐私

- 解码在本地 Go 后端执行。
- 文件内容始终留在本机。
- 剪贴板复制与导出只写入用户选择的结果内容。
- 应用未实现遥测或远程解码服务。

## 当前发布范围

当前已实现内容覆盖解码核心、嵌套解析、gRPC 头处理、分隔流、结果检查 UI、导出能力、运行时防护以及 GitHub Actions smoke build。