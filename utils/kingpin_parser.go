package utils

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/alecthomas/kingpin/v2"
)

var (
	ErrLabelDup  = errors.New("duplicate label")
	ErrLabelName = errors.New("bad label name")
)

// Prometheus label names must:
// - Not start from number
// - Not use `__` prefix
// - Use only latin letters, digits and underscore
//
// See: https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
var labelNameConstraintRe = regexp.MustCompile(`^([^_0-9][^_][a-zA-Z]|(?:_)[a-zA-Z0-9]|[a-zA-Z])[a-zA-Z0-9_]*$`)

// LabelMappingFlag parse server metadata to label kingpin option
//
// Supported formats:
// - `label=key` - map metadata *key* value to *label*;
// - `key` - same as above if *label* equal to *key*.
//
// Accept multiple mappings separated by comma (',').
// One metadata key may be mapped into multiple labels, but not vice versa.
type LabelMappingFlag struct {
	Labels []string
	Keys   []string
}

func (s *LabelMappingFlag) Set(value string) error {
	if s.Labels == nil {
		s.Labels = make([]string, 0)
	}
	if s.Keys == nil {
		s.Keys = make([]string, 0)
	}

	if len(value) == 0 {
		return nil
	}

	for _, kv := range strings.Split(value, ",") {
		label, key, ok := strings.Cut(kv, "=")
		if !ok {
			key = label
		}

		if slices.Contains(s.Labels, label) {
			return fmt.Errorf("%w: %s", ErrLabelDup, label)
		}
		if !labelNameConstraintRe.MatchString(label) {
			return fmt.Errorf("%w: %s", ErrLabelName, label)
		}

		s.Labels = append(s.Labels, label)
		s.Keys = append(s.Keys, key)
	}

	return nil
}

func (s *LabelMappingFlag) String() string {

	buf := make([]string, 0, len(s.Labels))
	for i := range s.Labels {
		label, key := s.Labels[i], s.Keys[i]
		if label != key {
			buf = append(buf, strings.Join([]string{label, key}, "="))
		} else {
			buf = append(buf, label)
		}
	}

	return strings.Join(buf, ",")
}

func (s *LabelMappingFlag) IsCumulative() bool {
	return true
}

func (s *LabelMappingFlag) Extract(m map[string]string) []string {
	ret := make([]string, 0, len(s.Keys))
	for _, key := range s.Keys {
		ret = append(ret, m[key])
	}

	return ret
}

func LabelMapping(s kingpin.Settings) *LabelMappingFlag {
	ret := new(LabelMappingFlag)
	s.SetValue(ret)
	return ret
}
