package models

var (
	EKSEngine               = "eks"
	EKSSparkEngine          = "eks-spark"
	LocalEngine             = "local"
	DefaultEngine           = EKSEngine
	DefaultTaskType         = "task"
	MinCPU                  = int64(256)
	MaxCPU                  = int64(32000)
	MinMem                  = int64(512)
	MaxMem                  = int64(250000)
	TTLSecondsAfterFinished = int32(3600)
	Engines                 = []string{LocalEngine, EKSEngine, EKSSparkEngine}
	MaxLogLines             = int64(256)
	EKSBackoffLimit         = int32(0)
)

type RunStatus string

const (
	StatusRunning    RunStatus = "RUNNING"
	StatusQueued     RunStatus = "QUEUED"
	StatusNeedsRetry RunStatus = "NEEDS_RETRY"
	StatusPending    RunStatus = "PENDING"
	StatusStopped    RunStatus = "STOPPED"
)

type WorkerType string

const (
	RetryWorker  WorkerType = "retry"
	SubmitWorker WorkerType = "submit"
	StatusWorker WorkerType = "status"
)
