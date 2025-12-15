import path from 'path';
import { fileURLToPath, pathToFileURL } from 'url';
import { existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    const a = argv[i];
    if (!a.startsWith('--')) continue;
    const k = a.slice(2);
    const v = argv[i + 1];
    if (!v || v.startsWith('--')) {
      out[k] = true;
      continue;
    }
    out[k] = v;
    i += 1;
  }
  return out;
}

const args = parseArgs(process.argv.slice(2));
const endpoint = String(args.endpoint || args.url || 'http://127.0.0.1:6752/mcp');
const authHeader = args.auth ? String(args.auth) : '';

// Use the same SDK version/path as mcphub stack trace by default.
const DEFAULT_SDK_ESM_DIR =
  'D:/CodeRelated/AI/MCP/mcphub-github/mcphub/node_modules/.pnpm/@modelcontextprotocol+sdk@1.23.0_zod@3.25.76/node_modules/@modelcontextprotocol/sdk/dist/esm';
const sdkEsmDir = String(process.env.MCP_SDK_ESM_DIR || DEFAULT_SDK_ESM_DIR);

const clientIndexPath = path.join(sdkEsmDir, 'client', 'index.js');
const streamableHttpPath = path.join(sdkEsmDir, 'client', 'streamableHttp.js');

if (!existsSync(clientIndexPath) || !existsSync(streamableHttpPath)) {
  console.error('SDK ESM files not found.');
  console.error('Tried:');
  console.error(`- ${clientIndexPath}`);
  console.error(`- ${streamableHttpPath}`);
  console.error('You can override via env: MCP_SDK_ESM_DIR=<.../sdk/dist/esm>');
  process.exit(1);
}

const { Client } = await import(pathToFileURL(clientIndexPath).href);
const { StreamableHTTPClientTransport } = await import(pathToFileURL(streamableHttpPath).href);

function headerPreview(headers) {
  try {
    const o = {};
    if (!headers) return o;
    if (headers instanceof Headers) {
      for (const [k, v] of headers.entries()) o[k] = v;
      return o;
    }
    if (Array.isArray(headers)) {
      for (const [k, v] of headers) o[k] = v;
      return o;
    }
    return { ...headers };
  } catch {
    return {};
  }
}

async function loggingFetch(url, init) {
  const method = init?.method || 'GET';
  const headers = headerPreview(init?.headers);
  console.log('\n=== fetch ===');
  console.log('method:', method);
  console.log('url:', String(url));
  console.log('headers:', headers);

  const res = await fetch(url, init);
  const ct = res.headers.get('content-type') || '';
  console.log('status:', res.status);
  console.log('content-type:', ct);
  const sid = res.headers.get('mcp-session-id');
  if (sid) console.log('mcp-session-id:', sid);

  // Avoid consuming SSE streams.
  if (!ct.includes('text/event-stream')) {
    const text = await res
      .clone()
      .text()
      .catch(() => '');
    console.log('body(0..500):', String(text).slice(0, 500));
  }
  return res;
}

const options = {
  fetch: loggingFetch,
};

if (authHeader) {
  options.requestInit = { headers: { Authorization: authHeader } };
}

const transport = new StreamableHTTPClientTransport(new URL(endpoint), options);
const client = new Client({ name: 'page-spy-api-mcp-test', version: '0.0.0' }, { capabilities: {} });

try {
  console.log('Connecting with StreamableHTTPClientTransport...');
  console.log('endpoint:', endpoint);
  await client.connect(transport);

  console.log('\nConnected. Calling listTools()...');
  const tools = await client.listTools();
  console.log(JSON.stringify(tools, null, 2));

  console.log('\nOK');
  await client.close();
  await transport.close();
} catch (e) {
  console.error('\nFAILED');
  console.error(e);
  try {
    await transport.close();
  } catch {
    // ignore
  }
  process.exit(1);
}
