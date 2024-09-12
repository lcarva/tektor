package validator

import (
	"fmt"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func ValidateResults(resultRefs []*v1.ResultRef, allTaskResults map[string][]v1.TaskResult) error {
	for _, resultRef := range resultRefs {
		results, found := allTaskResults[resultRef.PipelineTask]
		if !found {
			return fmt.Errorf("%s result from non-existent %s PipelineTask", resultRef.Result, resultRef.PipelineTask)
		}
		var result *v1.TaskResult
		for _, r := range results {
			if r.Name == resultRef.Result {
				result = &r
				break
			}
		}
		if result == nil {
			return fmt.Errorf("non-existent %s result from %s PipelineTask", resultRef.Result, resultRef.PipelineTask)
		}
		// TODO: Validate type
	}
	return nil
}
