package controlplane

import (
	"context"
	"gitlab.globoi.com/tks/gks/gks-operator/pkg/model/master"
	"gotest.tools/assert/cmp"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gksv1alpha1 "gitlab.globoi.com/tks/gks/gks-operator/pkg/apis/gks/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_controlplane")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ControlPlane Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileControlPlane{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("controlplane-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ControlPlane
	err = c.Watch(&source.Kind{Type: &gksv1alpha1.ControlPlane{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Deployment and requeue the owner ControlPlane
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{
		//IsController: true,
		//OwnerType:    &gksv1alpha1.ControlPlane{},
	}, predicate.GenerationChangedPredicate{Funcs: predicate.Funcs{DeleteFunc: func(e event.DeleteEvent) bool{

		if _, ok := e.Meta.GetLabels()["tier"]; ok {
			return true
		}
		return false
	}}})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Service and requeue the owner ControlPlane
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &gksv1alpha1.ControlPlane{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileControlPlane implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileControlPlane{}

// ReconcileControlPlane reconciles a ControlPlane object
type ReconcileControlPlane struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileControlPlane) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ControlPlane")

	instance := &gksv1alpha1.ControlPlane{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err){
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	clusterNamespacedName := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}

	serviceLoadBalancer, err := r.ensureLatestLoadBalancer(instance, clusterNamespacedName)
	loadBalancerHostNames := r.extractLoadBalancerHostNames(serviceLoadBalancer)

	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ensureLatestDeployment(instance, loadBalancerHostNames, clusterNamespacedName)

	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileControlPlane) createMaster(namspacedName types.NamespacedName, instance *gksv1alpha1.ControlPlane,
	loadBalancerHostnames []string)error{
	masterModel, err := master.NewMaster(namspacedName, instance.Spec.ControlPlaneMaster, loadBalancerHostnames,
		master.NewResourceSplitter())

	if err != nil {
		return err
	}

	masterDeployment, _ := masterModel.BuildDeployment()

	if err := controllerutil.SetControllerReference(instance, masterDeployment, r.scheme); err != nil {
		return err
	}

	if err := r.client.Create(context.TODO(), masterDeployment); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileControlPlane) extractLoadBalancerHostNames(loadBalancer *corev1.Service)[]string{
	hostnames := make([]string, len(loadBalancer.Status.LoadBalancer.Ingress))

	for index,ingress := range loadBalancer.Status.LoadBalancer.Ingress{
		hostnames[index] = ingress.Hostname
	}

	if len(hostnames) == 0 {
		hostnames = []string{loadBalancer.Spec.ClusterIP}
	}

	return hostnames
}

func (r *ReconcileControlPlane) createLoadBalancer(instance *gksv1alpha1.ControlPlane,
	namespacedName types.NamespacedName)(*corev1.Service, error){

	serviceLoadBalancer := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels: map[string]string{
				"app":"load-balancer",
				"cluster": namespacedName.Name,
				"tier": "control-plane",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceType("LoadBalancer"),
			Ports: []corev1.ServicePort{
				{ Port: 6443, TargetPort: intstr.FromInt(6443)},
			},
			Selector: map[string]string{
				"cluster": namespacedName.Name,
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, serviceLoadBalancer, r.scheme); err != nil {
		return nil, err
	}

	if err := r.client.Create(context.TODO(), serviceLoadBalancer); err != nil {
		return nil, err
	}

	return serviceLoadBalancer, nil
}

func (r *ReconcileControlPlane) ensureLatestDeployment(instance *gksv1alpha1.ControlPlane,
	loadBalancerHostnames []string, clusterNamespacedName types.NamespacedName)error {

	masterDeployment := &appsv1.Deployment{}

	err := r.client.Get(context.TODO(), clusterNamespacedName, masterDeployment)

	if err != nil {
		if errors.IsNotFound(err){
			return r.createMaster(clusterNamespacedName, instance, loadBalancerHostnames)
		}
		return err
	}

	desiredMasterModel, err := master.NewMaster(clusterNamespacedName, instance.Spec.ControlPlaneMaster,
		loadBalancerHostnames, master.NewResourceSplitter())

	if err != nil {
		return err
	}

	desiredMasterDeployment, _ := desiredMasterModel.BuildDeployment()

	deploymentEqual := cmp.DeepEqual(masterDeployment, desiredMasterDeployment)()

	if !deploymentEqual.Success() {
		if err = r.client.Update(context.TODO(), desiredMasterDeployment); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileControlPlane) ensureLatestLoadBalancer(instance *gksv1alpha1.ControlPlane,
	clusterNamespacedName types.NamespacedName)(*corev1.Service, error){

	serviceLoadBalancer := &corev1.Service{}

	err := r.client.Get(context.TODO(), clusterNamespacedName, serviceLoadBalancer)

	if err != nil {
		if errors.IsNotFound(err){
			serviceLoadBalancer, err = r.createLoadBalancer(instance,clusterNamespacedName)
			if err != nil {
				return nil, err
			}
			return serviceLoadBalancer, nil
		}
		return nil, err
	}

	return serviceLoadBalancer, nil
}
