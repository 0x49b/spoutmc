package kubernetes

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var spoutLoadbalancer = "spoutmc-lb"

func checkLoadbalancer() error {

	_, err := clientset.CoreV1().Services(spoutNamespace).Get(context.TODO(), spoutLoadbalancer, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return err
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

func createLoadbalancer() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: spoutLoadbalancer,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app": "demo",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "minecraft",
					Protocol:   corev1.ProtocolTCP,
					Port:       25565,                 // External port
					TargetPort: intstr.FromInt(25565), // Container port
				},
			},
		},
	}

	_, err := clientset.CoreV1().Services(spoutNamespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}
