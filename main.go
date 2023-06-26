/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"context"
	"flag"
	"fmt"

	// "time"

	"github.com/itchyny/gojq" // json parser

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

// Define flag variables
var externalTest = false

func main() {
	// Bind flags
	flag.BoolVar(&externalTest, "external", false, "use external to cluster configuration")

	flag.Parse()

	var config *rest.Config
	var err error
	var volsyncNamespace string = "volsync"

	if externalTest {
		kubeconfig := flag.String("kubeconfig", ".kubeconfig", "(optional) absolute path to the kubeconfig file")
		// if home := homedir.HomeDir(); home != "" {
		// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		// } else {
		// 	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		// }

		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Query pods
	pods, err := clientset.CoreV1().Pods(volsyncNamespace).List(context.TODO(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Unable to find volsync pod in %s namespace\n", volsyncNamespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod in namespace %s: %v\n",
			volsyncNamespace, statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
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
			fmt.Printf("volsync pod not found???\n")
		}
	} else {
		panic("No pods found")
	}
		
	// fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	// List all instances of replicationsource crd try 2
	// TODO: Process this data
	ctx := context.Background()
	dynamic := dynamic.NewForConfigOrDie(config)

	namespace := "default"
	items, err := GetResourcesDynamically(dynamic, ctx, "volsync.backube", "v1alpha1", "replicationsources", namespace)
	if err != nil {
		fmt.Printf("Error: %v", err)
		panic(err)
	}
	for _, backup := range items {
		fmt.Printf("%+v", backup)
	}

}

// Stolen code: https://itnext.io/generically-working-with-kubernetes-resources-in-go-53bce678f887
func GetResourcesByJq(dynamic dynamic.Interface, ctx context.Context, group string,
	version string, resource string, namespace string, jq string) (
	[]unstructured.Unstructured, error) {

	resources := make([]unstructured.Unstructured, 0)

	query, err := gojq.Parse(jq)
	if err != nil {
		return nil, err
	}

	items, err := GetResourcesDynamically(dynamic, ctx, group, version, resource, namespace)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		// Convert object to raw JSON
		var rawJson interface{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &rawJson)
		if err != nil {
			return nil, err
		}

		// Evaluate jq against JSON
		iter := query.Run(rawJson)
		for {
			result, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := result.(error); ok {
				if err != nil {
					return nil, err
				}
			} else {
				boolResult, ok := result.(bool)
				if !ok {
					fmt.Println("Query returned non-boolean value")
				} else if boolResult {
					resources = append(resources, item)
				}
			}
		}
	}
	return resources, nil
}

// Stolen code: https://itnext.io/generically-working-with-kubernetes-resources-in-go-53bce678f887
func GetResourcesDynamically(dynamic dynamic.Interface, ctx context.Context,
	group string, version string, resource string, namespace string) (
	[]unstructured.Unstructured, error) {

	resourceId := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	list, err := dynamic.Resource(resourceId).Namespace(namespace).
		List(ctx, metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}