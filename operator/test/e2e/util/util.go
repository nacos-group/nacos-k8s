package util

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func GetDynamicClient() dynamic.Interface {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return client
}

func GetClientSet() *kubernetes.Clientset {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}

func CreateFromYaml(clientSet *kubernetes.Clientset, dd dynamic.Interface, resource string, namespace string) error {
	obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode([]byte(resource), nil, nil)
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	gr, err := restmapper.GetAPIGroupResources(clientSet.Discovery())
	if err != nil {
		return err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		log.Fatal(err)
	}

	var dri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace != "" {
			unstructuredObj.SetNamespace(namespace)
		} else if unstructuredObj.GetNamespace() == "" {
			unstructuredObj.SetNamespace("default")
		}
		dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
	} else {
		dri = dd.Resource(mapping.Resource)
	}

	obj2, err := dri.Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("%s/%s created", obj2.GetKind(), obj2.GetName())
	return nil
}

func GetRestClient(groupversion schema.GroupVersion) *rest.RESTClient {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}
	// groupversion := schema.GroupVersion{
	// 	Group:   "k8s.io",
	// 	Version: "v1",
	// }
	config.GroupVersion = &groupversion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	//config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	restclient, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err)
	}
	return restclient
}
