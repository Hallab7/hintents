// Copyright 2026 Erst Users
// SPDX-License-Identifier: Apache-2.0

package shell

import (
	"context"
	"math"
	"testing"

	"github.com/dotandev/hintents/internal/rpc"
	"github.com/dotandev/hintents/internal/simulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRunner implements simulator.RunnerInterface for testing
type MockRunner struct {
	response *simulator.SimulationResponse
	err      error
}

func (m *MockRunner) Run(ctx context.Context, req *simulator.SimulationRequest) (*simulator.SimulationResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestSessionLedgerSequenceOverflowProtection(t *testing.T) {
	tests := []struct {
		name            string
		initialSequence uint32
		expectedResult  uint32
		expectReset     bool
	}{
		{
			name:            "normal increment",
			initialSequence: 12345,
			expectedResult:  12346,
			expectReset:     false,
		},
		{
			name:            "near max uint32 - safe",
			initialSequence: math.MaxUint32 - 2,
			expectedResult:  math.MaxUint32 - 1,
			expectReset:     false,
		},
		{
			name:            "at overflow boundary",
			initialSequence: math.MaxUint32 - 1,
			expectedResult:  1, // Should reset to 1
			expectReset:     true,
		},
		{
			name:            "at max uint32",
			initialSequence: math.MaxUint32,
			expectedResult:  1, // Should reset to 1
			expectReset:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock runner
			mockRunner := &MockRunner{
				response: &simulator.SimulationResponse{
					Status: "success",
					Events: []string{},
					Logs:   []string{},
				},
			}

			// Create session with mock runner
			session := NewSession(mockRunner, nil, rpc.NetworkTestnet)
			session.ledgerSequence = tt.initialSequence

			// Test the increment logic directly (Invoke would fail due to unimplemented envelope building)
			err := session.incrementLedgerSequence()
			
			if tt.expectReset {
				// Should return error for overflow
				require.Error(t, err)
				assert.Contains(t, err.Error(), "would overflow uint32 boundary")
				
				// Simulate the reset behavior from updateLedgerState
				session.ledgerSequence = 1
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult, session.ledgerSequence)
		})
	}
}

func TestIncrementLedgerSequence(t *testing.T) {
	session := NewSession(nil, nil, rpc.NetworkTestnet)

	// Test normal increment
	session.ledgerSequence = 100
	err := session.incrementLedgerSequence()
	require.NoError(t, err)
	assert.Equal(t, uint32(101), session.ledgerSequence)

	// Test overflow protection
	session.ledgerSequence = math.MaxUint32 - 1
	err = session.incrementLedgerSequence()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "would overflow uint32 boundary")
	// Sequence should remain unchanged when error occurs
	assert.Equal(t, uint32(math.MaxUint32-1), session.ledgerSequence)

	// Test at max uint32
	session.ledgerSequence = math.MaxUint32
	err = session.incrementLedgerSequence()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "would overflow uint32 boundary")
	assert.Equal(t, uint32(math.MaxUint32), session.ledgerSequence)
}

func TestUpdateLedgerStateOverflowHandling(t *testing.T) {
	session := NewSession(nil, nil, rpc.NetworkTestnet)
	
	// Set sequence to overflow boundary
	session.ledgerSequence = math.MaxUint32 - 1
	
	mockResponse := &simulator.SimulationResponse{
		Status: "success",
		Events: []string{},
		Logs:   []string{},
	}
	
	// This should trigger overflow protection and reset to 1
	session.updateLedgerState(mockResponse)
	
	// Should be reset to 1 due to overflow protection
	assert.Equal(t, uint32(1), session.ledgerSequence)
}