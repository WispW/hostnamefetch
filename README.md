# hostnamefetch

[CI workflow](.github/workflows/ci.yml) · [MIT License](LICENSE)

> neofetch 的主角是操作系统徽标；hostnamefetch 的主角是这台机器的名字。

`hostnamefetch` 是一个 neofetch 风格的系统信息工具。它保留 neofetch 的信息栏和
左右布局，但把左侧的发行版 ASCII 徽标换成用内置 5×5 字体渲染的 **hostname 横幅**。
适合用来区分多台机器、做 shell 启动画面，或者单纯让 `fetch` 更有辨识度。

## 预览

```text
##### ####        ####  ####   ###    qq341@Z3-PRO
   #      #       #   # #   # #   #   ------------
  #    ###   ###  ####  ####  #   #   Hostname: Z3-PRO
 #        #       #     #  #  #   #   OS: Arch Linux x86_64
##### ####        #     #   #  ###    Machine: PRIME Z370-A
                                      Kernel: 7.3.4-zen2-2-zen
                                      Uptime: 7 hours, 9 mins
                                      Packages: 1506 (pacman, dpkg)
                                      Shell: bash
                                      Resolution: 1920x1080
                                      DE: KDE Plasma
                                      WM: KWin
                                      Terminal: VS Code
                                      CPU: Intel i7-8700K (12) @ 4.9GHz
                                      GPU: NVIDIA GeForce RTX 2080
                                      Memory: 17109MiB / 64214MiB (26%)
                                      Local IP: 192.168.5.56
                                      Disk (/): 38G / 148G (27%)
```

预览使用 `--layout left --valign top --banner_chars ascii --disable cols`，
方便在 Markdown 中阅读。真实终端默认使用 `█` 字符、彩色标签和 16 色色块；信息列比横幅高时，
横幅默认在信息列中垂直居中（`--valign center`），`--valign top` 可改成上图样式。
配色由 hostname 的校验和决定：同一台机器每次运行颜色一致，不同机器通常不同。

## 特性

- **内置 5×5 块字体**，不依赖 `figlet` / `toilet`。
- **自动布局**：终端够宽时横幅在左、信息在右；不够宽时横幅在上、信息在下。
  左布局中横幅默认相对信息列垂直居中，可用 `--valign top` 改回顶部对齐。
  显式 `--layout left` 在终端过窄时会回退到上下布局并在 stderr 给出警告。
  超长 hostname 会自动分行，并优先在 `-`、`.`、`_`、`+`、`/` 处断开。
- **自动字符集回退**：UTF-8 终端使用 `█`，非 UTF-8 环境自动改用 `#`。
- **hostname 派生配色**：默认从 12 组配色中按 hostname 校验和选一组；
  也可以用 `--banner_colors` 手动指定。
- **信息项与 neofetch 对齐**：Hostname、FQDN、OS、Machine、Kernel、Uptime、
  Packages、Shell、Resolution、DE、WM、Terminal、CPU、GPU、Memory、
  Local IP、Disk、色块。
- **纯文本输出友好**：输出到管道/文件时自动关闭颜色并改用上下布局，
  不会留下光标控制序列；`--stdout` 只输出信息行，方便脚本使用。

## 环境要求

- `bash` 4.0 及以上
- 常见 Unix 工具：`awk`、`uname`、`df`、`hostname` 等
- 主要面向 Linux；macOS 做尽量兼容；Windows 通过 Git Bash 可运行，
  但会跳过部分无法采集的信息项。Linux、macOS、Windows 三种系统都在 CI 中跑测试。
- Windows 另有原生 `hostnamefetch.exe`（Go 实现），不需要 bash / Git Bash / WSL。
- 信息行不做截断；终端窄于最长信息行时会折行，建议宽度至少 80 列。

## 安装

一条命令安装到 `~/.local/bin`（需要该目录在 `PATH` 中）：

```sh
mkdir -p ~/.local/bin && curl -fsSL https://raw.githubusercontent.com/WispW/hostnamefetch/v2026.11.21.1/hostnamefetch -o ~/.local/bin/hostnamefetch && chmod +x ~/.local/bin/hostnamefetch
```

不安装、直接通过管道运行：

```sh
curl -fsSL https://raw.githubusercontent.com/WispW/hostnamefetch/v2026.11.21.1/hostnamefetch | bash
```

> `v2026.11.21.1` 是当前版本 tag；在发布前可把 URL 中的 tag 换成 `main`。

### Debian/Ubuntu（APT）

已提供第三方 APT 源：<https://repo.wennie99s.top>

```sh
sudo install -d -m 0755 /etc/apt/keyrings

curl -fsSL https://repo.wennie99s.top/wispw-archive-keyring.gpg \
  | sudo tee /etc/apt/keyrings/wispw.gpg >/dev/null

echo "deb [signed-by=/etc/apt/keyrings/wispw.gpg] https://repo.wennie99s.top stable main" \
  | sudo tee /etc/apt/sources.list.d/wispw.list

sudo apt update
sudo apt install hostnamefetch
```

### Windows（原生 exe）

从 [Releases](https://github.com/WispW/hostnamefetch/releases/latest) 下载
`hostnamefetch.exe`，在终端里直接运行：

```powershell
.\hostnamefetch.exe
```

- 单文件，无需 bash、Git Bash、WSL 或任何运行时
- 想在任何目录直接调用，把 exe 所在目录加入用户 `PATH`
- 首次运行可能被 SmartScreen 拦截，选择「更多信息 → 仍要运行」
- 信息项：Hostname、FQDN、OS、Machine、Kernel、Uptime、Shell、Terminal、
  Resolution、CPU、GPU、Memory、Local IP、Disk、色块（不显示 Packages / DE / WM）

## 用法

```text
hostnamefetch [options]
```

常用示例：

```sh
# 默认：用本机 hostname 渲染横幅
hostnamefetch

# 指定横幅文字（会自动转成大写）
hostnamefetch --banner_text "web-01"

# 强制横幅在上、信息在下
hostnamefetch --layout top

# 用 ASCII '#' 渲染，适合不支持 Unicode 的终端
hostnamefetch --banner_chars ascii

# 手动指定配色（ANSI 颜色 0-255）
hostnamefetch --banner_colors 2 6

# 隐藏部分信息项
hostnamefetch --disable gpu resolution cols

# 只输出信息，不带横幅和颜色
hostnamefetch --stdout
```

### 选项

| 选项 | 说明 |
| --- | --- |
| `-h`, `--help` | 显示帮助 |
| `-v`, `--version` | 显示版本 |
| `--banner <on\|off>` | 是否显示 hostname 横幅，默认 `on` |
| `--banner_text <text>` | 覆盖横幅文字（默认使用短 hostname） |
| `--banner_colors <c> [c2]` | 横幅/标签配色，ANSI 0-255；默认 `auto` |
| `--banner_chars <auto\|blocks\|ascii>` | 横幅填充字符，默认 `auto` |
| `--banner_spacing <0\|1>` | 字形之间是否留 1 格，默认 `1` |
| `--layout <auto\|left\|top>` | 横幅位置，默认 `auto` |
| `--valign <top\|center>` | 左布局中横幅的垂直对齐，默认 `center` |
| `--gap <num>` | 横幅与信息的间距，默认 `3` |
| `--disable <item>...` | 隐藏信息项，可一次写多个 |
| `--color_text <on\|off>` | 信息文字着色，默认 `on` |
| `--bold <on\|off>` | 粗体，默认 `on` |
| `--title_fqdn <on\|off>` | 标题使用 FQDN，默认 `off` |
| `--color_blocks <on\|off>` | 是否显示色块，默认 `on` |
| `--block_range <a> <b>` | 色块范围，默认 `0 15` |
| `--block_width <num>` | 色块宽度，默认 `3` |
| `--block_height <num>` | 色块高度，默认 `1` |
| `--separator <str>` | 标签分隔符，默认 `":"` |
| `--stdout` | 只输出纯文本信息（无横幅、无颜色） |

`--disable` 接受内部变量名：

```text
title underline hostname fqdn distro model kernel uptime packages shell
resolution de wm term cpu gpu memory local_ip disk cols
```

也支持几个友好别名：`host` → `hostname`，`os` → `distro`，
`machine` → `model`，`ip` → `local_ip`。

## 工作原理

1. **字体**：脚本内置 47 个 5×5 字形（A-Z、0-9 和常用标点），
   每个字形由 5 行组成，行内用 `|` 分隔。字体数据用 `#` 表示填充，
   渲染时按 `--banner_chars` 替换为 `█` 或保留 `#`。
2. **布局**：先采集所有信息行并计算信息列宽度，再决定 `left` 还是 `top`。
   左布局采用逐行拼接：横幅行 + 填充空格 + 信息行，因此不依赖光标移动序列，
   导出到文件时也不会错位。横幅较短时按行数差在顶部补空行，使其相对信息列居中。
3. **配色**：默认把 hostname 传给 `cksum`，对 12 组配色取模，
   得到一组稳定的双色方案。
4. **换行**：hostname 超过一行能容纳的宽度时，按标点边界切成多个横幅块，
   上下堆叠。

## 与 neofetch 的关系

信息采集函数、信息行组织方式（`info` / `print_info`）和整体左右布局参考了
[neofetch 7.1.0](https://github.com/dylanaraps/neofetch)（MIT License，
Copyright (c) 2015-2021 Dylan Araps）。`hostnamefetch` 不是 neofetch 的 fork，
不包含各发行版的 ASCII logo；它把 logo 位置留给了 hostname 本身。

## 测试

```sh
bash tests/smoke.sh
```

测试只读取仓库并在内存中渲染，不会修改任何文件。

## CI

`.github/workflows/ci.yml` 在 push 到 `main`、PR 和手动触发时运行：

| Job | 环境 | 说明 |
| --- | --- | --- |
| ShellCheck | `ubuntu-latest` | 静态检查脚本 |
| Ubuntu | `ubuntu-latest` | 系统 bash + 默认 awk（mawk） |
| Windows | `windows-latest` | pwsh 调用 Git Bash；`.gitattributes` 保证脚本使用 LF 换行 |
| Arch Linux | `archlinux:latest` 容器 | 在 Docker 中运行同一套测试 |
| macOS | `macos-latest` | 安装 bash 5 后运行测试 |

执行顺序固定为 ShellCheck → Ubuntu → Windows → Arch → macOS，使用 `needs` 串行；
任意一步失败，后续 job 会自动跳过。Ubuntu / Windows / Arch / macOS 四个 job 先做
`bash -n` 语法检查，再运行 `tests/smoke.sh`。ShellCheck 优先使用 runner
预装版本，缺失时自动安装；Ubuntu 和 Windows 不需要额外准备，Arch 只拉取官方基础镜像，
macOS 安装 bash 5。测试覆盖：

- 字形尺寸与横幅渲染；
- 长 hostname 的分块、换行和原文重组；
- 80 列终端下不溢出、信息列对齐；
- 垂直居中、CLI 参数校验和 `--stdout` 输出；
- macOS `system_profiler` 缓存、内存回退与窄终端警告。

如果希望 PR 在任一检查失败时禁止合并，还需要在仓库
Settings → Branches 中把这五个 job 设为 required status checks；
workflow 本身只负责执行和串联顺序。

## License

MIT。见 [LICENSE](LICENSE)。
