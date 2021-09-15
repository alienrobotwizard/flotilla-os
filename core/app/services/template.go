package services

import (
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"strings"
)

type TemplateService interface {
	GetTemplate(args *state.GetTemplateArgs) (models.Template, error)
	ListTemplates(args *state.ListArgs) (models.TemplateList, error)
	CreateTemplate(t models.Template) (models.Template, error)
}

type templateService struct {
	sm state.Manager
}

func NewTemplateService(c *config.Config, sm state.Manager) (TemplateService, error) {
	return &templateService{sm: sm}, nil
}

func (ts *templateService) GetTemplate(args *state.GetTemplateArgs) (models.Template, error) {
	return ts.sm.GetTemplate(args)
}

func (ts *templateService) ListTemplates(args *state.ListArgs) (models.TemplateList, error) {
	return ts.sm.ListTemplates(args)
}

func (ts *templateService) CreateTemplate(t models.Template) (models.Template, error) {
	if valid, reasons := t.IsValid(); !valid {
		return t, exceptions.MalformedInput{ErrorString: strings.Join(reasons, "\n")}
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
	if tl, err := ts.sm.ListTemplates(args); err != nil {
		return existing, err
	} else {
		if tl.Total > 0 {
			exists, existing = true, tl.Templates[0]
		}
	}

	if !exists {
		t.Version = 1
		return ts.sm.CreateTemplate(t)
	} else {
		if existing.Diff(t) {
			t.Version = existing.Version + 1
			return ts.sm.CreateTemplate(t)
		} else {
			return existing, nil
		}
	}
}
