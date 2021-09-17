package services

import (
	"bytes"
	"context"
	"github.com/Masterminds/sprig"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
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

func (er *ExecutionRequest) SetTemplateVersion(version int64) {
	er.TemplateVersion = &version
}

func (er *ExecutionRequest) SetCommand(cmd string) {
	er.Command = &cmd
}

func (er *ExecutionRequest) HasCommand() bool {
	return er.Command != nil && len(*er.Command) > 0
}

type ExecutionService interface {
	ListRuns(ctx context.Context, args *state.ListRunsArgs) (models.RunList, error)
	GetRun(ctx context.Context, runID string) (models.Run, error)
	Logs(ctx context.Context, runID string, lastSeen *string) (string, *string, error)
	Terminate(ctx context.Context, runID string) error
	CreateTemplateRun(ctx context.Context, args *ExecutionRequest) (models.Run, error)
}

func NewExecutionService(
	c *config.Config, sm state.Manager, executionEngines engines.Engines) (ExecutionService, error) {
	return &executionService{
		sm:   sm,
		engs: executionEngines,
	}, nil
}

type executionService struct {
	sm   state.Manager
	engs engines.Engines
}

func (es *executionService) ListRuns(ctx context.Context, args *state.ListRunsArgs) (models.RunList, error) {
	return es.sm.ListRuns(ctx, args)
}

func (es *executionService) GetRun(ctx context.Context, runID string) (models.Run, error) {
	run, err := es.sm.GetRun(ctx, runID)
	if err != nil && errors.Is(err, exceptions.ErrRecordNotFound) {
		err = MissingRunError(runID)
	}
	return run, err
}

func (es *executionService) Logs(ctx context.Context, runID string, lastSeen *string) (string, *string, error) {
	if run, err := es.GetRun(ctx, runID); err != nil {
		return "", nil, err
	} else {
		args := &state.GetTemplateArgs{TemplateID: run.TemplateID}
		tmpl, err := es.sm.GetTemplate(ctx, args)
		if err != nil {
			if errors.Is(err, exceptions.ErrRecordNotFound) {
				err = MissingTemplateError(args)
			}
			return "", nil, err
		}

		if engine, err := es.engineForRun(run); err != nil {
			return "", nil, err
		} else {
			return engine.Logs(tmpl, run, lastSeen)
		}
	}
}

func (es *executionService) Terminate(ctx context.Context, runID string) error {
	if run, err := es.GetRun(ctx, runID); err != nil {
		return err
	} else {
		if engine, err := es.engineForRun(run); err != nil {
			return err
		} else {
			return engine.Terminate(run)
		}
	}
}

func (es *executionService) CreateTemplateRun(ctx context.Context, args *ExecutionRequest) (run models.Run, err error) {
	gta := &state.GetTemplateArgs{
		TemplateID:      args.TemplateID,
		TemplateName:    args.TemplateName,
		TemplateVersion: args.TemplateVersion,
	}
	tmpl, err := es.sm.GetTemplate(ctx, gta)

	if err != nil {
		if errors.Is(err, exceptions.ErrRecordNotFound) {
			return run, MissingTemplateError(gta)
		}
		return run, err
	}

	// 1. Construct run from template
	if cmd, err := es.renderCommand(tmpl, args); err != nil {
		// TODO - likely malformed payload, needs to return as such
		return run, err
	} else {
		if !args.HasCommand() && len(cmd) > 0 {
			args.SetCommand(cmd)
		}
	}

	run = models.Run{
		Image:                 tmpl.Image,
		Status:                models.StatusQueued,
		Command:               args.Command,
		Memory:                args.Memory,
		Cpu:                   args.CPU,
		Gpu:                   args.GPU,
		Engine:                args.Engine,
		TemplateID:            &tmpl.TemplateID,
		ActiveDeadlineSeconds: args.ActiveDeadlineSeconds,
	}

	engine, err := es.engineForRun(run)
	if err != nil {
		return run, err
	}

	if run, err := es.sm.CreateRun(ctx, run); err != nil {
		return run, err
	} else {
		if err = engine.Enqueue(run); err != nil {
			return run, err
		}
		queued := time.Now()
		return es.sm.UpdateRun(ctx, run.RunID, models.Run{QueuedAt: &queued})
	}
}

func (es *executionService) engineForRun(run models.Run) (engines.Engine, error) {
	engineName := "local"
	if run.Engine != nil {
		engineName = *run.Engine
	}

	if engine, ok := es.engs.Get(engineName); !ok {
		return nil, EngineNotConfigured(engineName)
	} else {
		return engine, nil
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
