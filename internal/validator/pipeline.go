package validator

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/bundle"
	"sigs.k8s.io/yaml"
)

func ValidatePipeline(ctx context.Context, p v1.Pipeline) error {

	if err := p.Validate(ctx); err != nil {
		// TODO: These errors are quite cryptic. Find a way to make them nicer.
		return err
	}

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
		}
		// TODO: Add support for other resolvers and embedded task definitions.

		if err := ValidateParameters(params, paramSpecs); err != nil {
			return fmt.Errorf("ERROR: %s PipelineTask: %s", pipelineTask.Name, err)
		}

		// TODO: Validate workspaces.
		// TODO: Validate params and results usage.
	}

	return nil
}
