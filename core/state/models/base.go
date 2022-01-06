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

func (e *EnvList) ToMap() map[string]EnvVar {
	m := make(map[string]EnvVar, len(*e))
	for _, v := range *e {
		m[v.Name] = v
	}
	return m
}

func (e *EnvList) Merge(toMerge *EnvList) *EnvList {
	self := []EnvVar(*e)
	other := []EnvVar(*toMerge)
	together := append(self, other...)
	m := make(map[string]EnvVar)
	for _, v := range together {
		m[v.Name] = v
	}

	var merged EnvList
	for _, v := range m {
		merged = append(merged, v)
	}
	return &merged
}

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
