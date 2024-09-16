package validate

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift-pipelines/pipelines-as-code/pkg/cli"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/cmd/tknpac/resolve"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/params"
	"github.com/spf13/cobra"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/lcarva/tektor/internal/validator"
)

var ValidateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "Validate a Tekton resource",
	Example: "tekton validate /tmp/pipeline.yaml",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd.Context(), args[0])
	},
}

func run(ctx context.Context, fname string) error {
	f, err := os.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("reading %s: %w", fname, err)
	}

	var m metav1.TypeMeta
	if err := yaml.Unmarshal(f, &m); err != nil {
		return fmt.Errorf("unmarshaling %s as k8s resource: %w", fname, err)
	}

	key := fmt.Sprintf("%s/%s", m.APIVersion, m.Kind)
	switch key {
	case "tekton.dev/v1/Pipeline":
		var p v1.Pipeline
		if err := yaml.Unmarshal(f, &p); err != nil {
			return fmt.Errorf("unmarshalling %s as %s: %w", fname, key, err)
		}
		if err := validator.ValidatePipeline(ctx, p); err != nil {
			return err
		}
	case "tekton.dev/v1/PipelineRun":
		var pr v1.PipelineRun
		if err := yaml.Unmarshal(f, &pr); err != nil {
			return fmt.Errorf("unmarshalling %s as %s: %w", fname, key, err)
		}

		// TODO: Run it through PaC. Similar to:
		// 	tkn pac resolve -f <input> --no-generate-name -o <output>
		// Use the resolved file going forward.
		clients := params.New()
		ioStreams := cli.NewIOStreams()
		resolve.Command(clients, ioStreams)

		if err := validator.ValidatePipelineRun(ctx, pr); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s is not supported", key)
	}

	return nil
}
