package diff

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	helmspec "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/helm/chart-testing/v3/pkg/config"
	"github.com/helm/chart-testing/v3/pkg/exec"
	"github.com/helm/chart-testing/v3/pkg/tool"
	"github.com/helm/chart-testing/v3/pkg/util"
	"github.com/kylelemons/godebug/diff"
	log "github.com/sirupsen/logrus"
	fttool "github.com/stgrace/ft/pkg/tool"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

type CmdExecutor interface {
	RunCommand(cmdTemplate string, data interface{}) error
}

type Kubectl interface {
	CreateNamespace(namespace string) error
	DeleteNamespace(namespace string)
	WaitForDeployments(namespace string, selector string) error
	GetPodsforDeployment(namespace string, deployment string) ([]string, error)
	GetPods(args ...string) ([]string, error)
	GetEvents(namespace string) error
	DescribePod(namespace string, pod string) error
	Logs(namespace string, pod string, container string) error
	GetInitContainers(namespace string, pod string) ([]string, error)
	GetContainers(namespace string, pod string) ([]string, error)
}

type Git interface {
	FileExistsOnBranch(file string, remote string, branch string) bool
	Show(file string, remote string, branch string) (string, error)
	AddWorktree(path string, ref string) error
	RemoveWorktree(path string) error
	MergeBase(commit1 string, commit2 string) (string, error)
	ListChangedFiles(commit string) ([]string, error)
	GetURLForRemote(remote string) (string, error)
	ValidateRepository() error
}

type Helm interface {
	AddRepo(name string, url string, extraArgs []string) error
	Version() (string, error)
	TemplateWithValues(string, *v1.JSON, string, string, string) (string, error)
}

// DirectoryLister is the interface
//
// ListChildDirs lists direct child directories of parentDir given they pass the test function
type DirectoryLister interface {
	ListChildDirs(parentDir string, test func(string) bool) ([]string, error)
}
type Utils interface {
	LookupChartDir(chartDirs []string, dir string) (string, error)
}

type Testing struct {
	config                   config.Configuration
	helm                     Helm
	kubectl                  Kubectl
	git                      Git
	cmdExecutor              CmdExecutor
	accountValidator         AccountValidator
	directoryLister          DirectoryLister
	utils                    Utils
	previousRevisionWorktree string
}

// AccountValidator is the interface that wraps Git account validation
//
// Validate checks if account is valid on repoDomain
type AccountValidator interface {
	Validate(repoDomain string, account string) error
}

type DiffResult struct {
	NewHelmRelease HelmRelease
	OldHelmRelease HelmRelease
	Diff           string
}

type HelmRelease struct {
	HelmRelease *helmspec.HelmRelease
	Path        string
}

// NewTesting creates a new Testing struct with the given config.
func NewTesting(config config.Configuration, extraSetArgs string) (Testing, error) {
	procExec := exec.NewProcessExecutor(config.Debug)
	extraArgs := strings.Fields(config.HelmExtraArgs)

	testing := Testing{
		config:           config,
		helm:             fttool.NewHelm(procExec, extraArgs, strings.Fields(extraSetArgs)),
		git:              fttool.NewGit(procExec),
		kubectl:          tool.NewKubectl(procExec, config.KubectlTimeout),
		cmdExecutor:      tool.NewCmdTemplateExecutor(procExec),
		accountValidator: tool.AccountValidator{},
		directoryLister:  util.DirectoryLister{},
		utils:            util.Utils{},
	}

	versionString, err := testing.helm.Version()
	if err != nil {
		return testing, err
	}

	version, err := semver.NewVersion(versionString)
	if err != nil {
		return testing, err
	}

	if version.Major() < 3 {
		return testing, fmt.Errorf("minimum required Helm version is v3.0.0; found: %s", version)
	}
	return testing, nil
}

func (t *Testing) computeMergeBase() (string, error) {
	err := t.git.ValidateRepository()
	if err != nil {
		return "", errors.New("must be in a git repository")
	}

	return t.git.MergeBase(fmt.Sprintf("%s/%s", t.config.Remote, t.config.TargetBranch), t.config.Since)
}

func (t *Testing) parseHelmRelease(path string) (*HelmRelease, error) {
	helmRelease := helmspec.HelmRelease{}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("unable to read file: %v", path)
		return nil, err
	}
	err = yaml.Unmarshal([]byte(data), &helmRelease)
	if err != nil {
		log.Errorf("unable to unmarshal file %v in helmRelease", path)
		log.Errorf("error: %v", err)
		return nil, err
	}
	return &HelmRelease{HelmRelease: &helmRelease, Path: path}, nil

}

// ComputeChangedChartDirectories takes the merge base of HEAD and the configured remote and target branch and computes a
// slice of changed charts from that in the configured chart directories excluding those configured to be excluded.
func (t *Testing) ComputeChangedHelmReleases() ([]*DiffResult, error) {
	// cfg := t.config

	log.Debug("Merging base")
	mergeBase, err := t.computeMergeBase()
	if err != nil {
		return nil, err
	}
	log.Debug("Bases merged")

	worktreePath, err := os.MkdirTemp("./", "ct_previous_revision")
	if err != nil {
		return nil, fmt.Errorf("could not create previous revision directory: %w", err)
	}
	t.previousRevisionWorktree = worktreePath
	err = t.git.AddWorktree(worktreePath, mergeBase)
	if err != nil {
		return nil, fmt.Errorf("could not create worktree for previous revision: %w", err)
	}
	defer t.git.RemoveWorktree(worktreePath)

	allChangedFiles, err := t.git.ListChangedFiles(mergeBase)
	if err != nil {
		return nil, fmt.Errorf("failed creating diff: %w", err)
	}
	log.Infof("All changed HelmRelease files: %s", allChangedFiles)

	var changedHelmReleases []*HelmRelease
	for _, file := range allChangedFiles {

		hr, err := t.parseHelmRelease(file)
		if err != nil {
			log.Errorf("failed parsing a file: %v", err)
		}
		changedHelmReleases = append(changedHelmReleases, hr)
	}

	diffResults, err := t.processHelmReleases(changedHelmReleases)
	if err != nil {
		log.Errorf("Error processing diffs")
		return nil, err
	}
	return diffResults, nil
}

// getOldHelmReleaseVersion checks if an old version of the HelmRelease exists
func (t *Testing) getOldHelmReleaseVersion(chartPath string) (*HelmRelease, error) {
	cfg := t.config

	if !t.git.FileExistsOnBranch(t.computeCurrentRevisionPath(chartPath), cfg.Remote, cfg.TargetBranch) {
		fmt.Printf("Unable to find chart on %s. New chart detected.\n", cfg.TargetBranch)
		return nil, nil
	}

	helmRelease, err := t.parseHelmRelease(chartPath)
	if err != nil {
		return nil, fmt.Errorf("error reading Chart file: %v", err)
	}

	return helmRelease, nil
}

func (t *Testing) processHelmReleases(helmReleases []*HelmRelease) ([]*DiffResult, error) {
	var results []*DiffResult
	repoArgs := map[string][]string{}
	for _, repo := range t.config.HelmRepoExtraArgs {
		repoSlice := strings.SplitN(repo, "=", 2)
		name := repoSlice[0]
		repoExtraArgs := strings.Fields(repoSlice[1])
		repoArgs[name] = repoExtraArgs
	}
	for _, repo := range t.config.ChartRepos {
		repoSlice := strings.SplitN(repo, "=", 2)
		name := repoSlice[0]
		url := repoSlice[1]
		repoExtraArgs := repoArgs[name]
		if err := t.helm.AddRepo(name, url, repoExtraArgs); err != nil {
			return nil, fmt.Errorf("failed adding repo: %s=%s: %w", name, url, err)
		}
	}
	for _, helmRelease := range helmReleases {
		// Check for old HelmRelease
		oldHelmRelease, err := t.getOldHelmReleaseVersion(t.computePreviousRevisionPath(helmRelease.Path))
		if err != nil {
			log.Errorf("failed to parse old HelmRelease: %v", err)
		}
		if oldHelmRelease == nil {
			log.Debug("Found a new chart, skipping diff")
			log.Debugf("Helm release path: %s", helmRelease.Path)
			continue
		}
		result, err := t.checkDiff(oldHelmRelease, helmRelease)
		if err != nil {
			log.Errorf("error checking HelmRelease diff: %v", err)
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (t *Testing) computePreviousRevisionPath(fileOrDirPath string) string {
	return filepath.Join(t.previousRevisionWorktree, fileOrDirPath)
}

func (t *Testing) computeCurrentRevisionPath(fileOrDirPath string) string {
	return (strings.ReplaceAll(fileOrDirPath, t.previousRevisionWorktree[2:], ""))[1:]
}

func (t *Testing) checkDiff(oldHelmRelease, newHelmRelease *HelmRelease) (*DiffResult, error) {
	oldTemplate, err := t.helm.TemplateWithValues(oldHelmRelease.Path, oldHelmRelease.HelmRelease.Spec.Values, oldHelmRelease.HelmRelease.Spec.Chart.Spec.SourceRef.Name, oldHelmRelease.HelmRelease.Spec.Chart.Spec.Chart, oldHelmRelease.HelmRelease.Spec.Chart.Spec.Version)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Old template:\n %v", oldTemplate)
	newTemplate, err := t.helm.TemplateWithValues(newHelmRelease.Path, newHelmRelease.HelmRelease.Spec.Values, newHelmRelease.HelmRelease.Spec.Chart.Spec.SourceRef.Name, newHelmRelease.HelmRelease.Spec.Chart.Spec.Chart, newHelmRelease.HelmRelease.Spec.Chart.Spec.Version)
	if err != nil {
		return nil, err
	}
	fmt.Printf("New template:\n %v", newTemplate)
	helmDiff := diff.Diff(oldTemplate, newTemplate)
	diffResult := &DiffResult{*newHelmRelease, *oldHelmRelease, helmDiff}
	return diffResult, nil
}
