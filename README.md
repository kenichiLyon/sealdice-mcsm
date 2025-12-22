# Sealdice-MCSM-Bridge

Sealdice-MCSM-Bridge 是一个连接 Sealdice 海豹核心与 MCSManager 面板的桥接工具，允许通过海豹指令管理 MCSM 实例。

![CI/CD Pipeline](https://github.com/YOUR_USERNAME/sealdice-mcsm/actions/workflows/ci-cd.yml/badge.svg)

## 项目结构

- `server/`: Golang 后端服务，负责与 MCSM API 交互。
- `plugin/`: Sealdice JavaScript 插件，负责接收指令并与后端 WebSocket 通信。

## 功能特性

- **实例绑定**: 将海豹群组与 MCSM 实例绑定。
- **远程登录**: 支持通过二维码远程登录协议 (Relogin Workflow)。
- **指令控制**: 支持 Start, Stop, Status 等基础指令。

## 构建与部署

本项目使用 GitHub Actions 进行自动构建。
- **Server**: 自动交叉编译 Windows/Linux/macOS 版本。
- **Plugin**: 自动构建 JS 插件。
- **Release**: 每次 Push 自动生成 Release 并附带多平台压缩包。

## 开发

### Server
```bash
cd server
go run cmd/server/main.go
```

### Plugin
```bash
cd plugin/plugin
npm install
npm run build
```
