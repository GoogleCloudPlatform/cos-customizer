package partutil

import "testing"

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
