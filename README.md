<div align="center">

# MONITOR // CHENV

**基于 Beszel 二开的公开主机与网络监控看板。**<br>
*A Beszel-derived public dashboard for host and network monitoring.*

[![GitHub](https://img.shields.io/badge/GitHub-kerntau%2Fmonitor-181717?style=for-the-badge&logo=github)](https://github.com/kerntau/monitor)
[![Go](https://img.shields.io/badge/Go-1.27.1-00ADD8?style=for-the-badge&logo=go&logoColor=ffffff)](https://go.dev)
[![Node.js](https://img.shields.io/badge/Node.js-22-339933?style=for-the-badge&logo=nodedotjs&logoColor=ffffff)](https://nodejs.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=ffffff)](https://www.docker.com)
[![MIT](https://img.shields.io/badge/License-MIT-10B981?style=for-the-badge)](LICENSE)

</div>

---

## 01. PROJECT PURPOSE // 项目定位

monitor 是基于 Beszel v0.21.0 的公开监控看板二开，默认品牌为 chenv。它保留主机指标、网络探测、历史图表和 Agent 安装入口，并增加公开数据字段白名单、匿名只读接口和管理员设置。

在线预览：[ss.chenv.cn](https://ss.chenv.cn/)。

## 02. TECHNICAL ARCHITECTURE // 技术架构

~~~mermaid
graph TD
    A[Beszel Agent] --> B[Go Hub]
    C[Admin and public web UI] --> B
    B --> D[PocketBase data]
    B --> E[Public allowlisted API]
    E --> F[Anonymous dashboard]
    G[Network probes] --> B
~~~

- Go Hub 嵌入 internal/site 的前端产物。
- PocketBase 保存主机、网络监控和管理员数据。
- 匿名接口只返回公开主机和探测的字段白名单，并隐藏地址、端口和目标。
- compose.yml 默认将容器映射到本机 127.0.0.1:8091。

## 03. TECHNICAL MATRIX // 技术矩阵

| 模块 | 技术 | 说明 |
| --- | --- | --- |
| Hub | Go 1.27.1 + PocketBase | API、认证、任务和数据存储 |
| Agent | Beszel agent | CPU、内存、磁盘、网络和系统信息 |
| 前端 | internal/site | 监控表格、图表和管理界面 |
| 网络监控 | ICMP、TCP、HTTP、DNS | 过滤、历史、延迟和丢包 |
| 部署 | Docker Compose 或 Go 二进制 | 默认数据目录 monitor_data |
| 安全 | 超级管理员与字段白名单 | 公开接口匿名只读 |

## 04. REPOSITORY TOPOLOGY // 源码工程拓扑

~~~text
.
├── internal/site/             # 前端源码与构建产物
├── internal/cmd/hub/          # Go Hub 入口
├── internal/cmd/agent/        # Agent 入口
├── agent/                     # Agent 构建资源
├── docs/                      # 部署与验证文档
├── Dockerfile                 # 多阶段镜像构建
├── compose.yml                # 本地 Docker Compose
├── Makefile                   # Go/前端构建和测试
└── go.mod                     # Go 模块
~~~

## 05. CLONE & QUICK START // 极速装配指南

环境要求：Go 1.27.1、Node.js 22；Docker 部署需要 Docker Engine。

~~~bash
git clone https://github.com/kerntau/monitor.git
cd monitor
npm install --prefix internal/site
npm run build --prefix internal/site
go build -tags no_ui -o monitor ./internal/cmd/hub
./monitor superuser create admin@example.com 'change-this-password' --dir=./monitor_data
./monitor serve --http=127.0.0.1:8091 --dir=./monitor_data
~~~

也可以使用 Docker Compose：

~~~bash
cp .env.example .env
docker compose build
docker compose up -d
~~~

## 06. DEPLOYMENT & VERIFICATION // 部署与验证

~~~bash
make test
make build
~~~

前端改动后必须重新构建 Hub，因为前端产物会嵌入二进制。首次部署前备份 monitor_data/；通过反向代理或域名访问时保留 WebSocket 和请求头。

默认配置：

| 变量 | 默认值 | 作用 |
| --- | --- | --- |
| MONITOR_BIND | 127.0.0.1 | Docker 主机绑定地址 |
| MONITOR_PORT | 8091 | Docker 主机端口，容器内为 8090 |

## 07. LICENSE // 许可证

本项目基于 MIT License，许可证文本见 LICENSE。上游项目为 [henrygd/beszel](https://github.com/henrygd/beszel)。
