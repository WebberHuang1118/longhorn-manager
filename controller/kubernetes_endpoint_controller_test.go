package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/longhorn/longhorn-manager/types"
)

func TestUpdateShareManagerServiceFromEndpoint(t *testing.T) {
	const (
		shareManagerName = "pvc-80ea461a-5f25-4577-bb5a-14cd8f3c2532"
		endpointAddress  = "182.16.0.6"
	)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: shareManagerName,
		},
	}
	endpoint := &corev1.Endpoints{ // nolint: staticcheck
		ObjectMeta: metav1.ObjectMeta{
			Name: shareManagerName,
		},
		Subsets: []corev1.EndpointSubset{ // nolint: staticcheck
			{
				Addresses: []corev1.EndpointAddress{
					{IP: endpointAddress},
				},
			},
		},
	}

	if updated := updateShareManagerServiceFromEndpoint(service, endpoint); !updated {
		t.Fatal("expected the Service to be updated")
	}
	if service.Spec.LoadBalancerIP != endpointAddress {
		t.Fatalf("expected loadBalancerIP %q, got %q", endpointAddress, service.Spec.LoadBalancerIP)
	}
	if service.Labels[types.HarvesterLabelRWXVolumeService] != shareManagerName {
		t.Fatalf("expected label %q to equal %q, got %q",
			types.HarvesterLabelRWXVolumeService, shareManagerName, service.Labels[types.HarvesterLabelRWXVolumeService])
	}
	if updated := updateShareManagerServiceFromEndpoint(service, endpoint); updated {
		t.Fatal("expected an already synchronized Service to remain unchanged")
	}

	endpoint.Subsets = nil
	if updated := updateShareManagerServiceFromEndpoint(service, endpoint); !updated {
		t.Fatal("expected the stale loadBalancerIP to be cleared")
	}
	if service.Spec.LoadBalancerIP != "" {
		t.Fatalf("expected loadBalancerIP to be cleared, got %q", service.Spec.LoadBalancerIP)
	}
}

func TestGetEndpointAddress(t *testing.T) {
	endpoint := &corev1.Endpoints{ // nolint: staticcheck
		Subsets: []corev1.EndpointSubset{ // nolint: staticcheck
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "182.16.0.7"},
					{IP: "182.16.0.6"},
				},
				NotReadyAddresses: []corev1.EndpointAddress{
					{IP: "182.16.0.5"},
				},
			},
		},
	}

	if got := getEndpointAddress(endpoint, "182.16.0.7"); got != "182.16.0.7" {
		t.Fatalf("expected the current ready address to remain selected, got %q", got)
	}
	if got := getEndpointAddress(endpoint, "182.16.0.8"); got != "182.16.0.6" {
		t.Fatalf("expected the first sorted ready address, got %q", got)
	}
	if got := getEndpointAddress(&corev1.Endpoints{}, "182.16.0.8"); got != "" { // nolint: staticcheck
		t.Fatalf("expected no address, got %q", got)
	}
}

func TestGetShareManagerServiceLabels(t *testing.T) {
	const shareManagerName = "pvc-80ea461a-5f25-4577-bb5a-14cd8f3c2532"

	labels := getShareManagerServiceLabels(shareManagerName)
	if labels[types.HarvesterLabelRWXVolumeService] != shareManagerName {
		t.Fatalf("expected label %q to equal %q, got %q",
			types.HarvesterLabelRWXVolumeService, shareManagerName, labels[types.HarvesterLabelRWXVolumeService])
	}
	if labels[types.GetLonghornLabelKey(types.LonghornLabelShareManager)] != shareManagerName {
		t.Fatalf("expected the Longhorn share-manager label to remain set")
	}
}

func TestIsShareManagerService(t *testing.T) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: getShareManagerServiceLabels("pvc-test"),
		},
	}
	if !isShareManagerService(service) {
		t.Fatal("expected the share-manager Service to match")
	}
	if isShareManagerService(&corev1.Service{}) {
		t.Fatal("expected an unrelated Service not to match")
	}
}
