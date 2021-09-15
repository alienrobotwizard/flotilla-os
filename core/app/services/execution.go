package services

import (
	"bytes"
	"github.com/Masterminds/sprig"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
	"strings"
	"text/template"
	"time"
)

type ExecutionRequest struct {
	TemplateID            *string                `json:"template_id,omitempty"`
	TemplateName          *string                `json:"template_name,omitempty"`
	TemplateVersion       *int64                 `json:"template_version,omitempty"`
	Env                   *models.EnvList        `json:"env"`
	OwnerID               string                 `json:"owner_id"`
	Command               *string                `json:"command"`
	Memory                *int64                 `json:"memory"`
	CPU                   *int64                 `json:"cpu"`
	GPU                   *int64                 `json:"gpu"`
	Engine                *string                `json:"engine"`
	ActiveDeadlineSeconds *int64                 `json:"active_deadline_seconds,omitempty"`
	TemplatePayload       map[string]interface{} `json:"template_payload"`
	DryRun                *bool                  `json:"dry_run,omitempty"`
}

func (er *ExecutionRequest) SetCommand(cmd string) {
	er.Command = &cmd
}

func (er *ExecutionRequest) HasCommand() bool {
	return er.Command != nil && len(*er.Command) > 0
}

type ExecutionService interface {
	ListRuns(args *state.ListRunsArgs) (models.RunList, error)
	GetRun(runID string) (models.Run, error)
	Terminate(runID string) error
	CreateTemplateRun(args *ExecutionRequest) (models.Run, error)
}

func NewExecutionService(c *config.Config, sm state.Manager, engine engines.Engine) (ExecutionService, error) {
	return &executionService{
		sm:     sm,
		engine: engine,
	}, nil
}

type executionService struct {
	sm     state.Manager
	engine engines.Engine
}

func (es *executionService) ListRuns(args *state.ListRunsArgs) (models.RunList, error) {
	return es.sm.ListRuns(args)
}

func (es *executionService) GetRun(runID string) (models.Run, error) {
	return es.sm.GetRun(runID)
}

func (es *executionService) Terminate(runID string) error {
	if run, err := es.GetRun(runID); err != nil {
		return err
	} else {
		return es.engine.Terminate(run)
	}
}

func (es *executionService) CreateTemplateRun(args *ExecutionRequest) (run models.Run, err error) {
	template, err := es.sm.GetTemplate(&state.GetTemplateArgs{
		TemplateID:      args.TemplateID,
		TemplateName:    args.TemplateName,
		TemplateVersion: args.TemplateVersion,
	})
	if err != nil {
		return
	}

	// 1. Construct run from template
	if cmd, err := es.renderCommand(template, args); err != nil {
		// TODO - likely malformed payload, needs to return as such
		return run, err
	} else {
		if !args.HasCommand() && len(cmd) > 0 {
			args.SetCommand(cmd)
		}
	}

	run = models.Run{
		Image:                 template.Image,
		Status:                models.StatusQueued,
		Command:               args.Command,
		Memory:                args.Memory,
		Cpu:                   args.CPU,
		Gpu:                   args.GPU,
		Engine:                args.Engine,
		TemplateID:            &template.TemplateID,
		ActiveDeadlineSeconds: args.ActiveDeadlineSeconds,
	}

	if run, err := es.sm.CreateRun(run); err != nil {
		return run, err
	} else {
		if err = es.engine.Enqueue(run); err != nil {
			return run, err
		}
		queued := time.Now()
		return es.sm.UpdateRun(run.RunID, models.Run{QueuedAt: &queued})
	}
}

func (es *executionService) renderCommand(t models.Template, args *ExecutionRequest) (string, error) {
	var result bytes.Buffer

	executionPayload, err := t.MergeWithDefaults(args.TemplatePayload)

	schemaLoader := gojsonschema.NewGoLoader(t.Schema)
	documentLoader := gojsonschema.NewGoLoader(executionPayload)

	// Perform JSON schema validation to ensure that the request's template
	// payload conforms to the template's JSON schema.
	validationResult, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return "", err
	}
	if validationResult != nil && validationResult.Valid() != true {
		var res []string
		for _, resultError := range validationResult.Errors() {
			res = append(res, resultError.String())
		}
		return "", errors.New(strings.Join(res, "\n"))
	}

	// Create a new template string based on the template.Template.
	textTemplate, err := template.New("command").Funcs(sprig.TxtFuncMap()).Parse(t.CommandTemplate)
	if err != nil {
		return "", err
	}

	// Dump payload into the template string.
	if err = textTemplate.Execute(&result, executionPayload); err != nil {
		return "", err
	}

	return result.String(), nil
}
