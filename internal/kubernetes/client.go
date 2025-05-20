package kubernetes

import (
	"context"
	"fmt"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/utils/pointer"
	"path/filepath"
	"runtime"
	"spoutmc/internal/log"
	"time"
)

var kubeconfig string
var logger = log.GetLogger()
var spoutNamespace = "spoutmc"

func StartKubeClient() error {

	go func() {

		// Kube Context for lima on macOS
		if runtime.GOOS == "darwin" {
			kubeconfig = filepath.Join(homedir.HomeDir(), ".lima", "k3s", "copied-from-guest", "kubeconfig.yaml")
		} else {
			kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}

		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Error(err.Error())
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			logger.Error(err.Error())
		}

		deploymentsClient := clientset.AppsV1().Deployments(spoutNamespace)

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "spoutmc-deployment",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "demo",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "web",
								Image: "nginx:1.12",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 80,
									},
								},
							},
						},
					},
				},
			},
		}

		logger.Info("🚢 Creating Deployment")

		result, err := deploymentsClient.Create(context.Background(), deployment, metav1.CreateOptions{})
		if err != nil {
			logger.Error("🚢 " + err.Error())
		}

		logger.Info(fmt.Sprintf("🚢 Created deployment %s/%s", result.GetObjectMeta().GetNamespace(), result.GetObjectMeta().GetName()))

		for {
			pods, err := clientset.CoreV1().Pods(spoutNamespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				logger.Error(err.Error())
			}
			logger.Info(fmt.Sprintf("🚢 There are %d pods in the namespace %s", len(pods.Items), spoutNamespace))
			for _, pod := range pods.Items {
				logger.Info(fmt.Sprintf("🚢 Pod Name: %s, Pod Namespace: %s", pod.Name, pod.Namespace))
				for _, container := range pod.Spec.Containers {
					logger.Info(fmt.Sprintf("🚢 Container Name: %s", container.Name))
					if container.Name == "spoutmc" {
						logger.Info(fmt.Sprintf("🚢 SpoutMC Pod Name: %s, SpoutMC Pod Namespace: %s", pod.Name, pod.Namespace))
					}

				}

			}

			namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
			var namespaceNames []string
			for _, namespace := range namespaces.Items {
				namespaceNames = append(namespaceNames, namespace.Name)
			}

			if !slices.Contains(namespaceNames, spoutNamespace) {
				nsName := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "spoutmc",
					},
				}

				_, err := clientset.CoreV1().Namespaces().Create(context.Background(), nsName, metav1.CreateOptions{})
				if err != nil {
					logger.Error("🚢 " + err.Error())
				}

			}

			time.Sleep(10 * time.Second)
		}
	}()

	return nil

}
