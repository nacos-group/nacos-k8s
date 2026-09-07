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

package v1alpha1

import (
	"encoding/json"
	"testing"
)

func TestCertificationBooleanRoundTrip(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		expectedEnabled *bool
		expectedCache   *bool
	}{
		{name: "omitted", input: `{}`},
		{
			name:            "explicit true",
			input:           `{"enabled":true,"cache_enabled":true}`,
			expectedEnabled: booleanPointer(true),
			expectedCache:   booleanPointer(true),
		},
		{
			name:            "explicit false",
			input:           `{"enabled":false,"cache_enabled":false}`,
			expectedEnabled: booleanPointer(false),
			expectedCache:   booleanPointer(false),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var certification Certification
			if err := json.Unmarshal([]byte(testCase.input), &certification); err != nil {
				t.Fatalf("unmarshal certification: %v", err)
			}
			assertBooleanPointer(t, "enabled", certification.Enabled, testCase.expectedEnabled)
			assertBooleanPointer(t, "cache_enabled", certification.CacheEnabled, testCase.expectedCache)

			encoded, err := json.Marshal(certification)
			if err != nil {
				t.Fatalf("marshal certification: %v", err)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatalf("unmarshal encoded certification: %v", err)
			}
			assertBooleanField(t, fields, "enabled", testCase.expectedEnabled)
			assertBooleanField(t, fields, "cache_enabled", testCase.expectedCache)
		})
	}
}

func assertBooleanField(t *testing.T, fields map[string]interface{}, name string, expected *bool) {
	t.Helper()
	value, found := fields[name]
	if expected == nil {
		if found {
			t.Fatalf("%s was serialized with value %v, expected it to be omitted", name, value)
		}
		return
	}
	if !found || value != *expected {
		t.Fatalf("%s = %v, expected %t", name, value, *expected)
	}
}

func assertBooleanPointer(t *testing.T, name string, actual *bool, expected *bool) {
	t.Helper()
	if expected == nil {
		if actual != nil {
			t.Fatalf("%s = %t, expected nil", name, *actual)
		}
		return
	}
	if actual == nil || *actual != *expected {
		t.Fatalf("%s = %v, expected %t", name, actual, *expected)
	}
}

func booleanPointer(value bool) *bool {
	return &value
}
