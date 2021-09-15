package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"reflect"
)

func NewTemplateID() string {
	uid := uuid.New().String()
	return fmt.Sprintf("tpl-%s", uid[4:])
}

type Template struct {
	TemplateID      string             `json:"template_id" gorm:"primaryKey; type:varchar"`
	TemplateName    string             `json:"template_name" gorm:"type:varchar; index:name_and_version_ix,unique; not null"`
	Version         int64              `json:"version" gorm:"index:name_and_version_ix,unique"`
	Schema          TemplateJSONSchema `json:"schema" gorm:"type:jsonb"`
	CommandTemplate string             `json:"command_template"`
	Defaults        TemplatePayload    `json:"defaults" gorm:"type:jsonb"`
	AvatarURI       string             `json:"avatar_uri" gorm:"type:varchar"`
	TemplateResources
}

// TODO - this is in DIRE need of tests
func (t *Template) Diff(other Template) bool {
	if t.TemplateName != other.TemplateName {
		return true
	}
	if t.CommandTemplate != other.CommandTemplate {
		return true
	}
	if t.Image != other.Image {
		return true
	}
	if t.Memory != nil && other.Memory != nil && *t.Memory != *other.Memory {
		return true
	}
	if t.Gpu != nil && other.Gpu != nil && *t.Gpu != *other.Gpu {
		return true
	}

	if t.Cpu != nil && other.Cpu != nil && *t.Cpu != *other.Cpu {
		return true
	}

	if t.Env != nil && other.Env != nil {
		tEnv := *t.Env
		otherEnv := *other.Env
		if len(tEnv) != len(otherEnv) {
			return true
		}

		for i, e := range tEnv {
			if e != otherEnv[i] {
				return true
			}
		}
	}

	if reflect.DeepEqual(t.Defaults, other.Defaults) == false {
		if len(t.Defaults) != len(other.Defaults) && len(t.Defaults) > 0 {
			return true
		}
	}

	if t.AvatarURI != other.AvatarURI {
		return true
	}

	if t.Ports != nil && other.Ports != nil {
		tPorts := *t.Ports
		otherPorts := *other.Ports
		if len(tPorts) != len(otherPorts) {
			return true
		}

		for i, e := range tPorts {
			if e != otherPorts[i] {
				return true
			}
		}
	}

	if t.Tags != nil && other.Tags != nil {
		tTags := *t.Tags
		otherTags := *other.Tags
		if len(tTags) != len(otherTags) {
			return true
		}

		for i, e := range tTags {
			if e != otherTags[i] {
				return true
			}
		}
	}

	if reflect.DeepEqual(t.Schema, other.Schema) == false {
		return true
	}

	return false
}

func (t Template) MergeWithDefaults(userPayload map[string]interface{}) (TemplatePayload, error) {
	err := utils.MergeMaps(&userPayload, t.Defaults)
	return userPayload, err
}

func (t *Template) IsValid() (bool, []string) {
	conditions := []struct {
		condition bool
		reason    string
	}{
		{len(t.TemplateName) == 0, "string [template_name] must be specified"},
		{len(t.Schema) == 0, "schema must be specified"},
		{len(t.CommandTemplate) == 0, "string [command_template] must be specified"},
		{len(t.Image) == 0, "string [image] must be specified"},
		{t.Memory == nil, "int [memory] must be specified"},
	}

	valid := true
	var reasons []string
	for _, cond := range conditions {
		if cond.condition {
			valid = false
			reasons = append(reasons, cond.reason)
		}
	}
	return valid, reasons
}

func (t *Template) BeforeCreate(tx *gorm.DB) (err error) {
	t.TemplateID = NewTemplateID()

	if t.Memory == nil {
		t.Memory = &MinMem
	}

	if t.Cpu == nil {
		t.Cpu = &MinCPU
	}

	if t.Defaults == nil {
		t.Defaults = make(map[string]interface{})
	}
	return
}

type TemplateJSONSchema map[string]interface{}

func (tjs *TemplateJSONSchema) Scan(value interface{}) error {
	if value != nil {
		s := value.([]uint8)
		json.Unmarshal(s, &tjs)
	}
	return nil
}

// Value to db
func (tjs TemplateJSONSchema) Value() (driver.Value, error) {
	res, _ := json.Marshal(tjs)
	return res, nil
}

type TemplatePayload map[string]interface{}

func (tjs *TemplatePayload) Scan(value interface{}) error {
	if value != nil {
		s := value.([]uint8)
		json.Unmarshal(s, &tjs)
	}
	return nil
}

// Value to db
func (tjs TemplatePayload) Value() (driver.Value, error) {
	res, _ := json.Marshal(tjs)
	return res, nil
}

type TemplateResources struct {
	Image                      string     `json:"image" type:"index,varchar"`
	Memory                     *int64     `json:"memory,omitempty"`
	Gpu                        *int64     `json:"gpu,omitempty"`
	Cpu                        *int64     `json:"cpu,omitempty"`
	Env                        *EnvList   `json:"env" gorm:"type:jsonb"`
	AdaptiveResourceAllocation *bool      `json:"adaptive_resource_allocation,omitempty"`
	Ports                      *PortsList `json:"ports,omitempty" gorm:"type:jsonb"`
	Tags                       *Tags      `json:"tags,omitempty" gorm:"type:jsonb"`
}

type TemplateList struct {
	Total     int64      `json:"total"`
	Templates []Template `json:"templates"`
}
