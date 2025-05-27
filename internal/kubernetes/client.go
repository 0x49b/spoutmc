package kubernetes

import (
	"context"
	"fmt"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"runtime"
	"spoutmc/internal/log"
	"time"
)

var kubeconfig string
var config *rest.Config
var logger = log.GetLogger()
var spoutNamespace = "spoutmc"
var clientset *kubernetes.Clientset
var skyblockDeployment = "skyblock"
var err error

func init() {

}

func StartKubeClient() error {

	// Kube Context for lima on macOS
	if runtime.GOOS == "darwin" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".lima", "k3s", "copied-from-guest", "kubeconfig.yaml")
	} else {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Error(err.Error())
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err.Error())
	}

	go func() {

		if err := checkNamespace(); err != nil {
			if err := createNamespace(); err != nil {
				logger.Error("🚢 " + err.Error())
			}
		}

		if err := checkLoadbalancer(); err != nil {
			err := createLoadbalancer()
			if err != nil {
				logger.Error("🚢 " + err.Error())
			}
		}

		if err := checkDeployment(skyblockDeployment); err != nil {
			err := createDeployment(skyblockDeployment)
			if err != nil {
				logger.Error("🚢 " + err.Error())
			}
		}

		if err := restartDeployment(skyblockDeployment); err != nil {
			logger.Error("🚢 " + err.Error())
		} else {
			logger.Info("🚢 deployment restarted: " + skyblockDeployment)
		}

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

			getMetrics()

			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

func ResetKube(namespace string) error {
	err := DeleteNamespace(namespace)
	if err != nil {
		return err
	}

	err = StartKubeClient()
	if err != nil {
		return err
	}

	return nil
}
