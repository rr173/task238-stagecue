package model

import "time"

// Now 返回当前 UTC 时间（统一时间源）。
func Now() time.Time { return time.Now().UTC() }

// ZeroTime 返回零值时间（用于未冻结/未发布占位）。
func ZeroTime() time.Time { return time.Time{} }

// SourceKind 表示事件来源种类。
type SourceKind string

const (
	SourceScript    SourceKind = "script"     // 场次脚本事件（权威时间源）
	SourceCueLog    SourceKind = "cue_log"    // 灯光提示触发日志
	SourceDeviceLog SourceKind = "device_log" // 设备响应日志（灯具/机械/幕布）
)

// ActorKind 表示舞台上需要被保护的对象类别。
type ActorKind string

const (
	ActorPerformer ActorKind = "performer" // 演员安全区
	ActorCurtain   ActorKind = "curtain"   // 幕布
	ActorMechanism ActorKind = "mechanism" // 机械装置
)

// EventRole 表示事件在时序复核中的角色。
type EventRole string

const (
	RoleCueStart EventRole = "cue_start" // 灯光提示开始（如暗场）
	RoleCueEnd   EventRole = "cue_end"   // 灯光提示结束
	RoleMoveIn   EventRole = "move_in"   // 演员/物体进入
	RoleMoveOut  EventRole = "move_out"  // 演员/物体离开
	RoleMechMove EventRole = "mech_move" // 机械动作
	RoleCurtain  EventRole = "curtain"   // 幕布动作
)

// Rehearsal 演练版本：导入中 → 待复核 → 可发布 → 冻结。
type Rehearsal struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Show        string    `json:"show"`
	State       State     `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
	ImportedAt  time.Time `json:"imported_at"`
	FrozenAt    time.Time `json:"frozen_at,omitempty"`
}

// StageEvent 单条带源时间戳的事件，需经时钟校正后才能进入统一时间线。
type StageEvent struct {
	ID           string    `json:"id"`
	RehearsalID  string    `json:"rehearsal_id"`
	Source       SourceKind `json:"source"`
	Seq          int       `json:"seq"` // 提示序号（仅在 cue_log 下有意义），同 rehearsal 幂等
	Actor        ActorKind `json:"actor"`
	Role         EventRole `json:"role"`
	Label        string    `json:"label"`
	RawTimestamp int64     `json:"raw_timestamp"` // 源时钟下的毫秒时间戳
	CorrectedAt  int64     `json:"corrected_at"`  // 校正后统一时钟毫秒时间戳
	Device       string    `json:"device"`        // 设备标识（灯具/机械编号）
	CreatedAt    time.Time `json:"created_at"`
}

// ClockSkew 一个源的时钟偏差（毫秒），统一时间线基准则为 0。
type ClockSkew struct {
	RehearsalID string     `json:"rehearsal_id"`
	Source      SourceKind `json:"source"`
	SkewMs      int64      `json:"skew_ms"`
	Baseline     bool       `json:"baseline"`
}

// Constraint 舞台约束：草稿 → 生效 → 豁免 → 废止。
// 描述一类互斥关系：在某一 actor 的安全窗口内，照明动作与机械动作不得重叠。
type Constraint struct {
	ID          string    `json:"id"`
	RehearsalID string    `json:"rehearsal_id"`
	Name        string    `json:"name"`
	Actor       ActorKind `json:"actor"`
	State       State     `json:"state"`
	MinGapMs    int64     `json:"min_gap_ms"` // 照明结束到机械开始的最小安全间隔
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

// Conflict 检测到的时序冲突。
type Conflict struct {
	ID            string    `json:"id"`
	RehearsalID   string    `json:"rehearsal_id"`
	ConstraintID  string    `json:"constraint_id"`
	Actor         ActorKind `json:"actor"`
	CueEventID    string    `json:"cue_event_id"`
	MechEventID   string    `json:"mech_event_id"`
	OverlapMs     int64     `json:"overlap_ms"`
	AtMs          int64     `json:"at_ms"`
	Resolved      bool      `json:"resolved"`
	CreatedAt     time.Time `json:"created_at"`
}

// Waiver 豁免理由登记。
type Waiver struct {
	ID         string    `json:"id"`
	ConflictID string    `json:"conflict_id"`
	Reason     string    `json:"reason"`
	Evidence   string    `json:"evidence"`
	CreatedAt  time.Time `json:"created_at"`
}

// CuePackage 提示包版本：草稿 → 复核 → 发布 → 替代。
type CuePackage struct {
	ID            string    `json:"id"`
	RehearsalID   string    `json:"rehearsal_id"`
	Version       int       `json:"version"`
	State         State     `json:"state"`
	SnapshotDigest string   `json:"snapshot_digest"` // 完整演练输入的摘要（不可变）
	ReleasedAt    time.Time `json:"released_at,omitempty"`
	SupersededBy  string    `json:"superseded_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
