
# Kubernetes Lease

Kubernetes provides a built-in leader election mechanism using the **Lease API**.

The **Lease** object ensures that only one replica holds the lease at any given time, allowing it to perform the task while others wait for the next opportunity.

Here’s an example of how to use Kubernetes’ built-in leader election using the **Lease** resource in your application.


```go

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/retry"
)

func main() {

	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to use kubeconfig if not running inside a cluster
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			fmt.Printf("Failed to create Kubernetes client config: %v", err)
			os.Exit(1)
		}
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Failed to create Kubernetes clientSet: %v", err)
		os.Exit(1)
	}

	// Define the lease lock for leader election
	leaseLock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "example-lease",
			Namespace: "default", 
		},
		Client: clientSet.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: os.Getenv("POD_NAME"),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up leader election
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          leaseLock,
		// The duration that the leader holds the lock. Other pods can only acquire the lock if the current leader does not renew it within this time.
		LeaseDuration: 15 * time.Second,
		// The time by which the leader must renew its lock.
		RenewDeadline: 10 * time.Second,
		// The interval at which the leader tries to acquire or renew the lock.
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(c context.Context) {
				// When the pod becomes the leader
				fmt.Println("I am the leader, performing the task...")
				executeTask(c)
			},
			OnStoppedLeading: func() {
				// when leadership is lost
				fmt.Println("I am not the leader anymore, stopping tasks...")
			},
		},
	})
}

// only the leader should execute this
func executeTask(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Leadership is lost or context is canceled
			fmt.Println("Context canceled, stopping task...")
			return
		case <-ticker.C:
			// Execute task every tick
			fmt.Println("Leader executing task...")
		}
	}
}


```


### RBAC Permissions

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default
  name: leader-election-role
rules:
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "watch", "list", "delete", "update", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: leader-election-binding
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: leader-election-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
```


### Ensure that the POD_NAME environment variable is injected into each pod
```yaml
env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
```
