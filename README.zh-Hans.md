<p align="center">
  <a href="https://github.com/gythialy/fermata">
    <img src=".github/assets/banner.png" alt="Fermata / 停琴——菜单栏里的休息提醒。A break, held." width="100%">
  </a>
</p>

# Fermata · 停琴

**macOS、Windows、Linux 上的休息提醒。** 它待在菜单栏或系统托盘里；短休息和
长休息按自己的节奏到来，占用几分钟屏幕，然后安静地退回去。灵感来自 [Stretchly](https://github.com/hovancik/stretchly)，
用 Go 与 [MyGo](https://github.com/egoist/mygo) 写成：单进程、原生菜单栏、没有 WebView。

[English](README.md) · 简体中文

[![Go 1.27](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)](https://github.com/gythialy/fermata/blob/main/go.mod)
[![CI](https://github.com/gythialy/fermata/actions/workflows/ci.yml/badge.svg)](https://github.com/gythialy/fermata/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-3E5F4E.svg)](https://github.com/gythialy/fermata/blob/main/LICENSE)
[![macOS 13+ · Windows 11 · Linux](https://img.shields.io/badge/macOS%2013%2B%20%C2%B7%20Windows%2011%20%C2%B7%20Linux-171A17)](https://github.com/gythialy/fermata/blob/main/docs/development.md#requirements)
[![Nightly](https://github.com/gythialy/fermata/actions/workflows/nightly.yml/badge.svg)](https://github.com/gythialy/fermata/actions/workflows/nightly.yml)

## 它能做什么

- **可坚持的节奏。** 默认番茄钟：每 25 分钟一次 5 分钟短休息，每四次短休息
  之后一次 15 分钟长休息。两种都可调，但至少保留一种。
- **一块休息的场，不是对话框。** 每块屏幕一个窗口，占屏幕 85%，里面只有
  延长记号、一条建议、一条随休息推进的发丝线、倒计时，以及同一时刻唯一的
  出口——`延迟`、然后`跳过`、然后等待你完成的`结束`。键盘上 `⌘X` 效果相同。
- **严格模式说到做到。** 不能延迟、不能跳过、不能关窗，休息结束前应用拒绝退出。
  锁屏或 Mac 睡眠仍会结束这次休息。
- **不必离开菜单栏就能暂停。** 30 分钟、1 小时、2 小时、5 小时、直到明早，
  或无限期。离开电脑、专注模式、锁屏、以及你排除的应用也会自动暂停。
- **到点的休息会等一等。** 前台应用全屏（最多 1 小时），或启用后你正在输入
  （最多 30 秒）时，休息暂缓，条件解除后以完整时长开始。全屏检测只读窗口边界，不需要屏幕录制权限。
- **安静而完整的设置。** 通用、日程、外观、关于；英文、简体中文或跟随系统；
  登录时启动；菜单栏样式（记号／时间／圆环）；休息场颜色与不透明度；声音与
  音量；应用排除；休息健康；恢复默认。
- **状态弹窗代替一长串菜单项。** 左键点图标（或休息预告）看倒计时和快捷操作；
  右键保留经典菜单。
- **等待中的倒计时会被记住。** 十分钟内重启应用，倒计时继续；休息进行中或
  暂停中则不恢复。
- **更新保持安静。** 后台查询 GitHub Releases，在「关于」里提示；不会自行下载
  或安装。

## 截图

<p align="center">
  <a href="docs/images/zh-Hans/break-mini.png"><img src="docs/images/zh-Hans/break-loop.png" width="840" alt="休息窗动图：内容落定，发丝线随休息推进，倒计时逐秒减少，出口从「延迟」变为「跳过」"></a>
</p>

占屏幕 85% 的休息场：延长记号、一条建议、一条随休息推进的发丝线、倒计时，
以及同一时刻唯一的出口。动图由应用自己逐帧渲染，点击可看全尺寸静止图。

| 短休息 | 长休息 |
|:-:|:-:|
| <a href="docs/images/zh-Hans/break-mini.png"><img src="docs/images/zh-Hans/break-mini.png" width="400" alt="短休息：深绿色的场、延长记号、一条建议、发丝线与倒计时"></a> | <a href="docs/images/zh-Hans/break-long.png"><img src="docs/images/zh-Hans/break-long.png" width="400" alt="长休息：标题、正文、更长的发丝线与十五分钟"></a> |
| 每 25 分钟五分钟。 | 四次短休息后十五分钟。 |

### 应用各处

| 状态弹窗 | 首次启动 |
|:-:|:-:|
| <a href="docs/images/zh-Hans/popover.png"><img src="docs/images/zh-Hans/popover.png" width="270" alt="状态弹窗：倒计时、刻度尺和快捷操作"></a> | <a href="docs/images/zh-Hans/welcome.png"><img src="docs/images/zh-Hans/welcome.png" width="450" alt="欢迎窗，带一份休息的真实预览"></a> |
| 菜单栏里的倒计时与快捷操作。 | 第一次休息之前，先看一份真实预览。 |

### 设置

| 通用 | 日程 |
|:-:|:-:|
| <a href="docs/images/zh-Hans/settings-general.png"><img src="docs/images/zh-Hans/settings-general.png" width="400" alt="通用设置：登录时启动、语言、菜单栏样式与自动暂停"></a> | <a href="docs/images/zh-Hans/settings-schedule.png"><img src="docs/images/zh-Hans/settings-schedule.png" width="400" alt="日程设置：长短休息时长、延迟与跳过"></a> |
| 登录时启动、语言、菜单栏样式、自动暂停。 | 长短休息时长、延迟与跳过。 |

| 外观 | 关于 |
|:-:|:-:|
| <a href="docs/images/zh-Hans/settings-appearance.png"><img src="docs/images/zh-Hans/settings-appearance.png" width="400" alt="外观设置：休息场颜色与不透明度、声音与音量"></a> | <a href="docs/images/zh-Hans/settings-about.png"><img src="docs/images/zh-Hans/settings-about.png" width="400" alt="关于：版本、更新与 MIT 许可"></a> |
| 休息场颜色与不透明度、声音与音量。 | 版本、更新与许可。 |

## 安装

### macOS

macOS 13 或更新，通过 [Homebrew](https://brew.sh)：

```sh
brew tap gythialy/tap
brew install --cask fermata
```

cask 位于 [gythialy/homebrew-tap](https://github.com/gythialy/homebrew-tap)。
Fermata 为 ad-hoc 签名、未公证；cask 会清除下载隔离，若首次启动仍被 macOS
拦截，右键应用选择「打开」。更新用 `brew upgrade --cask fermata`。

### Windows

从 [Releases 页面](https://github.com/gythialy/fermata/releases/latest)下载，
x64 与 ARM64 各有：

- `fermata-setup-<版本>-amd64.exe`（或 `-arm64.exe`）：为当前用户安装到
  `%LOCALAPPDATA%\Programs\Fermata`，带开始菜单快捷方式与卸载项。
- `Fermata-<版本>-windows-<架构>-portable.zip`：解压即用，不写注册表安装信息。

Windows 版本尚未代码签名，首次运行 SmartScreen 会拦：选「更多信息 → 仍要运行」。
Windows 的托盘没有标题，所以图标只能显示记号或圆环，不显示倒计时。

### Linux

同一页面提供 x86-64 与 ARM64 的包：

- `fermata_<版本>_<架构>.deb`：Debian、Ubuntu
- `fermata-<版本>-1.<架构>.rpm`：Fedora、RHEL、openSUSE
- `fermata-<版本>-linux-<架构>.tar.gz` 与 `install.sh`：装到 `~/.local`，不需要 root
- `PKGBUILD-<架构>`：Arch Linux（`makepkg -si`）

需要 GTK 3；托盘图标还需要 AppIndicator（`libayatana-appindicator3`）。Linux 的
托盘只支持菜单，所以浮层从菜单项打开。

### 平台差异

- 全屏应用运行时暂缓休息、读取专注/勿扰状态，仅 macOS 支持；系统做不到的设置项
  会直接隐藏。
- 应用排除规则需要进程列表：macOS 与 Linux 提供，Windows 不提供。
- 结束休息的快捷键：macOS 是 ⌘X，其余平台是 Ctrl+X。

从源码构建见 [开发指南](https://github.com/gythialy/fermata/blob/main/docs/development.md)。

### Nightly（每夜构建）

每晚从 `main` 构建一个 DMG：未签名、未公证，可能随时是坏的。
[下载 Fermata-nightly.dmg](https://github.com/gythialy/fermata/releases/download/nightly/Fermata-nightly.dmg)。

## 名字的由来

*Fermata* 是乐谱上的延长记号：让一个音停得比写下的时值更久，乐队等着，然后继续。
中文名 **停琴** 取自南朝齐诗人谢朓《移病还园示亲属诗》：「停琴伫凉月，灭烛听归鸿。」
——停下琴弦，在凉月里站一会儿，吹灭蜡烛听归雁。两个名字说的是同一个动作；
记号里弧下的那一点，是按住琴弦的手指。

## 许可

MIT，见 [LICENSE](https://github.com/gythialy/fermata/blob/main/LICENSE)。Fermata
灵感来自 [Stretchly](https://github.com/hovancik/stretchly)（BSD-2-Clause），
未复用其代码；休息建议与声音是原创的。Spectral 与 Noto Serif SC 以
SIL Open Font License 1.1 分发。

## 文档

设计、行为与实现都写成了文档。从[文档索引](https://github.com/gythialy/fermata/blob/main/docs/README.md)开始，或直接进入：

| 文档 | 内容 |
|---|---|
| [产品](https://github.com/gythialy/fermata/blob/main/docs/product.md) | 目标、范围、决策、路线图与视觉画布 |
| [功能规范](https://github.com/gythialy/fermata/blob/main/docs/functional-spec.md) | 规范性的行为规则与验收用例 |
| [架构](https://github.com/gythialy/fermata/blob/main/docs/architecture.md) | 包结构、状态机、数据流、测试与构建 |
| [设计系统](https://github.com/gythialy/fermata/blob/main/docs/design-system.md) | Held Note v2：记号、令牌、组件与界面 |
| [开发](https://github.com/gythialy/fermata/blob/main/docs/development.md) | 运行、构建、安装、测试、命令行、i18n、素材 |

环境要求、从源码构建和完整命令行见[开发指南](https://github.com/gythialy/fermata/blob/main/docs/development.md)。
