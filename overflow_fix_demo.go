// Copyright 2026 Erst Users
// SPDX-License-Identifier: Apache-2.0

// Package main demonstrates the XDR integer overflow fix
package main

import (
	"encoding/base64"
	"fmt"
	"math"

	"github.com/dotandev/hintents/internal/simulator"
)

func main() {
	fmt.Println("XDR Integer Overflow Fix Demonstration")
	fmt.Println("=====================================")
	
	validator := simulator.NewValidator(false)
	validXDR := base64.StdEncoding.EncodeToString([]byte("demo xdr data"))

	// Test cases demonstrating the overflow protection
	testCases := []struct {
		name     string
		sequence uint32
		expected string
	}{
		{"Normal sequence", 12345, "PASS"},
		{"Large but safe sequence", math.MaxUint32 - 10, "PASS"},
		{"Near overflow boundary", math.MaxUint32 - 2, "PASS"},
		{"At overflow boundary", math.MaxUint32 - 1, "FAIL - Overflow Risk"},
		{"Max uint32", math.MaxUint32, "FAIL - Overflow Risk"},
		{"Zero sequence", 0, "FAIL - Invalid Sequence"},
	}

	fmt.Printf("Testing ledger sequences near uint32 boundary (MaxUint32 = %d):\n\n", math.MaxUint32)

	for _, tc := range testCases {
		req := &simulator.SimulationRequest{
			EnvelopeXdr:    validXDR,
			ResultMetaXdr:  validXDR,
			LedgerSequence: tc.sequence,
			Timestamp:      1735689600,
		}

		err := validator.ValidateRequest(req)
		
		status := "PASS"
		errorMsg := ""
		if err != nil {
			status = "FAIL"
			errorMsg = fmt.Sprintf(" - %s", err.Error())
		}

		fmt.Printf("%-25s | Sequence: %-10d | %s%s\n", 
			tc.name, tc.sequence, status, errorMsg)
	}

	fmt.Println("\nOverflow Protection Summary:")
	fmt.Println("- Sequences >= (MaxUint32 - 1) are rejected to prevent overflow")
	fmt.Println("- Session increment logic safely handles overflow by resetting to 1")
	fmt.Println("- Comprehensive test coverage ensures edge cases are handled")
}