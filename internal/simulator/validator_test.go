// Copyright 2026 Erst Users
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLedgerSequence(t *testing.T) {
	tests := []struct {
		name        string
		sequence    uint32
		strictMode  bool
		expectError bool
		errorCode   string
	}{
		{
			name:        "valid sequence",
			sequence:    12345,
			strictMode:  false,
			expectError: false,
		},
		{
			name:        "zero sequence",
			sequence:    0,
			strictMode:  false,
			expectError: true,
			errorCode:   "ERR_INVALID_SEQUENCE",
		},
		{
			name:        "max safe uint32",
			sequence:    math.MaxUint32 - 1,
			strictMode:  false,
			expectError: true,
			errorCode:   "ERR_OVERFLOW_RISK",
		},
		{
			name:        "max uint32",
			sequence:    math.MaxUint32,
			strictMode:  false,
			expectError: true,
			errorCode:   "ERR_OVERFLOW_RISK",
		},
		{
			name:        "near overflow boundary",
			sequence:    math.MaxUint32 - 10,
			strictMode:  false,
			expectError: true,
			errorCode:   "ERR_OVERFLOW_RISK",
		},
		{
			name:        "strict mode - exceeds reasonable limit",
			sequence:    1000000001,
			strictMode:  true,
			expectError: true,
			errorCode:   "ERR_VALUE_TOO_HIGH",
		},
		{
			name:        "strict mode - at reasonable limit",
			sequence:    1000000000,
			strictMode:  true,
			expectError: false,
		},
		{
			name:        "strict mode - below reasonable limit",
			sequence:    999999999,
			strictMode:  true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(tt.strictMode)
			err := validator.validateLedgerSequence(tt.sequence)

			if tt.expectError {
				require.Error(t, err)
				validationErr, ok := err.(*ValidationError)
				require.True(t, ok, "expected ValidationError")
				assert.Equal(t, tt.errorCode, validationErr.Code)
				assert.Equal(t, "ledger_sequence", validationErr.Field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLedgerSequenceEdgeCases(t *testing.T) {
	validator := NewValidator(false)

	// Test sequences near uint32 boundaries
	edgeCases := []uint32{
		4294967293, // MaxUint32 - 2
		4294967294, // MaxUint32 - 1 (should fail)
		4294967295, // MaxUint32 (should fail)
	}

	for i, sequence := range edgeCases {
		t.Run(fmt.Sprintf("edge_case_%d", i), func(t *testing.T) {
			err := validator.validateLedgerSequence(sequence)
			if sequence >= 4294967294 {
				require.Error(t, err)
				validationErr, ok := err.(*ValidationError)
				require.True(t, ok)
				assert.Equal(t, "ERR_OVERFLOW_RISK", validationErr.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}