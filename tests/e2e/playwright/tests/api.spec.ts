import { test, expect } from '@playwright/test';

const KONG_URL = process.env.KONG_URL || 'http://localhost:8000';
const LLM_API_URL = process.env.LLM_API_URL || 'http://localhost:8080';

test.describe('Authentication API', () => {
  test('should issue guest token', async ({ request }) => {
    const response = await request.post(`${KONG_URL}/auth/guest-login`, {
      headers: { 'Content-Type': 'application/json' },
      data: {},
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data).toHaveProperty('access_token');
    expect(data).toHaveProperty('expires_in');
    expect(typeof data.access_token).toBe('string');
  });

  test('should reject empty body', async ({ request }) => {
    const response = await request.post(`${KONG_URL}/auth/guest-login`, {
      headers: { 'Content-Type': 'application/json' },
      data: 'invalid',
    });

    expect(response.status()).toBeGreaterThanOrEqual(400);
  });
});

test.describe('Models API', () => {
  let token: string;

  test.beforeEach(async ({ request }) => {
    const response = await request.post(`${KONG_URL}/auth/guest-login`, {
      headers: { 'Content-Type': 'application/json' },
      data: {},
    });
    const data = await response.json();
    token = data.access_token;
  });

  test('should list available models', async ({ request }) => {
    const response = await request.get(`${KONG_URL}/v1/models`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data).toHaveProperty('data');
    expect(Array.isArray(data.data)).toBeTruthy();
  });

  test('should require authentication', async ({ request }) => {
    const response = await request.get(`${KONG_URL}/v1/models`);
    expect(response.status()).toBe(401);
  });
});

test.describe('Chat Completions API', () => {
  let token: string;
  let modelId: string;

  test.beforeEach(async ({ request }) => {
    // Get token
    const tokenRes = await request.post(`${KONG_URL}/auth/guest-login`, {
      headers: { 'Content-Type': 'application/json' },
      data: {},
    });
    const tokenData = await tokenRes.json();
    token = tokenData.access_token;

    // Get model
    const modelsRes = await request.get(`${KONG_URL}/v1/models`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    const modelsData = await modelsRes.json();
    modelId = modelsData.data?.[0]?.id || 'jan-v1-4b';
  });

  test('should create chat completion', async ({ request }) => {
    const response = await request.post(`${KONG_URL}/v1/chat/completions`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        model: modelId,
        messages: [{ role: 'user', content: 'Say hello' }],
        stream: false,
        max_tokens: 20,
      },
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data).toHaveProperty('id');
    expect(data).toHaveProperty('choices');
    expect(Array.isArray(data.choices)).toBeTruthy();
    expect(data.choices[0]).toHaveProperty('message');
  });

  test('should stream chat completion', async ({ request }) => {
    const response = await request.post(`${KONG_URL}/v1/chat/completions`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        model: modelId,
        messages: [{ role: 'user', content: 'Count from 1 to 3' }],
        stream: true,
      },
    });

    expect(response.ok()).toBeTruthy();
    // Streaming responses return text/event-stream content-type
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('text/event-stream');
  });

  test('should reject invalid model', async ({ request }) => {
    const response = await request.post(`${KONG_URL}/v1/chat/completions`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        model: 'non-existent-model',
        messages: [{ role: 'user', content: 'Hello' }],
      },
    });

    expect(response.status()).toBeGreaterThanOrEqual(400);
  });
});

test.describe('Conversations API', () => {
  let token: string;

  test.beforeEach(async ({ request }) => {
    const response = await request.post(`${KONG_URL}/auth/guest-login`, {
      headers: { 'Content-Type': 'application/json' },
      data: {},
    });
    const data = await response.json();
    token = data.access_token;
  });

  test('should list conversations', async ({ request }) => {
    const response = await request.get(`${KONG_URL}/v1/conversations`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data).toHaveProperty('data');
    expect(Array.isArray(data.data)).toBeTruthy();
  });

  test('should create conversation', async ({ request }) => {
    const response = await request.post(`${KONG_URL}/v1/conversations`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: { title: 'Test Conversation' },
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data).toHaveProperty('id');
    expect(data).toHaveProperty('title', 'Test Conversation');
  });

  test('should add message to conversation', async ({ request }) => {
    // Create conversation first
    const createRes = await request.post(`${KONG_URL}/v1/conversations`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: { title: 'Test Conv' },
    });
    const conv = await createRes.json();

    // Add message
    const response = await request.post(
      `${KONG_URL}/v1/conversations/${conv.id}/items`,
      {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        data: {
          items: [{
            type: 'message',
            role: 'user',
            content: [{ type: 'input_text', text: 'Hello world' }],
          }],
        },
      }
    );

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data).toHaveProperty('data');
    expect(Array.isArray(data.data)).toBeTruthy();
  });
});

test.describe('Media API', () => {
  let token: string;

  test.beforeEach(async ({ request }) => {
    const response = await request.post(`${KONG_URL}/auth/guest-login`, {
      headers: { 'Content-Type': 'application/json' },
      data: {},
    });
    const data = await response.json();
    token = data.access_token;
  });

  test('should list media files', async ({ request }) => {
    const response = await request.get(`${KONG_URL}/media/v1/files`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data).toHaveProperty('data');
    expect(Array.isArray(data.data)).toBeTruthy();
  });

  test('should require authentication', async ({ request }) => {
    const response = await request.get(`${KONG_URL}/media/v1/files`);
    expect(response.status()).toBe(401);
  });
});

test.describe('Health Checks', () => {
  test('Kong should be healthy', async ({ request }) => {
    const response = await request.get(`${KONG_URL}/health`, { ignoreHTTPSErrors: true });
    // Kong returns 200 or 503 depending on upstream health
    expect([200, 503]).toContain(response.status());
  });

  test('LLM API should be healthy', async ({ request }) => {
    const response = await request.get(`${LLM_API_URL}/v1/healthz`);
    expect(response.status()).toBe(200);
  });
});