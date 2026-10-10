# 验证记录

本地验证对象：生产数据副本和独立测试库。生产程序尚未替换。

## 已执行

- TypeScript 检查、Lingui 编译与 Vite 生产构建。
- Go Hub 编译，使用 `no_ui` 排除原 PocketBase 管理网页。
- chenv 后端权限与匿名数据脱敏测试；检查普通账户／旧令牌、未公开主机与探测、集合接口及管理员接口不可绕过权限。
- 主机与探测 CRUD、公开排序、Agent 专用令牌、超级管理员通用令牌、主题／favicon 配置、密码修改及旧会话失效测试。
- 上游相关 Agent 连接、网络监控、记录清理及主机管理测试。
- 浏览器自动化：单台／多台、原版新增系统／安装 UI、网络表格／历史／比较控件、Logo 连击、登录失败／退出／会话过期、展示设置和手机布局。

测试命令：

```powershell
$env:CHENV_MODE='false'
go test -tags 'testing no_ui' ./internal/hub ./internal/alerts ./internal/records ./internal/hub/systems -run 'TestChenv|TestAgentConnect|TestAgentWebSocketIntegration|TestNetworkMonitor|TestApiRoutesAuthentication|TestHandleSystem|TestDeleteOld' -count=1
npm.cmd --prefix internal/site run test:e2e
```

Windows Hub 测试编译间接导入 Agent；上游 Windows Agent 的嵌入资源需要相应的 .NET／smartctl 构建资源。本地占位文件只用于 Hub 测试编译，不代表 Agent 可执行文件已构建；本项目的 Docker 镜像只构建 Hub。

## 环境限制

当前电脑没有 Docker CLI／Docker Engine，因此尚未实际执行 Docker 镜像构建和 Compose 启动，部署文件需要在有 Docker 的主机上验证。当前本地数据副本已暂停主机，验证了迁移和原有 Agent 协议相关测试；未把生产 Agent 切换到新 Hub。

Vite 提示部分原版共享模块体积较大，不影响构建通过。依赖审计存在上游问题，未为修复审计而大幅升级原版前端依赖。
