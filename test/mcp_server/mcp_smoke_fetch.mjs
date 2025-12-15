import process from 'process';

function getArg(name, def) {
  const idx = process.argv.indexOf(name);
  if (idx >= 0 && idx + 1 < process.argv.length) return process.argv[idx + 1];
  return def;
}

function normalizeBaseUrl(u) {
  const s = String(u || '').trim();
  if (!s) return '';
  return s.endsWith('/') ? s.slice(0, -1) : s;
}

async function postJson(url, body, headers) {
  const res = await fetch(url, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  const text = await res.text();
  return { status: res.status, text };
}

async function main() {
  const baseUrl = normalizeBaseUrl(getArg('--baseUrl', 'http://127.0.0.1:6752'));
  const endpoint = normalizeBaseUrl(getArg('--endpoint', `${baseUrl}/mcp`));
  const addressArg = String(getArg('--address', '') || '').trim();
  const secretArg = String(getArg('--secret', '') || '').trim();

  const auth = getArg('--auth', '');
  const headers = {
    Accept: 'application/json, text/event-stream',
    'Content-Type': 'application/json',
    'mcp-protocol-version': '2025-03-26',
  };
  if (auth) headers.Authorization = auth;

  const initReq = {
    jsonrpc: '2.0',
    id: 1,
    method: 'initialize',
    params: {
      protocolVersion: '2025-03-26',
      capabilities: {},
      clientInfo: { name: 'pagespy-mcp-smoke', version: '0.0.0' },
    },
  };

  const toolsReq = { jsonrpc: '2.0', id: 2, method: 'tools/list', params: {} };
  const resourcesReq = { jsonrpc: '2.0', id: 3, method: 'resources/list', params: {} };

  console.log(JSON.stringify({ endpoint }, null, 2));

  const initRes = await postJson(endpoint, initReq, headers);
  console.log(JSON.stringify({ step: 'initialize', status: initRes.status }, null, 2));
  console.log(initRes.text);

  const toolsRes = await postJson(endpoint, toolsReq, headers);
  console.log(JSON.stringify({ step: 'tools/list', status: toolsRes.status }, null, 2));
  console.log(toolsRes.text);

  const resourcesRes = await postJson(endpoint, resourcesReq, headers);
  console.log(JSON.stringify({ step: 'resources/list', status: resourcesRes.status }, null, 2));
  console.log(resourcesRes.text);

  const call = async (id, name, args) => {
    const req = { jsonrpc: '2.0', id, method: 'tools/call', params: { name, arguments: args } };
    const r = await postJson(endpoint, req, headers);
    console.log(JSON.stringify({ step: `tools/call:${name}`, status: r.status }, null, 2));
    console.log(r.text);
    return r;
  };

  const listRoomsRes = await call(10, 'list_rooms', {});
  let firstAddress = '';
  try {
    const parsed = JSON.parse(listRoomsRes.text);
    const contentText = parsed?.result?.content?.[0]?.text;
    const rooms = JSON.parse(contentText || '[]');
    firstAddress = rooms?.[0]?.address || '';
  } catch {
    // ignore
  }

  const address = addressArg || firstAddress;
  if (address) {
    await call(11, 'read_room_debug_log', {
      address,
      secret: secretArg || undefined,
      timeoutMs: 800,
      limit: 20,
      format: 'json',
    });
  } else {
    console.log('No room address found. You can pass one via --address <roomAddress>.');
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
