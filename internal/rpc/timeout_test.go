// Copyright 2026 Erst Users
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalTimeout_ExpiresBeforeAllNodes(t *testing.T) {
	// Create slow servers that take 2 seconds each
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server3.Close()

	// Create client with 3 second global timeout (should timeout before trying all 3 servers)
	client, err := NewClient(
		WithAltURLs([]string{server1.URL, server2.URL, server3.URL}),
		WithTotalTimeout(3*time.Second),
		WithRequestTimeout(5*time.Second), // Individual request timeout longer than global
	)
	require.NoError(t, err)

	start := time.Now()
	ctx := context.Background()
	
	_, err = client.GetTransaction(ctx, "test-hash")
	
	elapsed := time.Since(start)
	
	// Should fail with timeout error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global timeout exceeded")
	
	// Should timeout around 3 seconds, not 6+ seconds (2s per server * 3 servers)
	assert.Less(t, elapsed, 4*time.Second, "Should timeout before trying all servers")
	assert.Greater(t, elapsed, 2*time.Second, "Should have tried at least one server")
}

func TestGlobalTimeout_SucceedsWithinTimeout(t *testing.T) {
	// First server fails quickly, second succeeds
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate successful transaction response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"hash": "test-hash",
			"ledger": 123,
			"envelope_xdr": "test-envelope",
			"result_xdr": "test-result",
			"result_meta_xdr": "test-meta",
			"successful": true
		}`)
	}))
	defer server2.Close()

	client, err := NewClient(
		WithAltURLs([]string{server1.URL, server2.URL}),
		WithTotalTimeout(10*time.Second),
	)
	require.NoError(t, err)

	start := time.Now()
	ctx := context.Background()
	
	resp, err := client.GetTransaction(ctx, "test-hash")
	
	elapsed := time.Since(start)
	
	// Should succeed
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-hash", resp.Hash)
	
	// Should complete quickly (well under the 10s timeout)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestGlobalTimeout_NoTimeoutSet(t *testing.T) {
	// Server that takes a long time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client without global timeout (should use default behavior)
	client, err := NewClient(
		WithAltURLs([]string{server.URL}),
		WithRequestTimeout(2*time.Second),
	)
	require.NoError(t, err)

	// Verify default timeout is set
	assert.Equal(t, 60*time.Second, client.Config.TotalTimeout)

	ctx := context.Background()
	_, err = client.GetTransaction(ctx, "test-hash")
	
	// Should fail with connection error, not timeout
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "global timeout exceeded")
}

func TestGlobalTimeout_ZeroTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with zero timeout (should disable global timeout)
	client, err := NewClient(
		WithAltURLs([]string{server.URL}),
		WithTotalTimeout(0),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = client.GetTransaction(ctx, "test-hash")
	
	// Should fail with connection error, not timeout
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "global timeout exceeded")
}

func TestGlobalTimeout_ContextAlreadyCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(
		WithAltURLs([]string{server.URL}),
		WithTotalTimeout(10*time.Second),
	)
	require.NoError(t, err)

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	_, err = client.GetTransaction(ctx, "test-hash")
	
	// Should fail with context cancelled error
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGlobalTimeout_NetworkConfigDefault(t *testing.T) {
	// Test that predefined network configs have default timeout
	assert.Equal(t, 60*time.Second, TestnetConfig.TotalTimeout)
	assert.Equal(t, 60*time.Second, MainnetConfig.TotalTimeout)
	assert.Equal(t, 60*time.Second, FuturenetConfig.TotalTimeout)
}

func TestWithTotalTimeout_BuilderOption(t *testing.T) {
	client, err := NewClient(
		WithNetwork(Testnet),
		WithTotalTimeout(30*time.Second),
	)
	require.NoError(t, err)
	
	assert.Equal(t, 30*time.Second, client.Config.TotalTimeout)
}

func TestGlobalTimeout_GetLedgerHeader(t *testing.T) {
	// Create slow servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	client, err := NewClient(
		WithAltURLs([]string{server1.URL, server2.URL}),
		WithTotalTimeout(3*time.Second),
	)
	require.NoError(t, err)

	start := time.Now()
	ctx := context.Background()
	
	_, err = client.GetLedgerHeader(ctx, 123)
	
	elapsed := time.Since(start)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global timeout exceeded")
	assert.Less(t, elapsed, 4*time.Second)
}

func TestGlobalTimeout_SimulateTransaction(t *testing.T) {
	// Create slow servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	client, err := NewClient(
		WithAltURLs([]string{server1.URL, server2.URL}),
		WithTotalTimeout(3*time.Second),
	)
	require.NoError(t, err)

	start := time.Now()
	ctx := context.Background()
	
	_, err = client.SimulateTransaction(ctx, "test-envelope-xdr")
	
	elapsed := time.Since(start)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global timeout exceeded")
	assert.Less(t, elapsed, 4*time.Second)
}

func TestGlobalTimeout_GetHealth(t *testing.T) {
	// Create slow servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	client, err := NewClient(
		WithAltURLs([]string{server1.URL, server2.URL}),
		WithTotalTimeout(3*time.Second),
	)
	require.NoError(t, err)

	start := time.Now()
	ctx := context.Background()
	
	_, err = client.GetHealth(ctx)
	
	elapsed := time.Since(start)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global timeout exceeded")
	assert.Less(t, elapsed, 4*time.Second)
}