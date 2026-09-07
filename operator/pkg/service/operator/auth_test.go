/*
Copyright 2026.

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

package operator

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	nacosgroupv1alpha1 "nacos.io/nacos-operator/api/v1alpha1"
)

func TestBuildStatefulsetCertificationBooleanTriState(t *testing.T) {
	testCases := []struct {
		name                 string
		certification        nacosgroupv1alpha1.Certification
		expectedEnabled      *string
		expectedCacheEnabled *string
		expectedToken        *string
		expectedTokenExpire  *string
	}{
		{name: "omitted"},
		{
			name: "explicit true",
			certification: nacosgroupv1alpha1.Certification{
				Enabled:            boolPointer(true),
				CacheEnabled:       boolPointer(true),
				Token:              "test-token",
				TokenExpireSeconds: "3600",
			},
			expectedEnabled:      stringPointer("true"),
			expectedCacheEnabled: stringPointer("true"),
			expectedToken:        stringPointer("test-token"),
			expectedTokenExpire:  stringPointer("3600"),
		},
		{
			name: "explicit false",
			certification: nacosgroupv1alpha1.Certification{
				Enabled:            boolPointer(false),
				CacheEnabled:       boolPointer(false),
				Token:              "test-token",
				TokenExpireSeconds: "3600",
			},
			expectedEnabled:      stringPointer("false"),
			expectedCacheEnabled: stringPointer("false"),
			expectedToken:        stringPointer("test-token"),
			expectedTokenExpire:  stringPointer("3600"),
		},
		{
			name: "cache false without client override",
			certification: nacosgroupv1alpha1.Certification{
				CacheEnabled: boolPointer(false),
			},
			expectedCacheEnabled: stringPointer("false"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			env := buildCertificationTestEnvironment(t, testCase.certification)
			assertEnvironmentValue(t, env, "NACOS_AUTH_ENABLE", testCase.expectedEnabled)
			assertEnvironmentValue(t, env, "NACOS_AUTH_CACHE_ENABLE", testCase.expectedCacheEnabled)
			assertEnvironmentValue(t, env, "NACOS_AUTH_TOKEN", testCase.expectedToken)
			assertEnvironmentValue(t, env, "NACOS_AUTH_TOKEN_EXPIRE_SECONDS",
				testCase.expectedTokenExpire)
		})
	}
}

func TestSetDefaultCertificationOnlyForExplicitTrue(t *testing.T) {
	testCases := []struct {
		name          string
		enabled       *bool
		expectDefault bool
	}{
		{name: "omitted"},
		{name: "explicit false", enabled: boolPointer(false)},
		{name: "explicit true", enabled: boolPointer(true), expectDefault: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nacos := &nacosgroupv1alpha1.Nacos{
				Spec: nacosgroupv1alpha1.NacosSpec{
					Certification: nacosgroupv1alpha1.Certification{Enabled: testCase.enabled},
				},
			}
			setDefaultCertification(nacos)
			if testCase.expectDefault {
				if nacos.Spec.Certification.Token == "" ||
					nacos.Spec.Certification.TokenExpireSeconds != "18000" {
					t.Fatalf("explicit true did not retain certification defaults")
				}
				return
			}
			if nacos.Spec.Certification.Token != "" || nacos.Spec.Certification.TokenExpireSeconds != "" {
				t.Fatalf("certification defaults were added without explicit true")
			}
		})
	}
}

func buildCertificationTestEnvironment(t *testing.T,
	certification nacosgroupv1alpha1.Certification) []corev1.EnvVar {
	t.Helper()
	replicas := int32(1)
	nacos := &nacosgroupv1alpha1.Nacos{
		ObjectMeta: metav1.ObjectMeta{Name: "test-nacos", Namespace: "default"},
		Spec: nacosgroupv1alpha1.NacosSpec{
			Type:          TYPE_STAND_ALONE,
			Replicas:      &replicas,
			Database:      nacosgroupv1alpha1.Database{TypeDatabase: "embedded"},
			Certification: certification,
		},
	}
	scheme := runtime.NewScheme()
	if err := nacosgroupv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add Nacos API to scheme: %v", err)
	}
	statefulSet := (&KindClient{scheme: scheme}).buildStatefulset(nacos)
	return statefulSet.Spec.Template.Spec.Containers[0].Env
}

func assertEnvironmentValue(t *testing.T, env []corev1.EnvVar, name string, expected *string) {
	t.Helper()
	var values []string
	for i := range env {
		if env[i].Name == name {
			values = append(values, env[i].Value)
		}
	}
	if expected == nil {
		if len(values) != 0 {
			t.Fatalf("%s appeared %d times, expected it to be omitted", name, len(values))
		}
		return
	}
	if len(values) != 1 || values[0] != *expected {
		t.Fatalf("%s = %v, expected exactly one value %q", name, values, *expected)
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
