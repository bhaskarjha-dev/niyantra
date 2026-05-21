// Niyantra Dashboard — Overview Tab Renderer
import { serverConfig, latestQuotaData } from '../core/state';
import { esc, formatTimeAgo, formatDurationSec, showToast } from '../core/utils';
import { downloadAPIFile, downloadBackup, fetchOverview, fetchSubscriptions, fetchUsage, claimOverageBonus, fetchStatus } from '../core/api';
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
    '<h3>Monthly Recurring Spend</h3>' +
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
    '<button class="btn-add" id="download-csv-btn" style="padding:6px 12px;font-size:12px">📥 CSV</button>' +
    '<button class="btn-add" id="download-json-btn" style="padding:6px 12px;font-size:12px">📦 Redacted JSON</button>' +
    '<button class="btn-add" id="download-backup-btn" style="padding:6px 12px;font-size:12px">💾 DB Backup</button>' +
    '<button class="btn-add" id="generate-report-btn" style="padding:6px 12px;font-size:12px">📊 Monthly Report</button>' +
    '</div></div>';

  var providerHTML = '';

  var costKPIHTML = '<div id="cost-kpi-container"></div>';
  var tokenAnalyticsHTML = '<div id="token-analytics-container" class="overview-card full-width"></div>';
  var gitCostsHTML = '<div id="git-costs-container" class="overview-card full-width"></div>';
  var heatmapHTML = '<div id="heatmap-container" class="overview-card full-width"></div>';

  var eligibleAccount = null;
  if (latestQuotaData && latestQuotaData.accounts) {
    eligibleAccount = latestQuotaData.accounts.find(function(acc: any) {
      return acc.planTier && acc.planTier.toLowerCase() === 'ultra' && acc.hasClaimedBonus2026 === 0;
    });
  }
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

  el.innerHTML = bannerHTML + safeToSpendHTML + countdownHTML + advisorHTML + costKPIHTML + tokenAnalyticsHTML + gitCostsHTML + heatmapHTML + providerHTML + insightsHTML + claudeHTML + spendHTML + calendarHTML + linksHTML + exportHTML;

  wireSafeToSpendButtons(openBudgetModal);

  var claimBtn = el.querySelector('.io-claim-btn');
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

  var reportBtn = document.getElementById('generate-report-btn');
  if (reportBtn) {
    reportBtn.addEventListener('click', function() { downloadReport(); });
  }
  var backupBtn = document.getElementById('download-backup-btn');
  if (backupBtn) {
    backupBtn.addEventListener('click', function() {
      downloadBackup().catch(function(err) {
        alert(err.message || 'Backup failed');
      });
    });
  }
  var csvBtn = document.getElementById('download-csv-btn');
  if (csvBtn) {
    csvBtn.addEventListener('click', function() {
      downloadAPIFile('/api/export/csv', 'niyantra-export.csv').catch(function(err) {
        alert(err.message || 'CSV export failed');
      });
    });
  }
  var jsonBtn = document.getElementById('download-json-btn');
  if (jsonBtn) {
    jsonBtn.addEventListener('click', function() {
      downloadAPIFile('/api/export/json', 'niyantra-export.json').catch(function(err) {
        alert(err.message || 'JSON export failed');
      });
    });
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
