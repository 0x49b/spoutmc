package kubernetes

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

func getMetrics() {

	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		logger.Error("🚢 " + err.Error())
	}

	podList, err := clientset.CoreV1().Pods(spoutNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=" + skyblockDeployment,
	})

	// Get metrics for each pod
	for _, pod := range podList.Items {
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(spoutNamespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if err != nil {
			logger.Error(fmt.Sprintf("🚢 Error getting metrics for pod %s: %v", pod.Name, err))
			continue
		}
		for _, container := range podMetrics.Containers {
			logger.Info(fmt.Sprintf("🚢 Pod: %s, Container: %s, CPU: %s, Memory: %s",
				pod.Name, container.Name,
				container.Usage.Cpu().String(),
				container.Usage.Memory().String()))
		}
	}

}
