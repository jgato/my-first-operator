# My first operator

Just playing/learning to create a K8S operator in go.

I will create a dummy operator that creates pods to open a shell inside

It is important to use an operator-sdk and golang version compatible. In this case:
 * Go 1.16
 * Operator-sdk 1.16
 
This is pretty different from the book I was reading. So it has changed a lot. First point the GOPATH to something like:

```
/home/jgato/Projects-src/playing/kubernetes/src/
```
From here you can create your fist operator

## Creating the template

```
mkdir $GOPATH/my-first-operator
cd $GOPATH/my-first-operator
operator-sdk init --domain jgato.io  --repo github.com/jgato/my-first-operator --plugins go/v3
```
 * domain: it is your apis domain
 * repo: used by the go.mod to refer your module
 * the plugin points to a go operator, instead of a helm or ansible one. Depending on the version the scafolding structure would be different

```
tree
.
├── config
│   ├── default
│   │   ├── kustomization.yaml
│   │   ├── manager_auth_proxy_patch.yaml
│   │   └── manager_config_patch.yaml
│   ├── manager
│   │   ├── controller_manager_config.yaml
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── manifests
│   │   └── kustomization.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   └── monitor.yaml
│   ├── rbac
│   │   ├── auth_proxy_client_clusterrole.yaml
│   │   ├── auth_proxy_role_binding.yaml
│   │   ├── auth_proxy_role.yaml
│   │   ├── auth_proxy_service.yaml
│   │   ├── kustomization.yaml
│   │   ├── leader_election_role_binding.yaml
│   │   ├── leader_election_role.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   └── scorecard
│       ├── bases
│       │   └── config.yaml
│       ├── kustomization.yaml
│       └── patches
│           ├── basic.config.yaml
│           └── olm.config.yaml
├── Dockerfile
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
├── main.go
├── Makefile
└── PROJECT

```

In the main.go you will see how the operator is created. Actually, it creates the manager that will connect to the cluster, and provides some interfaces for metrics and health: main.go

```
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "a9b2e162.jgato.io",
	})
```

The manager it is in charge of enabling the Controller. Each controller is in charge of watching one or many resources. In our case, we only have one controller in charge of MyOwnShell resource and the resources derivated. In the controllers file you will have something like:

```
func (r *MyOwnShellReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myfirstoperatorv1alpha1.MyOwnShell{}).
		Complete(r)
}
```
Here, it is only watching for myfirstoperatorv1alpha1.MyOwnShell resources (created later). But we also add other resources to watch, like the deployments created by that resource. The manager is in charge of loading that controller.


## Adding APIs and CRs

Now you can add different APIS for your CRs:

```
operator-sdk create api --group my-first-operator --version v1alpha1 --kind MyOwnShell  --resource --controller

```
 * We can group the kind of resources
 * We give it an alpha version
 * The name for the kind of resource in that group
 * Create the resource and the controller for the resource

The first, now you have a controller for that resource: controller/myownshell_controller.go:



You can see the groups in the new file: api/v1alpha1/groupversion_info.go

```golang
var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "my-first-operator.jgato.io", Version: "v1alpha1"}
...
...
```

then in: api/v1alpha1/myownshell_types.go you can see your Kind resource definition:

```
type MyOwnShellSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	Foo string `json:"foo,omitempty"`
}

type MyOwnShellStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster

}
```
and for curiosity:

```
type MyOwnShell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyOwnShellSpec   `json:"spec,omitempty"`
	Status MyOwnShellStatus `json:"status,omitempty"`
}

```
you see the usual structure of Manifest: metadata, spec and status.

We will only modify the spec to add a size, with the number of shell pods to be created:

```
Size int32 `json:"size"`
```

And we update the project:

```
make generate
```

The above makefile target will invoke the controller-gen utility to update the api/v1alpha1/zz_generated.deepcopy.go file to ensure our API’s Go type definitons implement the runtime.Object interface that all Kind types must implement.

And we generate the manifets (CRDs) based on the APIS

```
 make manifests
```

resulting in the file: config/crd/bases/my-first-operator.jgato.io_myownshells.yaml where the resource is defined with OPENAPI specification. The important here, is that you will see in the spec of the resource the new field size:
```
            properties:
              size:
                description: Foo is an example field of MyOwnShell. Edit myownshell_types.go
                  to remove/update
                format: int32
                type: integer
            required:
            - size
```

So now the hard part, to code the reconcile loop to manage our resources

## Understanding the Controller and Reconciler


**Be aware about the difference of an Reconciler object and the Reconcile function for the Reconciler**
Reconcile loop function is part of a Reconciler object. Every Controller has a Reconciler object with a Reconcile() method that implements the reconcile loop. 

From package reconciler (SDK) Code [here](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.9.2/pkg/reconcile/reconcile.go#L102)

The Reconciler Interface:
```
type Reconciler interface {
	Reconcile(context.Context, Request) (Result, error)
}
```

So how do we implement that interface inside our controller?. We define our own kind of Reconciler, but MyOwnShellReconciler is not of type Reconciler, because it does not implement all the methods (well, just the Reconcile one):

```
type MyOwnShellReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}
```
It is just an struct with some attributes inside. How to add the interface Methods?:

```
func (r *MyOwnShellReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
 // logic here
}
```
 * req only contains name of the request and the namespace (like an unique identifier)
 * cxt ....

So every time a change happens. Our Reconciler will be notified, but he only know that something happened with req.NameSpace/req.Name . It is up to the reconciler "to find out" what happened: looking for resources that should exists or not.

We add the Reconcile function definition to the struct MyOwnShellReconciler. Now, it implements the Reconciler interface. This is how in Golang Interfaces classes are implemented. More [here](https://tour.golang.org/methods/10). 

From now on, MyOwnShellReconciler struct implements the interface Reconciler. So the Controller/Manager can consider this as a Reconciler, and it knows it has a reconcile loop.

So we have a Reconciler, with a Reconcile loop. 

Now we configure the Controller to watch on our resources:

```
func (r *MyOwnShellReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myfirstoperatorv1alpha1.MyOwnShell{}).
		Complete(r)
}

```
Yes, we are also adding that function to our struct. 

Now the manager can init the Controller (using the SetupWithManager), that has a Reconciler, that will watch any changes on our resources. Great, we have it. 

The last point: our Reconciler (MyOwnShellReconciler) also has a client, which is provided by the Manager during initialization. So, the Reconcile loop function, which is inside an object MyOwnShellReconciler has access to the K8S client, which is an attribute of the class MyOwnShellReconciler. Inside the loop you can see things like:

```
r.Get(ctx, req.NamespacedName, myownshell)
```

This is done with the client received.



## Inside the Reconcile loop

*Typically, reconcile is triggered by a Controller in response to cluster Events (e.g. Creating, Updating, Deleting Kubernetes objects) or external Events (GitHub Webhooks, polling external sources, etc). Reconciler implements a Kubernetes API for a specific Resource by Creating, Updating or Deleting Kubernetes objects, or by making changes to systems external to the cluster (e.g. cloudproviders, github, etc). *

So the Reconcile loop receives a context and the object (from the ones that your controller is watching). It returns a result and an error. The result and error combination makes the controller aware, if it everything is ok, or if it is need it to requeue the object again later (something pending).

Edit the controller: controller/myownshell_controller.go to add all the logic. By default is empty, so I have copied the reconcile loop from [here](https://github.com/operator-framework/operator-sdk/blob/latest/testdata/go/v3/memcached-operator/controllers/memcached_controller.go) as an example from the tutorial.

But I had to do several modifications because I am creating my own resources. 

Here I just retrieve the object that has changed:

```
	// Fetch the  instance
	myownshell := &myfirstoperatorv1alpha1.MyOwnShell{}
	log.Info("%v\n", myownshell)

	err := r.Get(ctx, req.NamespacedName, myownshell)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Memcached")
		return ctrl.Result{}, err
	}
```

Basically it gets all the info for the object looking for possible errors:
 * The object does not exists. This would be ok. Because the loop was executed after deleting an object. This is managed by K8S, you dont need to do anything. So.. nothing else must be done
 * Other errors: you tell the controller something went wrong and you will retry later. 

This makes pretty nothing, but we can try it, just to detect something happened with our resources. 

## Running the operator locally

Before proceeding, the Reconcile loop has some markers about the RBAC roles that will be need it:

```
//+kubebuilder:rbac:groups=my-first-operator.jgato.io,resources=myownshells,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my-first-operator.jgato.io,resources=myownshells/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my-first-operator.jgato.io,resources=myownshells/finalizers,verbs=update
```

From our current code, it is too much and maybe the first line is need it. But, lets keep it as it is. This markers will be used to generate the RBAC manifests to deploy the Operator. If we make changes, remember to invoke **make manifests**

In this tutorial I dont go in deep about how to deploy the Operator. But basically, you will have a SA, some Roles (based on that markers) and Bindings, and CRD. Of course, by the moment we will not deploy the Operator, we will run it locally. But, SA,Roles, CRDs need to be deployed in the cluster.

The deployment will build a Docker Image with the code of the Operator. By default, it will use the image name as you initialization domain. In this case: jgato.io/my-first-operator, which looks great as K8S API extension, but it is not ok for my Docker Hub Registry. So, edit the Makefile and change IMAGE_TAG_BASE to jgato/my-first-operator. That it will work as the Registry. 

```
IMAGE_TAG_BASE ?= jgato/my-first-operator
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)
``` 

and build and push:

```
make docker-build docker-push
```



Run it locally. But be sure your kubectl is pointing to a good cluster candidate and context.

```
make build install run

```

 * build: builds the go code
 * install: deploys in the cluster your CRD (not sure if also roles)
 * run: run it locally
To run it locally, you dont need to build and push the Docker Image every time.

Now we create our own CRD. There are some examples in config/samples

```
apiVersion: my-first-operator.jgato.io/v1alpha1
kind: MyOwnShell
metadata:
  name: myownshell-sample
spec:
  # Add fields here
  size: 3

```

If you apply this CRD you will see some logs but nothing really happens. Well:
```
k get myownshells
NAME                AGE
myownshell-sample   7m59s

```
The resource exists, but it does not create a thing. BUt it works.

## Adding some logic

From the previous Reconcile loop, we just did a look for the object but nothing else. Lets add some logic to make a deployment of pods with size argument defined in our CRD.

```

found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: myownshell.Name, Namespace: myownshell.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForMyOwnShell(myownshell)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}
```
This project looks for deployments owned by CR myownshell. If not, it creates a new one:

```
dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "busybox:latest",
						Name:    "busybox",
						Command: []string{"sh", "-c", "echo The app is running, You can now ssh into for a while! && sleep 99999999 "},
					}},
				},
			},
		},
	}
```

and the deployment is linked to our resource by:

```
	// Set MyOwnShell instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
```

Here you will see that our CR will create a Deployment with Pods of Busybox and an sleep. That will allow us to open a Shell inside during a time :) 

Remember to add the watcher for Deployments owned by our Resource.
```
func (r *MyOwnShellReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myfirstoperatorv1alpha1.MyOwnShell{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
```
Build and deploy everything again :)


## Interacting with the Operator

If everything is ok we have our functionality ready.

We can see the CR myownshell, which has created a deployment, which has deployed 3 pods.

```
$> k get myownshells
NAME                AGE
myownshell-sample   36m

─> k get deployments.apps 
NAME                READY   UP-TO-DATE   AVAILABLE   AGE
myownshell-sample   3/3     3            3           25s

─> k get pods
NAME                                 READY   STATUS    RESTARTS   AGE
myownshell-sample-67888b4d74-452vr   1/1     Running   0          30s
myownshell-sample-67888b4d74-fbcjs   1/1     Running   0          30s
myownshell-sample-67888b4d74-sbkbv   1/1     Running   0          30s


```

More about the deployment and how this is marked by the owner (I have simplified the output):
```
-> k get deployments.apps myownshell-sample  -o yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myownshell-sample
  namespace: default
  ownerReferences:
  - apiVersion: my-first-operator.jgato.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: MyOwnShell
    name: myownshell-sample
    uid: ac77f5eb-e8ab-4058-8dc3-9a49712172e0
  resourceVersion: "16151"
  uid: 01dbb17a-1224-4d80-b507-f4225ac45cf6
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myownshell
      myownshell_cr: myownshell-sample
  template:
    metadata:
      labels:
        app: myownshell
        myownshell_cr: myownshell-sample
    spec:
      containers:
      - command:
        - sh
        - -c
        - 'echo The app is running, You can now ssh into for a while! && sleep 99999999 '
        image: busybox:latest
        name: busybox


```

Interesting the ownerReferences with an UID. Lets look at the owner:

```
─> k get myownshells myownshell-sample  -o yaml
apiVersion: my-first-operator.jgato.io/v1alpha1
kind: MyOwnShell
metadata:
  name: myownshell-sample
  namespace: default
  uid: ac77f5eb-e8ab-4058-8dc3-9a49712172e0
spec:
  size: 3

```

Now it is easy to see how the owner is linked to its resources.

If we delete a Pod from the Deployment, it will be generated. But this is just done by the K8S deployment logic. The Operator does nothing. Of course it will be invoked because a change happened but we did not created any logic about it.

## Adding even more logic

### Allowing updates

Not the Operator finds the deployment, if not, it creates it. But, what happens if the original resource is updated? If you manually update the deployment size, I can guess that K8S will update the number of pods. But this is not the proper way. You should manage and update the size, with the CR MyOwnShell. 

Now, with the actual code, the Reconciler only check for existing Deployment, and create it, if it does not exists. But, it should detect if existing, but the size should be updated.

The code it is really easy, we take the MyOwnShell CR size, and we check if it is the same of the Deployment Replicas. If not, we update the Deployment Replicas number, and we let K8S to do all the stuff about the Deployment

```
	// Ensure the deployment size is the same as the spec
	size := myownshell.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

```

### Adding finalizers

Now, when a MyOwnShell is deleted we cannot make anything. Well, we dont need it. K8S will delete the Resource and all its related Resources. 

For testing purpose, we will add a finalizer to be executed, before K8S make the job of deleting our pods. Just to log it.

We add the Finalizer
```
	if !controllerutil.ContainsFinalizer(myownshell, myOwnShellFinalizer) {
		controllerutil.AddFinalizer(myownshell, myOwnShellFinalizer)
		err = r.Update(ctx, myownshell)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

```

Pretty simple. And now we add the code that will control the Finalizer when deleting the Resource:

```
isMyOwnShellMarkedToBeDeleted := myownshell.GetDeletionTimestamp() != nil
	if isMyOwnShellMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(myownshell, myOwnShellFinalizer) {
			// We do some logic
			log.Info("Successfully finalized MyOwnShell")

			// We say everything goes well. But we should control
			// if something goes wrong, we dont delete finalizeer
			// in order to retry later
			if false {
				return ctrl.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(myownshell, myOwnShellFinalizer)
			err := r.Update(ctx, myownshell)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}


```
We see that is MarkedToBeDeleted, but not deleted yet. It will not be deleted until we make our logic and execute the RemoveFinalizer.


Now you delete a MyOwnShell Resource you will see in the logs:

```
2021-09-12T17:08:04.517+0200	INFO	controller-runtime.manager.controller.myownshell	Successfully finalized MyOwnShell	{"reconciler group": "my-first-operator.jgato.io", "reconciler kind": "MyOwnShell", "name": "myownshell-sample", "namespace": "default"}

```

We have added the Finalizer to the MyOwnShell, but it could be also added to other resources like our Deployment:

```
	// Add the finalizer for the Deployment
	if !controllerutil.ContainsFinalizer(found, myOwnShellFinalizer) {
		controllerutil.AddFinalizer(found, myOwnShellFinalizer)
		err = r.Update(ctx, found)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
```

```
─> k get deployments.apps myownshell-sample -o yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
  creationTimestamp: "2021-09-07T15:21:28Z"
  finalizers:
  - myownshell.my-first-operator.jgato.io/finalizer
  generation: 1
  name: myownshell-sample
  namespace: default
  ownerReferences:
...
...
```

If I delete the deployment it will be blocked the deletion. Why? There is no logic for putting away the Finalizer. We only have logic for detecting a delete on MyOwnShell Resource. But to add this logic is pretty simple and we can use the same logic than before, but extending it to the deployment


