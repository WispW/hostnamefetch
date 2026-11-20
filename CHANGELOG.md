# 更新日志

本文件记录各版本的主要变更。

## [v2026.11.21.1] - 2026-11-21

### 新增

- Windows 原生 `hostnamefetch.exe`（Go 实现）：单文件、无需 Git Bash/WSL，
  采集 OS/机型/CPU/GPU/内存/分辨率/本地 IP/磁盘/终端/Shell，渲染与 bash 版一致
- 发布时自动把 exe 附加到对应 Release

## [v2026.11.21] - 2026-11-21

### 新增

- 内置 5×5 块字体渲染 hostname 横幅，不依赖 figlet / toilet
- neofetch 风格信息项：Hostname、FQDN、OS、Machine、Kernel、Uptime、Packages、Shell、Resolution、DE、WM、Terminal、CPU、GPU、Memory、Local IP、Disk、色块
- 自动左右/上下布局；长 hostname 按标点分块换行
- 左布局横幅相对信息列垂直居中
- hostname 校验和派生配色；非 UTF-8 自动回退 ASCII
- Linux / macOS / Windows Git Bash 兼容

### 变更

- 安装改为一行命令
- 版本号采用 CalVer `2026.11.21`

### 修复

- macOS/BSD 磁盘容量解析（避免 `df -P` 的 512-byte 块）
- macOS `system_profiler` 缓存，resolution/GPU 只调用一次
- `color 7`、`--disable title`、未知 `--disable`、内存回退、窄终端布局警告
