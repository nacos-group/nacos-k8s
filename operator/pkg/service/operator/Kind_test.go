package operator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	nacosv1alpha1 "nacos.io/nacos-operator/api/v1alpha1"
)

func TestClientServiceIPv4SingleStack(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := nacosv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	client := &KindClient{scheme: scheme}
	nacos := &nacosv1alpha1.Nacos{ObjectMeta: metav1.ObjectMeta{
		Name: "nacos", Namespace: "test", UID: "nacos-owner", Labels: map[string]string{"environment": "test"},
	}}
	svc := client.buildClientService(nacos)
	if !reflect.DeepEqual(svc.Spec.IPFamilies, []corev1.IPFamily{corev1.IPv4Protocol}) ||
		svc.Spec.IPFamilyPolicy == nil || *svc.Spec.IPFamilyPolicy != corev1.IPFamilyPolicySingleStack {
		t.Fatalf("expected IPv4 SingleStack, got families=%v policy=%v", svc.Spec.IPFamilies, svc.Spec.IPFamilyPolicy)
	}
	if svc.Name != "nacos-client" || svc.Namespace != "test" || !svc.Spec.PublishNotReadyAddresses {
		t.Fatalf("client Service identity or discovery behavior changed: %+v", svc)
	}
	wantPorts := []corev1.ServicePort{
		{Name: "client", Port: 8848, Protocol: corev1.ProtocolTCP},
		{Name: "rpc", Port: 9848, Protocol: corev1.ProtocolTCP},
	}
	if !reflect.DeepEqual(svc.Spec.Ports, wantPorts) {
		t.Fatalf("unexpected ports: %+v", svc.Spec.Ports)
	}
	wantLabels := map[string]string{"app": "nacos", "middleware": "nacos", "component": "nacos", "environment": "test"}
	if !reflect.DeepEqual(svc.Spec.Selector, wantLabels) {
		t.Fatalf("unexpected selector: %v", svc.Spec.Selector)
	}
	owner := metav1.GetControllerOf(svc)
	if owner == nil || owner.UID != nacos.UID || owner.Kind != "Nacos" {
		t.Fatalf("client Service lost controller ownership: %+v", owner)
	}
}

func TestClusterStartupPingsBeforeStartingNacos(t *testing.T) {
	var replicas int32 = 4
	nacos := &nacosv1alpha1.Nacos{
		ObjectMeta: metav1.ObjectMeta{Name: "nacos", Namespace: "test"},
		Spec:       nacosv1alpha1.NacosSpec{Replicas: &replicas},
	}
	sts := &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "nacos"}}},
	}}}
	client := &KindClient{}
	command := client.buildStatefulsetCluster(nacos, sts).Spec.Template.Spec.Containers[0].Command
	if len(command) != 3 || command[0] != "sh" || command[1] != "-c" || !strings.Contains(command[2], "&& exec bin/docker-startup.sh") {
		t.Fatalf("unexpected startup command: %q", command)
	}

	// Execute the generated shell script without DNS, network access or real sleeps.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"bin/ping":              "#!/bin/sh\nprintf '%s %s %s\\n' \"$1\" \"$2\" \"$3\" >> \"$PING_LOG\"\n",
		"bin/sleep":             "#!/bin/sh\nexit 0\n",
		"bin/docker-startup.sh": "#!/bin/sh\necho nacos-started\n",
	}
	for name, script := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = dir
	logPath := filepath.Join(dir, "pings")
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(dir, "bin")+":"+os.Getenv("PATH"), "PING_LOG="+logPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("startup script failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "nacos-started") {
		t.Fatalf("startup script did not start Nacos: %s", out)
	}
	pings, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	for _, host := range []string{"nacos-0", "nacos-1", "nacos-2", "nacos-3"} {
		want += "-c 1 " + host + ".nacos-headless.test.svc.cluster.local\n"
	}
	if string(pings) != want {
		t.Fatalf("ping flags must precede each hostname: got %q, want %q", pings, want)
	}
}
