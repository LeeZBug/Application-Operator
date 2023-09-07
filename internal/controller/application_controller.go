/*
Copyright 2023 lizhejie.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dappsv1 "lizhejie.com/application-operator/api/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.lizhejie.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.lizhejie.com,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.lizhejie.com,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("控制器运行！！！")

	// TODO(user): your logic here

	// 获取CR资源
	app := &dappsv1.Application{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if errors.IsNotFound(err) {
			l.Info("The Application is not found")
			return ctrl.Result{}, nil
		}
		l.Error(err, "fail to get Application")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// 获取CR定义下的所有pod
	podList := &v1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(app.Namespace), client.MatchingLabels(app.Labels)); err != nil {
		l.Error(err, "Failed to list Pods")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// 当前pod数量
	currentReplicas := int32(len(podList.Items))
	l.Info(fmt.Sprintf("当前pod数量为%d", currentReplicas))
	// 期望pod数量

	desiredReplicas := app.Spec.Replicas
	l.Info(fmt.Sprintf("pod期望数量为%d", desiredReplicas))

	if currentReplicas < desiredReplicas {
		// Need to create more Pods
		for i := currentReplicas; i < desiredReplicas; i++ {
			pod := createPodForApplication(app)
			if err := r.Create(ctx, pod); err != nil {
				l.Error(err, "Failed to create Pod")
				return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
			}
			l.Info(fmt.Sprintf("Created Pod: %s", pod.Name))
		}
	} else if currentReplicas > desiredReplicas {
		// Need to delete excess Pods
		podsToDelete := podList.Items[currentReplicas-desiredReplicas:]
		for _, pod := range podsToDelete {
			if err := r.Delete(ctx, &pod); err != nil {
				l.Error(err, "Failed to delete Pod")
				return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
			}
			l.Info(fmt.Sprintf("Deleted Pod: %s", pod.Name))
		}
	}

	// return ctrl.Result{}, nil
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func createPodForApplication(app *dappsv1.Application) *v1.Pod {
	// Create and return a Pod object based on the Application's template and labels
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", app.Name, generateRandomString(5)), // You may want to generate a unique name
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
		Spec: app.Spec.Template.Spec,
	}
}

func generateRandomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dappsv1.Application{}).
		Complete(r)
}
