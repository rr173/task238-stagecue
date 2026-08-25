package model

import "errors"

// 领域错误，供 service / httpapi 映射为 HTTP 状态码。
var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource conflict")
	ErrInvalidState      = errors.New("invalid state transition")
	ErrDuplicateCueSeq   = errors.New("duplicate cue sequence number")
	ErrNegativeDuration  = errors.New("negative or zero duration")
	ErrUnknownRehearsal  = errors.New("unknown rehearsal")
	ErrFrozenRehearsal   = errors.New("rehearsal is frozen")
	ErrInvalidClockSkew  = errors.New("invalid clock skew value")
	ErrCircularRelation  = errors.New("circular device relation")
	ErrMissingWaiver     = errors.New("waiver evidence missing")
	ErrReleasedPackage   = errors.New("package already released")
)

// State 表示通用状态机状态字符串。
type State string

const (
	StateImporting  State = "importing"
	StatePending    State = "pending"
	StateReviewable State = "reviewable"
	StateFrozen     State = "frozen"
	StateValid      State = "valid"
	StateEarly      State = "early"
	StateLate       State = "late"
	StateConflict   State = "conflict"
	StateDraft      State = "draft"
	StateActive     State = "active"
	StateWaived     State = "waived"
	StateRevoked    State = "revoked"
	StateRehearsing State = "rehearsing"
	StateAudited    State = "audited"
	StateApproved   State = "approved"
	StateRejected   State = "rejected"
	StatePublished  State = "published"
	StateSuperseded State = "superseded"
)
