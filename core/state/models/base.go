package models

import (
	"database/sql/driver"
	"encoding/json"
)

type Tags []string
type PortsList []string

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type EnvList []EnvVar

func (e *EnvList) Scan(value interface{}) error {
	if value != nil {
		s := value.([]byte)
		json.Unmarshal(s, &e)
	}
	return nil
}

// Value to db
func (e EnvList) Value() (driver.Value, error) {
	res, _ := json.Marshal(e)
	return res, nil
}
