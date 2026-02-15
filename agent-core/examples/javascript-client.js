#!/usr/bin/env node
/**
 * Simple JavaScript client for agent-core HTTP API.
 */

const http = require('http');

const BASE_URL = 'http://localhost:8080';

function makeRequest(path, method, data) {
    return new Promise((resolve, reject) => {
        const postData = data ? JSON.stringify(data) : null;
        const options = {
            hostname: 'localhost',
            port: 8080,
            path: path,
            method: method,
            headers: { 'Content-Type': 'application/json' }
        };

        if (postData) {
            options.headers['Content-Length'] = Buffer.byteLength(postData);
        }

        const req = http.request(options, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                try {
                    resolve(JSON.parse(data));
                } catch (e) {
                    resolve(data);
                }
            });
        });

        req.on('error', reject);
        if (postData) req.write(postData);
        req.end();
    });
}

function streamRequest(path, data) {
    return new Promise((resolve, reject) => {
        const postData = JSON.stringify(data);
        const options = {
            hostname: 'localhost',
            port: 8080,
            path: path,
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Content-Length': Buffer.byteLength(postData)
            }
        };

        const req = http.request(options, (res) => {
            res.on('data', (chunk) => {
                const lines = chunk.toString().split('\n');
                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        try {
                            const event = JSON.parse(line.substring(6));
                            if (event.type === 'text_delta') {
                                process.stdout.write(event.payload?.Text || '');
                            } else if (event.type === 'tool_call') {
                                console.log(`\n  [calling ${event.payload?.ToolName}]`);
                            } else if (event.type === 'tool_result') {
                                console.log(`  [result: ${JSON.stringify(event.payload?.Result)}]`);
                            } else if (event.type === 'done') {
                                resolve();
                            }
                        } catch (e) {}
                    }
                }
            });
            res.on('end', () => resolve());
        });

        req.on('error', reject);
        req.write(postData);
        req.end();
    });
}

async function main() {
    console.log('JavaScript Client: Calculator Example');
    console.log('='.repeat(40));

    // Check health
    try {
        const health = await makeRequest('/health', 'GET');
        console.log(`Server: ${health.status}`);
    } catch (err) {
        console.log(`Server not running: ${err.message}`);
        console.log('Start with: go run cmd/http-server/main.go');
        return;
    }

    // Single calculation
    console.log('\nUser: What is 15 * 3?');
    process.stdout.write('Agent: ');

    await streamRequest('/agent/prompt', {
        message: 'What is 15 * 3?',
        tools: ['calculator'],
        stream: true
    });

    console.log('\n\nDone!');
}

main().catch(console.error);
