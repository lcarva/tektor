package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/bundle"
	"sigs.k8s.io/yaml"
)

func ValidatePipeline(ctx context.Context, p v1.Pipeline) error {

	if err := p.Validate(ctx); err != nil {
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

	allTaskResults := map[string][]v1.TaskResult{}
	allTaskResultRefs := map[string][]*v1.ResultRef{}

	// TODO: Check finally Tasks as well.
	for i, pipelineTask := range p.Spec.Tasks {
		fmt.Printf("%d: %s\n", i, pipelineTask.Name)
		params := pipelineTask.Params
		var paramSpecs v1.ParamSpecs

		if pipelineTask.TaskRef != nil && pipelineTask.TaskRef.Resolver == "bundles" {
			var params []v1.Param
			params = append(params, pipelineTask.TaskRef.Params...)
			// TODO: Do this only if the SA param is not set.
			params = append(params, v1.Param{Name: bundle.ParamServiceAccount, Value: *v1.NewStructuredValues("none")})
			opts, err := bundle.OptionsFromParams(ctx, params)
			if err != nil {
				return err
			}
			// TODO: Use local credentials
			var keychain authn.Keychain
			resolvedResource, err := bundle.GetEntry(ctx, keychain, opts)
			if err != nil {
				return err
			}

			var t v1.Task
			if err := yaml.Unmarshal(resolvedResource.Data(), &t); err != nil {
				return err
			}

			paramSpecs = t.Spec.Params

			allTaskResults[pipelineTask.Name] = t.Spec.Results
			allTaskResultRefs[pipelineTask.Name] = v1.PipelineTaskResultRefs(&pipelineTask)

		}
		// TODO: Add support for other resolvers and embedded task definitions.

		if err := ValidateParameters(params, paramSpecs); err != nil {
			return fmt.Errorf("ERROR: %s PipelineTask: %s", pipelineTask.Name, err)
		}

		// TODO: Validate workspaces.
	}

	// Verify result references in PipelineTasks are valid.
	for pipelineTaskName, resultRefs := range allTaskResultRefs {
		if err := ValidateResults(resultRefs, allTaskResults); err != nil {
			return fmt.Errorf("%s PipelineTask results: %w", pipelineTaskName, err)
		}
	}

	// Verify result references in Pipeline are valid.
	for _, pipelineResult := range p.Spec.Results {
		expressions, _ := pipelineResult.GetVarSubstitutionExpressions()
		resultRefs := v1.NewResultRefs(expressions)
		if err := ValidateResults(resultRefs, allTaskResults); err != nil {
			return fmt.Errorf("pipeline results: %w", err)
		}
	}

	return nil
}
