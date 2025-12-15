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

async function tryEndpoint(endpoint, auth) {
  const headers = {
    Accept: 'application/json, text/event-stream',
    'Content-Type': 'application/json',
    'mcp-protocol-version': '2025-03-26',
  };
  if (auth) headers.Authorization = auth;

  const body = {
    jsonrpc: '2.0',
    id: 1,
    method: 'initialize',
    params: {
      protocolVersion: '2025-03-26',
      capabilities: {},
      clientInfo: { name: 'pagespy-mcp-probe', version: '0.0.0' },
    },
  };

  try {
    const res = await fetch(endpoint, { method: 'POST', headers, body: JSON.stringify(body) });
    const text = await res.text();
    return { ok: res.ok, status: res.status, text };
  } catch (e) {
    return { ok: false, status: 0, text: String(e && e.message ? e.message : e) };
  }
}

async function main() {
  const baseUrl = normalizeBaseUrl(getArg('--baseUrl', 'http://127.0.0.1:6752'));
  const auth = getArg('--auth', '');

  const candidates = [
    `${baseUrl}/mcp`,
    `${baseUrl}/mcp/`,
    `${baseUrl}/api/v1/mcp`,
    `${baseUrl}/api/v1/mcp/`,
  ];

  const results = [];
  for (const c of candidates) {
    // eslint-disable-next-line no-await-in-loop
    const r = await tryEndpoint(c, auth);
    results.push({ endpoint: c, ...r });
    if (r.ok) break;
  }

  console.log(JSON.stringify({ baseUrl, results }, null, 2));
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});

