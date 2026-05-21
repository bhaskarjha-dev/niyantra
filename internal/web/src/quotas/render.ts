// Niyantra Dashboard — Quota Grid Rendering
// Sort, filter, tag strip, and account grid rendering.

import {
  GROUP_ORDER, GROUP_LABELS, GROUP_COLORS, GROUP_NAMES,
  GRID_COLUMNS, GRID_LABELS,
  expandedAccounts, collapsedProviders,
  quotaSortState, quotaSortStates, latestQuotaData, setLatestQuotaData,
  activeTagFilter, setActiveTagFilter,
  usageDataCache,
} from '../core/state';
import { esc, formatSeconds, formatCredits, formatTimeAgo, showToast } from '../core/utils';
import { claimOverageBonus, fetchStatus } from '../core/api';
import type { StatusResponse, AccountReadiness } from '../types/api';
import { renderAccountTags, renderAccountNote, renderCreditRenewal } from './features';
export function getGroupPct(acc: any, groupKey: string): number {
  if (!acc.groups) return -1;
  for (var i = 0; i < acc.groups.length; i++) {
    if (acc.groups[i].groupKey === groupKey) return acc.groups[i].remainingPercent;
  }
  return -1;
}

export function getAICredits(acc: any): number {
  if (acc.aiCredits && acc.aiCredits.length > 0) return acc.aiCredits[0].creditAmount;
  return -1;
}

// Get soonest reset time in seconds across all groups for an account
export function getSoonestResetSec(acc: any): number {
  var groups = acc.groups || [];
  var soonest = Infinity;
  for (var i = 0; i < groups.length; i++) {
    var t = groups[i].timeUntilResetSec;
    if (t !== undefined && t !== null && t < soonest) soonest = t;
  }
  return soonest === Infinity ? -1 : soonest;
}

export function allExhausted(acc: any): boolean {
  var grps = acc.groups || [];
  if (grps.length === 0) return false;
  for (var i = 0; i < grps.length; i++) {
    if (!grps[i].isExhausted && grps[i].remainingPercent > 0) return false;
  }
  return true;
}

// Bug 6: Determine status of Codex/Claude snapshot for filtering
export function getCodexClaudeStatus(snap: any): string {
  var fiveUsed = snap.fiveHourPct || 0;
  var sevenUsed = snap.sevenDayPct || 0;
  var fiveRem = Math.max(0, 100 - fiveUsed);
  var sevenRem = Math.max(0, 100 - sevenUsed);
  if (fiveRem === 0 && sevenRem === 0) return 'empty';
  if (fiveUsed >= 80 || sevenUsed >= 80) return 'low';
  return 'ready';
}

export function sortAccountsArray(accounts: any[]): any[] {
  var state = quotaSortStates.antigravity || quotaSortState;
  var col = state.column;
  var dir = state.direction;
  return accounts.slice().sort(function(a, b) {
    var va, vb;
    switch (col) {
      case 'account': va = a.email; vb = b.email; break;
      case 'claude_gpt':
      case 'gemini_unified':
      case 'unknown':
        va = getGroupPct(a, col); vb = getGroupPct(b, col); break;
      case 'credits':
        va = getAICredits(a); vb = getAICredits(b); break;
      case 'lastsnap':
        va = a.lastSeen ? new Date(a.lastSeen).getTime() : 0;
        vb = b.lastSeen ? new Date(b.lastSeen).getTime() : 0; break;
      case 'status':
        va = a.isReady ? 1 : 0; vb = b.isReady ? 1 : 0; break;
      case 'resetsIn':
        va = getSoonestResetSec(a); vb = getSoonestResetSec(b);
        // Treat -1 (no data) as very large so they sort last
        if (va < 0) va = 999999999;
        if (vb < 0) vb = 999999999;
        // Treat <= 0 (already reset / ready) as 0 so they sort first
        if (va <= 0) va = 0;
        if (vb <= 0) vb = 0;
        break;
      default: va = a.email; vb = b.email; break;
    }
    if (va === vb) return 0;
    var res = va > vb ? 1 : -1;
    return dir === 'asc' ? res : -res;
  });
}

export function sortProviderArray(array: any[], provider: string): any[] {
  var state = quotaSortStates[provider] || quotaSortState;
  var col = state.column;
  var dir = state.direction;
  return array.slice().sort(function(a, b) {
    var va, vb;
    if (provider === 'codex') {
      switch (col) {
        case 'account':
          va = a.email || a.accountId || '';
          vb = b.email || b.accountId || '';
          break;
        case 'plan':
          va = a.planType || '';
          vb = b.planType || '';
          break;
        case 'fiveHour':
          va = a.fiveHourPct || 0;
          vb = b.fiveHourPct || 0;
          break;
        case 'sevenDay':
          va = a.sevenDayPct || 0;
          vb = b.sevenDayPct || 0;
          break;
        case 'credits':
          va = a.creditsBalance || 0;
          vb = b.creditsBalance || 0;
          break;
        case 'lastsnap':
          va = a.capturedAt ? new Date(a.capturedAt).getTime() : 0;
          vb = b.capturedAt ? new Date(b.capturedAt).getTime() : 0;
          break;
        case 'status':
          va = getCodexClaudeStatus(a);
          vb = getCodexClaudeStatus(b);
          break;
        default:
          return 0;
      }
    } else if (provider === 'cursor') {
      switch (col) {
        case 'account':
          va = a.email || '';
          vb = b.email || '';
          break;
        case 'plan':
          va = a.planType || '';
          vb = b.planType || '';
          break;
        case 'premiumUsed':
          va = a.premiumUsed || 0;
          vb = b.premiumUsed || 0;
          break;
        case 'usage':
          va = a.usagePct || 0;
          vb = b.usagePct || 0;
          break;
        case 'lastsnap':
          va = a.capturedAt ? new Date(a.capturedAt).getTime() : 0;
          vb = b.capturedAt ? new Date(b.capturedAt).getTime() : 0;
          break;
        case 'status':
          va = getCursorStatus(a);
          vb = getCursorStatus(b);
          break;
        default:
          return 0;
      }

    } else if (provider === 'copilot') {
      switch (col) {
        case 'account':
          va = a.username || a.email || '';
          vb = b.username || b.email || '';
          break;
        case 'plan':
          va = a.plan || '';
          vb = b.plan || '';
          break;
        case 'premium':
          va = a.premiumPct || 0;
          vb = b.premiumPct || 0;
          break;
        case 'chat':
          va = a.chatPct || 0;
          vb = b.chatPct || 0;
          break;
        case 'lastsnap':
          va = a.capturedAt ? new Date(a.capturedAt).getTime() : 0;
          vb = b.capturedAt ? new Date(b.capturedAt).getTime() : 0;
          break;
        case 'status':
          va = getCopilotStatus(a);
          vb = getCopilotStatus(b);
          break;
        default:
          return 0;
      }
    } else {
      return 0;
    }

    if (va === vb) return 0;
    var res = va > vb ? 1 : -1;
    // Ascending by default, reverse if desc
    return dir === 'asc' ? res : -res;
  });
}

export function filterAccountsArray(accounts: any[]): any[] {
  var searchInput = document.getElementById('quota-search');
  var statusFilter = document.getElementById('quota-filter-status');
  var query = searchInput ? (searchInput as HTMLInputElement).value.toLowerCase() : '';
  var status = statusFilter ? (statusFilter as HTMLSelectElement).value : 'all';

  return accounts.filter(function(acc) {
    var matchesSearch = !query ||
      acc.email.toLowerCase().includes(query) ||
      (acc.planName || '').toLowerCase().includes(query);

    var matchesStatus = true;
    if (status === 'ready') matchesStatus = acc.isReady;
    else if (status === 'low') matchesStatus = !acc.isReady && !allExhausted(acc);
    else if (status === 'empty') matchesStatus = allExhausted(acc);

    // F4: Tag-based filtering
    var matchesTag = true;
    if (activeTagFilter) {
      var accTags = (acc.tags || '').split(',').map(function(t: any) { return t.trim().toLowerCase(); });
      matchesTag = accTags.indexOf(activeTagFilter) >= 0;
    }

    return matchesSearch && matchesStatus && matchesTag;
  });
}

export function updateSortHeaders(): void {
  document.querySelectorAll('.grid-header .sortable').forEach(function(el) {
    el.classList.remove('sort-active');
    var span = el.querySelector('.sort-indicator');
    if (span) span.textContent = '';
    
    var providerSection = el.closest('.provider-section');
    var provider = providerSection ? (providerSection as HTMLElement).dataset.provider : 'antigravity';
    var state = quotaSortStates[provider || 'antigravity'] || quotaSortState;
    
    if ((el as HTMLElement).dataset.sort === state.column) {
      el.classList.add('sort-active');
      if (span) span.textContent = state.direction === 'asc' ? '▾' : '▴';
    }
  });
}

function getHumanReadableBasis(basis: string): string {
  if (!basis) return '';
  switch (basis) {
    case 'contains_post_reset_estimate':
      return 'Estimated models restored following their reset window';
    case 'sprint_reset_high':
      return 'Optimistic 100% estimate (reset occurred <30m ago)';
    case 'sprint_reset_medium':
      return 'Optimistic 100% estimate (reset occurred <6h ago)';
    case 'sprint_reset_low':
      return 'Optimistic 100% estimate (reset occurred <24h ago)';
    case 'sprint_reset_stale':
      return 'Stale reset window (fallback to last observed value)';
    case 'snapshot_too_stale':
      return 'Snapshot is too old to confidently estimate';
    case 'post_reset_estimate':
      return 'Usage reset to 0% following provider reset boundary';
    default:
      return basis.replace(/_/g, ' ');
  }
}

function getHumanReadableConfidence(confidence: string): string {
  if (!confidence) return '';
  switch (confidence) {
    case 'high':
      return 'High (Recent reset or fresh capture)';
    case 'medium':
      return 'Medium (Less than 6h since reset/activity)';
    case 'low':
      return 'Low (Up to 24h since reset/activity)';
    case 'very_low':
      return 'Very Low (Stale data; needs refresh)';
    default:
      return confidence.charAt(0).toUpperCase() + confidence.slice(1);
  }
}

function renderQualityBadge(item: any): string {
  if (!item) return '';
  var label = '';
  // Estimates now use ~ prefix on percentage, so no badge needed
  if (item.unavailableReason) label = 'Unavailable';
  else if (item.isEstimated) {
    // Stale-specific labels still useful as small badges
    if (item.basis === 'snapshot_too_stale') label = 'Stale';
    else if (item.confidence === 'very_low') label = 'Stale';
    // All other estimates: handled by ~ prefix, no badge
    else return '';
  }
  else if (item.confidence && item.confidence !== 'high') label = item.confidence + ' confidence';
  if (!label) return '';
  var titleParts: string[] = [];
  if (item.basis) titleParts.push('Basis: ' + getHumanReadableBasis(item.basis));
  if (item.confidence) titleParts.push('Confidence: ' + getHumanReadableConfidence(item.confidence));
  if (item.unavailableReason) titleParts.push('Unavailable: ' + item.unavailableReason);
  return '<span class="data-quality-badge" title="' + esc(titleParts.join(' | ')) + '">' + esc(label) + '</span>';
}

// ════════════════════════════════════════════
//  F4: TAG-BASED FILTERING
// ════════════════════════════════════════════

export function getUniqueTagsFromData(data: any): Record<string, number> {
  var tagCounts: Record<string, number> = {};
  var accounts = data.accounts || [];
  for (var i = 0; i < accounts.length; i++) {
    var tags = (accounts[i].tags || '').split(',');
    for (var j = 0; j < tags.length; j++) {
      var t = tags[j].trim().toLowerCase();
      if (t) {
        tagCounts[t] = (tagCounts[t] || 0) + 1;
      }
    }
  }
  return tagCounts;
}

export function renderTagFilterStrip(data: any): void {
  var strip = document.getElementById('tag-filter-strip');
  if (!strip) return;

  var tagCounts = getUniqueTagsFromData(data);
  var tagNames = Object.keys(tagCounts).sort();

  // Only show strip if there are tags to filter by
  if (tagNames.length === 0) {
    // Bug fix: if active filter was set to a now-deleted tag, reset to show all
    if (activeTagFilter) {
      setActiveTagFilter(null);
    }
    strip.innerHTML = '';
    return;
  }

  // Bug fix: if the active tag no longer exists in the data, reset to show all
  if (activeTagFilter && tagNames.indexOf(activeTagFilter) < 0) {
    setActiveTagFilter(null);
  }

  var html = '<span class="tag-filter-label">🏷️ Filter:</span>';

  // "All" chip
  var allActive = !activeTagFilter ? ' active' : '';
  var totalAccounts = (data.accounts || []).length;
  html += '<button class="tag-filter-chip' + allActive + '" data-tag-filter="">' +
    'All <span class="tag-filter-count">' + totalAccounts + '</span></button>';

  // Tag chips
  for (var i = 0; i < tagNames.length; i++) {
    var tag = tagNames[i];
    var isActive = activeTagFilter === tag ? ' active' : '';
    html += '<button class="tag-filter-chip' + isActive + '" data-tag-filter="' + esc(tag) + '">' +
      esc(tag) + ' <span class="tag-filter-count">' + tagCounts[tag] + '</span></button>';
  }

  strip.innerHTML = html;
}

export function handleTagFilterClick(e: Event): void {
  var chip = (e.target as HTMLElement).closest('.tag-filter-chip');
  if (!chip) return;

  var tag = chip.getAttribute('data-tag-filter');
  setActiveTagFilter(tag || null);

  // Re-render with filter applied
  if (latestQuotaData) {
    renderTagFilterStrip(latestQuotaData);
    renderAccounts(latestQuotaData);
  }
}

export function renderAccounts(data: any): void {
  setLatestQuotaData(data);
  var grid = document.getElementById('account-grid');
  var countBadge = document.getElementById('account-count');
  var snapCount = document.getElementById('snap-count');
  if (!grid) return;

  // F4: Update tag filter strip on data refresh
  renderTagFilterStrip(data);

  // Q-M2: Visual indication on active filters
  var statusFilterEl = document.getElementById('quota-filter-status');
  var providerFilterEl = document.getElementById('quota-filter-provider');
  if (statusFilterEl) statusFilterEl.classList.toggle('filter-active', (statusFilterEl as HTMLSelectElement).value !== 'all');
  if (providerFilterEl) providerFilterEl.classList.toggle('filter-active', (providerFilterEl as HTMLSelectElement).value !== 'all');

  var acctCount = (data.accounts || []).length;
  var parts = [];
  if (acctCount > 0) parts.push(acctCount + ' Antigravity');
  if (data.codexSnapshots && data.codexSnapshots.length > 0) parts.push(data.codexSnapshots.length + ' Codex');
  if (data.claudeSnapshot) parts.push('1 Claude');
  if (data.cursorSnapshots && data.cursorSnapshots.length > 0) parts.push(data.cursorSnapshots.length + ' Cursor');
  if (data.copilotSnapshots && data.copilotSnapshots.length > 0) parts.push(data.copilotSnapshots.length + ' Copilot');
  if (countBadge) countBadge.textContent = parts.join(' · ') || '0 accounts';
  if (snapCount) snapCount.textContent = data.snapshotCount ? (data.snapshotCount + ' snapshots') : '';

  if (acctCount === 0 && (!data.codexSnapshots || data.codexSnapshots.length === 0) && !data.claudeSnapshot && (!data.cursorSnapshots || data.cursorSnapshots.length === 0) && (!data.copilotSnapshots || data.copilotSnapshots.length === 0)) {
    grid.innerHTML = '<div class="empty-state">' +
      '<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/><path d="M12 2v4M12 18v4M2 12h4M18 12h4"/></svg>' +
      '<p>No accounts tracked yet</p>' +
      '<p class="empty-hint">Click <strong>Snap Now</strong> to capture your first snapshot</p>' +
      '</div>';
    return;
  }

  var providerFilter = document.getElementById('quota-filter-provider');
  var pf = providerFilter ? (providerFilter as HTMLSelectElement).value : 'all';
  var html = '';
  if (acctCount > 0 && (pf === 'all' || pf === 'antigravity')) {
  var filtered = filterAccountsArray(data.accounts);
  var sorted = sortAccountsArray(filtered);
  var agCollapseClass = collapsedProviders.has('section-antigravity') ? ' collapsed' : '';
  var agChevron = collapsedProviders.has('section-antigravity') ? '▸' : '▾';
  var agReadyCount = 0;
  for (var i = 0; i < acctCount; i++) {
    if (data.accounts[i].isReady) agReadyCount++;
  }

  html += '<div class="provider-section" data-provider="antigravity"><div class="provider-header" data-toggle-provider="section-antigravity">' +
    '<div class="provider-header-left"><span class="provider-chevron" id="pchev-section-antigravity">' + agChevron + '</span>' +
    '<span class="provider-name">Antigravity</span>' +
    '<span class="provider-count">' + acctCount + ' account' + (acctCount !== 1 ? 's' : '') + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + agReadyCount + ' ready)</span></span></div></div>' +
    '<div class="provider-body' + agCollapseClass + '" id="section-antigravity">';
  // Dynamic Antigravity grid header
  html += '<div class="grid-header">' +
    '<div class="grid-col-account sortable" data-sort="account">Account <span class="sort-indicator"></span></div>';
  for (var gh = 0; gh < GRID_COLUMNS.length; gh++) {
    html += '<div class="grid-col-group sortable" data-sort="' + GRID_COLUMNS[gh] + '">' + (GRID_LABELS[gh] || GRID_COLUMNS[gh]) + ' <span class="sort-indicator"></span></div>';
  }
  html += '<div class="grid-col-credits sortable" data-sort="credits">AI Credits <span class="sort-indicator"></span></div>' +
    '<div class="grid-col-snap sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div>' +
    '<div class="grid-col-status sortable" data-sort="resetsIn">Status <span class="sort-indicator"></span></div></div>';
  for (var i = 0; i < sorted.length; i++) {
    var acc = sorted[i];
    var accId = 'acc-' + acc.accountId;
    var isExpanded = expandedAccounts.has(accId as any);

    var groupCells = '';
    // Pre-index models by group for Quick Adjust
    var modelIdsByGroup: Record<string, string[]> = {};
    var modelLabelsByGroup: Record<string, string[]> = {};
    if (acc.models) {
      for (var mi2 = 0; mi2 < acc.models.length; mi2++) {
        var mm = acc.models[mi2];
        var gk = mm.groupKey || 'unknown';
        if (!modelIdsByGroup[gk]) modelIdsByGroup[gk] = [];
        if (!modelLabelsByGroup[gk]) modelLabelsByGroup[gk] = [];
        modelIdsByGroup[gk].push(mm.modelId || '');
        modelLabelsByGroup[gk].push(mm.label || mm.modelId);
      }
    }

    // F3: Determine pinned group for this account (only if explicitly set)
    var pinnedKey = acc.pinnedGroup || '';
    var pinnedGroupData = null;
    if (pinnedKey) {
      var groups = acc.groups || [];
      for (var pg = 0; pg < groups.length; pg++) {
        if (groups[pg].groupKey === pinnedKey) { pinnedGroupData = groups[pg]; break; }
      }
    }

    for (var gi = 0; gi < GRID_COLUMNS.length; gi++) {
      var key = GRID_COLUMNS[gi];

      var g = null;
      var groups = acc.groups || [];
      for (var gj = 0; gj < groups.length; gj++) {
        if (groups[gj].groupKey === key) { g = groups[gj]; break; }
      }
      if (!g) {
        groupCells += '<div class="quota-cell"><span class="quota-pct">—</span></div>';
        continue;
      }
      var pct = Math.round(g.remainingPercent);
      var cls = 'good';
      if (g.isExhausted || pct === 0) cls = 'exhausted';
      else if (pct < 20) cls = 'warning';
      else if (pct < 50) cls = 'ok';
      
      // Q4: Mini progress bar under percentage
      var barCls = cls;

      // Group-level Quick Adjust — ±20 buttons and custom edit button, appear on hover
      var groupModelIds = (modelIdsByGroup[key] || []).join('|||');
      var groupModelLabels = (modelLabelsByGroup[key] || []).join('|||');
      var groupAdjust = '<span class="group-adjust" data-snap-id="' + acc.latestSnapshotId +
        '" data-group-key="' + key +
        '" data-group-model-ids="' + esc(groupModelIds) +
        '" data-group-model-labels="' + esc(groupModelLabels) +
        '" data-current-pct="' + pct + '">' +
        '<button class="gadj-btn" data-delta="-20" title="−20% all models in group">−20</button>' +
        '<button class="gadj-btn" data-delta="20" title="+20% all models in group">+20</button>' +
        '<button class="gadj-btn btn-custom" data-custom="true" title="Enter custom percentage for group">✏️</button>' +
        '</span>';

      // Gather rich metrics into a clean textual tooltip title for a pristine look
      var tooltipParts: string[] = [];
      if (g.timeUntilResetSec > 0) {
        tooltipParts.push('Reset in: ' + formatSeconds(g.timeUntilResetSec));
      }

      // F7: TTX badge data
      if (data.forecasts && data.forecasts[acc.accountId]) {
        var acctForecasts = data.forecasts[acc.accountId];
        for (var fi = 0; fi < acctForecasts.length; fi++) {
          if (acctForecasts[fi].groupKey === key && acctForecasts[fi].ttxLabel) {
            var ttxLabel = acctForecasts[fi].ttxLabel;
            if (ttxLabel && ttxLabel !== '') {
              tooltipParts.push('TTX: ' + ttxLabel);
            }
            break;
          }
        }
      }

      // F8: Cost data
      if (pct < 95 && data.estimatedCosts && data.estimatedCosts[acc.accountId]) {
        var acctCosts = data.estimatedCosts[acc.accountId];
        if (acctCosts.groups) {
          for (var ci = 0; ci < acctCosts.groups.length; ci++) {
            if (acctCosts.groups[ci].groupKey === key && acctCosts.groups[ci].hasData) {
              var costVal = acctCosts.groups[ci].estimatedCost || 0;
              if (costVal >= 0.01) {
                var costLabel = acctCosts.groups[ci].costLabel || '—';
                var hourly = acctCosts.groups[ci].hourlyLabel ? ' (' + acctCosts.groups[ci].hourlyLabel + ')' : '';
                tooltipParts.push('Estimated Cost: ' + costLabel + hourly);
              }
              break;
            }
          }
        }
      }

      var cellTitle = tooltipParts.join(' | ') || (GRID_LABELS[gi] || key);

      var pinnedStarHTML = (key === pinnedKey) ? '<span class="pinned-group-star" title="Pinned group" style="position: absolute; top: 4px; right: 4px; font-size: 12px; line-height: 1; z-index: 2;">⭐</span>' : '';

      groupCells += '<div class="quota-cell' + (key === 'gemini_unified' ? ' unified-pool-cell' : '') + '" title="' + esc(cellTitle) + '" style="display: flex; flex-direction: column; align-items: center; justify-content: center; position: relative;">' +
        pinnedStarHTML +
        '<span class="quota-pct ' + cls + '">' + (g.isEstimated ? '~' : '') + pct + '%' + '</span>' +
        renderQualityBadge(g) +
        '<div class="quota-minibar"><div class="quota-minibar-fill ' + barCls + '" style="width:' + pct + '%"></div></div>' +
        (g.timeUntilResetSec > 0 ? '<div class="reset-timer">↻ ' + formatSeconds(g.timeUntilResetSec) + '</div>' : '') +
        groupAdjust +
        '</div>';
    }


    // Q5: Health dots — visual status
    var dotCls = 'dot-ready';
    var badgeText = 'Ready';
    if (allExhausted(acc)) { dotCls = 'dot-empty'; badgeText = 'Empty'; }
    else if (!acc.isReady) { dotCls = 'dot-low'; badgeText = 'Low'; }

    var creditsCell = '<div class="credits-cell" style="position:relative">';
    if (acc.aiCredits && acc.aiCredits.length > 0) {
      var credits = acc.aiCredits[0].creditAmount;
      var creditCls = credits > 500 ? 'good' : credits > 100 ? 'ok' : 'warning';
      creditsCell += '<span class="credit-amount ' + creditCls + '" title="AI Credits">✦ ' +
        formatCredits(credits) + '</span>';
      // Credit renewal countdown
      creditsCell += renderCreditRenewal(acc.accountId, acc.creditRenewalDay);
    } else {
      creditsCell += '<span class="credit-amount muted">—</span>';
    }
    creditsCell += '</div>';

    var modelsHTML = '';
    if (acc.models && acc.models.length > 0) {
      // F3: Build group headers with pin stars in expanded view
      var groupedModels: Record<string, any[]> = {};
      for (var mi = 0; mi < acc.models.length; mi++) {
        var m = acc.models[mi];
        var gk2 = m.groupKey || 'unknown';
        if (!groupedModels[gk2]) groupedModels[gk2] = [];
        groupedModels[gk2].push(m);
      }

      var modelRows = '';
      for (var goi = 0; goi < GROUP_ORDER.length; goi++) {
        var groupKey2 = GROUP_ORDER[goi];
        var groupModels = groupedModels[groupKey2];
        if (!groupModels || groupModels.length === 0) continue;

        // F3: Group header with pin star
        var isPinned = pinnedKey === groupKey2;
        var starCls = isPinned ? 'pin-star pinned' : 'pin-star';
        var starTitle = isPinned ? 'Pinned — click to unpin' : 'Click to pin this group';
        var starChar = isPinned ? '★' : '☆';

        // Calculate group-level analytics for the expanded view
        var g2 = null;
        var groups2 = acc.groups || [];
        for (var gj2 = 0; gj2 < groups2.length; gj2++) {
          if (groups2[gj2].groupKey === groupKey2) { g2 = groups2[gj2]; break; }
        }

        var expandedBadges = '';
        if (g2) {
          var pct2 = Math.round(g2.remainingPercent);
          if (g2.timeUntilResetSec > 0) {
            expandedBadges += ' <span class="quota-reset" style="margin-left:8px">↻ ' + formatSeconds(g2.timeUntilResetSec) + '</span>';
          }

          // F7: TTX badge shows "~Xh" time-to-exhaustion
          if (data.forecasts && data.forecasts[acc.accountId]) {
            var acctForecasts2 = data.forecasts[acc.accountId];
            for (var fi2 = 0; fi2 < acctForecasts2.length; fi2++) {
              if (acctForecasts2[fi2].groupKey === groupKey2 && acctForecasts2[fi2].ttxLabel) {
                var ttxSev2 = acctForecasts2[fi2].severity || 'safe';
                var ttxLabel2 = acctForecasts2[fi2].ttxLabel;
                if (ttxLabel2 && ttxLabel2 !== '' && ttxSev2 !== 'none') {
                  expandedBadges += ' <span class="ttx-badge ttx-' + ttxSev2 + '" style="margin-left:6px" title="Time to exhaustion at current burn rate">' + esc(ttxLabel2) + '</span>';
                }
                break;
              }
            }
          }

          // F8: Cost badge shows estimated cost
          if (pct2 < 95 && data.estimatedCosts && data.estimatedCosts[acc.accountId]) {
            var acctCosts2 = data.estimatedCosts[acc.accountId];
            if (acctCosts2.groups) {
              for (var ci2 = 0; ci2 < acctCosts2.groups.length; ci2++) {
                if (acctCosts2.groups[ci2].groupKey === groupKey2 && acctCosts2.groups[ci2].hasData) {
                  var costVal2 = acctCosts2.groups[ci2].estimatedCost || 0;
                  if (costVal2 >= 0.01) {
                    var costLabel2 = acctCosts2.groups[ci2].costLabel || '—';
                    var costCls2 = 'cost-low';
                    if (costVal2 >= 10) costCls2 = 'cost-high';
                    else if (costVal2 >= 3) costCls2 = 'cost-medium';
                    var costTitle2 = 'Estimated cost this cycle';
                    if (acctCosts2.groups[ci2].hourlyLabel) {
                      costTitle2 += ' (' + acctCosts2.groups[ci2].hourlyLabel + ')';
                    }
                    expandedBadges += ' <span class="cost-badge ' + costCls2 + '" style="margin-left:6px" title="' + costTitle2 + '">' + esc(costLabel2) + '</span>';
                  }
                  break;
                }
              }
            }
          }
        }

        modelRows += '<div class="model-group-header">' +
          '<button class="' + starCls + '" data-pin-group="' + groupKey2 + '" data-pin-account="' + acc.accountId + '" title="' + starTitle + '">' + starChar + '</button>' +
          '<span class="model-group-name" style="color:' + (GROUP_COLORS[groupKey2] || 'var(--text-secondary)') + '">' + (GROUP_NAMES[groupKey2] || groupKey2) + '</span>' +
          expandedBadges +
          '</div>';

        for (var mi3 = 0; mi3 < groupModels.length; mi3++) {
          var m = groupModels[mi3];
        var mpct = Math.round(m.remainingPercent);
        var mcls = 'good';
        if (m.isExhausted || mpct === 0) mcls = 'exhausted';
        else if (mpct < 20) mcls = 'warning';
        else if (mpct < 50) mcls = 'ok';
        var color = GROUP_COLORS[m.groupKey] || '#94a3b8';
        var resetStr = m.resetSeconds > 0 ? ('↻ ' + formatSeconds(m.resetSeconds)) : '';

        // Intelligence badges from usage data
        var intellBadges = '';
        var usageModels = usageDataCache ? ((usageDataCache as any).models as any[] | undefined) : undefined;
        if (usageModels) {
          for (var ui = 0; ui < usageModels.length; ui++) {
            var um = usageModels[ui];
            if (um.modelId === m.modelId && um.accountId === acc.accountId && um.hasIntelligence) {
              var rateStr = (um.currentRate * 100).toFixed(1) + '%/hr';
              intellBadges += '<span class="rate-badge" title="Current consumption rate">' + rateStr + '</span>';
              if (um.projectedUsage > 0) {
                var projPct = Math.round(um.projectedUsage * 100);
                var projCls = projPct > 95 ? 'proj-danger' : (projPct > 80 ? 'proj-warn' : 'proj-ok');
                intellBadges += '<span class="proj-badge ' + projCls + '" title="Projected usage at reset">→' + projPct + '%</span>';
              }
              if (um.projectedExhaustion) {
                var exhaust = new Date(um.projectedExhaustion);
                var minsLeft = Math.round((exhaust.getTime() - Date.now()) / 60000);
                if (minsLeft > 0) {
                  intellBadges += '<span class="exhaust-badge" title="Projected exhaustion time">⚠ ' + (minsLeft > 60 ? Math.round(minsLeft/60) + 'h' : minsLeft + 'm') + '</span>';
                }
              }
              break;
            }
          }
        }

        // Quick Adjust controls — visible on hover (uses 20% increments and custom edit button for Antigravity)
        var adjustBtns = '<span class="adjust-controls" data-snap-id="' + acc.latestSnapshotId + '" data-model-id="' + esc(m.modelId || '') + '" data-model-label="' + esc(m.label || m.modelId) + '" data-current-pct="' + mpct + '">' +
          '<button class="adj-btn" data-delta="-20" title="−20%">−20</button>' +
          '<button class="adj-btn" data-delta="20" title="+20%">+20</button>' +
          '<button class="adj-btn btn-custom" data-custom="true" title="Enter custom percentage">✏️</button>' +
          '</span>';

        modelRows += '<div class="model-row">' +
          '<div class="model-indicator" style="background:' + color + '"></div>' +
          '<span class="model-label">' + esc(m.label || m.modelId) + '</span>' +
          '<div class="model-bar-track"><div class="model-bar-fill ' + mcls + '" style="width:' + mpct + '%"></div></div>' +
          '<span class="model-pct ' + mcls + '">' + (m.isEstimated ? '~' : '') + mpct + '%</span>' +
          renderQualityBadge(m) +
          adjustBtns +
          '<span class="model-reset">' + resetStr + '</span>' +
          intellBadges +
          '</div>';
        }
      }
      var expandedCls = isExpanded ? ' is-expanded' : '';
      modelsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '">' + modelRows +
        '<div class="account-actions">' +
        '<button class="btn-clear-snaps btn-warning" data-clear-account="' + acc.accountId + '" data-clear-email="' + esc(acc.email) + '" title="Delete all snapshots for this account">Clear Snapshots</button>' +
        '<button class="btn-delete-account btn-danger" data-delete-account="' + acc.accountId + '" data-delete-email="' + esc(acc.email) + '" title="Remove account and all its data">Remove Account</button>' +
        '</div></div>';
    }

    var chevronCls = isExpanded ? 'chevron expanded' : 'chevron';
    // Bug 5 fix: Dim based on quota readiness, not snap age.
    // Accounts with any depleted group get dimmed; fully ready = bright.
    // UX: Status class for CSS border signaling (replaces opacity dimming anti-pattern)
    var statusClass = '';
    if (allExhausted(acc)) statusClass = ' status-empty';
    else if (!acc.isReady) statusClass = ' status-low';
    else statusClass = ' status-ready';
    html += '<div class="account-card' + statusClass + '">' +
      '<div class="account-row" data-toggle="' + accId + '">' +
      '<div class="account-info">' +
      '<div class="account-email"><span class="' + chevronCls + '" id="chev-' + accId + '">▸</span> ' + esc(acc.email) + '</div>' +
      '<div class="account-meta" style="position:relative">' +
      (acc.planName ? '<span class="plan-badge">' + esc(acc.planName) + '</span>' : '') +
      renderAccountTags(acc) +
      renderAccountNote(acc) +
      '</div></div>' +
      groupCells +
      creditsCell +
      '<div class="snap-cell"><span class="snap-ago" title="' + esc(acc.lastSeen || '') + '">' + esc(acc.stalenessLabel) + '</span></div>' +
      '<div class="status-cell"><span class="health-dot ' + dotCls + '">● ' + badgeText + '</span>' + (function() {
        var rs = getSoonestResetSec(acc);
        if (rs <= 0) return '';
        return '<div class="reset-timer" style="margin-top:2px">↻ ' + formatSeconds(rs) + '</div>';
      })() + '</div>' +
      '</div>' +
      modelsHTML +
      '</div>';
  }

  html += '</div></div>'; // close provider-body + provider-section
  } // end if acctCount > 0

  // Bug 6 fix: Apply status filter to Codex/Claude sections too
  var sf = document.getElementById('quota-filter-status');
  var statusVal = sf ? (sf as HTMLSelectElement).value : 'all';
  if (data.codexSnapshots && data.codexSnapshots.length > 0 && (pf === 'all' || pf === 'codex')) {
    html += renderCodexProviderSection(data.codexSnapshots, statusVal, data.allAccounts || []);
  }
  if (data.claudeSnapshot && (pf === 'all' || pf === 'claude')) {
    var clStatus = getCodexClaudeStatus(data.claudeSnapshot);
    if (statusVal === 'all' || clStatus === statusVal) {
      html += renderClaudeProviderSection(data.claudeSnapshot);
    }
  }
  if (data.cursorSnapshots && data.cursorSnapshots.length > 0 && (pf === 'all' || pf === 'cursor')) {
    html += renderCursorProviderSection(data.cursorSnapshots, statusVal, data.allAccounts || []);
  }

  if (data.copilotSnapshots && data.copilotSnapshots.length > 0 && (pf === 'all' || pf === 'copilot')) {
    html += renderCopilotProviderSection(data.copilotSnapshots, statusVal, data.allAccounts || []);
  }

  // V3: Empty states when a specific provider is selected but has no data
  if (pf === 'antigravity' && acctCount === 0) {
    html += '<div class="provider-empty-state" data-provider="antigravity">' +
      '<span class="provider-empty-icon">⚡</span>' +
      '<p>No Antigravity accounts detected</p>' +
      '<p class="empty-hint">Open Windsurf and log in to start tracking quotas</p></div>';
  }
  if (pf === 'codex' && (!data.codexSnapshots || data.codexSnapshots.length === 0)) {
    html += '<div class="provider-empty-state" data-provider="codex">' +
      '<span class="provider-empty-icon">🤖</span>' +
      '<p>No Codex snapshots yet</p>' +
      '<p class="empty-hint">Install Codex CLI and click <strong>Snap Now</strong> to capture</p></div>';
  }
  if (pf === 'claude' && !data.claudeSnapshot) {
    html += '<div class="provider-empty-state" data-provider="claude">' +
      '<span class="provider-empty-icon">🔮</span>' +
      '<p>No Claude Code data yet</p>' +
      '<p class="empty-hint">Enable the Claude bridge in <strong>Settings</strong></p></div>';
  }
  if (pf === 'cursor' && (!data.cursorSnapshots || data.cursorSnapshots.length === 0)) {
    html += '<div class="provider-empty-state" data-provider="cursor">' +
      '<span class="provider-empty-icon">🖱️</span>' +
      '<p>No Cursor data yet</p>' +
      '<p class="empty-hint">Enable Cursor capture in <strong>Settings</strong> or click <strong>Snap Now</strong></p></div>';
  }

  if (pf === 'copilot' && (!data.copilotSnapshots || data.copilotSnapshots.length === 0)) {
    html += '<div class="provider-empty-state" data-provider="copilot">' +
      '<span class="provider-empty-icon">🐙</span>' +
      '<p>No GitHub Copilot data yet</p>' +
      '<p class="empty-hint">Add a PAT in <strong>Settings</strong> or click <strong>Snap Now</strong></p></div>';
  }

  var eligibleAccount = (data.accounts || []).find(function(acc: any) {
    return acc.planTier && acc.planTier.toLowerCase() === 'ultra' && acc.hasClaimedBonus2026 === 0;
  });
  var bannerHTML = '';
  if (eligibleAccount) {
    bannerHTML = '<div class="io-alert-card" data-account-id="' + eligibleAccount.accountId + '">' +
      '<div class="io-alert-content">' +
      '<div class="io-alert-title">✨ Google I/O 2026 Promotional Bonus</div>' +
      '<div class="io-alert-desc">Exclusive for Ultra members: Claim your $100 Overage Credit Bonus before it expires on <strong>May 25, 2026</strong>.</div>' +
      '</div>' +
      '<button class="io-claim-btn" data-claim-account-id="' + eligibleAccount.accountId + '">Claim $100 Bonus</button>' +
      '</div>';
  }

  grid.innerHTML = bannerHTML + html;

  var claimBtn = grid.querySelector('.io-claim-btn');
  if (claimBtn) {
    claimBtn.addEventListener('click', function(e) {
      var btn = e.currentTarget as HTMLButtonElement;
      var accId = parseInt(btn.getAttribute('data-claim-account-id') || '0', 10);
      if (accId > 0) {
        btn.disabled = true;
        btn.textContent = 'Claiming...';
        claimOverageBonus(accId).then(function() {
          showToast('✨ $100 Overage Bonus credit added!', 'success');
          fetchStatus().then(function(freshData) {
            document.dispatchEvent(new CustomEvent('niyantra:status-refreshed', { detail: { data: freshData } }));
            document.dispatchEvent(new CustomEvent('niyantra:overview-refresh'));
          }).catch(function() {});
        }).catch(function(err) {
          btn.disabled = false;
          btn.textContent = 'Claim $100 Bonus';
          showToast('❌ ' + (err.message || 'Claim failed'), 'error');
        });
      }
    });
  }

  // Wire up provider section collapse (state already baked into HTML)
  grid.querySelectorAll('.provider-header[data-toggle-provider]').forEach(function(hdr) {
    (hdr as HTMLElement).addEventListener('click', function() {
      var targetId = (hdr as HTMLElement).dataset.toggleProvider;
      var body = document.getElementById(targetId!);
      var chev = document.getElementById('pchev-' + targetId!);
      if (!body) return;
      var collapsed = body.classList.toggle('collapsed');
      if (chev) chev.textContent = collapsed ? '▸' : '▾';
      if (collapsed) {
        collapsedProviders.add(targetId!);
      } else {
        collapsedProviders.delete(targetId!);
      }
      localStorage.setItem('niyantra_collapsed_providers', JSON.stringify(Array.from(collapsedProviders)));
    });
  });
  updateSortHeaders();
}

export function renderCodexProviderSection(codexSnaps: any[], statusFilter: string, allAccounts: any[] = []): string {
  var cxCollapseClass = collapsedProviders.has('section-codex') ? ' collapsed' : '';
  var cxChevron = collapsedProviders.has('section-codex') ? '▸' : '▾';
  var cxReadyCount = 0;
  for (var i = 0; i < codexSnaps.length; i++) {
    if (getCodexClaudeStatus(codexSnaps[i]) === 'ready') cxReadyCount++;
  }

  var html = '<div class="provider-section" data-provider="codex">' +
    '<div class="provider-header" data-toggle-provider="section-codex">' +
    '<div class="provider-header-left">' +
    '<span class="provider-chevron" id="pchev-section-codex">' + cxChevron + '</span>' +
    '<span class="provider-name">\ud83e\udd16 Codex / ChatGPT</span>' +
    '<span class="provider-count">' + codexSnaps.length + ' account' + (codexSnaps.length !== 1 ? 's' : '') + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + cxReadyCount + ' ready)</span></span>' +
    '</div></div>' +
    '<div class="provider-body' + cxCollapseClass + '" id="section-codex">' +
    '<div class="grid-header grid-codex">' +
    '<div class="sortable" data-sort="account">Account <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="plan">Plan <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="fiveHour">Short-Term <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="sevenDay">Weekly <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="credits">Credits <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="status">Status <span class="sort-indicator"></span></div>' +
    '</div>';

  var sortedSnaps = sortProviderArray(codexSnaps, 'codex');
  var renderedCount = 0;
  for (var i = 0; i < sortedSnaps.length; i++) {
    var cs = sortedSnaps[i];
    var cxStatus = getCodexClaudeStatus(cs);
    if (statusFilter !== 'all' && cxStatus !== statusFilter) continue;
    renderedCount++;

    var fiveUsed = cs.fiveHourPct || 0;
    var fiveRem = Math.max(0, 100 - fiveUsed);
    var fiveCls = fiveRem > 50 ? 'good' : fiveRem > 20 ? 'ok' : fiveRem > 0 ? 'warning' : 'exhausted';
    var fiveReset = cs.fiveHourReset ? formatResetTime(cs.fiveHourReset) : '';
    var sevenUsed = cs.sevenDayPct ? cs.sevenDayPct : 0;
    var sevenRem = Math.max(0, 100 - sevenUsed);
    var sevenCls = sevenRem > 50 ? 'good' : sevenRem > 20 ? 'ok' : sevenRem > 0 ? 'warning' : 'exhausted';
    var sevenReset = cs.sevenDayReset ? formatResetTime(cs.sevenDayReset) : '';
    var capturedAgo = cs.capturedAt ? formatTimeAgo(cs.capturedAt) : '\u2014';
    var dotCls = (fiveUsed >= 80 || sevenUsed >= 80) ? 'dot-low' : 'dot-ready';
    var dotText = dotCls === 'dot-ready' ? 'Ready' : 'Low';
    var displayName = cs.email || (cs.accountId && cs.accountId.length > 12 ? cs.accountId.substring(0,6) + '..' + cs.accountId.slice(-6) : (cs.accountId || 'Codex'));
    var creditsStr = cs.creditsBalance !== null && cs.creditsBalance !== undefined ? cs.creditsBalance.toFixed(2) : String.fromCharCode(8212);

    var localAccId = cs.ownerAccountId || 0;
    var accId = 'acc-codex-' + cs.id;
    var isExpanded = expandedAccounts.has(accId as any);
    var chevronCls = isExpanded ? 'chevron expanded' : 'chevron';

    var chevronHTML = localAccId > 0 ? '<span class="' + chevronCls + '" id="chev-' + accId + '">▸</span> ' : '';
    var emailHTML = '<div class="account-email">' + chevronHTML + esc(displayName) + '</div>';
    var accData = localAccId > 0 ? allAccounts.find(function(a: any) { return a.id === localAccId; }) : null;
    var metaHTML = accData ? '<div class="account-meta" style="position:relative">' + renderAccountTags(accData) + renderAccountNote(accData) + '</div>' : '';

    var actionsHTML = '';
    if (localAccId > 0) {
      var expandedCls = isExpanded ? ' is-expanded' : '';
      actionsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '">' +
        '<div class="account-actions" style="margin-top:0">' +
        '<button class="btn-clear-snaps" data-clear-account="' + localAccId + '" data-clear-email="' + esc(displayName) + '" title="Delete all snapshots for this account">Clear Snapshots</button>' +
        '<button class="btn-delete-account" data-delete-account="' + localAccId + '" data-delete-email="' + esc(displayName) + '" title="Remove account and all its data">Remove Account</button>' +
        '</div></div>';
    }

    var toggleAttr = localAccId > 0 ? ' data-toggle="' + accId + '"' : '';

    var statusClass = '';
    if (cxStatus === 'empty') statusClass = ' status-empty';
    else if (cxStatus === 'low') statusClass = ' status-low';
    else statusClass = ' status-ready';

    html += '<div class="account-card' + statusClass + '"><div class="account-row grid-codex"' + toggleAttr + '>' +
      '<div class="account-info">' + emailHTML + metaHTML + '</div>' +
      '<div>' + (cs.planType ? '<span class="plan-badge">' + esc(cs.planType) + '</span>' : String.fromCharCode(8212)) + '</div>' +
      '<div class="quota-cell"><span class="quota-pct ' + fiveCls + '">' + estPrefix(isResetElapsed(cs.fiveHourReset)) + fiveRem.toFixed(0) + '%</span>' +
      renderProviderEstBadge(isResetElapsed(cs.fiveHourReset)) +
      '<div class="quota-minibar"><div class="quota-minibar-fill ' + fiveCls + '" style="width:' + fiveRem + '%"></div></div>' +
      (fiveReset ? '<span class="quota-reset">\u21bb ' + fiveReset + '</span>' : '') + '</div>' +
      '<div class="quota-cell"><span class="quota-pct ' + sevenCls + '">' + estPrefix(isResetElapsed(cs.sevenDayReset)) + sevenRem.toFixed(0) + '%</span>' +
      renderProviderEstBadge(isResetElapsed(cs.sevenDayReset)) +
      '<div class="quota-minibar"><div class="quota-minibar-fill ' + sevenCls + '" style="width:' + sevenRem + '%"></div></div>' +
      (sevenReset ? '<span class="quota-reset">\u21bb ' + sevenReset + '</span>' : '') + '</div>' +
      '<div class="credits-cell"><span class="credit-amount">' + creditsStr + '</span></div>' +
      '<div class="snap-cell"><span class="snap-ago">' + capturedAgo + '</span></div>' +
      '<div class="status-cell"><span class="health-dot ' + dotCls + '">\u25cf ' + dotText + '</span></div>' +
      '</div>' +
      actionsHTML +
      '</div>';
  }

  if (renderedCount === 0) return '';
  html += '</div></div>';
  return html;
}

export function renderClaudeProviderSection(cl: any): string {
  var clFive = cl.fiveHourPct || 0;
  var clFiveRem = Math.max(0, 100 - clFive);
  var clFiveCls = clFiveRem > 50 ? 'good' : clFiveRem > 20 ? 'ok' : clFiveRem > 0 ? 'warning' : 'exhausted';
  var clSeven = cl.sevenDayPct ? cl.sevenDayPct : 0;
  var clSevenRem = Math.max(0, 100 - clSeven);
  var clSevenCls = clSevenRem > 50 ? 'good' : clSevenRem > 20 ? 'ok' : clSevenRem > 0 ? 'warning' : 'exhausted';
  var clAgo = cl.capturedAt ? formatTimeAgo(cl.capturedAt) : '\u2014';
  var dotCls = (clFive >= 80 || clSeven >= 80) ? 'dot-low' : 'dot-ready';
  var dotText = dotCls === 'dot-ready' ? 'Ready' : 'Low';
  var clCollapseClass = collapsedProviders.has('section-claude') ? ' collapsed' : '';
  var clChevron = collapsedProviders.has('section-claude') ? '▸' : '▾';

  var clStatus = getCodexClaudeStatus(cl);
  var statusClass = '';
  if (clStatus === 'empty') statusClass = ' status-empty';
  else if (clStatus === 'low') statusClass = ' status-low';
  else statusClass = ' status-ready';

  return '<div class="provider-section" data-provider="claude">' +
    '<div class="provider-header" data-toggle-provider="section-claude">' +
    '<div class="provider-header-left">' +
    '<span class="provider-chevron" id="pchev-section-claude">' + clChevron + '</span>' +
    '<span class="provider-name">\ud83d\udd17 Claude Code</span>' +
    '<span class="provider-count">1 account \u00b7 Bridge <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + (getCodexClaudeStatus(cl) === 'ready' ? '1 ready' : '0 ready') + ')</span></span>' +
    '</div></div>' +
    '<div class="provider-body' + clCollapseClass + '" id="section-claude">' +
    '<div class="grid-header grid-claude">' +
    '<div>Source</div><div>Short-Term</div><div>Weekly</div><div>Last Snap</div><div>Status</div>' +
    '</div>' +
    '<div class="account-card' + statusClass + '"><div class="account-row grid-claude">' +
    '<div class="account-info"><div class="account-email">' + esc(cl.source || 'statusline') + '</div></div>' +
    '<div class="quota-cell"><span class="quota-pct ' + clFiveCls + '">' + estPrefix(isResetElapsed(cl.fiveHourReset)) + clFiveRem.toFixed(0) + '%</span>' +
    renderProviderEstBadge(isResetElapsed(cl.fiveHourReset)) +
    '<div class="quota-minibar"><div class="quota-minibar-fill ' + clFiveCls + '" style="width:' + clFiveRem + '%"></div></div></div>' +
    '<div class="quota-cell"><span class="quota-pct ' + clSevenCls + '">' + estPrefix(isResetElapsed(cl.sevenDayReset)) + clSevenRem.toFixed(0) + '%</span>' +
    renderProviderEstBadge(isResetElapsed(cl.sevenDayReset)) +
    '<div class="quota-minibar"><div class="quota-minibar-fill ' + clSevenCls + '" style="width:' + clSevenRem + '%"></div></div></div>' +
    '<div class="snap-cell"><span class="snap-ago">' + clAgo + '</span></div>' +
    '<div class="status-cell"><span class="health-dot ' + dotCls + '">\u25cf ' + dotText + '</span></div>' +
    '</div></div></div></div>';
}

export function formatResetTime(isoString: string | null): string {
  if (!isoString) return '';
  var reset = new Date(isoString);
  var now = new Date();
  var diffSec = (reset.getTime() - now.getTime()) / 1000;
  if (diffSec <= 0) return 'now';
  return formatSeconds(diffSec);
}

// Returns true if the given ISO reset timestamp is in the past
function isResetElapsed(isoString: string | null | undefined): boolean {
  if (!isoString) return false;
  return new Date(isoString).getTime() < Date.now();
}

// Renders estimation indicator for non-Antigravity providers when reset has elapsed
// Now returns empty string — tilde prefix on percentage handles this instead
function renderProviderEstBadge(resetElapsed: boolean): string {
  return '';
}

// Returns tilde prefix if reset has elapsed (for non-Antigravity providers)
function estPrefix(resetElapsed: boolean): string {
  return resetElapsed ? '~' : '';
}

export function getCursorStatus(snap: any): string {
  var usagePct = snap.usagePct || 0;
  var rem = Math.max(0, 100 - usagePct);
  if (rem === 0) return 'empty';
  if (usagePct >= 80) return 'low';
  return 'ready';
}

export function renderCursorProviderSection(cursorSnaps: any[], statusFilter: string, allAccounts: any[] = []): string {
  var crCollapseClass = collapsedProviders.has('section-cursor') ? ' collapsed' : '';
  var crChevron = collapsedProviders.has('section-cursor') ? '▸' : '▾';

  var crReadyCount = 0;
  for (var i = 0; i < cursorSnaps.length; i++) {
    if (getCursorStatus(cursorSnaps[i]) === 'ready') crReadyCount++;
  }

  var html = '<div class="provider-section" data-provider="cursor">' +
    '<div class="provider-header" data-toggle-provider="section-cursor">' +
    '<div class="provider-header-left">' +
    '<span class="provider-chevron" id="pchev-section-cursor">' + crChevron + '</span>' +
    '<span class="provider-name">\ud83d\uddb1\ufe0f Cursor</span>' +
    '<span class="provider-count">' + cursorSnaps.length + ' account' + (cursorSnaps.length !== 1 ? 's' : '') + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + crReadyCount + ' ready)</span></span>' +
    '</div></div>' +
    '<div class="provider-body' + crCollapseClass + '" id="section-cursor">' +
    '<div class="grid-header grid-cursor">' +
    '<div class="sortable" data-sort="account">Account <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="plan">Plan <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="premiumUsed">Premium Used <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="usage">Usage <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="status">Status <span class="sort-indicator"></span></div>' +
    '</div>';

  var sortedSnaps = sortProviderArray(cursorSnaps, 'cursor');
  var renderedCount = 0;
  for (var i = 0; i < sortedSnaps.length; i++) {
    var cs = sortedSnaps[i];
    var crStatus = getCursorStatus(cs);
    if (statusFilter !== 'all' && crStatus !== statusFilter) continue;
    renderedCount++;

    var usagePct = cs.usagePct || 0;
    var remaining = Math.max(0, 100 - usagePct);
    var cls = remaining > 50 ? 'good' : remaining > 20 ? 'ok' : remaining > 0 ? 'warning' : 'exhausted';
    var capturedAgo = cs.capturedAt ? formatTimeAgo(cs.capturedAt) : '\u2014';
    var dotCls = usagePct >= 80 ? 'dot-low' : 'dot-ready';
    var dotText = dotCls === 'dot-ready' ? 'Ready' : 'Low';
    var displayName = cs.email || 'Cursor';
    var usedStr = cs.premiumUsed !== undefined ? cs.premiumUsed : String.fromCharCode(8212);
    var limitStr = cs.premiumLimit !== undefined ? cs.premiumLimit : String.fromCharCode(8212);

    // Build per-model breakdown rows from modelsJson
    var modelRows = '';
    if (cs.modelsJson && cs.modelsJson !== '{}') {
      try {
        var models = typeof cs.modelsJson === 'string' ? JSON.parse(cs.modelsJson) : cs.modelsJson;
        var modelKeys = Object.keys(models);
        if (modelKeys.length > 0) {
          modelRows = '<div class="cursor-model-breakdown">';
          for (var mi = 0; mi < modelKeys.length; mi++) {
            var mKey = modelKeys[mi];
            var mVal = models[mKey];
            var mUsed = mVal.numRequests || 0;
            var mLimit = mVal.maxRequestUsage || 0;
            var mPct = mLimit > 0 ? (mUsed / mLimit * 100) : 0;
            var mRem = Math.max(0, 100 - mPct);
            var mCls = mRem > 50 ? 'good' : mRem > 20 ? 'ok' : mRem > 0 ? 'warning' : 'exhausted';
            modelRows += '<div class="cursor-model-row">' +
              '<span class="cursor-model-name">' + esc(mKey) + '</span>' +
              '<div class="quota-minibar"><div class="quota-minibar-fill ' + mCls + '" style="width:' + mRem + '%"></div></div>' +
              '<span class="cursor-model-usage">' + mUsed + '/' + mLimit + '</span>' +
              '</div>';
          }
          modelRows += '</div>';
        }
      } catch(e) { /* graceful degradation */ }
    }

    var localAccId = cs.accountId || 0;
    var accId = 'acc-cursor-' + cs.id;
    var isExpanded = expandedAccounts.has(accId as any);
    var chevronCls = isExpanded ? 'chevron expanded' : 'chevron';

    var chevronHTML = (localAccId > 0 || modelRows) ? '<span class="' + chevronCls + '" id="chev-' + accId + '">▸</span> ' : '';
    var emailHTML = '<div class="account-email">' + chevronHTML + esc(displayName) + '</div>';
    var accData = localAccId > 0 ? allAccounts.find(function(a: any) { return a.id === localAccId; }) : null;
    var metaHTML = accData ? '<div class="account-meta" style="position:relative">' + renderAccountTags(accData) + renderAccountNote(accData) + '</div>' : '';

    var actionsHTML = '';
    if (localAccId > 0 || modelRows) {
      var expandedCls = isExpanded ? ' is-expanded' : '';
      var accountActions = localAccId > 0 ?
        '<div class="account-actions">' +
        '<button class="btn-clear-snaps" data-clear-account="' + localAccId + '" data-clear-email="' + esc(displayName) + '" title="Delete all snapshots for this account">Clear Snapshots</button>' +
        '<button class="btn-delete-account" data-delete-account="' + localAccId + '" data-delete-email="' + esc(displayName) + '" title="Remove account and all its data">Remove Account</button>' +
        '</div>' : '';
      actionsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '">' +
        modelRows +
        accountActions +
        '</div>';
    }

    var toggleAttr = (localAccId > 0 || modelRows) ? ' data-toggle="' + accId + '"' : '';

    var statusClass = '';
    if (crStatus === 'empty') statusClass = ' status-empty';
    else if (crStatus === 'low') statusClass = ' status-low';
    else statusClass = ' status-ready';

    html += '<div class="account-card' + statusClass + '"><div class="account-row grid-cursor"' + toggleAttr + '>' +
      '<div class="account-info">' + emailHTML + metaHTML + '</div>' +
      '<div>' + (cs.planType ? '<span class="plan-badge">' + esc(cs.planType) + '</span>' : String.fromCharCode(8212)) + '</div>' +
      '<div class="quota-cell"><span class="quota-pct ' + cls + '">' + usedStr + ' / ' + limitStr + '</span>' +
      renderProviderEstBadge(isResetElapsed(cs.cycleEnd)) +
      '<div class="quota-minibar"><div class="quota-minibar-fill ' + cls + '" style="width:' + remaining + '%"></div></div></div>' +
      '<div class="quota-cell"><span class="quota-pct ' + cls + '">' + remaining.toFixed(0) + '% left</span></div>' +
      '<div class="snap-cell"><span class="snap-ago">' + capturedAgo + '</span></div>' +
      '<div class="status-cell"><span class="health-dot ' + dotCls + '">\u25cf ' + dotText + '</span></div>' +
      '</div>' +
      actionsHTML +
      '</div>';
  }

  if (renderedCount === 0) return '';
  html += '</div></div>';
  return html;
}



export function getCopilotStatus(snap: any): string {
  var premiumPct = snap.premiumPct || 0;
  var rem = Math.max(0, 100 - premiumPct);
  if (rem === 0) return 'empty';
  if (premiumPct >= 80) return 'low';
  return 'ready';
}

export function renderCopilotProviderSection(copilotSnaps: any[], statusFilter: string, allAccounts: any[] = []): string {
  var cpCollapseClass = collapsedProviders.has('section-copilot') ? ' collapsed' : '';
  var cpChevron = collapsedProviders.has('section-copilot') ? '▸' : '▾';

  var cpReadyCount = 0;
  for (var i = 0; i < copilotSnaps.length; i++) {
    if (getCopilotStatus(copilotSnaps[i]) === 'ready') cpReadyCount++;
  }

  var html = '<div class="provider-section" data-provider="copilot">' +
    '<div class="provider-header" data-toggle-provider="section-copilot">' +
    '<div class="provider-header-left">' +
    '<span class="provider-chevron" id="pchev-section-copilot">' + cpChevron + '</span>' +
    '<span class="provider-name">\ud83d\udc19 GitHub Copilot</span>' +
    '<span class="provider-count">' + copilotSnaps.length + ' account' + (copilotSnaps.length !== 1 ? 's' : '') + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + cpReadyCount + ' ready)</span></span>' +
    '</div></div>' +
    '<div class="provider-body' + cpCollapseClass + '" id="section-copilot">' +
    '<div class="grid-header grid-copilot">' +
    '<div class="sortable" data-sort="account">Account <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="plan">Plan <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="premium">Premium <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="chat">Chat <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div>' +
    '<div class="sortable" data-sort="status">Status <span class="sort-indicator"></span></div>' +
    '</div>';

  var sortedSnaps = sortProviderArray(copilotSnaps, 'copilot');
  var renderedCount = 0;
  for (var i = 0; i < sortedSnaps.length; i++) {
    var cp = sortedSnaps[i];
    var cpStatus = getCopilotStatus(cp);
    if (statusFilter !== 'all' && cpStatus !== statusFilter) continue;
    renderedCount++;

    // Premium interactions
    var premiumPct = cp.premiumPct || 0;
    var premiumRem = Math.max(0, 100 - premiumPct);
    var premiumCls = premiumRem > 50 ? 'good' : premiumRem > 20 ? 'ok' : premiumRem > 0 ? 'warning' : 'exhausted';

    // Chat usage
    var chatPct = cp.chatPct || 0;
    var chatRem = Math.max(0, 100 - chatPct);
    var chatCls = chatRem > 50 ? 'good' : chatRem > 20 ? 'ok' : chatRem > 0 ? 'warning' : 'exhausted';

    var capturedAgo = cp.capturedAt ? formatTimeAgo(cp.capturedAt) : '\u2014';
    var dotCls = premiumPct >= 80 ? 'dot-low' : 'dot-ready';
    var dotText = dotCls === 'dot-ready' ? 'Ready' : 'Low';
    var displayName = cp.username || cp.email || 'Copilot';

    var localAccId = cp.accountId || 0;
    var accId = 'acc-copilot-' + cp.id;
    var isExpanded = expandedAccounts.has(accId as any);
    var chevronCls = isExpanded ? 'chevron expanded' : 'chevron';

    var chevronHTML = localAccId > 0 ? '<span class="' + chevronCls + '" id="chev-' + accId + '">▸</span> ' : '';
    var emailHTML = '<div class="account-email">' + chevronHTML + esc(displayName) + '</div>';
    var accData = localAccId > 0 ? allAccounts.find(function(a: any) { return a.id === localAccId; }) : null;
    var metaHTML = accData ? '<div class="account-meta" style="position:relative">' + renderAccountTags(accData) + renderAccountNote(accData) + '</div>' : '';

    var actionsHTML = '';
    if (localAccId > 0) {
      var expandedCls = isExpanded ? ' is-expanded' : '';
      actionsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '">' +
        '<div class="account-actions" style="margin-top:0">' +
        '<button class="btn-clear-snaps" data-clear-account="' + localAccId + '" data-clear-email="' + esc(displayName) + '" title="Delete all snapshots for this account">Clear Snapshots</button>' +
        '<button class="btn-delete-account" data-delete-account="' + localAccId + '" data-delete-email="' + esc(displayName) + '" title="Remove account and all its data">Remove Account</button>' +
        '</div></div>';
    }

    var toggleAttr = localAccId > 0 ? ' data-toggle="' + accId + '"' : '';

    var statusClass = '';
    if (cpStatus === 'empty') statusClass = ' status-empty';
    else if (cpStatus === 'low') statusClass = ' status-low';
    else statusClass = ' status-ready';

    html += '<div class="account-card' + statusClass + '"><div class="account-row grid-copilot"' + toggleAttr + '>' +
      '<div class="account-info">' + emailHTML + metaHTML + '</div>' +
      '<div>' + (cp.plan ? '<span class="plan-badge">' + esc(cp.plan) + '</span>' : String.fromCharCode(8212)) + '</div>' +
      '<div class="quota-cell"><span class="quota-pct ' + premiumCls + '">' + premiumRem.toFixed(0) + '% left</span>' +
      renderProviderEstBadge(cp.capturedAt && new Date(cp.capturedAt).getUTCMonth() !== new Date().getUTCMonth()) +
      '<div class="quota-minibar"><div class="quota-minibar-fill ' + premiumCls + '" style="width:' + premiumRem + '%"></div></div></div>' +
      '<div class="quota-cell"><span class="quota-pct ' + chatCls + '">' + chatRem.toFixed(0) + '% left</span>' +
      renderProviderEstBadge(cp.capturedAt && new Date(cp.capturedAt).getUTCMonth() !== new Date().getUTCMonth()) +
      '<div class="quota-minibar"><div class="quota-minibar-fill ' + chatCls + '" style="width:' + chatRem + '%"></div></div></div>' +
      '<div class="snap-cell"><span class="snap-ago">' + capturedAgo + '</span></div>' +
      '<div class="status-cell"><span class="health-dot ' + dotCls + '">\u25cf ' + dotText + '</span></div>' +
      '</div>' +
      actionsHTML +
      '</div>';
  }

  if (renderedCount === 0) return '';
  html += '</div></div>';
  return html;
}
