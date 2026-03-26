// Copyright (c) 2026 dotandev
// SPDX-License-Identifier: MIT OR Apache-2.0

import { FallbackRPCClient } from './fallback-client';
import { RPCConfig } from '../config/rpc-config';

// Mock axios for testing
jest.mock('axios');
import axios from 'axios';
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('FallbackRPCClient Global Timeout', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        mockedAxios.create.mockReturnValue(mockedAxios);
    });

    it('should timeout before trying all nodes when global timeout is exceeded', async () => {
        const config: RPCConfig = {
            urls: ['http://slow1.com', 'http://slow2.com', 'http://slow3.com'],
            timeout: 5000, // Individual request timeout
            totalTimeout: 3000, // Global timeout (3 seconds)
            retries: 3,
            retryDelay: 1000,
            circuitBreakerThreshold: 5,
            circuitBreakerTimeout: 60000,
            maxRedirects: 5,
        };

        const client = new FallbackRPCClient(config);

        // Mock slow responses (2 seconds each)
        mockedAxios.mockImplementation(() => 
            new Promise((resolve) => {
                setTimeout(() => {
                    resolve({
                        status: 500,
                        statusText: 'Internal Server Error',
                        data: { error: 'Server error' },
                        headers: {},
                        config: {},
                    });
                }, 2000);
            })
        );

        const startTime = Date.now();
        
        await expect(client.request('/test')).rejects.toThrow('Global timeout exceeded');
        
        const elapsed = Date.now() - startTime;
        
        // Should timeout around 3 seconds, not 6+ seconds (2s per server * 3 servers)
        expect(elapsed).toBeLessThan(4000);
        expect(elapsed).toBeGreaterThan(2500); // Should have tried at least one server
    });

    it('should succeed within global timeout', async () => {
        const config: RPCConfig = {
            urls: ['http://fail.com', 'http://success.com'],
            timeout: 5000,
            totalTimeout: 10000, // 10 second global timeout
            retries: 3,
            retryDelay: 1000,
            circuitBreakerThreshold: 5,
            circuitBreakerTimeout: 60000,
            maxRedirects: 5,
        };

        const client = new FallbackRPCClient(config);

        let callCount = 0;
        mockedAxios.mockImplementation(() => {
            callCount++;
            if (callCount === 1) {
                // First call fails
                return Promise.reject(new Error('Network error'));
            } else {
                // Second call succeeds
                return Promise.resolve({
                    status: 200,
                    statusText: 'OK',
                    data: { result: 'success' },
                    headers: {},
                    config: {},
                });
            }
        });

        const startTime = Date.now();
        
        const result = await client.request('/test');
        
        const elapsed = Date.now() - startTime;
        
        expect(result).toEqual({ result: 'success' });
        expect(elapsed).toBeLessThan(2000); // Should complete quickly
    });

    it('should not apply global timeout when set to 0', async () => {
        const config: RPCConfig = {
            urls: ['http://slow.com'],
            timeout: 1000,
            totalTimeout: 0, // Disabled
            retries: 3,
            retryDelay: 1000,
            circuitBreakerThreshold: 5,
            circuitBreakerTimeout: 60000,
            maxRedirects: 5,
        };

        const client = new FallbackRPCClient(config);

        // Mock slow response
        mockedAxios.mockImplementation(() => 
            new Promise((resolve) => {
                setTimeout(() => {
                    resolve({
                        status: 500,
                        statusText: 'Internal Server Error',
                        data: { error: 'Server error' },
                        headers: {},
                        config: {},
                    });
                }, 500);
            })
        );

        await expect(client.request('/test')).rejects.toThrow('All RPC endpoints failed');
        // Should not throw global timeout error
    });

    it('should not apply global timeout when not configured', async () => {
        const config: RPCConfig = {
            urls: ['http://slow.com'],
            timeout: 1000,
            // totalTimeout not set
            retries: 3,
            retryDelay: 1000,
            circuitBreakerThreshold: 5,
            circuitBreakerTimeout: 60000,
            maxRedirects: 5,
        };

        const client = new FallbackRPCClient(config);

        // Mock slow response
        mockedAxios.mockImplementation(() => 
            new Promise((resolve) => {
                setTimeout(() => {
                    resolve({
                        status: 500,
                        statusText: 'Internal Server Error',
                        data: { error: 'Server error' },
                        headers: {},
                        config: {},
                    });
                }, 500);
            })
        );

        await expect(client.request('/test')).rejects.toThrow('All RPC endpoints failed');
        // Should not throw global timeout error
    });

    it('should check timeout between endpoint attempts', async () => {
        const config: RPCConfig = {
            urls: ['http://fast1.com', 'http://fast2.com', 'http://fast3.com'],
            timeout: 5000,
            totalTimeout: 1500, // Very short global timeout
            retries: 3,
            retryDelay: 1000,
            circuitBreakerThreshold: 5,
            circuitBreakerTimeout: 60000,
            maxRedirects: 5,
        };

        const client = new FallbackRPCClient(config);

        let callCount = 0;
        mockedAxios.mockImplementation(() => {
            callCount++;
            // Each call takes 800ms and fails
            return new Promise((resolve) => {
                setTimeout(() => {
                    resolve({
                        status: 500,
                        statusText: 'Internal Server Error',
                        data: { error: 'Server error' },
                        headers: {},
                        config: {},
                    });
                }, 800);
            });
        });

        const startTime = Date.now();
        
        await expect(client.request('/test')).rejects.toThrow('Global timeout exceeded');
        
        const elapsed = Date.now() - startTime;
        
        // Should timeout after first attempt (800ms) but before second attempt would complete
        expect(elapsed).toBeLessThan(2000);
        expect(callCount).toBeLessThanOrEqual(2); // Should not try all 3 endpoints
    });

    it('should use default global timeout from config parser', () => {
        const config: RPCConfig = {
            urls: ['http://test.com'],
            timeout: 30000,
            retries: 3,
            retryDelay: 1000,
            circuitBreakerThreshold: 5,
            circuitBreakerTimeout: 60000,
            maxRedirects: 5,
            totalTimeout: 60000, // Default from config parser
        };

        const client = new FallbackRPCClient(config);
        
        expect(client['config'].totalTimeout).toBe(60000);
    });
});