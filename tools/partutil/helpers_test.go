// Copyright 2020 Google LLC
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

package partutil

import (
	"errors"
	"testing"
)

func TestCheck(t *testing.T) {
	var testData = []struct {
		testName string
		err      error
		errStr   string
		want     bool
	}{
		{
			"error with msg",
			errors.New("testing error"),
			"msg:testing error",
			true,
		}, {
			"error with no msg",
			errors.New(""),
			"",
			true,
		}, {
			"nil error",
			nil,
			"",
			false,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			if Check(input.err, input.errStr) != input.want {
				t.Errorf("wrongly detect error %v", input.err)
			}
		})
	}
}

func TestConvertSizeToBytes(t *testing.T) {
	testData := []struct {
		testName string
		in       string
		want     int
		expErr   bool
	}{
		{
			"ValidInputSector",
			"4194304",
			2147483648,
			false,
		}, {
			"ValidInputB",
			"4194304B",
			4194304,
			false,
		}, {
			"ValidInputK",
			"500K",
			512000,
			false,
		}, {
			"ValidInputM",
			"456M",
			478150656,
			false,
		}, {
			"ValidInputG",
			"321G",
			344671125504,
			false,
		}, {
			"InvalidSuffix",
			"10T",
			0,
			true,
		}, {
			"InvalidNumber",
			"56AXM",
			0,
			true,
		}, {
			"EmptyString",
			"",
			0,
			true,
		}, {
			"IntOverflow",
			"654654654654654654654654654654654654654654654654321321654654654",
			0,
			true,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			res, err := ConvertSizeToBytes(input.in)
			if (err != nil) != input.expErr {
				if input.expErr {
					t.Fatalf("error not found in test %s", input.testName)
				} else {
					t.Fatalf("error in test %s", input.testName)
				}
			}
			if !input.expErr {
				if res != input.want {
					t.Fatalf("wrong result: %s to %d, exp: %d", input.in, res, input.want)
				}
			}
		})
	}
}
