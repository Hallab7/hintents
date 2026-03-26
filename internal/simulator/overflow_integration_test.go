// Copyright 2026 Erst Users
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"encoding/base64"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestXDRIntegerOverflowIntegration tests the complete overflow protection pipeline
func TestXDRIntegerOverflowIntegration(t *testing.T) {
	validXDR := base64.StdEncoding.EncodeToString([]byte("valid xdr data"))

	tests := []struct {
		name           string
		ledgerSequence uint32
		strictMode     bool
		expectError    bool
		errorCode      string
		description    string
	}{
		{
			name:           "normal_sequence",
			ledgerSequence: 12345,
			strictMode:     false,
			expectError:    false,
			description:    "Normal ledger sequence should pass validation",
		},
		{
			name:           "max_safe_sequence",
			ledgerSequence: math.MaxUint32 - 2,
			strictMode:     false,
			expectError:    false,
			description:    "Sequence at MaxUint32-2 should pass (can still increment once)",
		},
		{
			name:           "overflow_boundary_minus_1",
			ledgerSequence: math.MaxUint32 - 1,
			strictMode:     false,
			expectError:    true,
			errorCode:      "ERR_OVERFLOW_RISK",
			description:    "Sequence at MaxUint32-1 should fail (would overflow on increment)",
		},
		{
			name:           "max_uint32",
			ledgerSequence: math.MaxUint32,
			strictMode:     false,
			expectError:    true,
			errorCode:      "ERR_OVERFLOW_RISK",
			description:    "Sequence at MaxUint32 should fail (already at overflow)",
		},
		{
			name:           "strict_mode_reasonable_limit",
			ledgerSequence: 1000000001,
			strictMode:     true,
			expectError:    true,
			errorCode:      "ERR_VALUE_TOO_HIGH",
			description:    "Strict mode should enforce reasonable limits",
		},
		{
			name:           "zero_sequence",
			ledgerSequence: 0,
			strictMode:     false,
			expectError:    true,
			errorCode:      "ERR_INVALID_SEQUENCE",
			description:    "Zero sequence should always fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(tt.strictMode)

			req := &SimulationRequest{
				EnvelopeXdr:    validXDR,
				ResultMetaXdr:  validXDR,
				LedgerSequence: tt.ledgerSequence,
				Timestamp:      1735689600,
				LedgerEntries: map[string]string{
					validXDR: validXDR,
				},
			}

			err := validator.ValidateRequest(req)

			if tt.expectError {
				require.Error(t, err, tt.description)
				validationErr, ok := err.(*ValidationError)
				require.True(t, ok, "Expected ValidationError type")
				assert.Equal(t, tt.errorCode, validationErr.Code, "Error code mismatch")
				assert.Equal(t, "ledger_sequence", validationErr.Field, "Field name mismatch")
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestLedgerSequenceBoundaryValues tests specific boundary values around uint32 limits
func TestLedgerSequenceBoundaryValues(t *testing.T) {
	validator := NewValidator(false)
	validXDR := base64.StdEncoding.EncodeToString([]byte("valid xdr data"))

	// Test values around the uint32 boundary
	boundaryTests := []struct {
		sequence    uint32
		shouldPass  bool
		description string
	}{
		{4294967290, true, "MaxUint32 - 5 should pass"},
		{4294967291, true, "MaxUint32 - 4 should pass"},
		{4294967292, true, "MaxUint32 - 3 should pass"},
		{4294967293, true, "MaxUint32 - 2 should pass (last safe value)"},
		{4294967294, false, "MaxUint32 - 1 should fail (overflow risk)"},
		{4294967295, false, "MaxUint32 should fail (overflow risk)"},
	}

	for _, bt := range boundaryTests {
		t.Run(bt.description, func(t *testing.T) {
			req := &SimulationRequest{
				EnvelopeXdr:    validXDR,
				ResultMetaXdr:  validXDR,
				LedgerSequence: bt.sequence,
				Timestamp:      1735689600,
			}

			err := validator.ValidateRequest(req)

			if bt.shouldPass {
				assert.NoError(t, err, "Sequence %d should pass validation", bt.sequence)
			} else {
				require.Error(t, err, "Sequence %d should fail validation", bt.sequence)
				validationErr, ok := err.(*ValidationError)
				require.True(t, ok, "Expected ValidationError")
				assert.Equal(t, "ERR_OVERFLOW_RISK", validationErr.Code)
			}
		})
	}
}

// TestOverflowProtectionConstants verifies our constants are correct
func TestOverflowProtectionConstants(t *testing.T) {
	// Verify our understanding of uint32 limits
	assert.Equal(t, uint32(4294967295), uint32(math.MaxUint32), "MaxUint32 constant verification")
	
	// Verify our safe boundary calculation
	const maxSafeUint32 = uint32(4294967294) // math.MaxUint32 - 1
	assert.Equal(t, uint32(math.MaxUint32-1), maxSafeUint32, "Safe boundary calculation")
	
	// Verify that incrementing the safe boundary would cause overflow
	testVal := maxSafeUint32
	testVal++ // This should equal MaxUint32
	assert.Equal(t, uint32(math.MaxUint32), testVal, "Increment verification")
	
	// Verify that incrementing MaxUint32 would overflow to 0
	testVal = uint32(math.MaxUint32)
	testVal++ // This should overflow to 0
	assert.Equal(t, uint32(0), testVal, "Overflow verification")
}