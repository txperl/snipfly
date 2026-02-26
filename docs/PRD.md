# GoSnippet - 设计与实施 PRD

## Context

GoSnippet 是一个 Go 开发的轻量级 TUI 工具，灵感来自 [CodeLauncher](https://github.com/v2ex/launcher)。目标是管理本地文件夹中的代码片段（shell 脚本、TS/JS/Python 脚本等），通过 TUI 界面一键运行、停止、查看输出，无需反复打开终端输入命令。

核心差异：CodeLauncher 是 macOS 原生 GUI（Swift），GoSnippet 是跨平台 TUI（Go），更轻量。

---

## 产品设计

### CLI 用法

```
gosnippet                 # 扫描当前目录
gosnippet -l /path/to     # 扫描指定目录
gosnippet -g              # 扫描 ~/.gosnippet/snippets/（全局共享）
```

### 注解格式

每个代码片段文件头部用注释声明元数据：

```bash
#!/bin/bash
# @name: SSH Proxy
# @type: service
# @desc: Start SOCKS5 proxy via SSH
# @dir: ~/projects
# @env: SSH_HOST=example.com
# @env: SSH_PORT=22
```

```typescript
// @name: Clean Cache
// @type: oneshot
// @desc: Clean all build caches
```

| 注解           | 说明                           | 默认值             |
| -------------- | ------------------------------ | ------------------ |
| `@name`        | 显示名称                       | 文件名（去扩展名） |
| `@desc`        | 简短描述                       | 空                 |
| `@type`        | `oneshot` \| `service`         | `oneshot`          |
| `@dir`         | 工作目录（支持 `~`）           | 片段所在目录       |
| `@env`         | 环境变量 `KEY=VALUE`（可多条） | 继承当前环境       |
| `@interpreter` | 覆盖解释器                     | 自动推断           |

注释前缀支持 `#`（sh/py/rb）和 `//`（js/ts/go），解析器在遇到第一行非注释非空行时停止。

### 片段目录结构

子目录作为分组：

```
snippets/
├── network/
│   ├── ssh-proxy.sh        # @type: service
│   └── ping-test.sh        # @type: oneshot
└── dev/
    ├── dev-server.sh       # @type: service
    └── build.sh            # @type: oneshot
```

### 解释器解析优先级

1. `@interpreter` 注解（最高优先级）
2. Shebang 行（`#!/usr/bin/env bash`）
3. 扩展名映射：`.sh`→bash, `.ts`→`npx tsx`, `.js`→node, `.py`→python3, `.go`→`go run`, `.rb`→ruby, `.lua`→lua

### 进程状态

Service: `idle` → `running` → `stopped`（用户停止）/ `crashed`（非零退出）/ `exited`（正常退出）
Oneshot: `idle` → `running` → `done`（exit 0）/ `failed`（非零退出）

### 边界情况处理

- **标记为 service 但秒退**：显示退出码和输出，标记为 exited/crashed，可重新启动
- **标记为 oneshot 但长时间运行**：持续显示输出，允许手动 `S` 停止
- **崩溃处理**：不自动重启，显示崩溃状态和输出日志
- **退出确认**：有运行中的 service 时按 `Q` 弹出确认对话框，确认后停止所有进程再退出
- **信号处理**：注册 SIGINT/SIGTERM/SIGHUP 处理器，确保子进程不会成为孤儿

### TUI 界面

```
┌────────────────────────────────────────────────┐
│  GoSnippet                          [Q]uit     │
├────────────────┬───────────────────────────────┤
│ network/       │ @name: SSH Proxy              │
│  ● ssh-proxy   │ @desc: Start SOCKS5 proxy     │
│    ping-test   │ @type: service                │
│ dev/           │───────────────────────────────│
│  ● dev-server  │ $ Starting SSH proxy...       │
│    build.sh    │ Listening on :1080            │
│                │ Connected to remote           │
│                │ > Ready.                      │
├────────────────┴───────────────────────────────┤
│ [Enter] Run  [S]top  [R]estart  [/] Filter    │
│ [↑↓/jk] Navigate  [Tab] Switch Panel  [Q] Quit│
└────────────────────────────────────────────────┘
```

- 左面板 30% 宽度（min 20, max 40 列）
- 右面板顶部显示当前片段的 metadata 区域（`@key: value` 格式，灰色文本，不随内容滚动）
  - 显示字段顺序：`@name` → `@desc` → `@type` → `@dir` → `@env` → `@interpreter`
  - 仅显示有值且非默认值的字段（`@name` 始终显示）
  - 底部一条 `─` 分隔线（subtle 颜色）与输出内容区分
  - 切换片段时自动更新，viewport 高度自动适配
- 右面板下方显示选中片段的实时输出（viewport 可滚动）
- 状态图标：`●` running(green), `✗` crashed/failed(red), `✓` done(gray), `■` stopped, ` ` idle
- Tab 切换面板焦点（output 面板可滚动查看历史）
- 输出自动跟踪最新（sticky bottom），手动上滚后暂停自动跟踪

---

## 技术架构

### 依赖（全部使用最新 v2）

- `charm.land/bubbletea/v2` v2.0.0 — TUI 框架（Elm Architecture）
- `charm.land/bubbles/v2` v2.0.0 — viewport 等组件
- `charm.land/lipgloss/v2` v2.0.0 — 样式/布局

仅 Charm 生态 v2，无其他外部依赖。

### bubbletea v2 关键 API 变化（实现时注意）

**View() 返回值变化**：

```go
// v1: View() string
// v2: View() tea.View — 声明式视图
func (m AppModel) View() tea.View {
    v := tea.NewView(content)
    v.AltScreen = true                    // 替代 tea.EnterAltScreen() cmd
    v.MouseMode = tea.MouseModeCellMotion // 替代 tea.EnableMouseCellMotion() cmd
    return v
}
```

**键盘消息变化**：

```go
// v1: tea.KeyMsg + msg.String()
// v2: tea.KeyPressMsg + msg.Key.Code / msg.Key.Text
case tea.KeyPressMsg:
    switch {
    case msg.Key.Code == tea.KeyEscape: ...
    case msg.Key.Text == "q": ...
    }
```

**鼠标消息拆分**：`tea.MouseClickMsg`, `tea.MouseWheelMsg`, `tea.MouseMotionMsg`

**bubbles/viewport 变化**：

- `LineUp()`/`LineDown()` → `ScrollUp()`/`ScrollDown()`
- Width/Height 改为 getter/setter 方法
- 新增水平滚动支持

**lipgloss v2 变化**：

- 颜色类型改为 `color.Color`
- 背景检测需手动调用 `HasDarkBackground()`
- 样式是确定性的（deterministic），无隐式 stdout 检测

### 项目结构

```
GoSnippet/
├── main.go                         # CLI 入口，flag 解析，程序引导
├── go.mod / go.sum
├── internal/
│   ├── snippet/
│   │   ├── snippet.go              # 数据模型（Snippet, SnippetType, ProcessState）
│   │   ├── parser.go               # 注解解析器（正则提取 @key: value）
│   │   ├── scanner.go              # 目录扫描器（WalkDir，一层子目录分组）
│   │   └── interpreter.go          # 解释器解析（注解 > shebang > 扩展名映射）
│   ├── runner/
│   │   ├── runner.go               # 进程管理器（Start/Stop/Restart/StopAll）
│   │   ├── process.go              # 单进程封装（exec.Cmd + 状态机 + 输出捕获）
│   │   └── buffer.go               # 环形缓冲区（最近 1000 行，线程安全）
│   └── tui/
│       ├── app.go                  # 根 Model（消息路由，布局计算，键盘处理）
│       ├── list.go                 # 左面板（自定义列表，分组头+状态图标）
│       ├── output.go               # 右面板（viewport 包装，auto-scroll）
│       ├── metadata.go             # 输出面板顶部 metadata 渲染（@key: value 格式）
│       ├── statusbar.go            # 底部状态栏（快捷键提示 + 进程信息）
│       ├── confirm.go              # 退出确认对话框
│       ├── styles.go               # lipgloss 样式定义
│       └── messages.go             # 自定义 tea.Msg 类型
└── examples/                       # 示例片段
    ├── network/
    │   └── echo-server.sh
    └── tools/
        └── hello.sh
```

### 关键设计

**输出流式传输（性能关键路径）**：

- 进程 stdout/stderr 通过 goroutine 逐行读取，写入 `RingBuffer`
- 每次写入后调用 `program.Send(OutputMsg)`
- TUI 端做 50ms 节流（throttle）：首个 OutputMsg 启动 tick，50ms 内后续消息仅标记 pending
- tick 到期后批量刷新 viewport 内容 → 最多 20fps 刷新，不论输出速率

**进程组管理**：

- `SysProcAttr{Setpgid: true}` 创建进程组
- Stop 时先 SIGTERM 整个进程组，5s 超时后 SIGKILL
- 确保脚本的子进程也能被正确终止

**不使用 bubbles/list 组件**：

- 内置 list 有自己的标题栏、分页、过滤 UI，与自定义分屏布局冲突
- 自定义列表组件完全控制分组头、状态图标、选中渲染

**Runner → TUI 通信**：

- Runner 持有 `*tea.Program` 引用（通过 `SetProgram()` 注入）
- 后台 goroutine 通过 `program.Send()` 发送 OutputMsg / ProcessExitedMsg
- 这是 runner goroutine 与 TUI Update 循环之间的唯一通信方式

---

## 实施计划

### Phase 1: 数据模型与解析

**文件**: `internal/snippet/snippet.go`, `parser.go`, `scanner.go`, `interpreter.go`

- 定义 Snippet / SnippetType / ProcessState 类型
- 实现注解解析器（正则匹配 `# @key: value` 和 `// @key: value`）
- 实现目录扫描器（WalkDir，一层子目录，已知扩展名 + shebang 检测）
- 实现解释器解析链（注解 > shebang > 扩展名映射）

### Phase 2: 进程运行器

**文件**: `internal/runner/buffer.go`, `process.go`, `runner.go`

- 实现线程安全的 RingBuffer（1000 行，环形写入，count 变更检测）
- 实现 Process 封装（exec.Cmd + 进程组 + stdout/stderr goroutine 流式读取 + onOutput/onExit 回调）
- 实现 Runner 管理器（Start/Stop/Restart/StopAll/HasRunning/GetBuffer）

### Phase 3: TUI 框架搭建

**文件**: `internal/tui/messages.go`, `styles.go`, `app.go`, `main.go`

- 定义所有消息类型（OutputMsg, ProcessExitedMsg, ConfirmQuitMsg 等）
- 定义 lipgloss 样式
- 实现最小 AppModel（静态分屏布局 + WindowSizeMsg 响应式布局）
- 完成 main.go：flag 解析 → 目录扫描 → Runner 创建 → tea.NewProgram 启动 → 信号处理

### Phase 4: 列表面板

**文件**: `internal/tui/list.go`

- 实现 ListModel（ListItem 包含分组头和片段条目）
- j/k/↑/↓ 导航（跳过分组头）
- 滚动偏移管理
- 状态图标 + 颜色渲染 + 选中高亮

### Phase 5: 进程执行与输出面板

**文件**: `internal/tui/output.go`，更新 `app.go`

- 实现 OutputModel（viewport 包装 + auto-scroll + sticky bottom）
- Enter 启动 / S 停止 / R 重启
- OutputMsg 节流处理 → OutputThrottleTickMsg 批量刷新
- ProcessExitedMsg 处理
- Tab 切换面板焦点
- 列表选择变更时切换输出内容

### Phase 6: 状态栏与退出确认

**文件**: `internal/tui/statusbar.go`, `confirm.go`

- 底部状态栏（快捷键提示 + 当前片段状态/PID/退出码）
- 退出确认对话框（居中覆盖，Y/N 操作）

### Phase 7: 边界情况与打磨

- 处理 service 秒退场景
- 处理 oneshot 长时间运行场景
- 解释器解析失败时在输出面板显示错误
- `@dir` 路径 `~` 展开
- 创建 examples/ 示例片段

### Phase 8（可选）: 搜索/过滤

- `/` 键激活搜索模式（textinput 组件）
- 按片段名称子串过滤列表
- Esc 退出搜索并清除过滤

---

## 验证方案

1. **单元测试**：注解解析器（多种注释风格）、目录扫描器（测试 fixture）、RingBuffer（环形写入/读取）、解释器解析链
2. **集成测试**：创建 `examples/` 示例片段：
   - `examples/tools/hello.sh` — oneshot，打印 hello 后退出
   - `examples/network/echo-server.sh` — service，循环打印时间
3. **手动验证**：
   - `go run main.go examples/` 启动 TUI
   - 导航列表、启动/停止 service、运行 oneshot
   - 验证输出实时流式显示、auto-scroll、手动滚动
   - 验证退出确认对话框
   - 验证崩溃状态显示
