package validator

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/hashicorp/go-multierror"
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
		if pipelineTask.TaskRef != nil && pipelineTask.TaskRef.Resolver == "bundles" {
			var params []v1.Param
			params = append(params, pipelineTask.TaskRef.Params...)
			// TODO: Do this only if the SA param is not set.
			params = append(params, v1.Param{Name: bundle.ParamServiceAccount, Value: *v1.NewStructuredValues("none")})
			// ParamServiceAccount
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

			if err := validatePipelineTaskParameters(pipelineTask.Params, t.Spec.Params); err != nil {
				return fmt.Errorf("ERROR: %s PipelineTask: %s", pipelineTask.Name, err)
			}

			// TODO: Validate workspaces.

			// TODO: Remove this obviously...
			// break
		}
	}

	return nil
}

func validatePipelineTaskParameters(pipelineTaskParams []v1.Param, taskParams []v1.ParamSpec) error {
	var err error
	for _, pipelineTaskParam := range pipelineTaskParams {
		taskParam, found := getTaskParam(pipelineTaskParam.Name, taskParams)
		if !found {
			err = multierror.Append(err, fmt.Errorf(
				"%q parameter is not defined by the Task",
				pipelineTaskParam.Name))
			continue
		}

		// Tekton uses the "string" type for parameters by default.
		taskParamType := string(taskParam.Type)
		if taskParamType == "" {
			taskParamType = "string"
		}
		pipelineTaskParamType := string(pipelineTaskParam.Value.Type)
		if pipelineTaskParamType == "" {
			pipelineTaskParamType = "string"
		}

		if pipelineTaskParamType != taskParamType {
			err = multierror.Append(err, fmt.Errorf(
				"%q parameter has the incorrect type, got %q, want %q",
				pipelineTaskParam.Name, pipelineTaskParamType, taskParamType))
		}
	}

	// Verify all "required" parameters are fulfilled.
	for _, taskParam := range taskParams {
		if taskParam.Default != nil {
			// Task parameters with a default value are not required.
			continue
		}
		if _, found := getPipelineTaskParam(taskParam.Name, pipelineTaskParams); !found {
			err = multierror.Append(err, fmt.Errorf("%q parameter is required", taskParam.Name))
		}
	}

	return err
}

func getPipelineTaskParam(name string, pipelineTaskParams []v1.Param) (v1.Param, bool) {
	for _, pipelineTaskParam := range pipelineTaskParams {
		if pipelineTaskParam.Name == name {
			return pipelineTaskParam, true
		}
	}
	return v1.Param{}, false
}

func getTaskParam(name string, taskParams []v1.ParamSpec) (v1.ParamSpec, bool) {
	for _, taskParam := range taskParams {
		if taskParam.Name == name {
			return taskParam, true
		}
	}
	return v1.ParamSpec{}, false
}
