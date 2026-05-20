// Niyantra Dashboard — Global State
// Shared state variables used across all modules.

import type { ServerConfig, QuotaSortState, StatusResponse, GroupColorMap, GroupNameMap, PresetEntry } from '../types/api';

export const GROUP_ORDER: string[] = ['claude_gpt', 'gemini_unified', 'unknown'];
export const GROUP_LABELS: string[] = ['Claude + GPT', 'Gemini Pool', 'Other'];
export const GRID_COLUMNS: string[] = ['claude_gpt', 'gemini_unified'];
export const GRID_LABELS: string[] = ['Claude + GPT', 'Gemini Pool'];
export const GROUP_COLORS: GroupColorMap = { claude_gpt: '#D97757', gemini_unified: '#3B82F6', unknown: '#64748B' };
export const GROUP_NAMES: GroupNameMap = { claude_gpt: 'Claude + GPT', gemini_unified: 'Gemini Pool', unknown: 'Other' };

// Track which accounts are expanded (survives re-renders)
export const expandedAccounts: Set<number> = new Set();

// Track which provider sections are collapsed (survives re-renders)
const savedCollapsed = typeof localStorage !== 'undefined' ? localStorage.getItem('niyantra_collapsed_providers') : null;
export const collapsedProviders: Set<string> = new Set(savedCollapsed ? JSON.parse(savedCollapsed) : []);

// Track collapsed providers in the Subscriptions tab (survives re-renders)
const savedSubCollapsed = typeof localStorage !== 'undefined' ? localStorage.getItem('niyantra_collapsed_subs_providers') : null;
export const collapsedSubProviders: Set<string> = new Set(savedSubCollapsed ? JSON.parse(savedSubCollapsed) : []);

// Platform presets (loaded from API)
export let presetsData: PresetEntry[] = [];
export function setPresetsData(data: PresetEntry[]): void { presetsData = data; }

// F4: Active tag filter for Quotas tab (null = show all)
export let activeTagFilter: string | null = null;
export function setActiveTagFilter(val: string | null): void { activeTagFilter = val; }

// Usage intelligence cache (populated by fetchUsage)
export let usageDataCache: Record<string, unknown> | null = null;
export function setUsageDataCache(data: Record<string, unknown> | null): void { usageDataCache = data; }

// Quota sort state
export let quotaSortState: QuotaSortState = { column: 'account', direction: 'asc' };
export let quotaSortStates: Record<string, QuotaSortState> = {
  antigravity: { column: 'account', direction: 'asc' },
  codex: { column: 'account', direction: 'asc' },
  cursor: { column: 'account', direction: 'asc' },
  copilot: { column: 'account', direction: 'asc' }
};

// Latest quota data cache
export let latestQuotaData: StatusResponse | null = null;
export function setLatestQuotaData(data: StatusResponse | null): void { latestQuotaData = data; }

// Server config cache (loaded from /api/config)
export let serverConfig: ServerConfig = {};
export function setServerConfig(key: string, value: string): void { serverConfig[key] = value; }

// Snap-in-progress flag
export let snapInProgress: boolean = false;
export function setSnapInProgress(val: boolean): void { snapInProgress = val; }
