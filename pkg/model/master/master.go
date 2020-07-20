package master

import (
	"gitlab.globoi.com/tks/gks/gks-operator/pkg/apis/gks/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"strings"
)

type Master struct{
	settings v1alpha1.ControlPlaneMaster
	namespacedName types.NamespacedName
	apiServer apiServer
	scheduler Scheduler
	controllerManager ControllerManager
}

func NewMaster(namespacedName types.NamespacedName, settings v1alpha1.ControlPlaneMaster, loadBalancerHostnames []string,
	splitter ResourceSplitter)(*Master,error) {

	advertiseAddress := strings.Join(loadBalancerHostnames, ",")

	otherComponentsDivisorResourcesStrategy := func(res int)int{
		return res/3
	}
	otherComponentsResources,err := splitter.split(settings.ResourceRequirements,otherComponentsDivisorResourcesStrategy)

	if err != nil {
		return nil, err
	}

	apiServerDivisorResourcesStrategy := func(res int)int{
		return otherComponentsDivisorResourcesStrategy(res) + res%3
	}
	apiServerResources,err := splitter.split(settings.ResourceRequirements,apiServerDivisorResourcesStrategy)

	if err != nil {
		return nil, err
	}

	return &Master{
		settings: settings,
		namespacedName: namespacedName,
		apiServer: newAPIServer(
			advertiseAddress,
			settings.ServiceClusterIPRange,
			settings.AdmissionPlugins,
			*apiServerResources,
		),
		scheduler: NewScheduler(*otherComponentsResources),
		controllerManager: NewControllerManager(
			namespacedName.Name, settings.ServiceClusterIPRange,settings.ClusterCIDR, *otherComponentsResources,
		),
	}, nil
}

func (master *Master) BuildDeployment()*appsv1.Deployment{
	replicas := int32(master.settings.Count)

	return &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Namespace: master.namespacedName.Namespace,
			Name: master.namespacedName.Name,
			Labels: master.buildPodLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: master.buildPodLabels(),
			},
			Template: master.buildPod(),
		},
	}
}

func (master *Master) EqualDeployment(deployment *appsv1.Deployment)bool{
	currentDeployment := master.BuildDeployment()

	if !reflect.DeepEqual(currentDeployment.Labels, deployment.Labels){
		return false
	}

	return true
}

func (master *Master) buildPod()corev1.PodTemplateSpec{

	return corev1.PodTemplateSpec{
		ObjectMeta: v1.ObjectMeta{
			Namespace: master.namespacedName.Namespace,
			Labels: master.buildPodLabels(),
		},
		Spec: corev1.PodSpec{
			Volumes: master.buildVolumes(),
			Containers: []corev1.Container{
				master.apiServer.BuildContainer(),
				master.scheduler.BuilderContainer(),
				master.controllerManager.BuilderContainer(),
			},
		},
	}
}

func (master *Master) buildPodLabels()map[string]string{
	return map[string]string{
		"app":"master",
		"cluster": master.namespacedName.Name,
		"tier": "control-plane",
	}
}

func (master *Master) buildVolumes()[]corev1.Volume{

	return []corev1.Volume{
		master.buildSecretVolume("ca", "ca-certs"),
		master.buildSecretVolume("kubernetes", master.settings.MasterSecretName),
		master.buildSecretVolume("encryption", master.settings.EncryptionSecretName),
	}
}

func (*Master) buildSecretVolume(volumeName, secretName string)corev1.Volume{
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}

