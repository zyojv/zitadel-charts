package smoke_test_test

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/mridang/wilhelm/assert"
	setup "github.com/zitadel/zitadel-charts/test/smoke/support"
	"github.com/zitadel/zitadel-charts/test/support"
)

// configMapDataMatches builds a ConfigMapAssertion that asserts the rendered
// ZITADEL config (stored under the "zitadel-config-yaml" key) satisfies the
// given gomega matcher.
func configMapDataMatches(matcher gomega.OmegaMatcher) assert.ConfigMapAssertion {
	return assert.ConfigMapAssertion{
		Data: assert.Matching[map[string]string](
			gomega.HaveKeyWithValue("zitadel-config-yaml", matcher),
		),
	}
}

//goland:noinspection ALL
func TestInstrumentationDisabledByDefault(t *testing.T) {
	t.Parallel()

	support.WithNamespace(t, func(env *support.Env) {
		releaseName := setup.InstallZitadel(t, env, "instr-disabled", nil)

		// With no instrumentation values set, the chart must not render an
		// Instrumentation section into the ZITADEL config.
		env.AssertPartial(t, releaseName+"-config-yaml", configMapDataMatches(
			gomega.Not(gomega.ContainSubstring("Instrumentation:")),
		))
	})
}

//goland:noinspection ALL
func TestInstrumentationMatrix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		setValues map[string]string
		// expect are substrings that must be present in the rendered config.
		expect []string
		// reject are substrings that must NOT be present in the rendered config.
		reject []string
	}{
		{
			name: "trace-http",
			setValues: map[string]string{
				"instrumentation.trace.enabled":      "true",
				"instrumentation.trace.exporterType": "http",
				"instrumentation.trace.endpoint":     "alloy.vm.svc.cluster.local:4318",
				"instrumentation.trace.insecure":     "true",
			},
			expect: []string{
				"Instrumentation:",
				"Trace:",
				"Exporter:",
				"Type: http",
				"Endpoint: alloy.vm.svc.cluster.local:4318",
				"Insecure: true",
			},
		},
		{
			name: "trace-grpc-with-service-name",
			setValues: map[string]string{
				"instrumentation.serviceName":        "zitadel",
				"instrumentation.trace.enabled":      "true",
				"instrumentation.trace.exporterType": "grpc",
				"instrumentation.trace.endpoint":     "otel-collector.monitoring.svc.cluster.local:4317",
			},
			expect: []string{
				"ServiceName: zitadel",
				"Type: grpc",
				"Endpoint: otel-collector.monitoring.svc.cluster.local:4317",
				"Insecure: false",
			},
		},
		{
			name: "trace-fraction-and-batch-duration",
			setValues: map[string]string{
				"instrumentation.trace.enabled":          "true",
				"instrumentation.trace.endpoint":         "otel:4317",
				"instrumentation.trace.batchDuration":    "2s",
				"instrumentation.trace.trustRemoteSpans": "true",
				// Fraction is a number in the schema; --set-string keeps the
				// literal but Helm coerces it back to a numeric YAML scalar.
				"instrumentation.trace.fraction": "0.5",
			},
			expect: []string{
				"BatchDuration: 2s",
				"TrustRemoteSpans: true",
				"Fraction: 0.5",
			},
		},
		{
			name: "user-configmap-config-wins",
			setValues: map[string]string{
				"instrumentation.trace.enabled":  "true",
				"instrumentation.trace.endpoint": "chart.example:4317",
				// A user-supplied configmapConfig value must override the
				// chart-generated instrumentation config.
				"zitadel.configmapConfig.Instrumentation.Trace.Exporter.Endpoint": "user.example:4317",
			},
			expect: []string{
				"Endpoint: user.example:4317",
			},
			reject: []string{
				"Endpoint: chart.example:4317",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			support.WithNamespace(t, func(env *support.Env) {
				releaseName := setup.InstallZitadel(t, env, tc.name, tc.setValues)

				matchers := make([]gomega.OmegaMatcher, 0, len(tc.expect)+len(tc.reject))
				for _, s := range tc.expect {
					matchers = append(matchers, gomega.ContainSubstring(s))
				}
				for _, s := range tc.reject {
					matchers = append(matchers, gomega.Not(gomega.ContainSubstring(s)))
				}

				env.AssertPartial(t, releaseName+"-config-yaml", configMapDataMatches(
					gomega.And(matchers...),
				))
			})
		})
	}
}
