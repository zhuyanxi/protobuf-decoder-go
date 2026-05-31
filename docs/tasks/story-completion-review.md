# Story 完成情况 Review

日期：2026-05-31

范围：`docs/tasks/wails-desktop-rewrite-jira-stories.md` 中 Story 1-16 的实现、测试、文档、CI/release 配置。

## Review 结论

整体功能链路已完成：Wails 桌面工程、Go decoder、输入规范化、wire parser、候选解释、nested/gRPC/delimited 支持、前端工作区、导出、运行时限制、CI/release workflow、用户文档均已落地。

仍有需要修复或完善的点，优先级如下。

## 必须修复

### 1. Clean checkout 下前端 CI/构建缺少 Wails bindings（已修复）

- 影响 Story：Story 1、Story 11、Story 15。
- 严重级别：High。
- 修复状态：已从 `.gitignore` 移除 `frontend/wailsjs/` 忽略规则，Wails generated bindings 现在会作为可提交文件进入版本控制。
- 修复前证据：
  - `frontend/src/App.tsx:3` 直接 import `../wailsjs/go/main/App` 和 `../wailsjs/go/models`。
  - `.gitignore` 曾忽略 `frontend/wailsjs/`。
  - `.github/workflows/ci.yml:57` 运行 `npm test`，`.github/workflows/ci.yml:60` 运行 `npm run build`，但 frontend job 未先生成 Wails bindings。
  - `rtk git archive --format=tar HEAD | tar -t | rg '^(frontend/wailsjs|frontend/src/App.tsx|frontend/package.json|go.mod)'` 输出中没有 `frontend/wailsjs`。
- 风险：当前本地 workspace 因 ignored generated files 存在可通过测试；干净 clone 或 GitHub Actions frontend job 可能因模块缺失失败。
- 修复说明：
  - 已采用方案 A：移除 `.gitignore` 中 `frontend/wailsjs/`，提交生成的 bindings，使 frontend tests/build 在 clean checkout 可运行。
  - 提交时需包含 `frontend/wailsjs/go/main/App.d.ts`、`frontend/wailsjs/go/main/App.js`、`frontend/wailsjs/go/models.ts`、`frontend/wailsjs/runtime/package.json`、`frontend/wailsjs/runtime/runtime.d.ts`、`frontend/wailsjs/runtime/runtime.js`。
  - 已重新验证 `npm --prefix frontend test` 与 `npm --prefix frontend run build`。

### 2. Go 版本要求与文档不一致

- 影响 Story：Story 1、Story 15、Story 16。
- 严重级别：High。
- 证据：
  - `go.mod:3` 为 `go 1.26.1`。
  - `README.md:33`、`README.zh-CN.md:33`、`docs/platform-install.md:5` 写 `Go 1.21+`。
  - `.github/workflows/ci.yml:30`、`.github/workflows/ci.yml:86`、`.github/workflows/release.yml:40` 使用 `go-version-file: go.mod`。
- 风险：用户按 README 用 Go 1.21 构建会失败或被要求升级；CI/release runner 实际跟随 Go 1.26.1，而不是文档承诺的最低版本。
- 建议修复：
  - 若项目确实只需要 Go 1.21：将 `go.mod` 降到 `go 1.21`，并用 Go 1.21 环境验证 `go test ./...` 与 `wails build`。
  - 若项目决定要求 Go 1.26.1：更新 README、中文 README、platform install、Story 依赖说明和 release notes 模板中的版本要求。

## 建议完善

### 3. 前端测试覆盖过薄（已完善）

- 影响 Story：Story 11、Story 12、Story 13、Story 14、Story 15。
- 严重级别：Medium。
- 修复前证据：`frontend/src/App.test.tsx` 只有一个 smoke test，验证控件文案存在。
- 修复状态：已扩展为 5 个 App 行为测试，覆盖 decode 成功渲染、warnings、field detail、nested 展开、copy/export、decode error 清理旧结果、large file confirm 取消分支。
- 验证：`rtk npm --prefix frontend test` 通过，`src/App.test.tsx` 5 个测试全部通过；`rtk npm --prefix frontend run build` 通过。

### 4. `ExportResult` / `CopyResultJSON` 缺少空结果防御（已修复）

- 影响 Story：Story 13。
- 严重级别：Medium。
- 修复前证据：`ExportResult` 和 `buildExportPayload` 可接受空 `DecodeResult{}` 并生成空报告；前端当前会 disabled 按钮，但后端公开 Wails API 未做同等校验。
- 修复状态：已增加 `hasExportableResult(result DecodeResult) bool`，对 `Parts`、`Error`、`Leftover`、`Warnings`、`InputSize` 全空的结果返回 `no decode result to export`。
- 覆盖范围：`CopyResultJSON`、`ExportResult`、`buildExportPayload` 共用空结果防御。
- 验证：新增空结果 formatter/API 单元测试；`rtk go test ./...` 通过 59 个测试；`rtk npm --prefix frontend test` 与 `rtk npm --prefix frontend run build` 通过。

### 5. `MaxFields` 语义需明确或改为全局计数

- 影响 Story：Story 5、Story 7、Story 14、Story 16。
- 严重级别：Medium。
- 证据：`internal/decoder/decode.go` 中 `decodeBytesAtDepth` 每次进入 nested message 都新建 `fieldIndex := 0`，因此 `MaxFields` 当前是单个 message 层级内限制，不是整次 decode 的全局字段总数限制。
- 风险：用户阅读 `MaxFields = 256` 可能理解为整次 decode 最多 256 个字段；实际 nested payload 可在多个 message 层级分别达到该限制。恶意宽嵌套 payload 会增加 CPU/内存压力，虽仍受 `MaxDepth` 和 `MaxBytes` 约束。
- 建议修复：
  - 若产品希望全局限制：引入共享 field counter，递归和 delimited stream 共用同一计数器。
  - 若产品接受 per-message 限制：更新 README、中文 README、troubleshooting 和 Story 文档，明确 `MaxFields` 是 per message boundary；补测试锁定语义。

### 6. 边界测试可再补两类

- 影响 Story：Story 7、Story 10、Story 14。
- 严重级别：Low。
- 现状：核心 wire type、nested、gRPC、delimited、非法输入已有测试；本地 `rtk go test ./...` 通过 56 个测试。
- 建议新增：
  - Empty LENDELIM payload，例如 `1a00`，验证不触发 nested 递归，且保留 bytes/string 候选语义。
  - Nested + MaxFields 边界，验证第 5 点确定后的 per-message 或 global 行为。

## 已确认不是问题

- gRPC compressed flag：`internal/decoder/decode.go` 已有 `ErrUnsupportedGRPCCompression`，`TestDecodeBytesRejectsCompressedGRPCPayload` 已覆盖。
- Release template：`README.md` 已链接 `.github/release-template.md`，平台 runner 说明在 `.github/platform-release-notes.md`。
- 平台签名/安装限制：`docs/platform-install.md` 已明确 macOS 未自动签名/公证、Windows 当前上传 raw artifact、Linux WebKitGTK 依赖、CI 不做 GUI launch smoke。

## 本次验证命令

```sh
rtk git status --short
rtk git ls-files frontend/wailsjs
rtk git check-ignore -v frontend/wailsjs/go/main/App.d.ts frontend/wailsjs/go/models.ts
rtk go version
rtk git archive --format=tar HEAD | tar -t | rg '^(frontend/wailsjs|frontend/src/App.tsx|frontend/package.json|go.mod)'
rtk go test ./...
rtk npm --prefix frontend test
rtk npm --prefix frontend run build
rtk wails build
```

结果：

- 当前 workspace：Go tests、frontend tests、frontend build、macOS arm64 Wails build 均通过。
- Wails bindings：已取消 `.gitignore` 忽略规则，`frontend/wailsjs` 当前作为待提交文件出现在工作区；提交后 clean checkout 会包含这些 bindings。
- 当前本机 Go：`go1.26.1 darwin/arm64`，与文档 `Go 1.21+` 不一致。

## Story 覆盖状态

| Story | 状态 | 说明 |
| --- | --- | --- |
| Story 1 | 已修复 | Wails 工程可构建；Wails generated bindings 已取消忽略，clean checkout 前端构建风险已处理。 |
| Story 2 | 已完成 | API/JSON contract 已实现，64-bit display value 使用 string；可继续加强精度断言。 |
| Story 3 | 已完成 | hex/base64/auto/file 输入与错误处理已有测试。 |
| Story 4 | 已完成 | reader 边界和错误场景已有测试。 |
| Story 5 | 需明确 | parser 主流程完成；`MaxFields` 全局/单 message 语义需定稿。 |
| Story 6 | 已完成 | varint/fixed/string/bytes 候选解释已实现并测试。 |
| Story 7 | 需完善 | nested parsing 已实现；建议补 empty LENDELIM 与 MaxFields 语义测试。 |
| Story 8 | 已完成 | gRPC header、截断、compressed flag 均已覆盖。 |
| Story 9 | 已完成 | delimited stream 与错误路径已有测试。 |
| Story 10 | 需完善 | golden tests 已覆盖主要路径；建议补新增边界 fixture。 |
| Story 11 | 已修复 | UI 已实现；Wails generated bindings 已取消忽略，前端独立 test/build 可使用提交后的 binding 文件；App 行为测试已扩展。 |
| Story 12 | 已完善 | 树表/详情/raw hex 已实现；新增 decode result、field detail、nested 展开回归测试。 |
| Story 13 | 已完善 | 导出/复制已实现；新增前端 copy/export 调用测试，后端已拒绝空结果。 |
| Story 14 | 需明确 | limit/loading/guardrail 已实现；`MaxFields` 语义需明确或改全局。 |
| Story 15 | 需修复 | workflow 已实现；binding 缺失风险已修复，仍需处理 Go version 跟文档不一致。 |
| Story 16 | 需同步 | 文档完整；Go version 与 `MaxFields` 语义需按最终实现同步。 |