package persistentvolumeclaim

import (
	"fmt"

	"github.com/longhorn/longhorn-manager/datastore"
	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
	"github.com/longhorn/longhorn-manager/scheduler"
	"github.com/longhorn/longhorn-manager/types"
	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/longhorn/longhorn-manager/webhook/admission"
	werror "github.com/longhorn/longhorn-manager/webhook/error"
)

var log = logrus.New()

type pvcValidator struct {
	admission.DefaultValidator
	ds *datastore.DataStore
}

func NewValidator(ds *datastore.DataStore) admission.Validator {
	return &pvcValidator{ds: ds}
}

func (v *pvcValidator) Resource() admission.Resource {
	return admission.Resource{
		Name:       "persistentvolumeclaims",
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   corev1.SchemeGroupVersion.Group,
		APIVersion: corev1.SchemeGroupVersion.Version,
		ObjectType: &corev1.PersistentVolumeClaim{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Update,
		},
	}
}

func (v *pvcValidator) Update(request *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	// Log the start of the update validation
	log.Infof("Starting PVC update validation for request: %s", request.UID)

	oldPVC, ok := oldObj.(*corev1.PersistentVolumeClaim)
	if !ok {
		log.Errorf("Invalid old object type: expected *corev1.PersistentVolumeClaim, got %T", oldObj)
		return werror.NewInvalidError(fmt.Sprintf("invalid old object: expected *corev1.PersistentVolumeClaim, got %T", oldObj), "")
	}

	newPVC, ok := newObj.(*corev1.PersistentVolumeClaim)
	if !ok {
		log.Errorf("Invalid new object type: expected *corev1.PersistentVolumeClaim, got %T", newObj)
		return werror.NewInvalidError(fmt.Sprintf("invalid new object: expected *corev1.PersistentVolumeClaim, got %T", newObj), "")
	}

	// Log the old and new PVC objects for comparison
	log.Infof("Old PVC: %+v", oldPVC)
	log.Infof("New PVC: %+v", newPVC)

	// Log the old and new PVC sizes in a more human-readable format
	oldSize := oldPVC.Spec.Resources.Requests[corev1.ResourceStorage]
	newSize := newPVC.Spec.Resources.Requests[corev1.ResourceStorage]
	log.Infof("PVC size change detected: Old Size = %s, New Size = %s", oldSize.String(), newSize.String())

	if oldSize.Cmp(newSize) == 0 {
		log.Infof("PVC size has not changed; skipping further validation.")
		return nil
	}

	pv, err := v.ds.GetPersistentVolumeRO(newPVC.Spec.VolumeName)
	if err != nil {
		log.Errorf("Failed to get PersistentVolume: %v", err)
		return werror.NewInternalError(err.Error())
	}

	// Log the PV details
	log.Infof("PersistentVolume details: Name=%s, Driver=%s", pv.Name, pv.Spec.CSI.Driver)

	if pv.Spec.CSI == nil || pv.Spec.CSI.Driver != types.LonghornDriverName {
		log.Infof("PersistentVolume is not a Longhorn volume; skipping validation.")
		return nil
	}

	volume, err := v.ds.GetVolumeRO(pv.Spec.CSI.VolumeHandle)
	if err != nil {
		log.Errorf("Failed to get Longhorn volume: %v", err)
		return werror.NewInternalError(err.Error())
	}

	// Log the Longhorn volume details
	log.Infof("Longhorn volume details: Name=%s, Size=%d", volume.Name, volume.Spec.Size)

	return v.validateExpansionSize(oldPVC, newPVC, volume)
}

func (v *pvcValidator) validateExpansionSize(oldPVC *corev1.PersistentVolumeClaim, newPVC *corev1.PersistentVolumeClaim, volume *longhorn.Volume) error {
	oldSize := oldPVC.Spec.Resources.Requests[corev1.ResourceStorage]
	oldSizeInt64, ok := oldSize.AsInt64()
	if !ok {
		return werror.NewInternalError(fmt.Sprintf("unable to convert old size '%v' to int64", oldSize))
	}

	newSize := newPVC.Spec.Resources.Requests[corev1.ResourceStorage]
	newSizeInt64, ok := newSize.AsInt64()
	if !ok {
		return werror.NewInternalError(fmt.Sprintf("unable to convert new size '%v' to int64", newSize))
	}

	replicaScheduler := scheduler.NewReplicaScheduler(v.ds)
	if _, err := replicaScheduler.CheckReplicasSizeExpansion(volume, oldSizeInt64, newSizeInt64); err != nil {
		return werror.NewForbiddenError(err.Error())
	}

	return nil
}
