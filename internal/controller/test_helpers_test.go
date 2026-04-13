package controller

import (
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

type runtimeObject = client.Object

func toClientObjects(objs []runtimeObject) []client.Object {
	out := make([]client.Object, 0, len(objs))
	out = append(out, objs...)
	return out
}

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = observabilityv1.AddToScheme(s)
	return s
}
