# monitor

基于 Beszel v0.21.0 的公开监控看板二开，保留 MIT 许可证及原版权声明。
项目仓库：[kerntau/monitor](https://github.com/kerntau/monitor)，主分支：`main`。默认站点品牌为 **chenv**，可在后台设置名称、favicon 和主题。
在线预览：[ss.chenv.cn](https://ss.chenv.cn/)

## 界面与基础功能

直接复用 Beszel 的 Logo、导航、登录布局、主机表格、“添加系统”弹窗、网络监控表格、监控弹窗、安装菜单和历史图表。没有重新设计主题或另写一套 CSS。

- 公开看板免登录；首页直接显示 Oracle 详情（Oracle 未公开时显示排序第一的公开主机），无公开主机显示空状态。隐藏“全部服务器”链接，点击品牌回首页；保留其他主机的 `/system/:id` 详情链接。
- 保留 CPU、内存、磁盘、网络、负载、在线状态、Agent 自动识别的主机名／硬件和历史图表。
- 管理员可以新增、编辑、暂停／恢复、删除主机；Docker／二进制、公钥／令牌、Compose／docker run／Linux／Homebrew／Windows／FreeBSD 安装入口保留。
- 通用令牌支持临时和持久模式，保留指纹清除与令牌轮换；自动注册的主机默认未公开。
- 网络监控保留过滤、视图／排序、ICMP／TCP／HTTP／DNS、批量新增、多主机添加、编辑、暂停／恢复、删除、复制批量配置、间隔、响应、1 小时平均／最大／最小、丢包、证书、更新时间，以及原版历史与目标／系统比较。
- 公开网络表格不显示“目标”列，不再生成“探测 1”占位名称；后台仍保留目标输入。探测显示名称和公开开关在原版监控弹窗中配置。
- 浏览器偏好保存在本地，不写入共享账户。
- 普通账户入口、注册、OAuth、告警执行、通知、心跳、容器／SMART／高级运维页面关闭，已有记录保留。

## 本地预览

默认本地访问：<http://127.0.0.1:8091/>，管理员：<http://127.0.0.1:8091/chenv-admin>。
连续点击 Logo 5 次（每次间隔不超过 3 秒）也能打开管理员登录，手机同样支持。
管理员使用现有 PocketBase 超级管理员邮箱和密码。公开导航不提供管理员链接。

数据库、私钥和备份保存在部署者自己的数据目录中，不随源码分发。测试现有数据时请使用副本，并暂停主机，避免同时连接生产 Agent。

本地构建需要 Node.js 22 和 Go 1.27.1。Windows PowerShell：

```powershell
git clone https://github.com/kerntau/monitor.git
cd monitor
npm.cmd --prefix internal/site ci
npm.cmd --prefix internal/site run build
New-Item -ItemType Directory -Force .tmp | Out-Null
go build -tags no_ui -o .tmp/monitor.exe ./internal/cmd/hub
.tmp/monitor.exe superuser create admin@example.com 'your-own-password' --dir=./monitor_data
.tmp/monitor.exe serve --http=127.0.0.1:8091 --dir=./monitor_data
```

修改页面后重新构建 Hub，因为前端产物嵌入二进制中；先停止运行中的旧进程再替换二进制。

## Docker

```sh
git clone https://github.com/kerntau/monitor.git
cd monitor
cp .env.example .env
docker compose build
docker compose up -d
```

数据持久化在 `./monitor_data`。默认只绑定本机 8091；用反向代理／域名访问时保留 WebSocket 与请求头。首次空库创建超级管理员：

```sh
docker compose exec monitor /monitor superuser create admin@example.com 'your-own-password' --dir=/monitor_data
```

该命令中的邮箱／密码仅为占位内容，请使用自己的信息。已有库无需另建普通用户或 AUTO_LOGIN 账户。登录后台后新增服务器、复制 Agent 安装配置，再在“展示设置”中开启公开显示。

默认 `MONITOR_BIND=127.0.0.1`、`MONITOR_PORT=8091`，可在 `.env` 中修改。通过其他设备直接访问时可将 `MONITOR_BIND` 改为 `0.0.0.0`；使用域名部署时可保持本机绑定并配置 HTTPS 反向代理。

## 接口与隐私

公开接口：`/api/chenv/public/config`、`systems`、`systems/:id`、`monitors`、`monitor-stats?ids=...&range=...`。匿名只读、字段白名单、查询范围有界。

管理接口：`/api/chenv/admin`，全部验证 `_superusers`。超级管理员认证与公开数据请求分开。普通用户和旧令牌不能读取原始集合；`/_/` 网页入口关闭。

默认在服务端隐藏 IP、连接地址、端口、探测目标与 DNS 地址。主机名和硬件由 Agent 提供，继续显示；主机名若实际是 IP 则脱敏。比较使用服务端产生的 opaque key，不向访客传递目标地址。未公开主机／探测及其历史不可从匿名接口读取。

显示开关只能控制页面内容，浏览器地址栏由部署域名决定。保留原服务器 ID 和 `/system/:id` 链接。

## 迁移、更新与回滚

详见 [部署与回滚](docs/DEPLOYMENT.md) 和 [验证记录](docs/VALIDATION.md)。上线前备份整个数据目录和原程序。Docker 部署文件已提供，当前开发环境没有 Docker Engine，尚未实际运行镜像构建验证。

上游：<https://github.com/henrygd/beszel>。许可证：[MIT](LICENSE)。
