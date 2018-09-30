# Pre-requisites

Get kubebuilder stable release:

```
curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v1.0.4/kubebuilder_1.0.4_linux_amd64.tar.gz
```

Install go and prepare your gopath and project:

```
export GOPATH=$HOME/projects/go
mkdir -p $GOPATH/src/github.com/feloy/operator && cd $_
```

Install dep:
```
go get -u github.com/golang/dep/cmd/dep
```

Add $HOME/projects/go/bin to your PATH

# Create the project

Initialize the project:

```
kubebuilder init --domain anevia.com --license none --owner Anevia
```

At this point, you get a project template with a `Makefile`, a `Dockerfile` a basic `manager` and some default yaml files.

`6a22201`

# API

The role of a Custom Resource Definition is to define a type of object so the user can create instances of this type to declare what he wants.

You will see later that the role of the operator is to read in these objects and to operate on the cluster to do what is described in the objects.

## Create a custom resource

```
kubebuilder create api --group cluster --version v1 --kind CdnCluster
```
This will create a resource under `pkg/apis` and an operator under `pkg/controller`.

The created files are:

- the **generated** CRD in yaml format:
  ```
  config/crds/cluster_v1_cdncluster.yaml
  ```

- the **generated** role and binding necessary for operator execution in the cluster:
  ```
  config/rbac/rbac_role.yaml
  config/rbac/rbac_role_binding.yaml
  ```

- a **generated** sample custom resource
  ```
  config/samples/cluster_v1_cdncluster.yaml
  ```

- the **sources** for the new custom resource:
  ```
  pkg/apis/
  ├ addtoscheme_cluster_v1.go
  ├ apis.go
  └ cluster
   ├ group.go
   └ v1
     ├ cdncluster_types.go # the structure definition
     ├ cdncluster_types_test.go # testing the structure
     ├ doc.go
     ├ register.go
     ├ v1_suite_test.go
     └ zz_generated.deepcopy.go
  ```

- the **sources** for the operator:
  ```
  pkg/controller/
  ├ add_cdncluster.go
  ├ cdncluster
  │ ├ cdncluster_controller.go # the reconcile function
  │ ├ cdncluster_controller_suite_test.go
  │ └ cdncluster_controller_test.go # testing the reconcile func
  └ controller.go
  ```

`0e342f1`

## Deploying the sample Custom resource definition

Verify no CRD is deployed:
```
kubectl get crd
```

Deploy CRD:
```
$ make install
CRD manifests generated under '.../config/crds' 
RBAC manifests generated under '.../config/rbac' 
kubectl apply -f config/crds
customresourcedefinition.apiextensions.k8s.io/cdnclusters.cluster.anevia.com created
$ kubectl get crd
cdnclusters.cluster.anevia.com   5s
```

Create a new instance of the custom resource with the provided sample:
```
$ kubectl get cdncluster.cluster.anevia.com
No resources found.
$ kubectl apply -f config/samples/cluster_v1_cdncluster.yaml
cdncluster.cluster.anevia.com "cdncluster-sample" created
$ kubectl get cdncluster.cluster.anevia.com
NAME                AGE
cdncluster-sample   5s
```

You can now delete it:
```
$ kubectl delete cdncluster.cluster.anevia.com cdncluster-sample
cdncluster.cluster.anevia.com "cdncluster-sample" deleted
```

## Customizing the custom resource definition

You can customize the CRD by editing the file `pkg/apis/cluster/v1/cdncluster_types.go`.

The specs part is editable in the `CdnClusterSpec` structure while the status part is editable in the `CdnClusterStatus` one.

Let's add a `Role` field in the specs, and a `State` field in the status:

```go
// CdnClusterSpec defines the desired state of CdnCluster
type CdnClusterSpec struct {
    // Role of the CDN cluster, can be 'balancer' or 'cache'
    Role string `json:"role"`
}

// CdnClusterStatus defines the observed state of CdnCluster
type CdnClusterStatus struct {
    // State of the CDN cluster
    State string `json:"state"`
}
```

Note that fields must have json tags.

You can re-generate the yaml files used to deploy the CRD, and examine the differences:
```diff
$ make manifests
$ git diff config/crds/cluster_v1_cdncluster.yaml 
diff --git a/config/crds/cluster_v1_cdncluster.yaml b/config/crds/cluster_v1_cdncluster.yaml
index 8d0dcbb..fe0efaf 100644
--- a/config/crds/cluster_v1_cdncluster.yaml
+++ b/config/crds/cluster_v1_cdncluster.yaml
@@ -21,8 +21,18 @@ spec:
         metadata:
           type: object
         spec:
+          properties:
+            role:
+              type: string
+          required:
+          - role
           type: object
         status:
+          properties:
+            state:
+              type: string
+          required:
+          - state
           type: object
       type: object
   version: v1
```

You can see that the `role` and `state` properties have been added to the definition of the CRD, and are marked as **required**.

`55f03d9`

## Making a field not required

If you want a field to be not required, you can use the `omitempty` flag in the json tag associated with this field:
```go
// CdnClusterStatus defines the observed state of CdnCluster
type CdnClusterStatus struct {
    State string `json:"state,omitempty"`
}
```

then re-generate the manifests again:
```diff
$ make manifests
$ git diff config/crds/cluster_v1_cdncluster.yaml 
diff --git a/config/crds/cluster_v1_cdncluster.yaml b/config/crds/cluster_v1_cdncluster.yaml
index fe0efaf..f663eba 100644
--- a/config/crds/cluster_v1_cdncluster.yaml
+++ b/config/crds/cluster_v1_cdncluster.yaml
@@ -31,8 +31,6 @@ spec:
           properties:
             state:
               type: string
-          required:
-          - state
           type: object
       type: object
   version: v1
```

The `state` field is not required anymore.

`f160593`

## Completing the Custom resource definition

We want our CDN clusters to redirect requests to *source clusters* depending on some condition on the path of the requested URL. For this, we add a list of `sources` to the definition of a CDN cluster and a source is defined by the name of the source CDN cluster and the path condition to redirect to this cluster.

The list of sources cannot be omitted (but can be an empty array), and a path condition can be omitted, in the case of a default source cluster (the one selected if no other path condition in other sources matches):
```go
// CdnClusterSource defines a source cluster of a cluster
type CdnClusterSource struct {
    // The name of the source cluster
    Name string `json:"name"`
    // The path condition to enter this cluster,
    // can be omitted for the default source
    PathCondition string `json:"pathCondition,omitempty"`
}

// CdnClusterSpec defines the desired state of CdnCluster
type CdnClusterSpec struct {
    // Role must be 'balancer' or 'cache'
    Role string `json:"role"`
    // Sources is the list of source clusters for this cluster
    Sources []CdnClusterSource `json:"sources"`
}
```

Then re-egnerate the CRD and install it:

```
make manifests
make install
```

`ef17bc4`

## Creating sample custom resource instances

Here, we create in `config/samples/cluster_v1_cdncluster.yaml` three instances of CDN clusters. A first instance of balancers, which will have two sources, one cluster of caches for Live requests and another for VOD requests:

```yaml
apiVersion: cluster.anevia.com/v1
kind: CdnCluster
metadata:
  name: balancer
spec:
  role: balancer
  sources:
  - name: cache-live
    pathCondition: ^/live/
  - name: cache-vod
    pathCondition: ^/vod/

---

apiVersion: cluster.anevia.com/v1
kind: CdnCluster
metadata:
  name: cache-live
spec:
  role: cache
  sources: []

---

apiVersion: cluster.anevia.com/v1
kind: CdnCluster
metadata:
  name: cache-vod
spec:
  role: cache
  sources: []
```

To deploy the instances: 
```
$ kubectl apply -f config/samples/cluster_v1_cdncluster.yaml 
cdncluster.cluster.anevia.com "balancer" created
cdncluster.cluster.anevia.com "cache-live" created
cdncluster.cluster.anevia.com "cache-vod" created
```

`285d502`

## Testing the creation of CdnCluster instances

```
$ make test
... spec.sources in body must be of type array: "null" ...
```

In the tests, we create a CDN cluster with this command:

```go
created := &CdnCluster{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}
```

In Go, an omitted field in a struct is equivalent to its zero value, so the precedent instruction is equivalent to:
```go
created := &CdnCluster{
  ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
  Spec: CdnClusterSpec{
    Role: "",
    Sources: nil,
  },
}
```

The Kubernetes API does not accept a nil value for the Sources with an array type; you have to define the sources with an empty array, for example:
```go
created := &CdnCluster{
  ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
  Spec: CdnClusterSpec{
    Role: "",
    Sources: []CdnClusterSource{},
  },
}
```
or with a more complete specification:
```go
created := &CdnCluster{
    ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
    Spec: CdnClusterSpec{
        Role: "balancer",
        Sources: []CdnClusterSource{
            {
                Name:          "cache-live",
                PathCondition: "^/live/",
            },
            {
                Name:          "cache-vod",
                PathCondition: "^/vod/",
            },
        },
    },
}
  ```

This time, the tests should pass:
```
$ make test
ok  	operator/pkg/apis/cluster/v1
```

`671d726`

## Creating custom resources with the controller-runtime client

At this time, you can create new CDN clusters with YAML files and the `kubectl` command. If you want to create new CDN clusters
from a Go application, you will need a specific clientset for this resource.

The kubebuilder team has created a client to work with Kubernetes entities, the controller-runtime client.

In the following Go program, we can see how to use the client to create a native ConfigMap, and a custom CdnCluster:

```go
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

  // Create the clientset
  config, _ := config.GetConfig()
  clientset, err := client.New(config, client.Options{})
  if err != nil {
    fmt.Printf("%s\n", err.Error())
  }

  // Create a native ConfigMap
  cmap := &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{Name: "cmap", Namespace: "default"},
  }
  err = clientset.Create(context.Background(), cmap)
  if err != nil {
    fmt.Printf("%s\n", err.Error())
  }

  // Add the cluster scheme to the current scheme
  apis.AddToScheme(scheme.Scheme)

  // Create a CdnCluster
  cdn := &cdnclusterv1.CdnCluster{
    ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
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
```

`6277280`

# Operator

The role of the operator is to read in *Custom Resources* what is declared by the user and operate the cluster to do what is described in these custom resources.

## The Reconcile function

The kubebuilder created for us a basic operator that creates a Deployment deploying an `nginx` container for each custom resource created.

The main part that you have to change is the `Reconcile` function defined in the `pkg/apis/cdncluster_controller.go` file.

The Reconcile function is called every time a change occurs in the cluster that could interest the operator, and is called with a parameter containing the name and namespace of a custom resource.

The job of the Reconcile function is to read what is expected by the user in the custom resource *Specs* and to make changes in the cluster to reflect these expectations. Another role of the Reconcile function is to keep up to date the *Status* part of the custom resource.

The different steps:
- get the `CdnCluster` custom resource the reconcile function is called for:
  ```go
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
  ```
  - if the CdnCluster resource is not present anymore, ignore the call to the Reconcile function: there is nothing special to do when a resource is deleted, because the garbage collector will handle the deletion
  - if an error occurred, try again later.
- create in memory the objects the operator would like to deploy, here a Deployment:
  ```go
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
- define the owner of the created objects (here, the Deployment) to the CdnCluster resource, so the garbage collector can handle the deletion of these objects when the CdnCluster is deleted:

  ```go
  if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
    return reconcile.Result{}, err
  }
  ```
- try to find the objects (here, the deployment) in the Kubernetes cluster, by namespace and name:
  ```go
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
  } else if err != nil {
    return reconcile.Result{}, err
  }
  ```
  - if the objects are not found in the cluster, create them,
  - if an error occurred during finding or creating the objects, try again later.
- compare the specs of the objects found in the cluster with the specs of the objects the operator would like to deploy:
  ```go
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
  ```
  - if a difference is found, update the objects deployed in the cluster with the object the operator want to deploy,
  - if an error occurred during updating, try again later.

## Defining on which events to call the Reconcile function

It is your responsability to define on which events the Reconcile function will be called.

For this, you can update the `add` function in the `cdncluster_controller.go` file.

The sample controller already defines that the Reconcile function will be called:
- every time a `CdnCluster` resource is modified, using the `handler.EnqueueRequestForObject` event handler:
  ```go
  // Watch for changes to CdnCluster
  err = c.Watch(&source.Kind{Type: &clusterv1.CdnCluster{}}, &handler.EnqueueRequestForObject{})
  ```
  In this case, the namespace and name passed as argument to the Reconcile function will be those of the `CdnCluster` itself.
- every time a Deployment owned by a `CdnCluster` is modified, using the `handler.EnqueueRequestForOwner` event handler:
  ```go
  // watch for Deployment created by CdnCluster changes
  err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
    IsController: true,
    OwnerType:    &clusterv1.CdnCluster{},
  })
  ```
  In this case, the namespace and name passed as argument to the Reconcile function will be those of the CdnCluster owning the Deployment.

## Testing the Reconcile function

A framework is provided by kubebuilder to test the Reconcile function.

The framework is based on the `controller-runtime` `envtest` framework: https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/envtest.

The `TestMain` function defined in `cdncluster_controller_suite_test.go` file will be the only function called when running the `go test` command.

This function will first start a Kubernetes test environment and install our CRD on it,
then run all the tests before to stop the test environment.

Each test has to prepare the environment with the following code:
```go
func TestReconcile(t *testing.T) {
  g := gomega.NewGomegaWithT(t)

  // Setup the Manager and Controller.
  // Wrap the Controller Reconcile function
  // so it writes each request to a channel when it is finished.
  mgr, err := manager.New(cfg, manager.Options{})
  g.Expect(err).NotTo(gomega.HaveOccurred())
  c := mgr.GetClient()

  recFn, requests := SetupTestReconcile(newReconciler(mgr))
  g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())
  defer close(StartTestManager(mgr, g))
}
```

It is for example possible to create some object in Kubernetes and test that the Reconcile function is called with the expected parameter:

```go
instance := &clusterv1.CdnCluster{
  ObjectMeta: metav1.ObjectMeta{
    Name: "foo",
    Namespace: "default",
  },
  Spec: clusterv1.CdnClusterSpec{
    Sources: []clusterv1.CdnClusterSource{},
  },
}

// Create the CdnCluster object
// and expect the Reconcile to be called
// with the instance namespace and name as parameter
err = c.Create(context.TODO(), instance)
g.Expect(err).NotTo(gomega.HaveOccurred())
defer c.Delete(context.TODO(), instance)

var expectedRequest = reconcile.Request{
  NamespacedName: types.NamespacedName{
    Name: "foo",
    Namespace: "default",
  },
}
const timeout = time.Second * 5

g.Eventually(requests, timeout)
 .Should(gomega.Receive(gomega.Equal(expectedRequest)))
```

It is also possible to test that, when a `CdnCluster` is created, that a
Deployment is created with the expected name:
```go
// Expect that a Deployment is created
deploy := &appsv1.Deployment{}
var depKey = types.NamespacedName{
  Name:      "foo-deployment",
  Namespace: "default",
}
g.Eventually(func() error {
  return c.Get(context.TODO(), depKey, deploy)
}, timeout).Should(gomega.Succeed())
```

Let's now test that, when a Deployment created by a CdnCluster is deleted, the Reconcile function is called and the Deployment is created again:
```go
// Delete the Deployment and expect 
// Reconcile to be called for Deployment deletion
// and Deployment to be created again
g.Expect(c.Delete(context.TODO(), deploy)).NotTo(gomega.HaveOccurred())
g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
g.Eventually(func() error {
  return c.Get(context.TODO(), depKey, deploy)
}, timeout).Should(gomega.Succeed())
```

`8d10c42`

## Adding dependencies between CdnCluster

As defined in the `CdnClusterSpec`, a CdnCluster relies on sources. We would like the operator to wait that the sources of a CdnCluster are created before it creates the CdnCluster itself.

The different things to modify in the operator are:
- define a structure to store the dependencies between CDN clusters:
  ```go
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
  ```
- in the `Reconcile` function, if one of the source `CdnCluster` does not exist in the Kubernetes cluster yet, do not continue,
  ```go
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
        // Source not found, return.
        return reconcile.Result{}, nil
      }
      // Error reading the object - requeue the request.
      return reconcile.Result{}, err
    }
  }
  ```
- in the `add` function, indicate that when a CdnCluster is modified, we want to call the Reconcile function for the CdnClusters which define this modified CdnCluster as a source. For this, we use the `handler.EnqueueRequestsFromMapFunc` event handler, that can be used to return an arbitrary list of objects:
  ```go
  func add(mgr manager.Manager, r reconcile.Reconciler) error {
    [...]
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
  ```
  In this case, the namespace and name passed as argument to the Reconcile function will be those of the CDN clusters parent of the modified CDN cluster.

`c39cd61`

## Testing the CdnCluster dependencies

```go
func TestReconcileCreatedAfterSource(t *testing.T) {
  g := gomega.NewGomegaWithT(t)

  // Setup the Manager and Controller.
  // Wrap the Controller Reconcile function
  // so it writes each request to a channel when it is finished.
  mgr, err := manager.New(cfg, manager.Options{})
  g.Expect(err).NotTo(gomega.HaveOccurred())
  c := mgr.GetClient()
  recFn, requests := SetupTestReconcile(newReconciler(mgr))
  g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())
  defer close(StartTestManager(mgr, g))

  // Create the CdnCluster object
  // and expect the Reconcile to be called
  // with the instance namespace and name as parameter
  instanceParent := &clusterv1.CdnCluster{
    ObjectMeta: metav1.ObjectMeta{
      Name:      "foo3",
      Namespace: "default",
    },
    Spec: clusterv1.CdnClusterSpec{
      Sources: []clusterv1.CdnClusterSource{
        {
          Name:          "asource",
          PathCondition: "/live/",
        },
      },
    },
  }
  err = c.Create(context.TODO(), instanceParent)
  g.Expect(err).NotTo(gomega.HaveOccurred())
  defer c.Delete(context.TODO(), instanceParent)
  var expectedRequest = reconcile.Request{
    NamespacedName: types.NamespacedName{
  	  Name:      "foo3",
  	  Namespace: "default",
    },
  }
  const timeout = time.Second * 5
  g.Eventually(requests, timeout)
   .Should(gomega.Receive(gomega.Equal(expectedRequest)))

  // Expect that a Deployment is not created
  deploy := &appsv1.Deployment{}
  var depKey = types.NamespacedName{
    Name:      "foo3-deployment",
    Namespace: "default",
  }
  g.Eventually(func() error {
    return c.Get(context.TODO(), depKey, deploy)
  }, timeout).ShouldNot(gomega.Succeed())

  // Create the CdnCluster object
  // and expect the Reconcile to be called
  // with the instance namespace and name as parameter
  instanceSource := &clusterv1.CdnCluster{
    ObjectMeta: metav1.ObjectMeta{
      Name:      "asource",
      Namespace: "default",
    },
    Spec: clusterv1.CdnClusterSpec{
      Sources: []clusterv1.CdnClusterSource{},
    },
  }
  err = c.Create(context.TODO(), instanceSource)
  g.Expect(err).NotTo(gomega.HaveOccurred())
  defer c.Delete(context.TODO(), instanceSource)
  var expectedRequestSource = reconcile.Request{
    NamespacedName: types.NamespacedName{
      Name:      "asource",
      Namespace: "default",
    },
  }
  g.Eventually(requests, timeout)
   .Should(gomega.Receive(gomega.Equal(expectedRequestSource)))

  // Expect that a Deployment is created
  deploy = &appsv1.Deployment{}
  depKey = types.NamespacedName{
    Name:      "asource-deployment",
    Namespace: "default",
  }
  g.Eventually(func() error {
    return c.Get(context.TODO(), depKey, deploy)
  }, timeout).Should(gomega.Succeed())

  // Expect the Reconcile function to be called for the parent cluster
  g.Eventually(requests, timeout)
   .Should(gomega.Receive(gomega.Equal(expectedRequest)))

  // Expect that a Deployment is created
  deploy = &appsv1.Deployment{}
  depKey = types.NamespacedName{
    Name:      "foo3-deployment",
    Namespace: "default",
  }
  g.Eventually(func() error {
    return c.Get(context.TODO(), depKey, deploy)
  }, timeout).Should(gomega.Succeed())
}
```
`9a3f67d`

## Sending Events

The Operator can send events and attach them to CdnCluster instances.

First, add a `record.EventRecorder` field in the `ReconcileCdnCluster` struct, that will reference the recorder
the operator will use to send events:
```go
// ReconcileCdnCluster reconciles a CdnCluster object
type ReconcileCdnCluster struct {
  client.Client
  scheme   *runtime.Scheme
  recorder record.EventRecorder
}
```
Second, get the recorder from the `Manager`.

To be able to fake the recorder, we can pass a recorder as argument to the newReconciler method. This way, we will be able to pass the mgr.GetRecorder("CdnCluster") one during real execution, and a fake recorder during tests:

```go
func Add(mgr manager.Manager) error {
  return add(mgr, newReconciler(mgr, mgr.GetRecorder("CdnCluster")))
 }

func newReconciler(mgr manager.Manager, recorder record.EventRecorder) reconcile.Reconciler {
  return &ReconcileCdnCluster{
    Client:   mgr.GetClient(),
    scheme:   mgr.GetScheme(),
    recorder: recorder,
  }
}
```
Finally, use `Event` or `Eventf` methods on this recorder to send events; one when the Reconcile function returns because sources are not ready:
```go
if errors.IsNotFound(err) {
  // Source not found, inform with an event and return.
  r.recorder.Eventf(instance, "Normal", "SourceNotFound", "Source %s not found, will retry later", source.Name)
  return reconcile.Result{}, nil
}
```
another one when the Reconcile function succeeds to create the deployment:
```go
log.Printf("Creating Deployment %s/%s\n", deploy.Namespace, deploy.Name)
err = r.Create(context.TODO(), deploy)
if err != nil {
  return reconcile.Result{}, err
}
r.recorder.Eventf(instance, "Normal", "DeploymentCreated", "The Deployment %s has been created", deploy.Name)
```

## A demo with Events

Start the operator from outside the cluster:
```
make run
```
Then, on another terminal:
```
$ cd config/samples

$ kubectl apply -f balancer.yaml 
cdncluster.cluster.anevia.com "balancer" created

$ kubectl describe cdncluster
Name:         balancer
Namespace:    default
[...]
Events:
  Type    Reason          Age   From        Message
  ----    ------          ----  ----        -------
  Normal  SourceNotFound  3s    CdnCluster  Source cache-live not found, will retry later



$ kubectl apply -f cache-live.yaml 
cdncluster.cluster.anevia.com "cache-live" created

$ kubectl describe cdncluster
Name:         balancer
Namespace:    default
[...]
Events:
  Type    Reason          Age   From        Message
  ----    ------          ----  ----        -------
  Normal  SourceNotFound  24s   CdnCluster  Source cache-live not found, will retry later
  Normal  SourceNotFound  3s    CdnCluster  Source cache-vod not found, will retry later

Name:         cache-live
Namespace:    default
[...]
Events:
  Type    Reason             Age   From        Message
  ----    ------             ----  ----        -------
  Normal  DeploymentCreated  3s    CdnCluster  The Deployment cache-live-deployment has been created



$ kubectl describe cdncluster
Name:         balancer
Namespace:    default
[...]
Events:
  Type    Reason             Age   From        Message
  ----    ------             ----  ----        -------
  Normal  SourceNotFound     34s   CdnCluster  Source cache-live not found, will retry later
  Normal  SourceNotFound     13s   CdnCluster  Source cache-vod not found, will retry later
  Normal  DeploymentCreated  2s    CdnCluster  The Deployment balancer-deployment has been created


Name:         cache-live
Namespace:    default
[...]
Events:
  Type    Reason             Age   From        Message
  ----    ------             ----  ----        -------
  Normal  DeploymentCreated  13s   CdnCluster  The Deployment cache-live-deployment has been created


Name:         cache-vod
Namespace:    default
[...]
Events:
  Type    Reason             Age   From        Message
  ----    ------             ----  ----        -------
  Normal  DeploymentCreated  2s    CdnCluster  The Deployment cache-vod-deployment has been created

```

## Setting State of CDN clusters

We previoulsy added a `State` field in the `CdnClusterStatus` structure, in the `pkg/apis/cluster/v1/cdncluster_types.go` file:

```go
// CdnClusterStatus defines the observed state of CdnCluster
type CdnClusterStatus struct {
  // State of the CDN cluster
  State string `json:"state,omitempty"`
}
```

We can create a method to set the state of a CDN cluster:
```go
// setState changes the State of CDN cluster, if necessary
func (r *ReconcileCdnCluster) setState(cdncluster *clusterv1.CdnCluster, newState string) error {
  if cdncluster.Status.State != newState {
    cdncluster.Status.State = newState
    return r.Update(context.TODO(), cdncluster)
  }
  return nil
}
```

and use this method in the Reconcile function, for rxample to set the state as `WaitingSource`:
```go
// Source not found, inform with an event and return.
r.recorder.Eventf(instance, "Normal", "SourceNotFound", "Source %s not found, will retry later", source.Name)
err = r.setState(instance, "WaitingSource")
if err != nil {
  return reconcile.Result{}, err
}
```

## Testing that the State is set

In the test, after the Deployment has been created, we can test that the state has been set:
```go
// Expect that a Deployment is created
deploy := &appsv1.Deployment{}
var depKey = types.NamespacedName{
  Name:      "foo2-deployment",
  Namespace: "default",
}
g.Eventually(func() error {
  return c.Get(context.TODO(), depKey, deploy)
}, timeout).Should(gomega.Succeed())

// Get CDN cluster and expect the state to be "Deploying"
c.Get(context.TODO(), types.NamespacedName{
  Name:      "foo2",
  Namespace: "default",
}, instance)
g.Expect(instance.Status.State).To(gomega.Equal("Deploying"))
```

and when waiting for sources, that the state is correct:
```go
// Get CDN cluster from cluster and expect the state to be "WaitingSource"
c.Get(context.TODO(), types.NamespacedName{
  Name:      "foo3",
  Namespace: "default",
}, instanceParent)
g.Expect(instanceParent.Status.State).To(gomega.Equal("WaitingSource"))

// Expect that a Deployment is not created
deploy := &appsv1.Deployment{}
var depKey = types.NamespacedName{
  Name:      "foo3-deployment",
  Namespace: "default",
}
```

## Testing that an event is sent

We previously declared the reconciler recorder in the newReconciler function, with mgr.GetRecorder("CdnCluster").

To be able to fake the recorder, we can instead pass a recorder as argument to the newReconciler method. This way, we will be able to pass the mgr.GetRecorder("CdnCluster") one during real execution, and a fake recorder during tests:

```diff
func Add(mgr manager.Manager) error {
- return add(mgr, newReconciler(mgr))
+ return add(mgr, newReconciler(mgr, mgr.GetRecorder("CdnCluster")))
 }

-func newReconciler(mgr manager.Manager) reconcile.Reconciler {
+func newReconciler(mgr manager.Manager, recorder record.EventRecorder) reconcile.Reconciler {
  return &ReconcileCdnCluster{
    Client:   mgr.GetClient(),
    scheme:   mgr.GetScheme(),
-   recorder: mgr.GetRecorder("CdnCluster"),
+   recorder: recorder,
  }
}
 ```

 and in the tests:
 ```diff
- recFn, requests := SetupTestReconcile(newReconciler(mgr))
+ eventRecorder := record.NewFakeRecorder(1024)
+ recFn, requests := SetupTestReconcile(newReconciler(mgr, eventRecorder))
 ```

 Now, using the `NewFakeRecorder` provider provided by the `record` package, an `Event` channel is accessible in this recorder where the events are stored and from which we can get emitted events:

```go
var eventReceived string
select {
case eventReceived = <-eventRecorder.Events:
}
g.Expect(eventReceived).To(gomega.Equal("Normal SourceNotFound Source asource not found, will retry later"))
```