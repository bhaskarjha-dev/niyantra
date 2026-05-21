// Niyantra Dashboard — Entry Point
// All core functionality is imported from modules.

import {
  GROUP_ORDER, GROUP_LABELS, GROUP_COLORS, GROUP_NAMES,
  expandedAccounts, collapsedProviders,
  presetsData, setPresetsData,
  activeTagFilter, setActiveTagFilter,
  usageDataCache, setUsageDataCache,
  quotaSortState, latestQuotaData, setLatestQuotaData,
  serverConfig, setServerConfig,
  snapInProgress, setSnapInProgress,
} from './core/state';

import {
  formatSeconds, formatCredits, formatNumber, currencySymbol,
  esc, showToast, updateTimestamp, refreshTimestampDisplay,
  formatTimeAgo, formatPollInterval, formatDurationSec,
} from './core/utils';

import {
  installAuthenticatedFetch,
  fetchStatus, triggerSnap,
  fetchSubscriptions, createSubscription, updateSubscription, deleteSubscription,
  fetchOverview, fetchPresets, fetchUsage, downloadBackup,
} from './core/api';

import { initTheme, initTabs } from './core/theme';

import {
  renderAccounts, filterAccountsArray, sortAccountsArray,
  updateSortHeaders, renderTagFilterStrip, handleTagFilterClick,
  getCodexClaudeStatus, formatResetTime, allExhausted,
} from './quotas/render';
import { setupToggle, initQuotas } from './quotas/expand';
import {
  renderAccountTags, renderAccountNote,
  renderCreditRenewal, updateAccountMeta, initAccountMetaHandlers,
  setRenderAccounts,
} from './quotas/features';

import { loadSubscriptions, initModal, initSearch } from './subscriptions';

import { getBudget, setBudget, getCurrency, updateConfig, loadConfig, initBudget, closeBudget, renderBudgetAlert } from './overview/budget';
import { loadOverview } from './overview/overview';

import { initSnapDropdown, handleSnap } from './advanced/snap';
import { updateChartTheme, loadHistoryChart, populateChartAccountSelect, filterChartAccounts } from './charts/history';
import { initSettings } from './settings/settings';
import { loadMode } from './settings/mode';
import { loadDataSources } from './settings/data';
import { loadActivityLog } from './settings/activity';
import { loadModelPricing } from './settings/pricing';
import { initKeyboardShortcuts } from './advanced/keyboard';
import { initCommandPalette } from './advanced/palette';
import { loadSystemAlerts } from './advanced/alerts';
import { emptyQuotas } from './core/emptyStates';

// ════════════════════════════════════════════
//  INIT
// ════════════════════════════════════════════


document.addEventListener('DOMContentLoaded', function() {
  installAuthenticatedFetch();
  initTheme();
  var isInitialTabChange = true;

  // Tab-change event: domain modules react to tab activation
  document.addEventListener('niyantra:tab-change', function(e) {
    var tab = (e as CustomEvent).detail.tab;
    if (tab === 'overview') {
      loadOverview();
    }
    if (tab === 'quotas') {
      if (!isInitialTabChange) {
        fetchStatus().then(function(data) {
          document.dispatchEvent(new CustomEvent('niyantra:status-refreshed', { detail: { data: data } }));
        }).catch(function() {});
      }
    }
    if (tab === 'subscriptions') {
      if (!isInitialTabChange) {
        loadSubscriptions();
      }
    }
    if (tab === 'settings') { loadActivityLog(); loadMode(); loadDataSources(); }
  });

  initTabs();
  isInitialTabChange = false;
  setRenderAccounts(renderAccounts);
  initQuotas();
  setupToggle();
  initModal();
  initBudget();
  initSettings();
  initSearch();
  initKeyboardShortcuts();
  initAccountMetaHandlers();

  // Global visibility change event listener for tab focus resilience
  document.addEventListener('visibilitychange', function() {
    if (document.visibilityState === 'visible') {
      var activeTab = document.querySelector('.tab-btn.active')?.getAttribute('data-tab');
      if (activeTab === 'quotas') {
        fetchStatus().then(function(data) {
          document.dispatchEvent(new CustomEvent('niyantra:status-refreshed', { detail: { data: data } }));
        }).catch(function() {});
      }
    }
  });

  // Status refreshed event: update UI and charts consistently
  document.addEventListener('niyantra:status-refreshed', function(e) {
    var data = (e as CustomEvent).detail.data;
    renderAccounts(data);
    populateChartAccountSelect(data);
    loadHistoryChart();
    updateTimestamp();

    if (localStorage.getItem('niyantra-active-tab') === 'overview') {
      document.dispatchEvent(new CustomEvent('niyantra:overview-refresh'));
    }
  });

  // Theme-change event: update chart colors
  document.addEventListener('niyantra:theme-change', function(e) {
    updateChartTheme((e as CustomEvent).detail.theme);
  });

  // Chart refresh: triggered by quota expand module after data changes
  document.addEventListener('niyantra:chart-refresh', function() {
    loadHistoryChart();
  });

  // Overview refresh: triggered by calendar navigation
  document.addEventListener('niyantra:overview-refresh', function() {
    loadOverview();
  });

  document.getElementById('snap-btn')!.addEventListener('click', handleSnap);
  var settingsBackupBtn = document.getElementById('settings-backup-btn');
  if (settingsBackupBtn) {
    settingsBackupBtn.addEventListener('click', function() {
      downloadBackup().catch(function(err) {
        alert(err.message || 'Backup failed');
      });
    });
  }
  initSnapDropdown();

  // Chart controls
  var chartProvider = document.getElementById('chart-provider');
  if (chartProvider) {
    chartProvider.addEventListener('change', function() {
      filterChartAccounts();
      loadHistoryChart();
    });
  }
  var chartAccount = document.getElementById('chart-account');
  if (chartAccount) {
    chartAccount.addEventListener('change', loadHistoryChart);
  }
  var chartRange = document.getElementById('chart-range');
  if (chartRange) {
    chartRange.addEventListener('change', loadHistoryChart);
  }
  var chartStart = document.getElementById('chart-start-date');
  if (chartStart) {
    chartStart.addEventListener('change', loadHistoryChart);
  }
  var chartEnd = document.getElementById('chart-end-date');
  if (chartEnd) {
    chartEnd.addEventListener('change', loadHistoryChart);
  }

  // Load quotas and usage intelligence
  Promise.all([fetchStatus(), fetchUsage()]).then(function(results) {
    var data = results[0];
    renderAccounts(data);
    updateTimestamp();
    populateChartAccountSelect(data);
    loadHistoryChart();

    // F3-UX: Show empty state if no accounts
    if (!data.accounts || data.accounts.length === 0) {
      var grid = document.getElementById('account-grid');
      if (grid) grid.innerHTML = emptyQuotas();
      // Wire up snap button in empty state
      var emptySnapBtn = document.getElementById('empty-snap-btn');
      if (emptySnapBtn) emptySnapBtn.addEventListener('click', handleSnap);
    }

    // Bug 1 fix: If no codex/claude data on first load, retry after 3s
    if (!data.codexSnapshot || !data.claudeSnapshot) {
      setTimeout(function() {
        fetchStatus().then(function(data2) {
          if (data2.codexSnapshot || data2.claudeSnapshot) {
            renderAccounts(data2);
          }
        }).catch(function() {});
      }, 3000);
    }
  }).catch(function(err) {
    console.error('Failed to load status:', err);
  });

  // Bug 3 fix: Pre-load subscriptions so tab is ready when first visited
  loadSubscriptions();

  // Load presets for the datalist
  fetchPresets().then(function(data) {
    setPresetsData(data.presets || []);
    var list = document.getElementById('preset-list');
    for (var i = 0; i < presetsData.length; i++) {
      var opt = document.createElement('option');
      opt.value = presetsData[i].platform;
      list!.appendChild(opt);
    }
  });

  // Load mode badge in header (manual/auto status)
  loadMode();

  // Init command palette
  initCommandPalette();

  // Phase 10: Load system alerts
  loadSystemAlerts();

  // Auto-capture polling is handled server-side by the agent.
  // Manual data refreshes on snap or page reload.

  // H2: Refresh relative timestamp every 30s
  setInterval(refreshTimestampDisplay, 30000);
});
