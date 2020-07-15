package master

import (
	"gitlab.globoi.com/tks/gks/gks-operator/pkg/apis/gks/v1alpha1"
	"gitlab.globoi.com/tks/gks/gks-operator/pkg/model/resources"
	kuberesources "k8s.io/apimachinery/pkg/api/resource"
	corev1 "k8s.io/api/core/v1"
)

type ResourceSplitter struct {}

func NewResourceSplitter()ResourceSplitter{
	return ResourceSplitter{}
}

func (splitter *ResourceSplitter) split(controlPlaneResources v1alpha1.ControlPlaneMasterResources,
	divisorStrategy func(res int)int)(*corev1.ResourceRequirements, error){

	res := corev1.ResourceRequirements{}
	requests := controlPlaneResources.ControlPlaneMasterResourcesRequests
	limits := controlPlaneResources.ControlPlaneMasterResourcesLimits

	cpuRequestsValue, err := resources.ConvertToIntegerMiliCores(requests.CPU)

	if err != nil {
		return nil, err
	}

	cpuLimitValue, 	err := resources.ConvertToIntegerMiliCores(limits.CPU)

	if err != nil {
		return nil, err
	}

	memoryRequestsValue, err := resources.ConvertToMebiBytes(requests.Memory)

	if err != nil {
		return nil, err
	}

	memoryLimitValue, err := resources.ConvertToMebiBytes(limits.Memory)

	if err != nil {
		return nil, err
	}

	res.Requests = corev1.ResourceList{
		"cpu": kuberesources.MustParse(
			resources.ConvertIntegerToStringMilicores(divisorStrategy(cpuRequestsValue))),
		"memory": kuberesources.MustParse(
			resources.ConvertIntegerToStringMebiBytes(divisorStrategy(memoryRequestsValue))),
	}

	res.Limits = corev1.ResourceList{
		"cpu": kuberesources.MustParse(
			resources.ConvertIntegerToStringMilicores(divisorStrategy(cpuLimitValue))),
		"memory": kuberesources.MustParse(
			resources.ConvertIntegerToStringMebiBytes(divisorStrategy(memoryLimitValue))),
	}

	return &res, nil
}
