package kubernetes

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"time"
)

func checkDeployment(deploymentName string) error {

	_, err := clientset.AppsV1().Deployments(spoutNamespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return err
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

func restartDeployment(deploymentName string) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"spoutmc.net/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339)))
	_, err := clientset.AppsV1().Deployments(spoutNamespace).Patch(context.TODO(), deploymentName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func createDeployment(deploymentName string) error {
	logger.Info("🚢 Creating Deployment")
	deploymentsClient := clientset.AppsV1().Deployments(spoutNamespace)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploymentName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  deploymentName,
							Image: "itzg/minecraft-server:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 25565,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "EULA",
									Value: "TRUE",
								},
								{
									Name:  "VERSION",
									Value: "1.21.5",
								}, {
									Name:  "ENABLE_RCON",
									Value: "FALSE",
								},
								{
									Name:  "CREATE_CONSOLE_IN_PIPE ",
									Value: "TRUE",
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := deploymentsClient.Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("🚢 Created deployment %s/%s", result.GetObjectMeta().GetNamespace(), result.GetObjectMeta().GetName()))
	return nil
}

func UpdateDeploymentReplicas(deploymentName string, replicas int32) error {
	logger.Info(fmt.Sprintf("🔄 Updating replicas for Deployment %s to %d", deploymentName, replicas))
	deploymentsClient := clientset.AppsV1().Deployments(spoutNamespace)

	// Retrieve the existing Deployment
	deployment, err := deploymentsClient.Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update the replicas
	deployment.Spec.Replicas = &replicas

	// Apply the update
	_, err = deploymentsClient.Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	logger.Info(fmt.Sprintf("✅ Successfully updated replicas for deployment %s to %d", deploymentName, replicas))
	return nil
}
