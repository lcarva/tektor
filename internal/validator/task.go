package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func ValidateTaskV1(ctx context.Context, t v1.Task) error {
	if err := t.Validate(ctx); err != nil {
		var allErrors error
		for _, e := range err.WrappedErrors() {
			details := e.Details
			if len(details) > 0 {
				details = " " + details
			}
			message := strings.TrimSuffix(e.Message, ": ")
			for _, p := range e.Paths {
				allErrors = multierror.Append(allErrors, fmt.Errorf("%v: %v%v", message, p, details))
			}
			if len(e.Paths) == 0 {
				allErrors = multierror.Append(allErrors, fmt.Errorf("%v: %v", message, details))
			}
		}
		return allErrors
	}

	return nil
}

func ValidateTaskV1Beta1(ctx context.Context, t v1beta1.Task) error {
	if err := t.Validate(ctx); err != nil {
		var allErrors error
		for _, e := range err.WrappedErrors() {
			details := e.Details
			if len(details) > 0 {
				details = " " + details
			}
			message := strings.TrimSuffix(e.Message, ": ")
			for _, p := range e.Paths {
				allErrors = multierror.Append(allErrors, fmt.Errorf("%v: %v%v", message, p, details))
			}
			if len(e.Paths) == 0 {
				allErrors = multierror.Append(allErrors, fmt.Errorf("%v: %v", message, details))
			}
		}
		return allErrors
	}

	return nil
}
