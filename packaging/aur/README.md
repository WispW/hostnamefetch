# AUR 打包

AUR 包名：`hostnamefetch`（源码包，不使用 `hostnamefetch-bin`）。

## 一次性准备

1. 注册/登录 AUR：<https://aur.archlinux.org/register>
2. 生成专用 SSH key：

   ```sh
   ssh-keygen -t ed25519 -f ~/.ssh/aur -C "aur@hostnamefetch" -N ""
   ```

3. 把 `~/.ssh/aur.pub` 的内容添加到 AUR 账号的 **My Account → SSH Public Key**。
4. 验证：

   ```sh
   ssh -T aur@aur.archlinux.org
   ```

   看到欢迎信息或 "Interactive shell is disabled" 即表示认证成功。
5. 在 GitHub 仓库 **Settings → Secrets and variables → Actions** 添加：

   - `AUR_SSH_PRIVATE_KEY`（必填）：`~/.ssh/aur` 私钥全文
   - `AUR_GIT_NAME`（可选）：AUR commit 作者名，默认 `WispW`
   - `AUR_GIT_EMAIL`（可选）：AUR commit 邮箱，默认 `wispw@users.noreply.github.com`

## 自动发布

`.github/workflows/aur.yml` 有两种触发方式：

- Release 发布后自动运行（`release: published`）
- Actions 页面手动运行 `AUR`，输入 tag，例如 `v2026.11.21`

工作流会：

1. 检出默认分支上的打包脚本（保证脚本是最新的）
2. 从指定 tag 下载 `hostnamefetch` 和 `LICENSE`
3. 计算 SHA256，生成 `PKGBUILD` 和 `.SRCINFO`
4. 用 `AUR_SSH_PRIVATE_KEY` 推送到 `ssh://aur@aur.archlinux.org/hostnamefetch.git`

首次手动运行会在 AUR 上创建该包；之后每次发布只需要发布 Release 或手动触发。

## 本地生成/验证

```sh
# 生成 PKGBUILD 和 .SRCINFO（需要 makepkg 或 Docker）
packaging/aur/update.sh v2026.11.21

# 在当前 Arch 环境构建测试包
cd packaging/aur
makepkg -f
namcap hostnamefetch-*.pkg.tar.zst  # 可选

# 如果 SSH 已配置，直接推送到 AUR
packaging/aur/update.sh v2026.11.21 --push
```

## 验证

```sh
yay -Ss hostnamefetch
yay -S hostnamefetch
hostnamefetch --version
pacman -Ql hostnamefetch
```

AUR 页面：<https://aur.archlinux.org/packages/hostnamefetch>

## 说明

- `pkgver` 直接取自 tag（去掉前缀 `v`），`pkgrel=1`；同日再次发布使用 `v2026.11.21.2` 这类 tag。
- `PKGBUILD.in` 和 `update.sh` 在仓库中维护；生成的 `PKGBUILD`、`.SRCINFO` 已被 `.gitignore` 忽略，AUR 仓库中的这两个文件才是发布源。
- AUR 只允许推送到 `master` 分支；`.SRCINFO` 每次都会重新生成。
- AUR 官方对自动化更新持谨慎态度，发布前建议用 `makepkg --printsrcinfo` / `makepkg -f` 检查生成结果。
