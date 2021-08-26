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

package main

import (
	"flag"
	"os"

	"nacos.io/nacos-operator/pkg/service/operator"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/klogr"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	"net/http"
	_ "net/http/pprof"

	nacosgroupv1alpha1 "nacos.io/nacos-operator/api/v1alpha1"
	"nacos.io/nacos-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nacosgroupv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {

	go func() {
		http.ListenAndServe("0.0.0.0:8090", nil)
	}()
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	//ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	ctrl.SetLogger(klogr.New())
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "219866ca.nacos.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	log := ctrl.Log.WithName("controllers").WithName("Nacos")
	clientset, _ := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err = (&controllers.NacosReconciler{
		Client:         mgr.GetClient(),
		Log:            log,
		Scheme:         mgr.GetScheme(),
		OperaterClient: operator.NewOperatorClient(log, clientset, mgr.GetScheme(), mgr.GetClient()),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Nacos")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
