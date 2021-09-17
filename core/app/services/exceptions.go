package services

import (
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/state"
)

func MissingTemplateError(args *state.GetTemplateArgs) error {
	var identifier string
	if args.TemplateID != nil {
		identifier = fmt.Sprintf("id: %s", *args.TemplateID)
	} else {
		identifier = fmt.Sprintf("name: %s and version: %d", *args.TemplateName, *args.TemplateVersion)
	}
	return fmt.Errorf("%w: template not found for %s", exceptions.ErrRecordNotFound, identifier)
}

func TemplateValidationError(reasons string) error {
	return fmt.Errorf("%w: validation failed, reasons: [%s]", exceptions.ErrMalformedInput, reasons)
}

func MissingRunError(runID string) error {
	return fmt.Errorf("%w: run with id: %s not found", exceptions.ErrRecordNotFound, runID)
}

func EngineNotConfigured(engineName string) error {
	return fmt.Errorf("%w: engine with name: %s not configured", exceptions.ErrMalformedInput, engineName)
}
