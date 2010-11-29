// Copyright 2010 Gary Burd
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package web

import (
	"testing"
	"reflect"
)

type ParseCookieValuesTest struct {
	values []string
	m      StringsMap
}

var ParseCookieValuesTests = []ParseCookieValuesTest{
	{[]string{"a=b"}, StringsMap{"a": []string{"b"}}},
	{[]string{"a=b; c"}, StringsMap{"a": []string{"b"}}},
	{[]string{"a=b; =c"}, StringsMap{"a": []string{"b"}}},
	{[]string{"a=b; ; "}, StringsMap{"a": []string{"b"}}},
	{[]string{"a=b; c=d"}, StringsMap{"a": []string{"b"}, "c": []string{"d"}}},
	{[]string{"a=b; c=d"}, StringsMap{"a": []string{"b"}, "c": []string{"d"}}},
	{[]string{"a=b;c=d"}, StringsMap{"a": []string{"b"}, "c": []string{"d"}}},
	{[]string{" a=b;c=d "}, StringsMap{"a": []string{"b"}, "c": []string{"d"}}},
	{[]string{"a=b", "c=d"}, StringsMap{"a": []string{"b"}, "c": []string{"d"}}},
	{[]string{"a=b", "c=x=y"}, StringsMap{"a": []string{"b"}, "c": []string{"x=y"}}},
}

func TestParseCookieValues(t *testing.T) {
	for _, pt := range ParseCookieValuesTests {
		m := make(StringsMap)
		if err := parseCookieValues(pt.values, m); err != nil {
			t.Errorf("error parsing values %s", err)
		}
		if !reflect.DeepEqual(pt.m, m) {
			t.Errorf("values=%s,\nexpected %q\nactual   %q", pt.values, pt.m, m)
		}
	}
}

func TestSignValue(t *testing.T) {
	secret := "7d1355a24a7bc1ad97a01f0252a5ba23e8b0aa366f1aa4d2c84b78ccdd6743a7"
	context := "UID"
	expectedValue := "admin~role"
	actualValue, err := VerifyValue(secret, context, SignValue(secret, context, 3600, expectedValue))
	if err != nil || expectedValue != actualValue {
		t.Error("verify failed", err, actualValue)
	}
}
