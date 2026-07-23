package test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// renderOpt configures a helm template invocation for renderZitadelConfig.
type renderOpt struct {
	set     []string // --set key=value (string-typed)
	setJSON []string // --set-json key=value (typed, e.g. numbers)
}

// renderZitadelConfig renders the chart with the given overrides and returns
// the parsed ZITADEL config from the config ConfigMap. It exercises the chart
// through its public interface (helm template output) so the tests survive
// internal template refactors.
func renderZitadelConfig(t *testing.T, opt renderOpt) map[string]any {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller(0) failed")
	chartPath := filepath.Join(filepath.Dir(file), "..", "charts", "zitadel")

	args := []string{
		"template", "zitadel", chartPath,
		"--kube-version", "1.30.0",
		"--set", "zitadel.masterkey=dGVzdC1tYXN0ZXJrZXktZm9yLXZhbGlkYXRpb24=",
		"--show-only", "templates/configmap_zitadel.yaml",
	}
	for _, s := range opt.set {
		args = append(args, "--set", s)
	}
	for _, s := range opt.setJSON {
		args = append(args, "--set-json", s)
	}

	out, err := exec.Command("helm", args...).CombinedOutput()
	require.NoError(t, err, string(out))

	var cm struct {
		Data struct {
			Config string `yaml:"zitadel-config-yaml"`
		} `yaml:"data"`
	}
	require.NoError(t, yaml.Unmarshal(out, &cm))

	config := map[string]any{}
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data.Config), &config))
	return config
}

// traceExporter navigates to Instrumentation.Trace.Exporter in the rendered
// config, returning the exporter map and whether it was present.
func traceExporter(config map[string]any) (map[string]any, bool) {
	instr, ok := config["Instrumentation"].(map[string]any)
	if !ok {
		return nil, false
	}
	trace, ok := instr["Trace"].(map[string]any)
	if !ok {
		return nil, false
	}
	exporter, ok := trace["Exporter"].(map[string]any)
	return exporter, ok
}

// TestTraceStdoutExporterOmitsEndpoint verifies that a non-endpoint exporter
// (stdOut) does not render an empty Endpoint into the ZITADEL config. An empty
// Endpoint is meaningless for stdOut and pollutes the config.
func TestTraceStdoutExporterOmitsEndpoint(t *testing.T) {
	t.Parallel()

	config := renderZitadelConfig(t, renderOpt{set: []string{
		"instrumentation.trace.enabled=true",
		"instrumentation.trace.exporterType=stdOut",
	}})

	exporter, ok := traceExporter(config)
	require.True(t, ok, "expected Instrumentation.Trace.Exporter to be rendered")

	_, hasEndpoint := exporter["Endpoint"]
	require.False(t, hasEndpoint,
		"stdOut exporter must not render an Endpoint, got: %v", exporter["Endpoint"])
}

// TestTraceGrpcExporterRendersEndpoint verifies that when an endpoint is
// supplied for an endpoint-based exporter (grpc/http), it is rendered into the
// ZITADEL config so ZITADEL knows where to push traces.
func TestTraceGrpcExporterRendersEndpoint(t *testing.T) {
	t.Parallel()

	config := renderZitadelConfig(t, renderOpt{set: []string{
		"instrumentation.trace.enabled=true",
		"instrumentation.trace.exporterType=grpc",
		"instrumentation.trace.endpoint=otel-collector:4317",
	}})

	exporter, ok := traceExporter(config)
	require.True(t, ok, "expected Instrumentation.Trace.Exporter to be rendered")

	require.Equal(t, "otel-collector:4317", exporter["Endpoint"],
		"grpc exporter must render the configured Endpoint")
}

// TestTraceStdoutExporterOmitsInsecure verifies that the Insecure flag, which
// only applies to endpoint-based exporters (grpc/http), is not rendered for a
// non-endpoint exporter like stdOut.
func TestTraceStdoutExporterOmitsInsecure(t *testing.T) {
	t.Parallel()

	config := renderZitadelConfig(t, renderOpt{set: []string{
		"instrumentation.trace.enabled=true",
		"instrumentation.trace.exporterType=stdOut",
	}})

	exporter, ok := traceExporter(config)
	require.True(t, ok, "expected Instrumentation.Trace.Exporter to be rendered")

	_, hasInsecure := exporter["Insecure"]
	require.False(t, hasInsecure,
		"stdOut exporter must not render Insecure, got: %v", exporter["Insecure"])
}

// TestTraceGrpcExporterRendersInsecure verifies that the Insecure flag is
// rendered for endpoint-based exporters, where it controls TLS towards the
// collector.
func TestTraceGrpcExporterRendersInsecure(t *testing.T) {
	t.Parallel()

	config := renderZitadelConfig(t, renderOpt{set: []string{
		"instrumentation.trace.enabled=true",
		"instrumentation.trace.exporterType=grpc",
		"instrumentation.trace.endpoint=otel-collector:4317",
		"instrumentation.trace.insecure=true",
	}})

	exporter, ok := traceExporter(config)
	require.True(t, ok, "expected Instrumentation.Trace.Exporter to be rendered")

	require.Equal(t, true, exporter["Insecure"],
		"grpc exporter must render the configured Insecure flag")
}

// TestTraceFractionRendersAsNumber verifies that a numeric fraction is accepted
// by the schema and rendered into the config. Fraction is typed as a number in
// values.schema.json, so it must be supplied via --set-json.
func TestTraceFractionRendersAsNumber(t *testing.T) {
	t.Parallel()

	config := renderZitadelConfig(t, renderOpt{
		set: []string{
			"instrumentation.trace.enabled=true",
			"instrumentation.trace.exporterType=grpc",
			"instrumentation.trace.endpoint=otel-collector:4317",
		},
		setJSON: []string{
			"instrumentation.trace.fraction=0.5",
		},
	})

	instr, ok := config["Instrumentation"].(map[string]any)
	require.True(t, ok, "expected Instrumentation to be rendered")
	trace, ok := instr["Trace"].(map[string]any)
	require.True(t, ok, "expected Instrumentation.Trace to be rendered")

	require.Equal(t, 0.5, trace["Fraction"],
		"Fraction must be rendered as a number")
}
