package main

import (
	"context"
	"fmt"

	"github.com/feloy/operator/pkg/apis"
	cdnclusterv1 "github.com/feloy/operator/pkg/apis/cluster/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {

	config, _ := config.GetConfig()
	clientset, err := client.New(config, client.Options{})
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	cmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cmap", Namespace: "default"},
	}
	err = clientset.Create(context.Background(), cmap)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	apis.AddToScheme(scheme.Scheme)

	cdn := &cdnclusterv1.CdnCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cdn", Namespace: "default"},
		Spec: cdnclusterv1.CdnClusterSpec{
			Role:    "balancer",
			Sources: []cdnclusterv1.CdnClusterSource{},
		},
	}
	err = clientset.Create(context.Background(), cdn)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
}
