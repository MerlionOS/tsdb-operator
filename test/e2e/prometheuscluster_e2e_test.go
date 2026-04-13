//go:build e2e
// +build e2e

/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MerlionOS/tsdb-operator/test/utils"
)

const (
	testNS      = "tsdb-e2e"
	clusterName = "demo"
)

var _ = Describe("PrometheusCluster lifecycle", Ordered, func() {
	BeforeAll(func() {
		// Make sure CRDs and the operator are present regardless of which
		// Describe ran first. Both commands are idempotent.
		_, err := utils.Run(exec.Command("make", "install"))
		Expect(err).NotTo(HaveOccurred(), "make install (CRDs)")
		_, err = utils.Run(exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage)))
		Expect(err).NotTo(HaveOccurred(), "make deploy (operator)")

		// Wait for the operator to be Ready before exercising CR behavior.
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "-n", "tsdb-operator-system",
				"rollout", "status", "deployment/tsdb-operator-controller-manager",
				"--timeout=10s")
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}, 3*time.Minute, 5*time.Second).Should(Succeed())

		_, _ = utils.Run(exec.Command("kubectl", "create", "ns", testNS))
	})

	AfterAll(func() {
		cmd := exec.Command("kubectl", "delete", "ns", testNS, "--wait=false")
		_, _ = utils.Run(cmd)
	})

	apply := func(manifest string) {
		cmd := exec.Command("kubectl", "apply", "-n", testNS, "-f", "-")
		cmd.Stdin = strings.NewReader(manifest)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "apply failed: %s", manifest)
	}

	getField := func(resource, name, jsonpath string) string {
		cmd := exec.Command("kubectl", "-n", testNS, "get", resource, name,
			"-o", fmt.Sprintf("jsonpath=%s", jsonpath))
		out, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		return out
	}

	It("provisions StatefulSet + Service + ConfigMap and reaches Active", func() {
		apply(fmt.Sprintf(`
apiVersion: observability.merlionos.org/v1
kind: PrometheusCluster
metadata:
  name: %s
spec:
  replicas: 1
  storage:
    size: 1Gi
`, clusterName))

		Eventually(func(g Gomega) {
			g.Expect(getField("statefulset", clusterName, "{.status.readyReplicas}")).To(Equal("1"))
			g.Expect(getField("configmap", clusterName+"-config", "{.data.prometheus\\.yml}")).NotTo(BeEmpty())
			g.Expect(getField("service", clusterName, "{.spec.clusterIP}")).To(Equal("None"))
			g.Expect(getField("prometheuscluster", clusterName, "{.status.phase}")).To(Equal("Active"))
		}, 3*time.Minute, 3*time.Second).Should(Succeed())
	})

	It("scales when spec.replicas changes", func() {
		cmd := exec.Command("kubectl", "-n", testNS, "patch", "prometheuscluster", clusterName,
			"--type=merge", "-p", `{"spec":{"replicas":2}}`)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			g.Expect(getField("statefulset", clusterName, "{.spec.replicas}")).To(Equal("2"))
		}, 2*time.Minute, 3*time.Second).Should(Succeed())
	})

	It("toggles --web.enable-admin-api when backup is enabled", func() {
		cmd := exec.Command("kubectl", "-n", testNS, "patch", "prometheuscluster", clusterName,
			"--type=merge", "-p", `{"spec":{"backup":{"enabled":true,"bucket":"x","schedule":"0 0 * * *"}}}`)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			args := getField("statefulset", clusterName, "{.spec.template.spec.containers[0].args}")
			g.Expect(args).To(ContainSubstring("--web.enable-admin-api"))
		}, 2*time.Minute, 3*time.Second).Should(Succeed())
	})

	It("runs finalizer on delete and fully removes the CR", func() {
		cmd := exec.Command("kubectl", "-n", testNS, "delete", "prometheuscluster", clusterName, "--wait=true")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "-n", testNS, "get", "prometheuscluster", clusterName)
			_, err := utils.Run(cmd)
			g.Expect(err).To(HaveOccurred(), "CR should be gone after finalizer runs")
		}, 1*time.Minute, 2*time.Second).Should(Succeed())
	})
})
