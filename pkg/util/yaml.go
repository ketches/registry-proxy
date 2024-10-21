package util

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// MarshalYAML marshals the given value into YAML format.
func MarshalYAML(v interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	err := encoder.Encode(v)
	return buffer.Bytes(), err
}

// UnmarshalYAML unmarshals the given YAML data into the given value.
func UnmarshalYAML(in []byte, out interface{}) error {
	return yaml.Unmarshal(in, out)
}
