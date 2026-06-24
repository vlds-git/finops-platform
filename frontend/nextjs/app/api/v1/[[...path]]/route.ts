import { NextRequest, NextResponse } from 'next/server';

const PRIMARY_URL = (process.env.API_GATEWAY_URL || 'http://finops-api-gateway:8080').replace(/\/?$/, '');
const FALLBACK_URLS = ['http://api-gateway:8080', 'http://host.docker.internal:8080', 'http://10.140.12.32:8080'];

async function proxy(request: NextRequest, { params }: { params: { path?: string[] } }) {
  const pathSegments = Array.isArray(params.path) ? params.path : [];
  const backendPath = pathSegments.join('/') || '';
  const search = request.nextUrl.search;
  const buildUrl = (base: string) => `${base}/api/v1/${backendPath}${search}`;
  let lastErr: unknown;

  for (const base of [PRIMARY_URL, ...FALLBACK_URLS]) {
    const targetUrl = buildUrl(base);
    console.log(`[API Proxy] ${request.method} ${targetUrl}`);

    const headers: Record<string, string> = {};
    const contentType = request.headers.get('content-type');
    if (contentType) headers['Content-Type'] = contentType;
    const authorization = request.headers.get('authorization');
    if (authorization) headers['Authorization'] = authorization;
    const accept = request.headers.get('accept');
    if (accept) headers['Accept'] = accept;

    let body: string | undefined;
    if (request.method !== 'GET' && request.method !== 'HEAD' && request.method !== 'OPTIONS') {
      try {
        body = await request.text();
      } catch (e) {
        console.error('[API Proxy] Failed to read request body:', e);
      }
    }

    try {
      const response = await fetch(targetUrl, {
        method: request.method,
        headers,
        body,
        redirect: 'manual',
      });

      const responseHeaders = new Headers();
      response.headers.forEach((value, key) => {
        const lower = key.toLowerCase();
        if (lower === 'content-encoding' || lower === 'transfer-encoding' || lower === 'connection') return;
        responseHeaders.set(key, value);
      });

      const responseBody = response.status === 204 || request.method === 'HEAD' ? null : await response.arrayBuffer();

      return new NextResponse(responseBody, {
        status: response.status,
        statusText: response.statusText,
        headers: responseHeaders,
      });
    } catch (err) {
      lastErr = err;
      console.error('[API Proxy] Error proxying to', targetUrl, err);
    }
  }

  return NextResponse.json({ error: 'API Gateway unavailable', primary: PRIMARY_URL, lastError: String(lastErr) }, { status: 502 });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const DELETE = proxy;
export const PATCH = proxy;
export const HEAD = proxy;
export const OPTIONS = proxy;
