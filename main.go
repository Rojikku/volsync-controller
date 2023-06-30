// TODO: Put applicable license here

package main

// TODO: Refine imports list
import (
	"context"
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	_ "github.com/joho/godotenv/autoload" // Load .env file automatically

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
var externalTest bool = false // In-Cluster config by default
var kubeconfig string = ".kubeconfig" // Use local kubeconfig
var searchNamespace string = "" // Search all by default
var volsyncNamespace string = "volsync" // Namespace where volsync pods are deployed

func init() {
	if value, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch value {
		case "trace":
			log.SetLevel(log.TraceLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		case "panic":
			log.SetLevel(log.PanicLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if value, ok := os.LookupEnv("LOG_FORMAT"); ok {
		switch value {
		case "json":
			log.SetFormatter(&log.JSONFormatter{})
		case "text":
			log.SetFormatter(&log.TextFormatter{
				DisableColors: false,
				FullTimestamp: true,
			})
		default:
			log.SetFormatter(&log.JSONFormatter{})
		}
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if value, ok := os.LookupEnv("NAMESPACE"); ok {
		searchNamespace = value
	}

	if value, ok := os.LookupEnv("VOLSYNC_NAMESPACE"); ok {
		volsyncNamespace = value
	}

	// Bind flags
	flag.BoolVar(&externalTest, "external", externalTest, "(optional) use external to cluster configuration (default false)")
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "(optional) absolute path to the kubeconfig file")
	flag.StringVar(&searchNamespace, "namespace", searchNamespace, "(optional) namespace to search for replicationsources (defaults to all)")
	flag.StringVar(&volsyncNamespace, "volsync-namespace", volsyncNamespace, "(optional) namespace where volsync pods are deployed")

	flag.Parse()
}

func main() {

	var config *rest.Config
	var err error

	if externalTest {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatal(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Query pods
	pods, err := clientset.CoreV1().Pods(volsyncNamespace).List(context.TODO(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		log.Errorf("Unable to find volsync pod in %s namespace\n", volsyncNamespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Errorf("Error getting pod in namespace %s: %v\n",
			volsyncNamespace, statusError.ErrStatus.Message)
	} else if err != nil {
		log.Panic(err.Error())
	}
	log.Infof("There are %d pods in %s", len(pods.Items), volsyncNamespace)

	// Verify the volsync pod is running
	if len(pods.Items) >= 1 {
		match := false
		for _, pod := range pods.Items {
			if pod.Name[:7] == "volsync" {
				log.Infof("Pod %s in namespace %s found", pod.Name, pod.Namespace)
				match = true
			}
		}
		if !match {
			log.Warn("volsync pod not found???")
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
	items, err := getResourcesDynamically(ctx, dynamic, "volsync.backube", "v1alpha1", "replicationsources", namespace)
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
