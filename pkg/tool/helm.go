package tool

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/helm/chart-testing/v3/pkg/exec"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type Helm struct {
	exec         exec.ProcessExecutor
	extraArgs    []string
	extraSetArgs []string
}

func NewHelm(exec exec.ProcessExecutor, extraArgs []string, extraSetArgs []string) Helm {
	return Helm{
		exec:         exec,
		extraArgs:    extraArgs,
		extraSetArgs: extraSetArgs,
	}
}

func (h Helm) TemplateWithValues(path string, values *v1.JSON, chartRepo, chart, version string) (string, error) {
	tempDir, err := os.MkdirTemp("./", "template")
	if err != nil {
		return "", fmt.Errorf("could not create temporary directory for Helm template: %w", err)
	}
	tempFile, err := ioutil.TempFile(tempDir, "template.json")
	if err != nil {
		return "", fmt.Errorf("could not create temporary file for Helm template: %w", err)
	}
	err = ioutil.WriteFile(tempFile.Name(), values.Raw, 0644)
	defer os.RemoveAll(tempDir)
	if err != nil {
		return "", fmt.Errorf("error creating temporary JSON file: %v", err)
	}

	return h.exec.RunProcessAndCaptureOutput("helm", "template", "test", fmt.Sprintf("%v/%v", chartRepo, chart), "-f", tempFile.Name(), "--version", version, h.extraArgs, h.extraSetArgs)
}

func (h Helm) AddRepo(name string, url string, extraArgs []string) error {
	log.Debugf("Name: %v", name)
	log.Debugf("URL: %v", url)
	log.Debugf("extraArgs: %v", extraArgs)
	return h.exec.RunProcess("helm", "repo", "add", name, url, extraArgs)
}

func (h Helm) Version() (string, error) {
	return h.exec.RunProcessAndCaptureStdout("helm", "version", "--template", "{{ .Version }}")
}
