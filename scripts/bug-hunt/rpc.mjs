import { randomUUID } from 'node:crypto';

const origin = process.env.BUCHFINK_BUGHUNT_URL || 'http://127.0.0.1:9250';
if (!/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(origin)) {
  throw new Error('Bug hunting requires a local test server.');
}

export async function call(method, ...args) {
  const response = await fetch(`${origin}/wails/runtime`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'x-wails-client-id': 'buchfink-bughunt' },
    body: JSON.stringify({ object: 0, method: 0, args: {
      'call-id': randomUUID(),
      methodName: `github.com/buchfink/buchfink/internal/wailsbridge.BuchfinkBridge.${method}`,
      args,
    } }),
  });
  const body = await response.text();
  let result;
  try { result = JSON.parse(body); } catch { result = body; }
  if (!response.ok) throw new Error(`${method}: ${typeof result === 'object' ? JSON.stringify(result) : result}`);
  return result;
}
