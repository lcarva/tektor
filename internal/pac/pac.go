package pac

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/openshift-pipelines/pipelines-as-code/pkg/formatting"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/git"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/params"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/params/info"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/params/settings"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/provider/github"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/resolve"
	"github.com/openshift-pipelines/pipelines-as-code/pkg/templates"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
	"sigs.k8s.io/yaml"
)

/*
This whole package is a huge hack. It is basically trying to implement the behavior when running:
	tkn pac resolve -f <input> --no-generate-name -o <output>
This inlines Task definitions from other files in the repo into the PipelineRun.
The implementation in github.com/openshift-pipelines/pipelines-as-code is not done in a share-friendly
manner. As such, the majority of the code here was copied and pasted from that repo.
*/

func ResolvePipelineRun(ctx context.Context, fname string, prName string) ([]byte, error) {
	run := params.New()
	errc := run.Clients.NewClients(ctx, &run.Info)
	zaplog, err := zap.NewProduction(
		zap.IncreaseLevel(zap.FatalLevel),
	)
	if err != nil {
		return nil, err
	}
	run.Clients.Log = zaplog.Sugar()

	if errc != nil {
		// Allow resolve to be run without a kubeconfig
		noConfigErr := strings.Contains(errc.Error(), "Couldn't get kubeConfiguration namespace")
		if !noConfigErr {
			return nil, errc
		}
	} else {
		// It's OK  if pac is not installed, ignore the error
		_ = run.UpdatePACInfo(ctx)
	}

	pacConfig := map[string]string{}
	if err := settings.ConfigToSettings(run.Clients.Log, run.Info.Pac.Settings, pacConfig); err != nil {
		return nil, err
	}

	params := map[string]string{}

	gitinfo := git.GetGitInfo(path.Dir(fname))
	if gitinfo.SHA != "" {
		params["revision"] = gitinfo.SHA
	}
	if gitinfo.URL != "" {
		params["repo_url"] = gitinfo.URL
		repoOwner, err := formatting.GetRepoOwnerFromURL(gitinfo.URL)
		if err != nil {
			return nil, fmt.Errorf("getting git repo owner: %w", err)
		}
		params["repo_owner"] = strings.Split(repoOwner, "/")[0]
		params["repo_name"] = strings.Split(repoOwner, "/")[1]
	}

	pacDir := path.Join(gitinfo.TopLevelPath, ".tekton")
	allTemplates := templates.ReplacePlaceHoldersVariables(enumerateFiles([]string{pacDir}), params)

	// We use github here but since we don't do remotetask we would not care
	providerintf := github.New()
	event := info.NewEvent()
	// Must change working dir to git repo so local fs resolver works
	if gitinfo.TopLevelPath != "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current working directory: %w", err)
		}
		if err := os.Chdir(gitinfo.TopLevelPath); err != nil {
			return nil, fmt.Errorf("changing working directory: %w", err)
		}
		defer func(wd string) {
			_ = os.Chdir(wd)
		}(wd)
	}

	ropt := &resolve.Opts{RemoteTasks: true}
	prs, err := resolve.Resolve(ctx, run, run.Clients.Log, providerintf, event, allTemplates, ropt)
	if err != nil {
		return nil, err
	}
	var pr *v1.PipelineRun
	for _, somePR := range prs {
		if somePR.Name == prName {
			pr = somePR
			break
		}
	}
	if pr == nil {
		return nil, fmt.Errorf("unable to find %q pipelinerun after pac resolution", prName)
	}

	pr.APIVersion = v1.SchemeGroupVersion.String()
	pr.Kind = "PipelineRun"
	d, err := yaml.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("marshaling pac resolved pipelinerun: %w", err)
	}

	return cleanRe.ReplaceAll(d, []byte("\n")), nil
}

// cleanedup regexp do as much as we can but really it's a lost game to try this
var cleanRe = regexp.MustCompile(`\n(\t|\s)*(creationTimestamp|spec|taskRunTemplate|metadata|computeResources):\s*(null|{})\n`)

func enumerateFiles(filenames []string) string {
	var yamlDoc string
	for _, paths := range filenames {
		if stat, err := os.Stat(paths); err == nil && !stat.IsDir() {
			yamlDoc += appendYaml(paths)
			continue
		}

		// walk dir getting all yamls
		err := filepath.Walk(paths, func(path string, fi os.FileInfo, err error) error {
			if filepath.Ext(path) == ".yaml" {
				yamlDoc += appendYaml(path)
			}
			return nil
		})
		if err != nil {
			log.Fatalf("Error enumerating files: %v", err)
		}
	}

	return yamlDoc
}

func appendYaml(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	s := string(b)
	if strings.HasPrefix(s, "---") {
		return s
	}
	return fmt.Sprintf("---\n%s", s)
}
