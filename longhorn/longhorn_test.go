package longhorn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestGetVolumeNameFromReplicaDataDirectoryName(t *testing.T) {
	type testCase struct {
		replicaName string

		expected    string
		expectError bool
	}
	testCases := map[string]testCase{
		"Valid replica name": {
			replicaName: "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95-971c46f6",
			expected:    "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95",
		},
		"Empty replica name": {
			replicaName: "",
			expectError: true,
		},
		"Invalid replica name": {
			replicaName: "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95-00",
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GetVolumeNameFromReplicaDataDirectoryName(testCase.replicaName)
			if testCase.expectError {
				assert.NotNil(t, err, Commentf(test.ErrErrorFmt, testName, err))
				return
			}
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))

			assert.Equal(t, result, testCase.expected, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetVolumeNameFromReplicaCRName(t *testing.T) {
	type testCase struct {
		replicaCRName string

		expected    string
		expectError bool
	}
	testCases := map[string]testCase{
		"Valid replica CR name": {
			replicaCRName: "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95-r-971c46f6",
			expected:      "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95",
		},
		"Volume name ending with a replica like suffix": {
			replicaCRName: "data-r-abcdef12-r-9f8e7d6c",
			expected:      "data-r-abcdef12",
		},
		"Single character volume name": {
			replicaCRName: "v-r-00000000",
			expected:      "v",
		},
		"Replica data directory name rather than CR name": {
			replicaCRName: "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95-971c46f6",
			expectError:   true,
		},
		"Engine name rather than replica CR name": {
			replicaCRName: "pvc-0e045ff8-4ea6-4573-889b-afc9aa147f95-e-971c46f6",
			expectError:   true,
		},
		"Random ID too short": {
			replicaCRName: "vol-1-r-short",
			expectError:   true,
		},
		"Random ID too long": {
			replicaCRName: "vol-1-r-toolongid123",
			expectError:   true,
		},
		"Empty volume name": {
			replicaCRName: "-r-5d8209ef",
			expectError:   true,
		},
		"Empty replica CR name": {
			replicaCRName: "",
			expectError:   true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GetVolumeNameFromReplicaCRName(testCase.replicaCRName)
			if testCase.expectError {
				assert.NotNil(t, err, Commentf(test.ErrErrorFmt, testName, err))
				return
			}
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))

			assert.Equal(t, result, testCase.expected, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestIsEngineProcess(t *testing.T) {
	type testCase struct {
		input    string
		expected bool
	}
	testCases := map[string]testCase{
		"Longhorn PVC": {
			input:    "pvc-5a8ee916-5989-46c6-bafc-ddbf7c802499-e-0",
			expected: true,
		},
		"Engine": {
			input:    "nginx-e-0",
			expected: true,
		},
		"Engine-2": {
			input:    "nginx-r-e-0",
			expected: true,
		},
		"Engine-3": {
			input:    "pvc-669e5426-8c62-42df-979d-1be22a30cd0a-e-cc7d5051",
			expected: true,
		},
		"Engine-4": {
			input:    "pvc-3308aae1-b3c4-4ea3-a6b8-d1fc16cea03b-e-8e24327e",
			expected: true,
		},
		"Replica": {
			input:    "nginx-r-0",
			expected: false,
		},
		"Replica-2": {
			input:    "nginx-e-r-0",
			expected: false,
		},
		"Invalid": {
			input:    "invalid-string",
			expected: false,
		},
		"Invalid-2": {
			input:    "-e-0",
			expected: false,
		},
		"Invalid-3": {
			input:    "abc-eee-0",
			expected: false,
		},
		"Invalid-4": {
			input:    "nginx-er-0",
			expected: false,
		},
		"Invalid-5": {
			input:    "nginx-e--0",
			expected: false,
		},
		"Invalid-6": {
			input:    "nginx-e-0-abcd",
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := IsEngineProcess(testCase.input)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}
