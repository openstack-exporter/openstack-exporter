//go:build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const defaultPrefix = "openstack"

var (
	reFirstCap = regexp.MustCompile(`(.)([A-Z][a-z]+)`)
	reAllCap   = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

type metricInfo struct {
	Name        string
	Exporter    string
	LocalName   string
	Labels      []string
	Slow        bool
	Deprecated  string
	DynamicNote string
}

type metricExample struct {
	Name     string
	Exporter string
	Lines    []string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		root  = flag.String("root", ".", "repository root")
		out   = flag.String("out", "docs/metrics.md", "output markdown path")
		check = flag.Bool("check", false, "check that the output file is current")
	)
	flag.Parse()

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	outPath := *out
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(rootPath, outPath)
	}

	metrics, err := collectMetrics(rootPath)
	if err != nil {
		return err
	}
	examples, err := collectMetricExamples(rootPath, metrics)
	if err != nil {
		return err
	}
	rendered := []byte(renderMetricsDoc(metrics, examples))

	if *check {
		existing, err := os.ReadFile(outPath)
		if err != nil {
			return err
		}
		if !bytes.Equal(existing, rendered) {
			return fmt.Errorf("%s is out of date; run go run ./script/generate-metrics-doc.go", outPath)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, rendered, 0o644)
}

func collectMetrics(root string) ([]metricInfo, error) {
	files, err := filepath.Glob(filepath.Join(root, "exporters", "*.go"))
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	var metrics []metricInfo
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			return nil, err
		}

		exporterName, err := exporterMetricNamespace(parsed)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if exporterName == "" {
			continue
		}

		metrics = append(metrics, descriptorMetrics(parsed, exporterName)...)
		if exporterName == "nova" {
			labels, ok, err := stringSliceAssignment(parsed, "serverStatusBaseLabels")
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			if !ok {
				return nil, fmt.Errorf("%s: serverStatusBaseLabels not found", file)
			}
			metrics = append(metrics, newMetric(exporterName, "server_status", labels, false, "", "labels can be extended with `--nova.metadata-extra-labels`"))
		}
	}

	seen := make(map[string]struct{}, len(metrics))
	for _, metric := range metrics {
		if _, ok := seen[metric.Name]; ok {
			return nil, fmt.Errorf("duplicate metric %s", metric.Name)
		}
		seen[metric.Name] = struct{}{}
	}
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})
	return metrics, nil
}

func collectMetricExamples(root string, metrics []metricInfo) ([]metricExample, error) {
	files, err := filepath.Glob(filepath.Join(root, "exporters", "*_test.go"))
	if err != nil {
		return nil, err
	}

	exporterForMetric := buildMetricExporterLookup(metrics)
	fset := token.NewFileSet()
	examplesByName := map[string]metricExample{}
	for _, file := range files {
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, text := range prometheusTextLiterals(parsed) {
			for _, example := range parsePrometheusExamples(text, exporterForMetric) {
				if _, exists := examplesByName[example.Name]; !exists {
					examplesByName[example.Name] = example
				}
			}
		}
	}

	examples := make([]metricExample, 0, len(examplesByName))
	for _, example := range examplesByName {
		examples = append(examples, example)
	}
	sort.Slice(examples, func(i, j int) bool {
		if examples[i].Exporter != examples[j].Exporter {
			return examples[i].Exporter < examples[j].Exporter
		}
		return examples[i].Name < examples[j].Name
	})
	return examples, nil
}

func descriptorMetrics(file *ast.File, exporterName string) []metricInfo {
	var metrics []metricInfo
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 || !isPrometheusDescPtr(field.Type) {
					continue
				}
				fieldName := field.Names[0].Name
				if !ast.IsExported(fieldName) {
					continue
				}

				var tag reflect.StructTag
				if field.Tag != nil {
					tag = reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				}
				name := tag.Get("metric")
				if name == "" {
					name = camelToSnake(fieldName)
				}

				metrics = append(metrics, newMetric(
					exporterName,
					name,
					splitLabels(tag.Get("labels")),
					tag.Get("slow") == "true",
					tag.Get("deprecated"),
					"",
				))
			}
		}
	}
	return metrics
}

func buildMetricExporterLookup(metrics []metricInfo) func(string) string {
	metricExporters := make(map[string]string, len(metrics))
	exporters := map[string]struct{}{}
	for _, metric := range metrics {
		metricExporters[metric.Name] = metric.Exporter
		exporters[metric.Exporter] = struct{}{}
	}

	exporterNames := make([]string, 0, len(exporters))
	for exporter := range exporters {
		exporterNames = append(exporterNames, exporter)
	}
	sort.Slice(exporterNames, func(i, j int) bool {
		if len(exporterNames[i]) != len(exporterNames[j]) {
			return len(exporterNames[i]) > len(exporterNames[j])
		}
		return exporterNames[i] < exporterNames[j]
	})

	return func(metricName string) string {
		if exporter, ok := metricExporters[metricName]; ok {
			return exporter
		}
		if strings.HasPrefix(metricName, defaultPrefix+"_exporter_") {
			return "exporter"
		}
		for _, exporter := range exporterNames {
			if strings.HasPrefix(metricName, defaultPrefix+"_"+exporter+"_") {
				return exporter
			}
		}
		return "other"
	}
}

func prometheusTextLiterals(file *ast.File) []string {
	var texts []string
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		text, ok := stringLiteral(lit)
		if ok && strings.Contains(text, "# HELP "+defaultPrefix+"_") {
			texts = append(texts, text)
		}
		return true
	})
	return texts
}

func parsePrometheusExamples(text string, exporterForMetric func(string) string) []metricExample {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var examples []metricExample
	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "# HELP "+defaultPrefix+"_") {
			i++
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			i++
			continue
		}

		metricName := fields[2]
		exampleLines := []string{line}
		i++
		if i < len(lines) {
			typeLine := strings.TrimSpace(lines[i])
			if strings.HasPrefix(typeLine, "# TYPE "+metricName+" ") {
				exampleLines = append(exampleLines, typeLine)
				i++
			}
		}

		var sample string
		for i < len(lines) {
			nextLine := strings.TrimSpace(lines[i])
			if strings.HasPrefix(nextLine, "# HELP "+defaultPrefix+"_") {
				break
			}
			if sample == "" && nextLine != "" && !strings.HasPrefix(nextLine, "#") {
				sample = nextLine
			}
			i++
		}
		if sample == "" {
			continue
		}
		exampleLines = append(exampleLines, sample)
		examples = append(examples, metricExample{
			Name:     metricName,
			Exporter: exporterForMetric(metricName),
			Lines:    exampleLines,
		})
	}
	return examples
}

func newMetric(exporterName, localName string, labels []string, slow bool, deprecated, dynamicNote string) metricInfo {
	return metricInfo{
		Name:        fmt.Sprintf("%s_%s_%s", defaultPrefix, exporterName, localName),
		Exporter:    exporterName,
		LocalName:   localName,
		Labels:      labels,
		Slow:        slow,
		Deprecated:  deprecated,
		DynamicNote: dynamicNote,
	}
}

func exporterMetricNamespace(file *ast.File) (string, error) {
	names := map[string]struct{}{}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || typeName(lit.Type) != "BaseOpenStackExporter" {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Name" {
				continue
			}
			value, ok := stringLiteral(kv.Value)
			if ok {
				names[value] = struct{}{}
			}
		}
		return true
	})

	switch len(names) {
	case 0:
		return "", nil
	case 1:
		for name := range names {
			return name, nil
		}
	}

	var sorted []string
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	return "", fmt.Errorf("multiple exporter metric namespaces found: %s", strings.Join(sorted, ", "))
}

func stringSliceAssignment(file *ast.File, name string) ([]string, bool, error) {
	var (
		values []string
		found  bool
		err    error
	)
	ast.Inspect(file, func(n ast.Node) bool {
		if found || err != nil {
			return false
		}
		switch node := n.(type) {
		case *ast.AssignStmt:
			for i, lhs := range node.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != name || i >= len(node.Rhs) {
					continue
				}
				values, err = stringSliceLiteral(node.Rhs[i])
				found = err == nil
				return false
			}
		case *ast.ValueSpec:
			for i, ident := range node.Names {
				if ident.Name != name || i >= len(node.Values) {
					continue
				}
				values, err = stringSliceLiteral(node.Values[i])
				found = err == nil
				return false
			}
		}
		return true
	})
	return values, found, err
}

func stringSliceLiteral(expr ast.Expr) ([]string, error) {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("expected string slice literal")
	}

	values := make([]string, 0, len(lit.Elts))
	for _, elt := range lit.Elts {
		value, ok := stringLiteral(elt)
		if !ok {
			return nil, fmt.Errorf("expected string literal in slice")
		}
		values = append(values, value)
	}
	return values, nil
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

func isPrometheusDescPtr(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := star.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Desc" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "prometheus"
}

func typeName(expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if ok {
		return ident.Name
	}
	return ""
}

func camelToSnake(s string) string {
	s = reFirstCap.ReplaceAllString(s, "${1}_${2}")
	s = reAllCap.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}

func splitLabels(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			labels = append(labels, part)
		}
	}
	return labels
}

func renderMetricsDoc(metrics []metricInfo, examples []metricExample) string {
	var b strings.Builder
	b.WriteString("# Metrics\n\n")
	b.WriteString("<!-- Code generated by go run ./script/generate-metrics-doc.go; DO NOT EDIT. -->\n")
	b.WriteString("<!-- markdownlint-disable MD013 -->\n\n")
	b.WriteString("This inventory is generated from exporter descriptor tags in `exporters/*.go`,\n")
	b.WriteString("with sample Prometheus lines extracted from fixture-backed exporter tests.\n")
	b.WriteString("Metric names use the default `--prefix=openstack`; replace the leading `openstack`\n")
	b.WriteString("if you run the exporter with a different prefix.\n\n")
	b.WriteString("Every service exporter also emits these scrape health metrics:\n")
	b.WriteString("`openstack_<exporter>_up`, `openstack_<exporter>_exporter_scrapes_total`,\n")
	b.WriteString("and `openstack_<exporter>_exporter_scrape_errors_total`.\n\n")
	b.WriteString("When `--collect-metric-time` is enabled, source fetch durations are emitted as\n")
	b.WriteString("`openstack_exporter_source_fetch_duration_seconds{service=\"<exporter>\",source=\"<source>\"}`.\n\n")
	b.WriteString("The standalone `/metrics` handler also registers `openstack_exporter_build_info`.\n\n")
	b.WriteString("Regenerate this file after changing metric descriptors:\n\n")
	b.WriteString("```sh\n")
	b.WriteString("go run ./script/generate-metrics-doc.go\n")
	b.WriteString("```\n\n")
	b.WriteString("Check that the generated file is current:\n\n")
	b.WriteString("```sh\n")
	b.WriteString("go run ./script/generate-metrics-doc.go -check\n")
	b.WriteString("```\n\n")

	writeSlowMetrics(&b, metrics)
	writeDeprecatedMetrics(&b, metrics)
	writeCollectedMetrics(&b, metrics)
	writePrometheusExamples(&b, examples)

	return b.String()
}

func writeSlowMetrics(b *strings.Builder, metrics []metricInfo) {
	b.WriteString("## Slow Metrics\n\n")
	b.WriteString("Slow metrics can be disabled with `--disable-slow-metrics` and individually\n")
	b.WriteString("re-enabled with `--enable-metric=<disable-key>`.\n\n")

	slow := filterMetrics(metrics, func(metric metricInfo) bool { return metric.Slow })
	if len(slow) == 0 {
		b.WriteString("No metrics are marked slow.\n\n")
		return
	}

	b.WriteString("Metric | Exporter | Disable key\n")
	b.WriteString("-------|----------|------------\n")
	for _, metric := range slow {
		fmt.Fprintf(b, "`%s` | %s | `%s-%s`\n", metric.LocalName, metric.Exporter, metric.Exporter, metric.LocalName)
	}
	b.WriteString("\n")
}

func writeDeprecatedMetrics(b *strings.Builder, metrics []metricInfo) {
	b.WriteString("## Deprecated Metrics\n\n")

	deprecated := filterMetrics(metrics, func(metric metricInfo) bool { return metric.Deprecated != "" })
	if len(deprecated) == 0 {
		b.WriteString("No metrics are marked deprecated.\n\n")
		return
	}

	b.WriteString("Metric | Deprecated since\n")
	b.WriteString("-------|-----------------\n")
	for _, metric := range deprecated {
		fmt.Fprintf(b, "`%s` | `%s`\n", metric.Name, metric.Deprecated)
	}
	b.WriteString("\n")
}

func writeCollectedMetrics(b *strings.Builder, metrics []metricInfo) {
	b.WriteString("## Collected Metrics\n\n")
	b.WriteString("`Labels` lists variable labels; `-` means the metric has no variable labels.\n\n")
	b.WriteString("`Key` is the exporter-metric value accepted by `--disable-metric` and `--enable-metric`.\n\n")
	b.WriteString("Name | Key | Labels | Notes\n")
	b.WriteString("-----|-----|--------|------\n")
	for _, metric := range metrics {
		fmt.Fprintf(b, "`%s` | `%s-%s` | %s | %s\n", metric.Name, metric.Exporter, metric.LocalName, labelsCell(metric.Labels), notesCell(metric))
	}
}

func writePrometheusExamples(b *strings.Builder, examples []metricExample) {
	b.WriteString("\n## Fixture-Backed Prometheus Samples\n\n")
	b.WriteString("These samples are extracted from exporter test expectations backed by\n")
	b.WriteString("JSON fixtures. Each metric family includes its `HELP` / `TYPE` lines and\n")
	b.WriteString("one representative sample line.\n")

	if len(examples) == 0 {
		b.WriteString("\nNo fixture-backed samples were found.\n")
		return
	}

	currentExporter := ""
	for _, example := range examples {
		if example.Exporter != currentExporter {
			currentExporter = example.Exporter
			fmt.Fprintf(b, "\n### %s\n", currentExporter)
		}
		fmt.Fprintf(b, "\n#### `%s`\n\n", example.Name)
		b.WriteString("```text\n")
		for _, line := range example.Lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("```\n")
	}
}

func filterMetrics(metrics []metricInfo, keep func(metricInfo) bool) []metricInfo {
	var filtered []metricInfo
	for _, metric := range metrics {
		if keep(metric) {
			filtered = append(filtered, metric)
		}
	}
	return filtered
}

func labelsCell(labels []string) string {
	if len(labels) == 0 {
		return "-"
	}
	return "`" + strings.Join(labels, ",") + "`"
}

func notesCell(metric metricInfo) string {
	var notes []string
	if metric.Slow {
		notes = append(notes, "slow")
	}
	if metric.Deprecated != "" {
		notes = append(notes, "deprecated since "+metric.Deprecated)
	}
	if metric.DynamicNote != "" {
		notes = append(notes, metric.DynamicNote)
	}
	if len(notes) == 0 {
		return "-"
	}
	return strings.Join(notes, "; ")
}
