/*
Copyright 2018 Anevia.
*/

package cdncluster

import (
	"context"
	"log"
	"reflect"

	clusterv1 "github.com/feloy/operator/pkg/apis/cluster/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ParentsList map[string][]string

func (o ParentsList) Add(source, parent string) {
	for _, p := range o[source] {
		if p == parent {
			return
		}
	}
	o[source] = append(o[source], parent)
}

var (
	// Parents is a map of parent CDN clusters for a CDN cluster
	Parents = ParentsList{}
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CdnCluster Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this cluster.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr, mgr.GetRecorder("CdnCluster")))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, recorder record.EventRecorder) reconcile.Reconciler {
	return &ReconcileCdnCluster{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: recorder,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cdncluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to CdnCluster
	err = c.Watch(&source.Kind{Type: &clusterv1.CdnCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by CdnCluster - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &clusterv1.CdnCluster{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.CdnCluster{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(mapObject handler.MapObject) []reconcile.Request {
				v, ok := mapObject.Object.(*clusterv1.CdnCluster)
				if ok {
					var res = []reconcile.Request{}
					for _, parent := range Parents[v.Name] {
						res = append(res, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      parent,
								Namespace: v.Namespace,
							},
						})
					}
					return res
				}
				return nil
			}),
		})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCdnCluster{}

// ReconcileCdnCluster reconciles a CdnCluster object
type ReconcileCdnCluster struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a CdnCluster object and makes changes based on the state read
// and what is in the CdnCluster.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.anevia.com,resources=cdnclusters,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileCdnCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the CdnCluster instance
	instance := &clusterv1.CdnCluster{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	for _, source := range instance.Spec.Sources {
		Parents.Add(source.Name, instance.Name)
	}

	// Verify that all sources exist
	// We do not continue until all sources exist
	for _, source := range instance.Spec.Sources {
		sourceInstance := &clusterv1.CdnCluster{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: source.Name, Namespace: instance.Namespace}, sourceInstance)
		if err != nil {
			if errors.IsNotFound(err) {
				// Source not found, inform with an event and return.
				r.recorder.Eventf(instance, "Normal", "SourceNotFound", "Source %s not found, will retry later", source.Name)
				return reconcile.Result{}, nil
			}
			// Error reading the object - requeue the request.
			return reconcile.Result{}, err
		}
	}

	// TODO(user): Change this to be the object type created by your controller
	// Define the desired Deployment object
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-deployment",
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"deployment": instance.Name + "-deployment"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"deployment": instance.Name + "-deployment"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// TODO(user): Change this for the object type created by your controller
	// Check if the Deployment already exists
	found := &appsv1.Deployment{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating Deployment %s/%s\n", deploy.Namespace, deploy.Name)
		err = r.Create(context.TODO(), deploy)
		if err != nil {
			return reconcile.Result{}, err
		}
		r.recorder.Eventf(instance, "Normal", "DeploymentCreated", "The Deployment %s has been created", deploy.Name)
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// TODO(user): Change this for the object type created by your controller
	// Update the found object and write the result back if there are any changes
	if !reflect.DeepEqual(deploy.Spec, found.Spec) {
		found.Spec = deploy.Spec
		log.Printf("Updating Deployment %s/%s\n", deploy.Namespace, deploy.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
