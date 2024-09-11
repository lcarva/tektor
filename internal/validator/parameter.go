package validator

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func ValidateParameters(params v1.Params, specs v1.ParamSpecs) error {
	return validatePipelineTaskParameters(params, specs)
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
