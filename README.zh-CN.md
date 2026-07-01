# mcmon

[English](README.md) | [简体中文](README.zh-CN.md)

mcmon 是一个轻量级的 Minecraft Java 版服务器桌面监控工具。它使用原版 status/ping 协议检测服务器，使用 SQLite 保存历史数据，并在内置桌面界面中展示在线状态、人数、延迟和丢包等指标。

这个桌面 app 可以独立使用，不需要部署 `mcmon-host`。如果你需要集中管理多台节点，也可以从 Remote 页面连接到 `mcmon-host`。

## 相关项目

- `mcmon`：本地桌面监控 app，适合个人或单机使用。
- [`mcmon-host`](https://github.com/Ctrl-Creeper/mcmon-host)：Linux-only 的中心面板和管理 API，用于集中配置节点和生成 `mcmon-agent` 一键安装命令。
- [`mcmon-agent`](https://github.com/Ctrl-Creeper/mcmon-agent)：无 UI 的跨平台轻量节点进程，向 `mcmon-host` 上报数据。

## 功能

- 基于 Wails 的原生桌面 app。
- Minecraft Java 版 status/ping 检测。
- online、players、latency、loss 四类指标。
- 每个指标都可以单独开关并设置检测周期。
- SQLite 本地历史数据。
- 可选后台运行，关闭窗口后继续监控。
- 可选连接 `mcmon-host` 查看远程部署数据。
- 支持浅色/深色主题和中英文界面切换。

## 开发要求

- Go 1.25.4 或兼容版本。
- Node.js/npm，Wails 构建过程需要。
- Wails v2.10.2，构建脚本会通过 `go run` 调用。

Linux 桌面构建需要 GTK/WebKitGTK 开发包。Ubuntu 24.04+：

```sh
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev pkg-config
```

Ubuntu 22.04 或较旧 Debian 系统可以使用 WebKitGTK 4.0：

```sh
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config
```

## 从源码运行

桌面 app：

```sh
go run .
```

CLI/server 模式：

```sh
go run ./cmd/mcmon
```

默认本地地址：

```text
http://127.0.0.1:8090
```

## 构建桌面 app

macOS 和 Linux：

```sh
./scripts/build-desktop.sh
```

macOS 产物：

```text
build/bin/mcmon.app
```

Windows PowerShell：

```powershell
.\scripts\build-desktop.ps1
```

## 构建 CLI

```sh
go build -o dist/mcmon ./cmd/mcmon
```

交叉编译示例：

```sh
GOOS=windows GOARCH=amd64 go build -o dist/mcmon.exe ./cmd/mcmon
GOOS=linux   GOARCH=amd64 go build -o dist/mcmon-linux ./cmd/mcmon
GOOS=darwin  GOARCH=arm64 go build -o dist/mcmon-mac ./cmd/mcmon
```

## 后台运行

在桌面 app 的 Settings 页面启用 `Run in background`。

也可以使用 CLI：

```sh
mcmon install -config /path/to/config.json
mcmon uninstall
```

后台服务只启动轻量 server 模式，不会打开 GUI：

```sh
serve -config /path/to/config.json
```

平台路径：

- macOS：`~/Library/LaunchAgents/com.mcmon.plist`
- Linux：`~/.config/systemd/user/mcmon.service`
- Windows：名为 `mcmon` 的 Scheduled Task

## 远程 Host

mcmon 可以独立监控本地配置的服务器，也可以在 Remote 页面连接到 `mcmon-host`。

Remote 设置支持：

- Host URL。
- 管理员用户名/密码登录。
- 可选 TOTP 2FA 验证码。
- 登录后转发 session token 到 host API。

未配置远程 Host 时，本地监控不受影响。

## 开发检查

```sh
go test ./...
go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2 doctor
```
