package backup

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// SPDYExecutor implements PodExecutor via the Kubernetes exec subresource.
type SPDYExecutor struct {
	Config    *rest.Config
	Clientset kubernetes.Interface
}

// NewSPDYExecutor builds a real-cluster exec'er.
func NewSPDYExecutor(cfg *rest.Config) (*SPDYExecutor, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes clientset: %w", err)
	}
	return &SPDYExecutor{Config: cfg, Clientset: cs}, nil
}

// Exec runs cmd in the given container and writes stdout into out.
func (e *SPDYExecutor) Exec(ctx context.Context, namespace, pod, container string, cmd []string, out io.Writer) error {
	req := e.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(e.Config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("new executor: %w", err)
	}
	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: out,
		Stderr: io.Discard,
	})
}
