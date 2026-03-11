[English](README.md) | 中文

# SnipFly CLI

<img src="docs/appicon.png" alt="SnipFly Icon" width="96">

就在一个 Terminal 标签页中，同时运行多个代码片段吧！

<!-- screenshot -->
<!-- <img src="screenshot.png" alt="SnipFly 截图" width="800"> -->

## 快速开始

首先安装 SnipFly：

```bash
brew install txperl/tap/snipfly
# 或
go install github.com/txperl/snipfly@latest
```

然后，在项目中创建 `.snipfly/` 目录并添加一个片段：

```bash
mkdir .snipfly
echo 'echo Hello from SnipFly!' > .snipfly/hello.sh
```

开始使用吧！

```bash
snipfly
```

##### `.snipfly/` 子目录

SnipFly 会自动从工作目录下的子目录（`.snipfly/`）中检测片段。

如果该目录不存在，SnipFly 将直接从工作目录中查找片段。

## 使用

```bash
# snipfly -h
Usage: snipfly [options] [directory]

Options:
  -e, --exact     使用精确目录，不自动检测 ./.snipfly/
  -g, --global    扫描全局片段目录（~）
  -v, --version   打印版本信息

# 运行当前目录中的片段
snipfly
snipfly .

# 运行指定目录中的片段
snipfly /path/to/project

# 运行全局片段
snipfly -g
```

### 快捷键

| 按键    | 操作           |
| ------- | -------------- |
| `↑`     | 向上移动       |
| `↓`     | 向下移动       |
| `Space` | 运行/停止片段  |
| `r`     | 重启片段       |
| `Tab`   | 切换到输出面板 |
| `q`     | 退出           |

## 片段

### 目录结构示例

```
project/
└── .snipfly/
    ├── deploy.sh            # 根级别（无分组）
    ├── dev/                 # 分组："dev"
    │   ├── server.sh
    │   └── watch.ts
    └── tools/               # 分组："tools"
        ├── lint.sh
        └── format.py
```

### 注解

在片段文件顶部以注释形式添加注解。解析器会读取以 `#` 或 `//` 开头的行，直到遇到第一个非注释、非空行。

| 注解           | 类型    | 默认值             | 描述                                  |
| -------------- | ------- | ------------------ | ------------------------------------- |
| `@name`        | string  | 文件名（无扩展名） | 列表中的显示名称                      |
| `@desc`        | string  | --                 | 简短描述                              |
| `@type`        | string  | `oneshot`          | `oneshot`、`service` 或 `interactive` |
| `@dir`         | string  | 片段所在目录       | 工作目录（支持 `~`）                  |
| `@env`         | string  | --                 | 环境变量 `KEY=VALUE`（可重复）        |
| `@interpreter` | string  | 自动检测           | 覆盖解释器命令                        |
| `@pty`         | boolean | `false`            | 分配伪终端                            |

示例：

```bash
# @name: Clock Server
# @desc: 每秒打印当前时间
# @type: service
# @dir: /tmp
# @env: TZ=UTC
# @env: LANG=en_US.UTF-8
# @pty: true

while true; do date; sleep 1; done
```

最简示例：

```bash
echo "Hello from SnipFly!"
```

## 从源码构建

```
git clone https://github.com/txperl/snipfly.git
cd snipfly
go build -o snipfly .
```

## 许可证

- [MIT](LICENSE)
