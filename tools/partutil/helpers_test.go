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
	"testing"
)

func TestConvertSizeToBytesFails(t *testing.T) {
	testData := []struct {
		testName string
		input    string
	}{
		{
			testName: "InvalidSuffix",
			input:    "10T",
		}, {
			testName: "InvalidNumber",
			input:    "56AXM",
		}, {
			testName: "EmptyString",
			input:    "",
		}, {
			testName: "IntOverflow",
			input:    "654654654654654654654654654654654654654654654654321321654654654",
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			_, err := ConvertSizeToBytes(input.input)
			if err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}
}

func TestConvertSizeToBytesPasses(t *testing.T) {
	testData := []struct {
		testName string
		input    string
		want     uint64
	}{
		{
			testName: "ValidInputSector",
			input:    "4194304",
			want:     2147483648,
		}, {
			testName: "ValidInputB",
			input:    "4194304B",
			want:     4194304,
		}, {
			testName: "ValidInputK",
			input:    "500K",
			want:     512000,
		}, {
			testName: "ValidInputM",
			input:    "456M",
			want:     478150656,
		}, {
			testName: "ValidInputG",
			input:    "321G",
			want:     344671125504,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			res, err := ConvertSizeToBytes(input.input)
			if err != nil {
				t.Fatalf("errorin test %s, error msg: (%v)", input.testName, err)
			}
			if res != input.want {
				t.Fatalf("wrong result: %q to %d, expect: %d", input.input, res, input.want)
			}
		})
	}
}

func TestPartNumIntToStringFails(t *testing.T) {
	_, err := PartNumIntToString("", 1)
	if err == nil {
		t.Fatal("error not found in test EmptyDiskName")
	}
}

func TestPartNumIntToStringPasses(t *testing.T) {
	testData := []struct {
		testName   string
		diskName   string
		partNumInt int
		want       string
	}{
		{
			testName:   "LetterEndDisk",
			diskName:   "/dev/sda",
			partNumInt: 1,
			want:       "1",
		},
		{
			testName:   "NumberEndDisk",
			diskName:   "/dev/loop5",
			partNumInt: 1,
			want:       "p1",
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			res, err := PartNumIntToString(input.diskName, input.partNumInt)
			if err != nil {
				t.Fatalf("error in test %s, error msg: (%v)", input.testName, err)
			}
			if res != input.want {
				t.Fatalf("error in test %s, wrong result: %q, expected: %q", input.testName, res, input.want)
			}
		})
	}
}

func TestConvertSizeToGBRoundUpFails(t *testing.T) {
	testData := []struct {
		testName string
		input    string
	}{
		{
			testName: "InvalidSuffix",
			input:    "10T",
		}, {
			testName: "InvalidNumber",
			input:    "56AXM",
		}, {
			testName: "EmptyString",
			input:    "",
		}, {
			testName: "IntOverflow",
			input:    "654654654654654654654654654654654654654654654654321321654654654",
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			_, err := ConvertSizeToGBRoundUp(input.input)
			if err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}
}

func TestConvertSizeToGBRoundUpPasses(t *testing.T) {
	testData := []struct {
		testName string
		input    string
		want     uint64
	}{
		{
			testName: "ValidInputSector",
			input:    "4194304",
			want:     2,
		}, {
			testName: "ValidInputB",
			input:    "4194304B",
			want:     1,
		}, {
			testName: "ValidInputK",
			input:    "500K",
			want:     1,
		}, {
			testName: "ValidInputM",
			input:    "456M",
			want:     1,
		}, {
			testName: "ValidInputG",
			input:    "321G",
			want:     321,
		},
		{
			testName: "ValidInputM2",
			input:    "2096M",
			want:     3,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			res, err := ConvertSizeToGBRoundUp(input.input)
			if err != nil {
				t.Fatalf("errorin test %s, error msg: (%v)", input.testName, err)
			}
			if res != input.want {
				t.Fatalf("wrong result: %q to %d, expect: %d", input.input, res, input.want)
			}
		})
	}
}
