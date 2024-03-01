package persistentvolumeclaim

import (
	"fmt"

	"github.com/longhorn/longhorn-manager/datastore"
	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/longhorn/longhorn-manager/webhook/admission"
	werror "github.com/longhorn/longhorn-manager/webhook/error"
)

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
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}

func (v *pvcValidator) Create(request *admission.Request, newObj runtime.Object) error {
	pvc, ok := newObj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return werror.NewInvalidError(fmt.Sprintf("%v is not a *corev1.PersistentVolumeClaim", newObj), "")
	}

	logrus.Infof("LH webhook pvc %v create", pvc.Name)
	return nil
}

func (v *pvcValidator) Update(request *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldPvc, ok := oldObj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return werror.NewInvalidError(fmt.Sprintf("%v is not a *corev1.PersistentVolumeClaim", oldObj), "")
	}
	_, ok = newObj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return werror.NewInvalidError(fmt.Sprintf("%v is not a *corev1.PersistentVolumeClaim", newObj), "")
	}

	logrus.Infof("LH webhook pvc %v update", oldPvc.Name)
	return nil
}
