// Niyantra Dashboard — Overview Tab Renderer
import { serverConfig, latestQuotaData } from '../core/state';
import { esc, formatTimeAgo, formatDurationSec } from '../core/utils';
import { fetchOverview, fetchSubscriptions, fetchUsage } from '../core/api';
import { openBudgetModal } from './budget';
import { renderServerInsights, loadAdvisorCard } from './insights';
import { loadCostKPI } from './cost';
import { loadHeatmap } from './heatmap';
import { renderRenewalCalendar } from './calendar';
import { formatResetTime } from '../quotas/render';
import { renderClaudeCodeCard, loadClaudeCardData, loadClaudeDeepUsage } from '../advanced/claude';
import { renderSessionsTimeline } from '../advanced/codex';
import { loadTokenAnalytics } from './tokenAnalytics';
import { loadGitCosts } from './gitCosts';
import { renderSafeToSpend, wireSafeToSpendButtons } from './safeToSpend';
import { renderStreakCard } from './streaks';
import { renderCountdowns, startCountdownRefresh } from './countdown';
import { loadAnomalies } from './anomalyCard';
import { downloadReport } from '../advanced/report';

export function loadOverview(): void {
  Promise.all([fetchOverview(), fetchSubscriptions('', ''), fetchUsage()]).then(function(results) {
    var data = results[0];
    var subsData = results[1] as any;
    var usageData = results[2];
    renderOverviewEnhanced(data, subsData.subscriptions || subsData || [], usageData);
  }).catch(function(err) {
    console.error('Failed to load overview:', err);
  });
}

export function renderOverviewEnhanced(data: any, subs: any[], usageData: any): void {
  var el = document.getElementById('overview-content');
  if (!el) return;

  var stats = data.stats || { totalMonthlySpend: 0, totalAnnualSpend: 0, byCategory: {}, byStatus: {} };
  var renewals = data.renewals || [];
  var links = data.quickLinks || [];
  var quotas = data.quotaSummary;
  var serverInsights = data.insights || [];

  var advisorHTML = '<div id="advisor-card-container"></div>';
  var insightsHTML = renderServerInsights(serverInsights);

  var safeToSpendHTML = renderSafeToSpend(
    usageData && usageData.budgetForecast ? usageData.budgetForecast : null,
    serverConfig['currency'] || 'USD'
  );

  var countdownContent = renderCountdowns(latestQuotaData);
  var countdownHTML = countdownContent ? '<div id="countdown-container" style="grid-column:1/-1">' + countdownContent + '</div>' : '';
  if (latestQuotaData) startCountdownRefresh(latestQuotaData);

  var cats = Object.keys(stats.byCategory);

  var spendHTML = '<div class="overview-card">' +
    '<h3>Monthly AI Spend</h3>' +
    '<div class="kpi-with-sparkline">' +
    '<div class="overview-big-number">$' + stats.totalMonthlySpend.toFixed(2) + '</div>' +
    '</div>';

  if (cats.length > 1) {
    cats.sort(function(a, b) {
      return (stats.byCategory[b].monthlySpend || 0) - (stats.byCategory[a].monthlySpend || 0);
    });
    for (var i = 0; i < cats.length; i++) {
      var c = stats.byCategory[cats[i]];
      spendHTML += '<div class="overview-category-row">' +
        '<span class="overview-category-name">' + esc(cats[i]) + '<span class="overview-category-count">' + c.count + ' subs</span></span>' +
        '<span class="overview-category-spend">$' + c.monthlySpend.toFixed(2) + '/mo</span>' +
        '</div>';
    }
  } else if (cats.length === 1) {
    var onlyCat = stats.byCategory[cats[0]];
    spendHTML += '<div class="overview-big-label">' + onlyCat.count + ' ' + cats[0] + ' subscription' + (onlyCat.count !== 1 ? 's' : '') + '</div>';
  }
  spendHTML += '</div>';

  var claudeHTML = renderClaudeCodeCard();

  var calendarHTML = '';
  if (renewals.length > 0) {
    calendarHTML = '<div id="renewal-calendar-container" class="overview-card full-width"></div>';
  }

  var linksHTML = '';
  if (links.length > 0) {
    if (links.length > 1 || (links.length === 1 && links[0].platform !== 'Antigravity')) {
      linksHTML = '<div class="overview-card full-width"><h3>Quick Links</h3>' +
        '<div class="quick-links-grid">';
      for (var pk = 0; pk < links.length; pk++) {
        var pl = links[pk];
        linksHTML += '<a class="quick-link" href="' + esc(pl.url) + '" target="_blank" rel="noopener">' +
          '🔗 ' + esc(pl.platform) + '</a>';
      }
      linksHTML += '</div></div>';
    }
  }

  var exportHTML = '<div class="overview-card full-width"><h3>Export</h3>' +
    '<p style="font-size:13px;color:var(--text-secondary);margin-bottom:12px">' +
    'Download a redacted JSON report or a full database backup.</p>' +
    '<div style="display:flex;gap:8px;flex-wrap:wrap">' +
    '<a class="btn-add" href="/api/export/csv" download style="text-decoration:none;display:inline-flex;padding:6px 12px;font-size:12px">📥 CSV</a>' +
    '<a class="btn-add" href="/api/export/json" download style="text-decoration:none;display:inline-flex;padding:6px 12px;font-size:12px">📦 Redacted JSON</a>' +
    '<a class="btn-add" href="/api/backup" download style="text-decoration:none;display:inline-flex;padding:6px 12px;font-size:12px">💾 DB Backup</a>' +
    '<button class="btn-add" id="generate-report-btn" style="padding:6px 12px;font-size:12px">📊 Monthly Report</button>' +
    '</div></div>';

  var providerHTML = '<div class="overview-card full-width"><h3>Provider Health</h3>';
  providerHTML += '<div class="provider-health-grid">';

  if (latestQuotaData && latestQuotaData.accounts && latestQuotaData.accounts.length > 0) {
    var accts = latestQuotaData.accounts;
    var readyCount = 0;
    for (var ai = 0; ai < accts.length; ai++) {
      if (accts[ai].isReady) readyCount++;
    }
    var healthPct = Math.round((readyCount / accts.length) * 100);
    var healthCls = healthPct >= 80 ? 'health-good' : healthPct >= 50 ? 'health-warn' : 'health-bad';
    providerHTML += '<div class="provider-health-row">' +
      '<span class="ph-name">⚡ Antigravity</span>' +
      '<span class="ph-count">' + accts.length + ' accounts</span>' +
      '<span class="ph-bar"><span class="ph-fill ' + healthCls + '" style="width:' + healthPct + '%"></span></span>' +
      '<span class="ph-stat ' + healthCls + '">' + readyCount + '/' + accts.length + ' ready</span>' +
      '</div>';
  }

  if (latestQuotaData && latestQuotaData.codexSnapshot) {
    var cs = latestQuotaData.codexSnapshot as any;
    var codexUsed = Math.max(cs.fiveHourPct || 0, cs.sevenDayPct || 0, cs.codeReviewPct || 0);
    var cxStatus = usageHealthClass(codexUsed);
    var cxLabel = cs.email || 'Codex account';
    providerHTML += '<div class="provider-health-row">' +
      '<span class="ph-name">🤖 Codex</span>' +
      '<span class="ph-count">' + esc(cxLabel) + '</span>' +
      '<span class="ph-bar"><span class="ph-fill ' + cxStatus + '" style="width:' + Math.max(0, 100 - codexUsed) + '%"></span></span>' +
      '<span class="ph-stat ' + cxStatus + '">' + esc(cs.planType || 'free') + '</span>' +
      '</div>';
  }

  if (latestQuotaData && latestQuotaData.claudeSnapshot) {
    var claude = latestQuotaData.claudeSnapshot as any;
    var claudeUsed = Math.max(claude.fiveHourPct || 0, claude.sevenDayPct || 0);
    var clStatus = usageHealthClass(claudeUsed);
    providerHTML += '<div class="provider-health-row">' +
      '<span class="ph-name">🔮 Claude Code</span>' +
      '<span class="ph-count">Bridge</span>' +
      '<span class="ph-bar"><span class="ph-fill ' + clStatus + '" style="width:' + Math.max(0, 100 - claudeUsed) + '%"></span></span>' +
      '<span class="ph-stat ' + clStatus + '">' + formatUsageSummary(claudeUsed) + '</span>' +
      '</div>';
  }

  if (latestQuotaData && latestQuotaData.cursorSnapshot) {
    var cursor = latestQuotaData.cursorSnapshot as any;
    var cursorStatus = usageHealthClass(cursor.usagePct || 0);
    providerHTML += '<div class="provider-health-row">' +
      '<span class="ph-name">🖱️ Cursor</span>' +
      '<span class="ph-count">' + esc(cursor.email || cursor.billingModel || 'Cursor') + '</span>' +
      '<span class="ph-bar"><span class="ph-fill ' + cursorStatus + '" style="width:' + Math.max(0, 100 - (cursor.usagePct || 0)) + '%"></span></span>' +
      '<span class="ph-stat ' + cursorStatus + '">' + esc(cursor.planTier || 'unknown') + '</span>' +
      '</div>';
  }

  if (latestQuotaData && latestQuotaData.geminiSnapshot) {
    var gemini = latestQuotaData.geminiSnapshot as any;
    var geminiStatus = usageHealthClass(gemini.overallPct || 0);
    providerHTML += '<div class="provider-health-row">' +
      '<span class="ph-name">✨ Gemini CLI</span>' +
      '<span class="ph-count">' + esc(gemini.email || gemini.projectId || 'Gemini') + '</span>' +
      '<span class="ph-bar"><span class="ph-fill ' + geminiStatus + '" style="width:' + Math.max(0, 100 - (gemini.overallPct || 0)) + '%"></span></span>' +
      '<span class="ph-stat ' + geminiStatus + '">' + esc(gemini.tier || 'unknown') + '</span>' +
      '</div>';
  }

  if (latestQuotaData && latestQuotaData.copilotSnapshot) {
    var copilot = latestQuotaData.copilotSnapshot as any;
    var copilotUsed = copilot.hasPremium ? (copilot.premiumPct || 0) : (copilot.chatPct || 0);
    var copilotStatus = usageHealthClass(copilotUsed);
    providerHTML += '<div class="provider-health-row">' +
      '<span class="ph-name">🐙 Copilot</span>' +
      '<span class="ph-count">' + esc(copilot.email || copilot.username || 'Copilot') + '</span>' +
      '<span class="ph-bar"><span class="ph-fill ' + copilotStatus + '" style="width:' + Math.max(0, 100 - copilotUsed) + '%"></span></span>' +
      '<span class="ph-stat ' + copilotStatus + '">' + esc(copilot.plan || 'unknown') + '</span>' +
      '</div>';
  }

  if (latestQuotaData && latestQuotaData.pluginSnapshots) {
    for (var psi = 0; psi < latestQuotaData.pluginSnapshots.length; psi++) {
      var pluginSnap = latestQuotaData.pluginSnapshots[psi] as any;
      var pluginStatus = usageHealthClass(pluginSnap.usagePct || 0);
      providerHTML += '<div class="provider-health-row">' +
        '<span class="ph-name">🧩 ' + esc(pluginSnap.provider || pluginSnap.pluginId || 'Plugin') + '</span>' +
        '<span class="ph-count">' + esc(pluginSnap.label || pluginSnap.email || pluginSnap.pluginId || 'Plugin source') + '</span>' +
        '<span class="ph-bar"><span class="ph-fill ' + pluginStatus + '" style="width:' + Math.max(0, 100 - (pluginSnap.usagePct || 0)) + '%"></span></span>' +
        '<span class="ph-stat ' + pluginStatus + '">' + esc(pluginSnap.plan || formatUsageSummary(pluginSnap.usagePct || 0)) + '</span>' +
        '</div>';
    }
  }
  providerHTML += '</div></div>';

  var costKPIHTML = '<div id="cost-kpi-container"></div>';
  var tokenAnalyticsHTML = '<div id="token-analytics-container" class="overview-card full-width"></div>';
  var gitCostsHTML = '<div id="git-costs-container" class="overview-card full-width"></div>';
  var heatmapHTML = '<div id="heatmap-container" class="overview-card full-width"></div>';
  var anomalyHTML = '<div id="anomaly-card-container"></div>';

  el.innerHTML = safeToSpendHTML + anomalyHTML + countdownHTML + advisorHTML + costKPIHTML + tokenAnalyticsHTML + gitCostsHTML + heatmapHTML + providerHTML + insightsHTML + claudeHTML + spendHTML + calendarHTML + linksHTML + exportHTML;

  wireSafeToSpendButtons(openBudgetModal);
  loadAnomalies();

  var reportBtn = document.getElementById('generate-report-btn');
  if (reportBtn) {
    reportBtn.addEventListener('click', function() { downloadReport(); });
  }

  if (serverConfig['claude_bridge'] === 'true') {
    loadClaudeCardData();
  } else {
    var cardBody = document.getElementById('claude-card-body');
    if (cardBody) cardBody.innerHTML = '';
  }

  loadClaudeDeepUsage();
  loadAdvisorCard();
  loadCostKPI();
  loadHeatmap();
  loadTokenAnalytics();
  loadGitCosts();

  if (renewals.length > 0) {
    renderRenewalCalendar(renewals, subs);
  }

  renderSessionsTimeline(el);
}

function usageHealthClass(usedPct: number): string {
  if (usedPct >= 80) return 'health-bad';
  if (usedPct >= 50) return 'health-warn';
  return 'health-good';
}

function formatUsageSummary(usedPct: number): string {
  return Math.max(0, 100 - usedPct).toFixed(0) + '% left';
}
