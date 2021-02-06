// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMapVar(t *testing.T) {
	var testData = []struct {
		testName string
		flags    []string
		want     map[string]string
	}{
		{
			"OneFlag",
			[]string{"a=A,b=B"},
			map[string]string{"a": "A", "b": "B"},
		},
		{
			"TwoFlag",
			[]string{"a=A,b=B", "c=C"},
			map[string]string{"a": "A", "b": "B", "c": "C"},
		},
		{
			"MultipleEquals",
			[]string{"a=A=B=C,d=D"},
			map[string]string{"a": "A=B=C", "d": "D"},
		},
	}
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			mv := newMapVar()
			for _, flag := range input.flags {
				if err := mv.Set(flag); err != nil {
					t.Fatalf("mapVar.Set(%s) = %s; want nil", flag, err)
				}
			}
			if diff := cmp.Diff(mv.m, input.want); diff != "" {
				t.Errorf("mapVar: got unexpected result with flags %v: diff (-got, +want):\n%v", input.flags, diff)
			}
		})
	}
}

func TestListVar(t *testing.T) {
	var testData = []struct {
		testName string
		flags    []string
		want     []string
	}{
		{
			"OneFlag",
			[]string{"a,b"},
			[]string{"a", "b"},
		},
		{
			"TwoFlag",
			[]string{"a", "b"},
			[]string{"a", "b"},
		},
	}
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			lv := &listVar{}
			for _, flag := range input.flags {
				if err := lv.Set(flag); err != nil {
					t.Fatalf("listVar.Set(%s) = %s; want nil", flag, err)
				}
			}
			if got := lv.l; !cmp.Equal(got, input.want) {
				t.Errorf("listVar: got unexpected result with flags %v: got %v, want %v", input.flags, got, input.want)
			}
		})
	}
}
