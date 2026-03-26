# Global Timeout for Multi-Node Failover

## Overview

The SDK now supports a global timeout feature that caps the total time spent across all RPC nodes during failover operations. This prevents scenarios where slow nodes cause the entire failover loop to take too long.

## Problem Solved

Previously, if all RPC nodes were slow (but not completely unresponsive), the failover loop could take an excessive amount of time. For example, with 3 nodes each taking 30 seconds to timeout, the total operation could take up to 90 seconds.

With the global timeout feature, you can set a maximum time limit for the entire failover operation, regardless of how many nodes are configured.

## Configuration

### Go SDK

#### Using NetworkConfig

```go
config := rpc.NetworkConfig{
    Name:              "custom",
    HorizonURL:        "https://horizon.stellar.org",
    NetworkPassphrase: "Public Global Stellar Network ; September 2015",
    SorobanRPCURL:     "https://soroban-rpc.stellar.org",
    TotalTimeout:      30 * time.Second, // 30 second global timeout
}

client, err := rpc.NewClient(rpc.WithNetworkConfig(config))
```

#### Using Builder Options

```go
client, err := rpc.NewClient(
    rpc.WithNetwork(rpc.Mainnet),
    rpc.WithAltURLs([]string{
        "https://horizon1.stellar.org",
        "https://horizon2.stellar.org", 
        "https://horizon3.stellar.org",
    }),
    rpc.WithTotalTimeout(45 * time.Second), // 45 second global timeout
)
```

#### Default Values

The predefined network configurations have a default global timeout of 60 seconds:
- `TestnetConfig.TotalTimeout = 60 * time.Second`
- `MainnetConfig.TotalTimeout = 60 * time.Second`
- `FuturenetConfig.TotalTimeout = 60 * time.Second`

### TypeScript SDK

#### Using RPCConfig

```typescript
import { FallbackRPCClient, RPCConfigParser } from './rpc';

const config = RPCConfigParser.loadConfig({
    rpc: ['https://rpc1.stellar.org', 'https://rpc2.stellar.org'],
    timeout: 30000,      // Individual request timeout (30s)
    totalTimeout: 60000, // Global timeout for all nodes (60s)
    retries: 3,
});

const client = new FallbackRPCClient(config);
```

#### Default Values

The TypeScript SDK has a default global timeout of 60 seconds (60000ms) when using `RPCConfigParser.loadConfig()`.

## Behavior

### When Global Timeout is Enabled

1. **Timeout Enforcement**: The SDK starts a timer when beginning the failover loop
2. **Context Cancellation**: If the global timeout is reached, the operation is cancelled immediately
3. **Error Response**: A timeout error is returned indicating the global timeout was exceeded
4. **Early Termination**: The SDK stops trying additional nodes once the timeout is reached

### When Global Timeout is Disabled

- Set `TotalTimeout` to `0` (Go) or `totalTimeout` to `0` (TypeScript) to disable
- The SDK will attempt all configured nodes regardless of total time spent
- Individual request timeouts still apply

## Affected Methods

The global timeout applies to all methods that use multi-node failover:

### Go SDK
- `GetTransaction()`
- `GetLedgerHeader()`
- `SimulateTransaction()`
- `GetHealth()`
- `GetLedgerEntries()` (single batch mode)

### TypeScript SDK
- All methods that use `FallbackRPCClient.request()`
- `getTransaction()`
- `simulateTransaction()`
- `getHealth()`
- `getLatestLedger()`

## Examples

### Scenario 1: Fast Failover

```go
// 3 nodes, each taking 2 seconds to fail
// Without global timeout: 6+ seconds total
// With 4-second global timeout: ~4 seconds total

client, _ := rpc.NewClient(
    rpc.WithAltURLs([]string{"slow1", "slow2", "slow3"}),
    rpc.WithTotalTimeout(4 * time.Second),
)

start := time.Now()
_, err := client.GetTransaction(ctx, "hash")
elapsed := time.Since(start) // ~4 seconds, not 6+
```

### Scenario 2: Success Within Timeout

```go
// First node fails quickly, second succeeds
// Total time: <1 second (well within 10s timeout)

client, _ := rpc.NewClient(
    rpc.WithAltURLs([]string{"fail-fast", "success"}),
    rpc.WithTotalTimeout(10 * time.Second),
)

resp, err := client.GetTransaction(ctx, "hash") // Success
```

## Error Handling

### Go SDK

```go
_, err := client.GetTransaction(ctx, "hash")
if err != nil {
    if strings.Contains(err.Error(), "global timeout exceeded") {
        // Handle global timeout
        log.Printf("Operation timed out after %v", client.Config.TotalTimeout)
    } else {
        // Handle other errors
        log.Printf("Other error: %v", err)
    }
}
```

### TypeScript SDK

```typescript
try {
    const result = await client.request('/transaction/hash');
} catch (error) {
    if (error.message.includes('Global timeout exceeded')) {
        // Handle global timeout
        console.log(`Operation timed out after ${config.totalTimeout}ms`);
    } else {
        // Handle other errors
        console.log(`Other error: ${error.message}`);
    }
}
```

## Best Practices

1. **Set Reasonable Timeouts**: Consider your application's latency requirements
2. **Monitor Metrics**: Track timeout occurrences to tune the timeout value
3. **Fallback Strategy**: Have a plan for when all nodes timeout
4. **Individual vs Global**: Set individual request timeouts shorter than global timeout
5. **Testing**: Test timeout behavior with slow/unresponsive test servers

## Migration

### Existing Code

Existing code continues to work without changes. The default 60-second global timeout provides reasonable protection against excessively long failover operations.

### Upgrading

To take advantage of custom global timeouts:

1. **Go**: Use `WithTotalTimeout()` builder option or set `TotalTimeout` in `NetworkConfig`
2. **TypeScript**: Pass `totalTimeout` to `RPCConfigParser.loadConfig()`

No breaking changes are introduced by this feature.