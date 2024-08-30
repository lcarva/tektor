package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lcarva/tekton-lint/internal/validator"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

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

	var m metav1.TypeMeta
	if err := yaml.Unmarshal(f, &m); err != nil {
		fmt.Printf("ERROR: cannot read file as a k8s resource: %s", err)
		os.Exit(1)
	}

	ctx := context.Background()

	key := fmt.Sprintf("%s/%s", m.APIVersion, m.Kind)
	switch key {
	case "tekton.dev/v1/Pipeline":
		var p v1.Pipeline
		if err := yaml.Unmarshal(f, &p); err != nil {
			fmt.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
		if err := validator.ValidatePipeline(ctx, p); err != nil {
			// TODO: Print to stderr
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		fmt.Printf("ERROR: %s is not supported", key)
		os.Exit(1)
	}

	fmt.Println("Success \\o/")
}
