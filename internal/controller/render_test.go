package controller

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

func TestRenderConfigNoRemoteWrite(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{}
	out := renderConfig(pc)
	if strings.Contains(out, "remote_write:") {
		t.Fatalf("unexpected remote_write block in output:\n%s", out)
	}
	if !strings.Contains(out, "scrape_configs:") {
		t.Fatalf("missing base scrape_configs:\n%s", out)
	}
}

func TestRenderConfigRemoteWriteBasicAuth(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{
			RemoteWrite: []observabilityv1.RemoteWriteSpec{{
				URL:                "https://thanos.example.com/api/v1/receive",
				Name:               "thanos",
				BasicAuthSecretRef: &corev1.LocalObjectReference{Name: "thanos-creds"},
			}},
		},
	}
	out := renderConfig(pc)
	for _, want := range []string{
		"remote_write:",
		`url: "https://thanos.example.com/api/v1/receive"`,
		`name: "thanos"`,
		"basic_auth:",
		"/etc/prometheus/secrets/thanos-creds/username",
		"/etc/prometheus/secrets/thanos-creds/password",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderConfigInlineAddsConfigMapFile(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{
			AdditionalScrapeConfigs: &observabilityv1.AdditionalScrapeConfigs{
				Inline: "- job_name: my-app\n  static_configs:\n    - targets: ['x:1']\n",
			},
		},
	}
	out := renderConfig(pc)
	if !strings.Contains(out, "/etc/prometheus/additional-scrape-configs.yml") {
		t.Fatalf("missing inline path:\n%s", out)
	}
}

func TestRenderConfigSecretRefAddsSecretMountPath(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{
			AdditionalScrapeConfigs: &observabilityv1.AdditionalScrapeConfigs{
				SecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "s"},
					Key:                  "scrapes.yaml",
				},
			},
		},
	}
	out := renderConfig(pc)
	if !strings.Contains(out, "/etc/prometheus/extra-secret/scrapes.yaml") {
		t.Fatalf("missing secret-mount path:\n%s", out)
	}
}

func TestRenderConfigNoScrapeConfigFilesWhenEmpty(t *testing.T) {
	out := renderConfig(&observabilityv1.PrometheusCluster{})
	if strings.Contains(out, "scrape_config_files:") {
		t.Fatalf("unexpected scrape_config_files block:\n%s", out)
	}
}

func TestWrapScrapeConfigsAddsRequiredKey(t *testing.T) {
	in := "- job_name: my-app\n  static_configs:\n    - targets: ['x:1']\n"
	out := wrapScrapeConfigs(in)
	expected := "scrape_configs:\n  - job_name: my-app\n    static_configs:\n      - targets: ['x:1']\n"
	if out != expected {
		t.Fatalf("unexpected wrap output:\nGOT:\n%s\nWANT:\n%s", out, expected)
	}
}

func TestRenderConfigRemoteWriteBearerToken(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{
			RemoteWrite: []observabilityv1.RemoteWriteSpec{{
				URL:                  "https://mimir.example.com/api/v1/push",
				BearerTokenSecretRef: &corev1.LocalObjectReference{Name: "mimir-token"},
			}},
		},
	}
	out := renderConfig(pc)
	if !strings.Contains(out, "bearer_token_file: /etc/prometheus/secrets/mimir-token/token") {
		t.Errorf("missing bearer_token_file in:\n%s", out)
	}
	if strings.Contains(out, "basic_auth:") {
		t.Errorf("unexpected basic_auth block:\n%s", out)
	}
}
