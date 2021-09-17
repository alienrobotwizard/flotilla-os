package services

import (
	"context"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"strings"
)

type TemplateService interface {
	GetTemplate(ctx context.Context, args *state.GetTemplateArgs) (models.Template, error)
	ListTemplates(ctx context.Context, args *state.ListArgs) (models.TemplateList, error)
	CreateTemplate(ctx context.Context, t models.Template) (models.Template, bool, error)
}

type templateService struct {
	sm state.Manager
}

func NewTemplateService(c *config.Config, sm state.Manager) (TemplateService, error) {
	return &templateService{sm: sm}, nil
}

func (ts *templateService) GetTemplate(ctx context.Context, args *state.GetTemplateArgs) (models.Template, error) {
	tmpl, err := ts.sm.GetTemplate(ctx, args)
	if err != nil && errors.Is(err, exceptions.ErrRecordNotFound) {
		err = MissingTemplateError(args)
	}
	return tmpl, err
}

func (ts *templateService) ListTemplates(ctx context.Context, args *state.ListArgs) (models.TemplateList, error) {
	return ts.sm.ListTemplates(ctx, args)
}

func (ts *templateService) CreateTemplate(ctx context.Context, t models.Template) (models.Template, bool, error) {
	if valid, reasons := t.IsValid(); !valid {
		return t, false, TemplateValidationError(strings.Join(reasons, "\n"))
	}

	limit := 1
	sortBy := "version"
	order := "desc"
	args := &state.ListArgs{Limit: &limit, SortBy: &sortBy, Order: &order}
	args.AddFilter("template_name", t.TemplateName)

	var (
		exists   bool
		existing models.Template
	)
	if tl, err := ts.sm.ListTemplates(ctx, args); err != nil {
		return existing, false, err
	} else {
		if tl.Total > 0 {
			exists, existing = true, tl.Templates[0]
		}
	}

	if !exists {
		t.Version = 1
		tmpl, err := ts.sm.CreateTemplate(ctx, t)
		return tmpl, true, err
	} else {
		if existing.Diff(t) {
			t.Version = existing.Version + 1
			tmpl, err := ts.sm.CreateTemplate(ctx, t)
			return tmpl, false, err
		} else {
			return existing, false, nil
		}
	}
}
