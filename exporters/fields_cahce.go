package exporters

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	fieldIndexCache   = make(map[reflect.Type]map[string]int)
	fieldIndexCacheMu sync.RWMutex
)

func buildIndex(t reflect.Type) map[string]int {
	fieldIndex := make(map[string]int)

	for i := 0; i < t.NumField(); i++ {
		tag := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			fieldIndex[tag] = i
		}
	}
	return fieldIndex
}

func getFieldIndex(t reflect.Type) map[string]int {
	fieldIndexCacheMu.RLock()
	if v, ok := fieldIndexCache[t]; ok {
		fieldIndexCacheMu.RUnlock()
		return v
	}
	fieldIndexCacheMu.RUnlock()

	idx := buildIndex(t)
	fieldIndexCacheMu.Lock()
	if existing, ok := fieldIndexCache[t]; ok {
		fieldIndexCacheMu.Unlock()
		return existing
	}
	fieldIndexCache[t] = idx
	fieldIndexCacheMu.Unlock()
	return idx
}

func resolveField(obj interface{}, field string) string {
	parts := strings.SplitN(field, ".", 2)
	t := reflect.TypeOf(obj)

	idx := getFieldIndex(t)

	index, ok := idx[parts[0]]
	if !ok {
		return ""
	}

	fieldVal := reflect.ValueOf(obj).Field(index)

	if len(parts) == 1 {
		return fmt.Sprintf("%v", fieldVal.Interface())
	}

	if fieldVal.Kind() != reflect.Map {
		return ""
	}

	mapVal := fieldVal.MapIndex(reflect.ValueOf(parts[1]))
	if !mapVal.IsValid() {
		return ""
	}
	if mapVal.Kind() == reflect.Interface {
		mapVal = mapVal.Elem()
	}

	return fmt.Sprintf("%v", mapVal.Interface())

}

func resolveExtraLabelValues(obj interface{}, spec *utils.ExtraLabelSpec) []string {
	if spec == nil {
		return nil
	}
	values := make([]string, len(spec.DynamicFields))
	for i, field := range spec.DynamicFields {
		values[i] = resolveField(obj, field)
	}
	return values
}

func computeMetricLabels(service string, metric Metric, extraLabels *utils.ExtraLabelsFlag) []string {

	labels := make([]string, 0)
	labels = append(labels, metric.Labels...)
	addLabels := extraLabels.Extract(service, metric.Name)
	if addLabels == nil {
		return labels
	}

	for _, l := range addLabels.DynamicLabels {
		if !slices.Contains(metric.Labels, l) {
			labels = append(labels, l)
		}
	}

	return labels
}

func computeConstantLabels(service string, metric Metric, extraLabels *utils.ExtraLabelsFlag) prometheus.Labels {
	promLabels := make(prometheus.Labels)

	addLabels := extraLabels.Extract(service, metric.Name)
	if addLabels == nil {
		return nil
	}

	if len(addLabels.StaticLabels) == 0 {
		return nil
	}

	for k, v := range addLabels.StaticLabels {
		promLabels[k] = v
	}

	return promLabels
}
