基于 Go 实现的戏剧舞台灯光提示时序复核 Web 项目，一款后端服务，完成场次脚本事件与时间校正、灯光提示与设备响应时序对齐、互斥照明与机械动作的冲突检测、豁免裁决与可发布提示包版本固化。

# BENZHI 评测说明

本项目为舞台灯光提示时序复核台（stagecue）。它导入演练日志（场次脚本事件、灯光提示、设备响应），对多源时钟做偏差校正，构建统一的提示时间线，检测相互排斥的照明与机械动作是否在演员、幕布、机械装置的安全窗口内执行，并支持豁免登记与提示包版本发布。

## 项目类型
- 领域：舞台技术 / 演出安全时序复核
- 形态：后端服务（HTTP API 前缀 `/api`），含 SQLite 持久化与重启恢复

## 标准命令
```bash
# 构建
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
# 静态检查
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
# 测试
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
# 启动服务
./stagecue --addr :8080 --db ./stagecue.db
# 自检（不启动长驻服务，真实落库并重启恢复后退出）
./stagecue --smoke-test
```

## HTTP API 入口
路由前缀统一为 `/api`，核心能力：
- 演练：`POST /api/rehearsals`、`GET /api/rehearsals/:id`、`POST /api/rehearsals/:id/events`、`GET /api/rehearsals/:id/timeline`
- 提示：`POST /api/cues`、`GET /api/cues/:id`、`PATCH /api/cues/:id/anchor`
- 约束：`POST /api/constraints`、`GET /api/constraints`
- 冲突：`GET /api/rehearsals/:id/conflicts`、`POST /api/conflicts/:id/waiver`
- 豁免：`GET /api/waivers`
- 发布：`POST /api/packages`、`GET /api/packages/:id`、`POST /api/packages/:id/publish`
- 自检：`GET /api/selfcheck`

## Docker 双架构
验收侧会分别构建 `linux/amd64` 与 `linux/arm64` 镜像，并以容器内的 `--smoke-test` 作为唯一自检判据。手工构建单个平台时使用仓库内置脚本：
```bash
bash build_benzhi_docker.sh stagecue-check linux/amd64
```
脚本参数依次为镜像名和目标平台；容器入口默认执行 `--smoke-test`，也可以显式运行该参数。

## --smoke-test 契约
`--smoke-test` 模式下：真实写入演练、事件、提示、约束；校正时钟偏差；构建时间线；检测一处暗场提示早于演员离开安全区的冲突；登记豁免；发布提示包；关闭并重新打开数据库验证重启恢复，最终以退出码 0 结束。该模式不启动 HTTP 长驻服务。
