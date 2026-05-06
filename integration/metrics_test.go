package integration

import (
	"bytes"
	"math"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type metricSample struct {
	labels map[string]string
	value  float64
}

func parseMetrics(body []byte) (map[string][]metricSample, error) {
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	samples := make(map[string][]metricSample, len(metricFamilies))
	for name, family := range metricFamilies {
		for _, metric := range family.GetMetric() {
			value, ok := metricValue(metric)
			if !ok {
				continue
			}
			sample := metricSample{
				labels: make(map[string]string, len(metric.GetLabel())),
				value:  value,
			}
			for _, label := range metric.GetLabel() {
				sample.labels[label.GetName()] = label.GetValue()
			}
			samples[name] = append(samples[name], sample)
		}
	}

	return samples, nil
}

func metricValue(metric *dto.Metric) (float64, bool) {
	switch {
	case metric.GetGauge() != nil:
		return metric.GetGauge().GetValue(), true
	case metric.GetCounter() != nil:
		return metric.GetCounter().GetValue(), true
	case metric.GetUntyped() != nil:
		return metric.GetUntyped().GetValue(), true
	default:
		return math.NaN(), false
	}
}

func findMetric(metricFamilies map[string][]metricSample, name string, labels map[string]string) (metricSample, bool) {
	for _, sample := range metricFamilies[name] {
		if labelsMatch(sample.labels, labels) {
			return sample, true
		}
	}
	return metricSample{}, false
}

func labelsMatch(got, want map[string]string) bool {
	for name, value := range want {
		if got[name] != value {
			return false
		}
	}
	return true
}
