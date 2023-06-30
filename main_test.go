package main

import (
	// "regexp"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestUnstructuredToRS calls main.unstructuredToRS with an unstructured example, checking
// for a valid return value.
func TestUnstructuredToRS(t *testing.T) {
	var data unstructured.Unstructured = unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "volsync.backube/v1alpha1",
			"kind":       "ReplicationSource",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "volsync",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"lastTransitionTime": "2020-07-22T15:00:00Z",
						"message":            "test",
						"reason":             "test",
						"status":             "test",
						"type":               "test",
					},
				},
				"lastSyncTime":     "2020-07-22T15:00:00Z",
				"lastSyncDuration": "10s",
				"latestMoverStatus": map[string]interface{}{
					"result": "test",
				},
			},
		},
	}
	rs, err := unstructuredToRS(data)
	if err != nil {
		t.Fatalf(`unstructuredToRS(data).err = %v, want match for nil`, err)
	}
	if rs.Metadata.Name != "test" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for test`, rs.Metadata.Name)
	}
	if rs.Metadata.Namespace != "volsync" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for volsync`, rs.Metadata.Namespace)
	}
	if rs.Status.Conditions[0].Message != "test" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for test`, rs.Status.Conditions[0].Message)
	}
	if rs.Status.Conditions[0].Reason != "test" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for test`, rs.Status.Conditions[0].Reason)
	}
	if rs.Status.Conditions[0].Status != "test" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for test`, rs.Status.Conditions[0].Status)
	}
	if rs.Status.Conditions[0].Type != "test" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for test`, rs.Status.Conditions[0].Type)
	}
	if rs.Status.LastSyncTime != "2020-07-22T15:00:00Z" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for 2020-07-22T15:00:00Z`, rs.Status.LastSyncTime)
	}
	if rs.Status.LastSyncDuration != "10s" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for 10s`, rs.Status.LastSyncDuration)
	}
	if rs.Status.LatestMoverStatus.Result != "test" {
		t.Fatalf(`unstructuredToRS(data) = %v, want match for test`, rs.Status.LatestMoverStatus.Result)
	}
}

// TestHelloEmpty calls greetings.Hello with an empty string,
// checking for an error.
func TestUnstructuredToRSEmpty(t *testing.T) {
	var data unstructured.Unstructured = unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	_, err := unstructuredToRS(data)
	if err == nil {
		t.Fatalf(`unstructuredToRS(data) = %v, want to error`, err)
	}
}
