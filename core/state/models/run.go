package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func NewRunID(engine *string) string {
	uid := uuid.New().String()
	return fmt.Sprintf("%s-%s", *engine, uid[len(*engine)+1:])
}

type Run struct {
	RunID                 string         `json:"run_id" gorm:"primaryKey; type:varchar"`
	TemplateID            *string        `json:"template_id,omitempty" gorm:"type:varchar"`
	Template              *Template      `json:"-" gorm:"references:TemplateID"`
	Alias                 string         `json:"alias" gorm:"type:varchar"`
	Image                 string         `json:"image" gorm:"type:varchar"`
	ClusterName           string         `json:"cluster" gorm:"type:varchar"`
	ExitCode              *int64         `json:"exit_code,omitempty"`
	Status                RunStatus      `json:"status" gorm:"type:varchar"`
	QueuedAt              *time.Time     `json:"queued_at,omitempty"`
	StartedAt             *time.Time     `json:"started_at,omitempty"`
	FinishedAt            *time.Time     `json:"finished_at,omitempty"`
	InstanceID            string         `json:"-" gorm:"type:varchar"`
	InstanceDNSName       string         `json:"-" gorm:"type:varchar"`
	GroupName             string         `json:"group_name" gorm:"type:varchar"`
	Env                   *EnvList       `json:"env,omitempty" gorm:"type:jsonb; index:,type:gin"`
	Command               *string        `json:"command,omitempty"`
	CommandHash           *string        `json:"command_hash,omitempty"`
	Memory                *int64         `json:"memory,omitempty"`
	MemoryLimit           *int64         `json:"memory_limit,omitempty"`
	Cpu                   *int64         `json:"cpu,omitempty"`
	CpuLimit              *int64         `json:"cpu_limit,omitempty"`
	Gpu                   *int64         `json:"gpu,omitempty"`
	ExitReason            *string        `json:"exit_reason,omitempty"`
	Engine                *string        `json:"engine,omitempty"`
	NodeLifecycle         *string        `json:"node_lifecycle,omitempty"`
	EphemeralStorage      *int64         `json:"ephemeral_storage,omitempty"`
	MaxMemoryUsed         *int64         `json:"max_memory_used,omitempty"`
	MaxCpuUsed            *int64         `json:"max_cpu_used,omitempty"`
	AttemptCount          *int64         `json:"attempt_count,omitempty"`
	SpawnedRuns           *SpawnedRuns   `json:"spawned_runs,omitempty" gorm:"type:jsonb"`
	RunExceptions         *RunExceptions `json:"run_exceptions,omitempty" gorm:"type:jsonb"`
	ActiveDeadlineSeconds *int64         `json:"active_deadline_seconds,omitempty"`
}

func (r *Run) BeforeCreate(tx *gorm.DB) (err error) {
	if r.Engine == nil {
		r.Engine = &DefaultEngine
	}

	r.RunID = NewRunID(r.Engine)
	return
}

type RunList struct {
	Total int64 `json:"total"`
	Runs  []Run `json:"history"`
}

type SpawnedRun struct {
	RunID string `json:"run_id"`
}

type SpawnedRuns []SpawnedRun

func (e RunExceptions) Value() (driver.Value, error) {
	res, _ := json.Marshal(e)
	return res, nil
}

func (e *RunExceptions) Scan(value interface{}) error {
	if value != nil {
		s := value.([]uint8)
		json.Unmarshal(s, &e)
	}
	return nil
}

type RunExceptions []string

func (e SpawnedRuns) Value() (driver.Value, error) {
	res, _ := json.Marshal(e)
	return res, nil
}

func (e *SpawnedRuns) Scan(value interface{}) error {
	if value != nil {
		s := []byte(value.(string))
		json.Unmarshal(s, &e)
	}
	return nil
}
