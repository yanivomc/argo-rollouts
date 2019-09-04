package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	yaml2json "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	yamlutil "github.com/argoproj/argo-rollouts/utils/yaml"
)

const (
	// CLIName is the name of the CLI
	cliName = "Rollout"
)

func newCommand() *cobra.Command {
	var command = cobra.Command{
		Use:   cliName,
		Short: "Rollout is a kustomize plugin to enable StrategicMergePatch on rollout CRD",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("Expect one arguement for the Rollout Patch")
			}

			patch, patchYaml, err := readInPatch(args[0])
			if err != nil {
				return errors.Wrap(err, "Can not read in patch from file")
			}

			resources, err := readInResources()
			if err != nil {
				return errors.Wrap(err, "Can not read in resources from stdin")
			}
			transformedResources := []*unstructured.Unstructured{}
			for i := range resources {
				resource := resources[i]
				if resource.GetKind() == patch.GetKind() && resource.GetAPIVersion() == patch.GetAPIVersion() && resource.GetName() == patch.GetName() {
					resource, err = patchResource(resource, patchYaml)
					if err != nil {
						return err
					}
				}
				transformedResources = append(transformedResources, resource)
			}
			writeOutResources(os.Stdout, transformedResources)

			return nil
		},
	}
	return &command
}

func patchResource(origObj *unstructured.Unstructured, patchYaml []byte) (*unstructured.Unstructured, error) {
	orig, err := json.Marshal(origObj)
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "Can't read orig json"))
	}

	patchJSON, err := yaml2json.YAMLToJSON([]byte(patchYaml))
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "Can't read patch yaml"))
	}

	newYaml, err := strategicpatch.StrategicMergePatch(orig, patchJSON, v1alpha1.Rollout{})
	fmt.Fprintf(os.Stderr, "Orig: %s\n", string(orig))
	fmt.Fprintf(os.Stderr, "Patch: %s\n", string(patchJSON))
	fmt.Fprintf(os.Stderr, "Patched: %s\n", string(newYaml))
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "Can't StrategicMergePatch"))
	}
	return yamlutil.UnmarshelObj(string(newYaml))
}

func readInPatch(filePath string) (*unstructured.Unstructured, []byte, error) {
	patchYaml, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Can not read in patch")
	}
	patch, err := yamlutil.UnmarshelObj(string(patchYaml))
	if err != nil {
		return nil, nil, errors.Wrap(err, "Can not unmarshel patch yaml")
	}
	return patch, patchYaml, nil
}

func readInResources() ([]*unstructured.Unstructured, error) {
	resourceYamls, err := ioutil.ReadFile(os.Stdin.Name())
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read stdin")
	}
	return yamlutil.SplitYAML(string(resourceYamls))
}

func writeOutResources(writer io.Writer, resources []*unstructured.Unstructured) error {
	for i := range resources {
		resource := resources[i]
		bytes, err := json.Marshal(resource)
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, string(bytes))

	}
	return nil
}

func main() {
	if err := newCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
