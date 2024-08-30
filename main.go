package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/bundle"
	"sigs.k8s.io/yaml"
)

// TODO: Must handle different versions of Pipelines and Tasks (just v1 and v1beta1 probably)
// TODO: Must also handle verifying a PipelineRun.
// TODO: Get rid of all the os.Exit calls.
// TODO: Check finally Tasks as well.

func main() {
	// TODO: Make this a proper CLI
	fname := os.Args[1]
	f, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	// TODO: Check that it is indeed a Pipeline before proceeding.

	var p v1.Pipeline
	if err := yaml.Unmarshal(f, &p); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := p.Validate(ctx); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
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
				fmt.Printf("ERROR: %s\n", err)
				os.Exit(1)
			}
			// TODO: Use local credentials
			var keychain authn.Keychain
			resolvedResource, err := bundle.GetEntry(ctx, keychain, opts)
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
				os.Exit(1)
			}

			var t v1.Task
			if err := yaml.Unmarshal(resolvedResource.Data(), &t); err != nil {
				fmt.Printf("ERROR: %s\n", err)
				os.Exit(1)
			}

			if err := validatePipelineTaskParameters(pipelineTask.Params, t.Spec.Params); err != nil {
				fmt.Printf("ERROR: %s PipelineTask: %s", pipelineTask.Name, err)
				os.Exit(1)
			}

			// TODO: Validate workspaces.

			// TODO: Remove this obviously...
			// break
		}
	}

	fmt.Println("Success \\o/")
}

func validatePipelineTaskParameters(pipelineTaskParams []v1.Param, taskParams []v1.ParamSpec) error {
	var result error
	for _, pipelineTaskParam := range pipelineTaskParams {
		taskParam, found := getTaskParam(pipelineTaskParam.Name, taskParams)
		if !found {
			result = multierror.Append(result, fmt.Errorf(
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
			result = multierror.Append(result, fmt.Errorf(
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
			result = multierror.Append(result, fmt.Errorf("%q parameter is required", taskParam.Name))
		}
	}

	return result
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
