package yaml

import (
	"fmt"
	"os"
	"regexp"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var diffSeparator = regexp.MustCompile(`\n---`)

// SplitYAML splits a YAML file into unstructured objects. Returns list of all unstructured objects
// found in the yaml. Panics if any errors occurred.
func SplitYAML(out string) ([]*unstructured.Unstructured, error) {
	parts := diffSeparator.Split(out, -1)
	var objs []*unstructured.Unstructured
	for _, part := range parts {
		obj, err := UnmarshelObj(part)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "Object read in: %v\n", obj)
		objs = append(objs, obj)
	}
	return objs, nil
}

func UnmarshelObj(str string) (*unstructured.Unstructured, error) {
	var objMap map[string]interface{}
	err := yaml.Unmarshal([]byte(str), &objMap)
	if err != nil {
		return nil, err
	}
	if len(objMap) == 0 {
		// handles case where theres no content between `---`
		return nil, nil
	}
	var obj unstructured.Unstructured
	err = yaml.Unmarshal([]byte(str), &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}
