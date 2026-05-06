package utils

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/alecthomas/kingpin/v2"
)

type ExtraLabelsFlag struct {
	Specs map[string]map[string]ExtraLabelSpec
}

type ExtraLabelSpec struct {
	DynamicLabels []string
	DynamicFields []string
	StaticLabels  map[string]string
}

var (
	ErrMissingLabels = errors.New("missing labels")
)

// func (f *ExtraLabelsFlag) String() string {
// 	serviceBuf := make([]string, 0, len(f.Specs))

// 	for service, sv := range f.Specs {
// 		metricBuf := make([]string, 0, len(sv))
// 		for metric, v := range sv {
// 			str := service + "." + metric + ":" + strings.Join(v.DynamicLabels, ",")
// 			for dLabelKey, dLabelsVal := range v.StaticLabels {
// 				str = str + "," + dLabelKey + "=" + dLabelsVal
// 			}
// 			metricBuf = append(metricBuf, str)
// 		}
// 		serviceBuf = append(serviceBuf, strings.Join(metricBuf, ","))
// 	}

// 	return strings.Join(serviceBuf, " ")
// }

// string function should be simple since its only purpose is to display in help
func (f *ExtraLabelsFlag) String() string {
	return "<service>.<metric>:<labels>"
}

func (f *ExtraLabelsFlag) Set(raw string) error {
	if f.Specs == nil {
		f.Specs = make(map[string]map[string]ExtraLabelSpec)
	}
	if len(raw) == 0 {
		return nil
	}

	parts := strings.SplitN(raw, ":", 2)

	if len(parts) != 2 {
		return fmt.Errorf("invalid format %q expected <service>.<metric>:<labels>", raw)
	}

	target := parts[0]
	labels := parts[1]
	if labels == "" {
		return ErrMissingLabels
	}

	targetParts := strings.Split(target, ".")
	if len(targetParts) != 2 {
		return fmt.Errorf("invalid format %q expected <service>.<metric>:<labels>", raw)
	}

	servicePart := targetParts[0]
	metricPart := targetParts[1]

	if f.Specs[servicePart] == nil {
		f.Specs[servicePart] = make(map[string]ExtraLabelSpec)
	}

	labelsParts := strings.Split(labels, ",")

	labelSpec := ExtraLabelSpec{}
	labelSpec.DynamicLabels = make([]string, 0)
	labelSpec.DynamicFields = make([]string, 0)
	labelSpec.StaticLabels = make(map[string]string)

	for _, v := range labelsParts {
		if strings.Contains(v, "=") {
			// static label
			sLabelPart := strings.SplitN(v, "=", 2)
			if len(sLabelPart) != 2 {
				return fmt.Errorf("invalid dynamic label format %q expected key=value", v)
			}

			// check if dupe
			if slices.Contains(labelSpec.DynamicLabels, sLabelPart[0]) {
				return fmt.Errorf("%w: %s", ErrLabelDup, sLabelPart[0])
			}

			if _, ok := labelSpec.StaticLabels[sLabelPart[0]]; ok {
				return fmt.Errorf("%w: %s", ErrLabelDup, sLabelPart[0])
			}

			if !labelNameConstraintRe.MatchString(sLabelPart[0]) {
				return fmt.Errorf("%w: %s", ErrLabelName, sLabelPart[0])
			}

			labelSpec.StaticLabels[sLabelPart[0]] = sLabelPart[1]
		} else {
			// dynamic labels
			// check if dupe
			if slices.Contains(labelSpec.DynamicFields, v) {
				return fmt.Errorf("%w: %s", ErrLabelDup, v)
			}
			labelName := strings.ReplaceAll(v, ".", "_")
			if _, ok := labelSpec.StaticLabels[labelName]; ok {
				return fmt.Errorf("%w: %s", ErrLabelDup, v)
			}

			if !labelNameConstraintRe.MatchString(labelName) {
				return fmt.Errorf("%w: %s", ErrLabelName, labelName)
			}

			labelSpec.DynamicFields = append(labelSpec.DynamicFields, v)
			labelSpec.DynamicLabels = append(labelSpec.DynamicLabels, labelName)
		}
	}

	f.Specs[servicePart][metricPart] = labelSpec
	return nil

}

func (s *ExtraLabelsFlag) IsCumulative() bool {
	return true
}

func ExtraLabelMapping(s kingpin.Settings) *ExtraLabelsFlag {
	ret := new(ExtraLabelsFlag)
	s.SetValue(ret)
	return ret
}

func (f *ExtraLabelsFlag) Extract(service string, metric string) *ExtraLabelSpec {
	if f == nil {
		return nil
	}
	if _, ok := f.Specs[service]; !ok {
		return nil
	}
	if _, ok := f.Specs[service][metric]; !ok {
		return nil
	}
	spec := f.Specs[service][metric]
	return &spec
}
