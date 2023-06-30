// TODO: Put applicable license here

package main

// TODO: Refine imports list
import (
	"context"
	"flag"
	"fmt"
	"log"

	// "time"

	// json parser

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

// TODO: Refine this struct to be more useful
type replicationSource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		CreationTimestamp string `json:"creationTimestamp"`
		Generation        int    `json:"generation"`
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		ResourceVersion   string `json:"resourceVersion"`
		SelfLink          string `json:"selfLink"`
		UID               string `json:"uid"`
	} `json:"metadata"`
	Status struct {
		Conditions []struct {
			LastTransitionTime string `json:"lastTransitionTime"`
			Message            string `json:"message"`
			Reason             string `json:"reason"`
			Status             string `json:"status"`
			Type               string `json:"type"`
		} `json:"conditions"`
		LastSyncTime      string `json:"lastSyncTime"`
		LastSyncDuration  string `json:"lastSyncDuration"`
		LatestMoverStatus struct {
			Result string `json:"result"`
		} `json:"latestMoverStatus"`
	} `json:"status"`
	Unk map[string]interface{} `json:"-"`
}

// Define flag variables
var externalTest = false // In-Cluster config by default
var kubeconfig = ".kubeconfig" // Use local kubeconfig
var searchNamespace = "" // Search all by default

func main() {
	// Bind flags
	flag.BoolVar(&externalTest, "external", false, "use external to cluster configuration")
	flag.StringVar(&kubeconfig, "kubeconfig", ".kubeconfig", "(optional) absolute path to the kubeconfig file")
	flag.StringVar(&searchNamespace, "namespace", "", "(optional) namespace to search for replicationsources (defaults to all)")

	flag.Parse()

	var config *rest.Config
	var err error
	var volsyncNamespace string = "volsync"

	if externalTest {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err.Error())
	}

	// Query pods
	pods, err := clientset.CoreV1().Pods(volsyncNamespace).List(context.TODO(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Unable to find volsync pod in %s namespace\n", volsyncNamespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting pod in namespace %s: %v\n",
			volsyncNamespace, statusError.ErrStatus.Message)
	} else if err != nil {
		log.Panic(err.Error())
	}
	fmt.Printf("There are %d pods in %s\n", len(pods.Items), volsyncNamespace)

	// Verify the volsync pod is running
	if len(pods.Items) >= 1 {
		match := false
		for _, pod := range pods.Items {
			if pod.Name[:7] == "volsync" {
				fmt.Printf("Pod %s in namespace %s found\n", pod.Name, pod.Namespace)
				match = true
			}
		}
		if !match {
			log.Printf("volsync pod not found???\n")
		}
	} else {
		log.Panic("No VolSync pods found")
	}

	// List all instances of replicationsource crd
	ctx := context.Background()
	dynamic := dynamic.NewForConfigOrDie(config)

	namespace := searchNamespace
	items, err := getResourcesAsRS(ctx, dynamic, namespace)
	if err != nil {
		log.Panic(err)
	}

	if namespace == "" {
		namespace = "total"
	}
	fmt.Printf("There are %d replicationsources in %s\n", len(items), namespace)

	fmt.Println("ReplicationSources:")
	for _, backup := range items {
		fmt.Printf("%s | %s | %s | %s\n", backup.Metadata.Name, backup.Metadata.Namespace, backup.Status.LatestMoverStatus.Result, backup.Status.LastSyncTime)
	}

}

func getResourcesAsRS(ctx context.Context, dynamic dynamic.Interface, namespace string) (
	[]replicationSource, error) {

	// Define var to return
	resources := make([]replicationSource, 0)

	// Get all replication sources requested
	items, err := getResourcesDynamically(ctx,dynamic, "volsync.backube", "v1alpha1", "replicationsources", namespace)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		// Convert unstructured object to typed ReplicationSource
		var rs replicationSource
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &rs)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rs)
	}
	return resources, nil
}

// Stolen code: https://itnext.io/generically-working-with-kubernetes-resources-in-go-53bce678f887
// GetResourcesDynamically returns a list of unstructured objects after querying the cluster
func getResourcesDynamically(ctx context.Context, dynamic dynamic.Interface,
	group string, version string, resource string, namespace string) (
	[]unstructured.Unstructured, error) {

	resourceID := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	list, err := dynamic.Resource(resourceID).Namespace(namespace).
		List(ctx, metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}
