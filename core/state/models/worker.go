package models

type Worker struct {
	WorkerType       string `json:"worker_type" gorm:"index:type_and_engine_ix,unique"`
	CountPerInstance int    `json:"count_per_instance"`
	Engine           string `json:"engine" gorm:"index:type_and_engine_ix,unique"`
}

type WorkersList struct {
	Total   int64    `json:"total"`
	Workers []Worker `json:"workers"`
}
