// Niyantra Dashboard — API Functions
// Fetch wrappers for all backend endpoints.

import { setUsageDataCache } from './state';
import type { StatusResponse, Subscription, OverviewResponse, PresetsResponse } from '../types/api';

const TOKEN_STORAGE_KEY = 'niyantra.dashboardToken';
const DEFAULT_API_TIMEOUT_MS = 30000;
let originalFetch: typeof fetch | null = null;

export function bootstrapDashboardToken(): string | null {
  var params = new URLSearchParams(window.location.search);
  var token = params.get('token') || params.get('auth_token');
  if (token) {
    sessionStorage.setItem(TOKEN_STORAGE_KEY, token);
    params.delete('token');
    params.delete('auth_token');
    var next = window.location.pathname + (params.toString() ? '?' + params.toString() : '') + window.location.hash;
    window.history.replaceState({}, document.title, next || '/');
    return token;
  }
  return sessionStorage.getItem(TOKEN_STORAGE_KEY);
}

export function dashboardToken(): string | null {
  return sessionStorage.getItem(TOKEN_STORAGE_KEY);
}

export function installAuthenticatedFetch(): void {
  if (originalFetch) return;
  bootstrapDashboardToken();
  originalFetch = window.fetch.bind(window);
  window.fetch = function(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    return apiFetch(input, init);
  };
}

export function apiFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  var fetchImpl = originalFetch || window.fetch.bind(window);
  var nextInit: RequestInit = Object.assign({}, init || {});
  var headers = new Headers(nextInit.headers || {});
  var url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url;
  var protectedLocalURL = isProtectedLocalURL(url);
  var timeoutHandle: number | undefined;
  if (protectedLocalURL && !nextInit.signal && typeof AbortController !== 'undefined') {
    var controller = new AbortController();
    nextInit.signal = controller.signal;
    timeoutHandle = window.setTimeout(function() {
      controller.abort();
    }, DEFAULT_API_TIMEOUT_MS);
  }
  if (protectedLocalURL && !headers.has('Authorization')) {
    var token = dashboardToken();
    if (token) headers.set('Authorization', 'Bearer ' + token);
  }
  nextInit.headers = headers;
  return fetchImpl(input, nextInit).then(function(res) {
    if (res.status === 401) {
      showAuthFailure();
    }
    return res;
  }).finally(function() {
    if (timeoutHandle !== undefined) window.clearTimeout(timeoutHandle);
  });
}

function isProtectedLocalURL(rawURL: string): boolean {
  var url = new URL(rawURL, window.location.href);
  if (url.origin !== window.location.origin) return false;
  return url.pathname.indexOf('/api/') === 0 || url.pathname === '/mcp';
}

function showAuthFailure(): void {
  var existing = document.getElementById('auth-error-banner');
  if (existing) return;
  var banner = document.createElement('div');
  banner.id = 'auth-error-banner';
  banner.className = 'auth-error-banner';
  banner.textContent = 'Dashboard API token is missing or invalid. Reopen the tokenized URL printed by niyantra serve.';
  document.body.prepend(banner);
}

// ── Quotas (auto-tracked) ──

export function fetchStatus(): Promise<StatusResponse> {
  return apiFetch('/api/status').then(function(res) {
    if (!res.ok) throw new Error('Failed to fetch status');
    return res.json();
  });
}

export function triggerSnap(): Promise<any> {
  return apiFetch('/api/snap', { method: 'POST' }).then(function(res) {
    return res.json().then(function(data: any) {
      if (!res.ok) throw new Error(data.error || 'Snap failed');
      return data;
    });
  });
}

// ── Subscriptions ──

export function fetchSubscriptions(status?: string, category?: string): Promise<any> {
  var params = new URLSearchParams();
  if (status) params.set('status', status);
  if (category) params.set('category', category);
  var url = '/api/subscriptions' + (params.toString() ? '?' + params : '');
  return apiFetch(url).then(function(res) { return res.json(); });
}

export function createSubscription(sub: Partial<Subscription>): Promise<Subscription> {
  return apiFetch('/api/subscriptions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(sub),
  }).then(function(res) {
    return res.json().then(function(data: any) {
      if (!res.ok) throw new Error(data.error || 'Create failed');
      return data;
    });
  });
}

export function updateSubscription(id: number, sub: Partial<Subscription>): Promise<Subscription> {
  return apiFetch('/api/subscriptions/' + id, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(sub),
  }).then(function(res) {
    return res.json().then(function(data: any) {
      if (!res.ok) throw new Error(data.error || 'Update failed');
      return data;
    });
  });
}

export function deleteSubscription(id: number): Promise<any> {
  return apiFetch('/api/subscriptions/' + id, { method: 'DELETE' }).then(function(res) {
    return res.json().then(function(data: any) {
      if (!res.ok) throw new Error(data.error || 'Delete failed');
      return data;
    });
  });
}

// ── Overview & Presets ──

export function fetchOverview(): Promise<OverviewResponse> {
  return apiFetch('/api/overview').then(function(res) { return res.json(); });
}

export function fetchPresets(): Promise<PresetsResponse> {
  return apiFetch('/api/presets').then(function(res) { return res.json(); });
}

// ── Usage Intelligence ──

export function fetchUsage(accountId?: number): Promise<any> {
  var url = '/api/usage';
  if (accountId) url += '?account=' + accountId;
  return apiFetch(url).then(function(res) { return res.json(); }).then(function(data: any) {
    setUsageDataCache(data);
    return data;
  });
}

export function downloadBackup(): Promise<void> {
  return apiFetch('/api/backup/create', { method: 'POST' }).then(function(res) {
    if (!res.ok) {
      return res.json().catch(function() { return {}; }).then(function(data: any) {
        throw new Error(data.error || 'Backup failed');
      });
    }
    return res.blob().then(function(blob) {
      var filename = filenameFromDisposition(res.headers.get('Content-Disposition')) || 'niyantra-backup.db';
      var url = URL.createObjectURL(blob);
      var a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    });
  });
}

export function downloadAPIFile(url: string, fallbackName: string): Promise<void> {
  return apiFetch(url).then(function(res) {
    if (!res.ok) {
      return res.json().catch(function() { return {}; }).then(function(data: any) {
        throw new Error(data.error || 'Download failed');
      });
    }
    return res.blob().then(function(blob) {
      var filename = filenameFromDisposition(res.headers.get('Content-Disposition')) || fallbackName;
      var objectURL = URL.createObjectURL(blob);
      var a = document.createElement('a');
      a.href = objectURL;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(objectURL);
    });
  });
}

function filenameFromDisposition(header: string | null): string | null {
  if (!header) return null;
  var match = /filename="([^"]+)"/.exec(header);
  return match ? match[1] : null;
}

export function claimOverageBonus(accountId: number): Promise<any> {
  return apiFetch('/api/accounts/' + accountId + '/claim-bonus', { method: 'POST' }).then(function(res) {
    return res.json().then(function(data: any) {
      if (!res.ok) throw new Error(data.error || 'Claim failed');
      return data;
    });
  });
}

