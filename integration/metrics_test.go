package integration

import (
	"bytes"
	"log"
	"math"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

type metricSample struct {
	labels map[string]string
	value  float64
}

type labels map[string]string

type metricSet struct {
	samples map[string][]metricSample
	body    string
}

func startExporter(t *testing.T, services ...string) func() {
	t.Helper()

	_, cleanup, err := startOpenStackExporter(services)
	if err != nil {
		t.Fatalf("Failed to start exporter: %v", err)
	}
	return cleanup
}

func scrapeMetrics(t *testing.T, context string) metricSet {
	t.Helper()

	_, bodyBytes, err := httpGetRetry(defaultMetricsURL, 10, t)
	if err != nil {
		t.Fatalf("Failed to fetch metrics%s: %v", contextSuffix(context), err)
	}

	body := string(bodyBytes)
	samples, err := parseMetrics(bodyBytes)
	if err != nil {
		failMetrics(t, body, "Failed to parse metrics response%s: %v", contextSuffix(context), err)
	}
	return metricSet{samples: samples, body: body}
}

func scrapeLoggedMetrics(t *testing.T, context string) metricSet {
	t.Helper()

	metrics := scrapeMetrics(t, context)
	t.Logf("Metrics response body:\n%s", metrics.body)
	return metrics
}

func parseMetrics(body []byte) (map[string][]metricSample, error) {
	parser := expfmt.NewTextParser(model.UTF8Validation)
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

func (m metricSet) requireMetric(t *testing.T, name string, labels map[string]string) metricSample {
	t.Helper()

	sample, ok := findMetric(m.samples, name, labels)
	if !ok {
		failMetrics(t, m.body, "Expected %s metric with labels %v not found", name, labels)
	}
	return sample
}

func (m metricSet) requireNoMetric(t *testing.T, name string, labels map[string]string) {
	t.Helper()

	if _, ok := findMetric(m.samples, name, labels); ok {
		failMetrics(t, m.body, "Expected %s metric with labels %v to be absent", name, labels)
	}
}

func (m metricSet) requireUp(t *testing.T, name string) {
	t.Helper()

	if sample := m.requireMetric(t, name, nil); sample.value != 1 {
		failMetrics(t, m.body, "%s should have value 1 indicating service is up, got %v", name, sample.value)
	}
}

func (m metricSet) requireAnyFamily(t *testing.T, names ...string) {
	t.Helper()

	for _, name := range names {
		if _, ok := m.samples[name]; ok {
			return
		}
	}
	failMetrics(t, m.body, "Expected one of metrics %v to be present", names)
}

func (m metricSet) requireLabels(t *testing.T, metricName string, match map[string]string, labels ...string) metricSample {
	t.Helper()

	sample := m.requireMetric(t, metricName, match)
	for _, label := range labels {
		if sample.labels[label] == "" {
			failMetrics(t, m.body, "Expected %s metric with labels %v to include non-empty %s label", metricName, match, label)
		}
	}
	return sample
}

func (m metricSet) requirePresentLabels(t *testing.T, metricName string, match map[string]string, labels ...string) metricSample {
	t.Helper()

	sample := m.requireMetric(t, metricName, match)
	for _, label := range labels {
		if _, ok := sample.labels[label]; !ok {
			failMetrics(t, m.body, "Expected %s metric with labels %v to include %s label", metricName, match, label)
		}
	}
	return sample
}

func (m metricSet) requireLabelValue(t *testing.T, metricName string, match map[string]string, label string, value string) metricSample {
	t.Helper()

	sample := m.requireMetric(t, metricName, match)
	if sample.labels[label] != value {
		failMetrics(t, m.body, "Expected %s metric with labels %v to have %s=%q, got %q", metricName, match, label, value, sample.labels[label])
	}
	return sample
}

func (m metricSet) requireSampleWithLabels(t *testing.T, metricName string, required ...string) metricSample {
	t.Helper()

	for _, sample := range m.samples[metricName] {
		ok := true
		for _, label := range required {
			if sample.labels[label] == "" {
				ok = false
				break
			}
		}
		if ok {
			return sample
		}
	}
	failMetrics(t, m.body, "No %s metric contained non-empty labels %v", metricName, required)
	return metricSample{}
}

func (m metricSet) requireMinValue(t *testing.T, name string, labels map[string]string, min float64) metricSample {
	t.Helper()

	sample := m.requireMetric(t, name, labels)
	if sample.value < min {
		failMetrics(t, m.body, "Expected %s metric with labels %v to be at least %v, got %v", name, labels, min, sample.value)
	}
	return sample
}

func (m metricSet) requireMinValueWithLabels(t *testing.T, name string, match map[string]string, min float64, labels ...string) {
	t.Helper()

	sample := m.requireMinValue(t, name, match, min)
	for _, label := range labels {
		if sample.labels[label] == "" {
			failMetrics(t, m.body, "Expected %s metric with labels %v to include non-empty %s label", name, match, label)
		}
	}
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

func failMetrics(t *testing.T, body string, msg string, args ...interface{}) {
	t.Helper()
	log.Printf("Metrics body:\n%s\n", body)
	t.Fatalf(msg, args...)
}

func contextSuffix(context string) string {
	if context == "" {
		return ""
	}
	return " " + context
}
