# task238-stagecue — 戏剧舞台灯光提示时序复核台

舞台技术人员在演练后需要确认灯光提示是否在演员、幕布与机械装置的安全窗口内执行。
本项目导入场次脚本事件、灯光提示与设备响应日志，校正多源时钟偏差，构建统一提示时间线，
检测相互排斥的照明与机械动作的时序冲突，支持豁免登记与可发布提示包版本固化。

## 业务闭环
1. 导入演练版本：场次脚本事件、灯光提示、设备响应日志。
2. 对齐模块校正各源时钟偏差，构建统一时间线。
3. 约束模块检测互斥的照明/机械动作是否在安全窗口内。
4. 设计师对冲突登记豁免理由（如艺术需要、备用机械）。
5. 复核通过后发布提示包版本（不可变快照），供演出执行。

## 核心状态机
- 演练版本：导入中 → 待复核 → 可发布 → 冻结
- 灯光提示：待对齐 → 有效 / 提前 / 延迟 / 冲突
- 舞台约束：草稿 → 生效 → 豁免 → 废止
- 提示包：草稿 → 复核 → 发布 → 替代

## 标准命令
```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
./stagecue --addr :8080 --db ./stagecue.db
./stagecue --smoke-test
```

## 持久化与重启恢复
使用 SQLite（modernc.org/sqlite，CGO 无关）持久化全部实体；服务关闭后重新打开同一数据库可恢复未完成对齐；同一提示序号幂等；发布包绑定完整演练输入。

## 模块责任
- `internal/rehearsal`：演练输入与事件管理
- `internal/align`：时钟校正与时间线构建
- `internal/constraint`：互斥约束与安全窗口检测
- `internal/waiver`：豁免理由登记
- `internal/cuepkg`：提示包版本发布
- `internal/service`：编排层
- `internal/httpapi`：HTTP 路由（前缀 `/api`）
