package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ValidatePipelineRun(ctx context.Context, pr v1.PipelineRun) error {
	if err := pr.Validate(ctx); err != nil {
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

	if pipelineSpec := pr.Spec.PipelineSpec; pipelineSpec != nil {
		p := v1.Pipeline{
			// Some name value is required for validation.
			ObjectMeta: metav1.ObjectMeta{Name: "noname"},
			Spec:       *pipelineSpec,
		}
		if err := ValidatePipeline(ctx, p); err != nil {
			return err
		}
	}
	return nil
}
