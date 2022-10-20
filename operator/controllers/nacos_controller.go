/*
Copyright 2021.

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

package controllers

import (
	"context"
	"reflect"
	"time"

	"nacos.io/nacos-operator/pkg/service/operator"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nacosgroupv1alpha1 "nacos.io/nacos-operator/api/v1alpha1"

	myErrors "nacos.io/nacos-operator/pkg/errors"
)

// NacosReconciler reconciles a Nacos object
type NacosReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	OperaterClient *operator.OperatorClient
}

// +kubebuilder:rbac:groups=nacos.io,resources=nacos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nacos.io,resources=nacos/status,verbs=get;update;patch
type reconcileFun func(nacos *nacosgroupv1alpha1.Nacos)

func (r *NacosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("nacos", req.NamespacedName)

	instance := &nacosgroupv1alpha1.Nacos{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// 工作逻辑入口 , 引发了painc，返回默认false，重新插入队列,5秒继续执行
	result := r.ReconcileWork(instance)
	if result == false {
		return reconcile.Result{
			Requeue:      !result,
			RequeueAfter: time.Second * 15}, nil
	} else {
		return reconcile.Result{}, nil
	}

}

func (r *NacosReconciler) ReconcileWork(instance *nacosgroupv1alpha1.Nacos) bool {
	// 处理全局异常处理中的异常
	defer func() {
		if err := recover(); err != nil {
			r.Log.Error(err.(error), "unknow error")
		}
	}()

	// 全局处理异常
	defer func() {
		if err := recover(); err != nil {
			// 可处理的异常
			r.globalExceptHandle(err, instance)
		}
	}()

	for _, fun := range []reconcileFun{
		r.OperaterClient.PreCheck,
		// 保证资源能够创建
		r.OperaterClient.MakeEnsure,
		// 检查并保障
		r.OperaterClient.CheckAndMakeHeal,
		// 保存状态
		r.OperaterClient.UpdateStatus,
	} {
		fun(instance)
	}

	return true
}

func filterByLabel(label map[string]string) bool {
	v := label["middleware"]
	if v != "nacos" {
		return false
	} else {
		return true
	}
}

func (r *NacosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nacosgroupv1alpha1.Nacos{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}

// 全局异常处理
func (r *NacosReconciler) globalExceptHandle(err interface{}, instance *nacosgroupv1alpha1.Nacos) {
	if reflect.TypeOf(err) == reflect.TypeOf(myErrors.NewErrMsg("")) {
		myerr := err.(*myErrors.Err)
		r.Log.V(0).Info("painc", "code", myerr.Code, "msg", myerr.Msg)
		switch myerr.Code {
		case myErrors.CODE_NORMAL:
			r.OperaterClient.StatusClient.UpdateStatus(instance)
			return
		}

		// 超时3分钟如果还未成功就显示异常
		if instance.Status.Phase != nacosgroupv1alpha1.PhaseCreating ||
			instance.CreationTimestamp.Add(time.Minute*3).Before(time.Now()) {
			r.OperaterClient.StatusClient.UpdateExceptionStatus(instance, myerr)
		}
	} else {
		// 未知的错误，把堆栈打印出来
		r.Log.Error(err.(error), "unknow error")
	}
}
