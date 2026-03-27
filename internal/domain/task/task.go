// Package task 定义诊断任务和结果的领域模型
package task

import "time"

// Status 任务状态
type Status string

const (
	StatusPending    Status = "pending"
	StatusRunning    Status = "running"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusRecovering Status = "recovering"
	StatusRecovered  Status = "recovered"
)

// Task 表示一次诊断任务
type Task struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`    // 所属租户 ID
	Input       string    `json:"input"`        // 用户输入或告警内容
	Source      string    `json:"source"`        // 来源: cli / web / webhook
	Status      Status    `json:"status"`
	Steps       []Step    `json:"steps"`         // Agent 推理步骤
	Diagnosis   *Diagnosis `json:"diagnosis,omitempty"`
	Recovery    *Recovery  `json:"recovery,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Step 是 Agent ReAct 循环中的一步
type Step struct {
	Index      int            `json:"index"`
	Think      string         `json:"think"`
	Action     *Action        `json:"action,omitempty"`
	Observe    string         `json:"observe,omitempty"`
	IsKeyStep  bool           `json:"is_key_step"`             // 标识是否为关键步骤（tool_call）
	ToolName   string         `json:"tool_name,omitempty"`     // 冗余字段，方便前端直接使用
	ToolParams map[string]any `json:"tool_params,omitempty"`   // 冗余字段，方便前端直接使用
	Timestamp  time.Time      `json:"timestamp"`
}

// Action 是工具调用
type Action struct {
	Tool   string         `json:"tool"`
	Params map[string]any `json:"params"`
}

// Diagnosis 诊断结论
type Diagnosis struct {
	RootCause   string   `json:"root_cause"`
	Confidence  float64  `json:"confidence"`
	AffectedServices []string `json:"affected_services"`
	Impact      string   `json:"impact"`
	Suggestions []string `json:"suggestions"`
}

// Recovery 恢复操作记录
type Recovery struct {
	Actions    []RecoveryAction `json:"actions"`
	Status     RecoveryStatus   `json:"status"`
	VerifiedAt *time.Time       `json:"verified_at,omitempty"`
}

// RecoveryAction 单次恢复动作
type RecoveryAction struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Result      string `json:"result"`
	Success     bool   `json:"success"`
}

// RecoveryStatus 恢复状态
type RecoveryStatus string

const (
	RecoveryPending   RecoveryStatus = "pending"
	RecoveryExecuting RecoveryStatus = "executing"
	RecoverySuccess   RecoveryStatus = "success"
	RecoveryFailed    RecoveryStatus = "failed"
	RecoverySkipped   RecoveryStatus = "skipped"
)

// TaskEvent 是 SSE 推送的事件，用于实时展示 Agent 思考过程
type TaskEvent struct {
	TaskID   string `json:"task_id"`
	TenantID string `json:"tenant_id,omitempty"` // 所属租户 ID
	Type     string `json:"type"`                // step / diagnosis / recovery / status
	Data     any    `json:"data"`
}
