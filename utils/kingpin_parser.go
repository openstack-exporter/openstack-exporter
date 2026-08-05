package utils

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
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

// RedactedLabelValue replaces the value of a mapped key that looks like a
// credential. It is constant, so the label still tells an operator whether the
// credential is set without publishing it to everyone who can read the metrics.
const RedactedLabelValue = "***"

// sensitiveKeyRe matches keys whose value is a credential, such as the Ironic
// driver_info ipmi_password and redfish_password, or a root_password left in
// the node extra dictionary.
//
// The match is anchored on the last segment on purpose: an unanchored
// `password` would also swallow legitimate fields such as
// `password_updated_at`, and silently redacting those would be worse than
// useless for alerting.
var sensitiveKeyRe = regexp.MustCompile(`(?i)(^|[._])(password|passwd|secret|token|private_key)$`)

// sensitiveKeyNames are credential-bearing keys that the pattern above cannot
// recognise by name. Ironic redacts these itself, so they are treated the same
// way here: image_url is routinely a pre-signed URL carrying credentials, and
// configdrive can embed injected secrets.
var sensitiveKeyNames = []string{"configdrive", "image_url"}

// isSensitive reports whether a mapped label or key looks credential-bearing.
// Both sides are checked, so hiding a secret behind an innocuous label name,
// as in `bmc=ipmi_password`, does not defeat the redaction.
func isSensitive(label, key string) bool {
	for _, s := range []string{label, key} {
		if sensitiveKeyRe.MatchString(s) {
			return true
		}
		leaf := s[strings.LastIndex(s, ".")+1:]
		if slices.Contains(sensitiveKeyNames, strings.ToLower(leaf)) {
			return true
		}
	}
	return false
}

// LabelMappingFlag parse server metadata to label kingpin option
//
// Supported formats:
// - `label=key` - map metadata *key* value to *label*;
// - `key` - same as above if *label* equal to *key*.
//
// Accept multiple mappings separated by comma (',').
// One metadata key may be mapped into multiple labels, but not vice versa.
//
// Labels and Keys are parallel: index i of one corresponds to index i of the
// other. Extract and ExtractAny rely on that to return values in the same order
// as the label names appended to a descriptor.
type LabelMappingFlag struct {
	Labels []string
	Keys   []string
	// Sensitive is parallel to Labels and Keys and marks the entries whose
	// value is redacted on extraction.
	Sensitive []bool

	// DeriveLabelFromLeaf makes the bare `key` form derive the label from the
	// last dot-separated segment of the key. It exists for mappings whose keys
	// are qualified with the source they come from, such as
	// `driver_info.deploy_kernel`, where the whole key is not a valid
	// Prometheus label name.
	DeriveLabelFromLeaf bool
}

// NewLabelMappingFlag returns an initialised mapping equivalent to one that
// parsed an empty flag value: non-nil and empty.
func NewLabelMappingFlag() *LabelMappingFlag {
	return &LabelMappingFlag{
		Labels:    []string{},
		Keys:      []string{},
		Sensitive: []bool{},
	}
}

func (s *LabelMappingFlag) Set(value string) error {
	if s.Labels == nil {
		s.Labels = make([]string, 0)
	}
	if s.Keys == nil {
		s.Keys = make([]string, 0)
	}
	if s.Sensitive == nil {
		s.Sensitive = make([]bool, 0)
	}

	if len(value) == 0 {
		return nil
	}

	for _, kv := range strings.Split(value, ",") {
		label, key, ok := strings.Cut(kv, "=")
		if !ok {
			key = label
			if s.DeriveLabelFromLeaf {
				if _, leaf, found := strings.Cut(label, "."); found {
					label = leaf[strings.LastIndex(leaf, ".")+1:]
				}
			}
		}

		if slices.Contains(s.Labels, label) {
			return fmt.Errorf("%w: %s", ErrLabelDup, label)
		}
		if !labelNameConstraintRe.MatchString(label) {
			return fmt.Errorf("%w: %s", ErrLabelName, label)
		}

		s.Labels = append(s.Labels, label)
		s.Keys = append(s.Keys, key)
		s.Sensitive = append(s.Sensitive, isSensitive(label, key))
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
	for i, key := range s.Keys {
		ret = append(ret, s.redact(i, m[key]))
	}

	return ret
}

// redact replaces a credential value with a constant marker, keeping the
// distinction between "set" and "not set" so operators can still alert on a
// missing credential without the value being exposed.
func (s *LabelMappingFlag) redact(i int, value string) string {
	if i < len(s.Sensitive) && s.Sensitive[i] && value != "" {
		return RedactedLabelValue
	}
	return value
}

// ValidateAgainst rejects mapped labels that collide with labels the metric
// already defines. A duplicate label name makes prometheus.NewDesc produce an
// invalid descriptor, which then panics when the metric is emitted, so this
// must fail at startup instead.
func (s *LabelMappingFlag) ValidateAgainst(reserved []string) error {
	for _, label := range s.Labels {
		if slices.Contains(reserved, label) {
			return fmt.Errorf("%w: %s is already a label of this metric", ErrLabelDup, label)
		}
	}
	return nil
}

// SensitiveLabels returns the labels whose values are redacted, so the caller
// can warn about them once after flag parsing.
func (s *LabelMappingFlag) SensitiveLabels() []string {
	var ret []string
	for i, sensitive := range s.Sensitive {
		if sensitive {
			ret = append(ret, s.Labels[i])
		}
	}
	return ret
}

// ExtractAny is Extract for untyped maps such as the Ironic node driver_info,
// instance_info, extra, properties and driver_internal_info dictionaries.
//
// Only top-level keys are looked up. Scalars are stringified; a missing key, a
// null and any composite value yield an empty string, so a nested object is
// never rendered into a label.
func (s *LabelMappingFlag) ExtractAny(m map[string]any) []string {
	ret := make([]string, 0, len(s.Keys))
	for i, key := range s.Keys {
		ret = append(ret, s.redact(i, labelValue(m[key])))
	}

	return ret
}

// NestedMap is a named map used by ExtractNestedAny. Mapping keys select a
// value from one of these maps with the form "name.key".
type NestedMap struct {
	Name   string
	Values map[string]any
}

// ValidateNestedKeys requires every mapping key to identify both the map and
// the top-level key it reads. The supplied map names form the allowlist, so a
// typo fails during construction instead of producing an empty label forever.
func (s *LabelMappingFlag) ValidateNestedKeys(names ...string) error {
	for _, key := range s.Keys {
		name, field, ok := strings.Cut(key, ".")
		if !ok || field == "" {
			return fmt.Errorf("mapping key %q must be qualified as <map>.<key>, one of %s",
				key, strings.Join(names, ", "))
		}
		if !slices.Contains(names, name) {
			return fmt.Errorf("unknown map %q in mapping key %q, want one of %s",
				name, key, strings.Join(names, ", "))
		}
	}
	return nil
}

// ExtractNestedAny resolves qualified keys against named untyped maps. It is
// the multi-map counterpart to ExtractAny: each key is split once at its first
// dot, the remainder is looked up as a top-level key, and the resulting value
// is rendered and redacted using the same rules.
//
// Call ValidateNestedKeys during construction to reject unknown map names.
// Missing maps and keys deliberately yield an empty value, matching Extract
// and ExtractAny.
func (s *LabelMappingFlag) ExtractNestedAny(maps ...NestedMap) []string {
	byName := make(map[string]map[string]any, len(maps))
	for _, m := range maps {
		byName[m.Name] = m.Values
	}

	ret := make([]string, 0, len(s.Keys))
	for i, key := range s.Keys {
		name, field, _ := strings.Cut(key, ".")
		ret = append(ret, s.redact(i, labelValue(byName[name][field])))
	}
	return ret
}

// labelValue renders a JSON-decoded value as a Prometheus label value.
func labelValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		// JSON numbers decode to float64; format them without an exponent so
		// that large values such as an image virtual_size stay readable.
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		// Objects and arrays are deliberately not rendered.
		return ""
	}
}

func LabelMapping(s kingpin.Settings) *LabelMappingFlag {
	ret := new(LabelMappingFlag)
	s.SetValue(ret)
	return ret
}

// QualifiedLabelMapping is LabelMapping for mappings whose keys are qualified
// with the dictionary they are read from, for example
// `driver_info.deploy_kernel`. The bare form derives the label from the last
// segment, so `driver_info.deploy_kernel` yields the label `deploy_kernel`.
func QualifiedLabelMapping(s kingpin.Settings) *LabelMappingFlag {
	ret := &LabelMappingFlag{DeriveLabelFromLeaf: true}
	s.SetValue(ret)
	return ret
}
