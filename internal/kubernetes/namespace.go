package kubernetes

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func checkNamespace() error {

	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), spoutNamespace, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		return err
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

func createNamespace() error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: spoutNamespace,
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	return err
}

func DeleteNamespace(namespace string) error {
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		// Namespace already deleted or doesn't exist
		return nil
	}
	return err
}
