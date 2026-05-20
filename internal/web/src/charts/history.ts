// Niyantra Dashboard — Quota History Chart
import { loadChartAnnotations } from './annotations';
var Chart = (window as any).Chart;


export var historyChart: any = null;

// M2: Update chart colors in-place on theme toggle (avoids destroy+rebuild flash)
export function updateChartTheme(theme: string): void {
  if (!historyChart) return;
  var isDark = theme !== 'light';
  var gridColor = isDark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.06)';
  var textColor = isDark ? '#94a3b8' : '#64748b';
  if (historyChart.options.scales && historyChart.options.scales.y) {
    historyChart.options.scales.y.grid.color = gridColor;
    historyChart.options.scales.y.ticks.color = textColor;
  }
  if (historyChart.options.scales && historyChart.options.scales.x) {
    historyChart.options.scales.x.grid.color = gridColor;
    historyChart.options.scales.x.ticks.color = textColor;
  }
  historyChart.update('none'); // 'none' = no animation, instant repaint
}

export var chartAccountsList: any[] = [];

export function loadHistoryChart(): void {
  if (typeof Chart === 'undefined') return; // CDN not loaded (offline)

  var provSel = document.getElementById('chart-provider') as HTMLSelectElement;
  var accSel = document.getElementById('chart-account') as HTMLSelectElement;
  var rangeSel = document.getElementById('chart-range') as HTMLSelectElement;

  var provider = provSel ? provSel.value : 'all';
  var accountId = accSel ? parseInt(accSel.value) || 0 : 0;
  var range = rangeSel ? rangeSel.value : '7d';

  // Toggle custom date range inputs visibility
  var customContainer = document.getElementById('chart-custom-range');
  if (customContainer) {
    if (range === 'custom') {
      customContainer.style.display = 'flex';
    } else {
      customContainer.style.display = 'none';
    }
  }

  var since = '';
  var until = '';

  var now = new Date();
  if (range === '24h') {
    since = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
  } else if (range === '7d') {
    since = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString();
  } else if (range === '30d') {
    since = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString();
  } else if (range === '90d') {
    since = new Date(now.getTime() - 90 * 24 * 60 * 60 * 1000).toISOString();
  } else if (range === 'custom') {
    var startInput = document.getElementById('chart-start-date') as HTMLInputElement;
    var endInput = document.getElementById('chart-end-date') as HTMLInputElement;
    
    // Set default custom dates if not already set (7 days ago to today)
    if (startInput && !startInput.value) {
      var dStart = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000);
      startInput.value = dStart.toISOString().split('T')[0];
    }
    if (endInput && !endInput.value) {
      endInput.value = new Date().toISOString().split('T')[0];
    }

    if (startInput && startInput.value) {
      since = new Date(startInput.value + 'T00:00:00').toISOString();
    }
    if (endInput && endInput.value) {
      until = new Date(endInput.value + 'T23:59:59').toISOString();
    }
  }

  var url = '/api/history?limit=1000';
  if (provider !== 'all') url += '&provider=' + provider;
  if (accountId > 0) url += '&account=' + accountId;
  if (since) url += '&since=' + encodeURIComponent(since);
  if (until) url += '&until=' + encodeURIComponent(until);

  fetch(url).then(function(res) { return res.json(); }).then(function(data) {
    var snapshots = data.snapshots || [];
    updateKPINumbers(snapshots);
    renderHistoryChart(snapshots);
  }).catch(function(err) {
    console.error('Failed to load history:', err);
  });
}

export function renderHistoryChart(snapshots: any[]): void {
  var container = document.querySelector('.chart-container');
  if (!container || typeof Chart === 'undefined') return;

  if (snapshots.length === 0) {
    container.innerHTML = '<div class="chart-empty">' +
      '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" opacity="0.4">' +
      '<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 17v-5M15 17V7M12 17v-3"/></svg>' +
      'No snapshot history found matching current filters.' +
      '</div>';
    return;
  }
  container.innerHTML = '<canvas id="history-chart"></canvas>';

  // Reverse so oldest is first (left-to-right timeline)
  snapshots = snapshots.slice().reverse();

  var labels = snapshots.map(function(s) {
    var d = new Date(s.capturedAt);
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) +
      ' ' + d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  });

  // Build datasets per group
  var groupData: Record<string, (number|null)[]> = {};
  var groupNames: Record<string, string> = {
    claude_gpt: 'Claude + GPT',
    gemini_unified: 'Gemini Pool',
    cursor: 'Cursor Quota',
    codex_5h: 'Codex 5-Hour',
    codex_7d: 'Codex 7-Day',
    copilot: 'Copilot Premium',
    unknown: 'Unknown'
  };
  var groupColors: Record<string, string> = {
    claude_gpt: '#D97757',
    gemini_unified: '#3B82F6',
    cursor: '#00E6FF',
    codex_5h: '#9B51E0',
    codex_7d: '#BB6BD9',
    copilot: '#2EA44F',
    unknown: '#64748B'
  };

  for (var i = 0; i < snapshots.length; i++) {
    var groups = snapshots[i].groups || [];
    for (var j = 0; j < groups.length; j++) {
      var g = groups[j];
      if (!(groupData as any)[g.groupKey]) (groupData as any)[g.groupKey] = [];
    }
  }

  var aiCreditsData = [];
  var hasAICredits = false;

  for (var i = 0; i < snapshots.length; i++) {
    var snap = snapshots[i];
    var groups = snap.groups || [];
    var seen: Record<string, boolean> = {};
    for (var j = 0; j < groups.length; j++) {
      var g = groups[j];
      if (!groupData[g.groupKey]) groupData[g.groupKey] = [];
      groupData[g.groupKey].push(Math.round(g.remainingPercent || 0));
      seen[g.groupKey] = true;
    }
    // Fill nulls for missing groups
    var keys = Object.keys(groupData);
    for (var k = 0; k < keys.length; k++) {
      if (!seen[keys[k]]) groupData[keys[k]].push(null);
    }

    // Capture AI credits
    if (snap.aiCredits && snap.aiCredits.length > 0) {
      aiCreditsData.push(snap.aiCredits[0].creditAmount);
      hasAICredits = true;
    } else {
      aiCreditsData.push(null);
    }
  }

  var datasets = [];
  var keys = Object.keys(groupData);
  for (var k = 0; k < keys.length; k++) {
    var key = keys[k];
    if (!key || !groupNames[key]) continue; // Skip unknown/empty groups
    datasets.push({
      label: groupNames[key],
      data: groupData[key],
      borderColor: groupColors[key] || '#94a3b8',
      backgroundColor: (groupColors[key] || '#94a3b8') + '15',
      yAxisID: 'y',
      fill: true,
      tension: 0.35,
      pointRadius: 0,
      pointHoverRadius: 6,
      pointHoverBorderWidth: 2,
      pointHoverBackgroundColor: groupColors[key] || '#94a3b8',
      pointHoverBorderColor: '#ffffff',
      borderWidth: 2.5,
    });
  }

  if (hasAICredits) {
    datasets.push({
      label: 'AI Credits',
      data: aiCreditsData,
      borderColor: '#fbbf24', // Amber
      backgroundColor: 'transparent',
      yAxisID: 'yCredits',
      borderDash: [6, 4],
      tension: 0.35,
      pointRadius: 0,
      pointHoverRadius: 6,
      pointHoverBorderWidth: 2,
      pointHoverBackgroundColor: '#fbbf24',
      pointHoverBorderColor: '#ffffff',
      borderWidth: 2.5,
    });
  }

  // Determine theme for chart
  var isDark = document.documentElement.getAttribute('data-theme') !== 'light';
  var gridColor = isDark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.06)';
  var textColor = isDark ? '#94a3b8' : '#64748b';

  if (historyChart) historyChart.destroy();

  var ctx = document.getElementById('history-chart');
  if (!ctx) return;

  historyChart = new Chart(ctx, {
    type: 'line',
    data: { labels: labels, datasets: datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: {
          position: 'bottom',
          labels: {
            color: textColor,
            font: { family: "'Inter', sans-serif", size: 11, weight: '500' },
            boxWidth: 8,
            boxHeight: 8,
            usePointStyle: true,
            pointStyle: 'circle',
            padding: 20
          }
        },
        tooltip: {
          backgroundColor: isDark ? 'rgba(15, 23, 42, 0.95)' : 'rgba(255, 255, 255, 0.95)',
          titleColor: isDark ? '#f8fafc' : '#0f172a',
          bodyColor: isDark ? '#cbd5e1' : '#475569',
          borderColor: isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.08)',
          borderWidth: 1,
          padding: 12,
          cornerRadius: 8,
          titleFont: { family: "'Inter', sans-serif", weight: '700', size: 12 },
          bodyFont: { family: "'Inter', sans-serif", size: 11 },
          multiKeyBackground: 'transparent',
          usePointStyle: true,
          boxWidth: 6,
          boxHeight: 6,
          boxPadding: 6,
          callbacks: {
            label: function(ctx: any) {
              if (ctx.dataset.yAxisID === 'yCredits') return ctx.dataset.label + ': ' + ctx.parsed.y.toLocaleString();
              return ctx.dataset.label + ': ' + ctx.parsed.y + '%';
            }
          }
        }
      },
      scales: {
        y: {
          type: 'linear',
          display: true,
          position: 'left',
          min: 0, max: 100,
          grid: { color: gridColor, drawTicks: false },
          ticks: { color: textColor, font: { family: "'Inter', sans-serif", size: 10 }, padding: 8, callback: function(v: any) { return v + '%'; } },
          border: { display: false }
        },
        yCredits: {
          type: 'linear',
          display: hasAICredits,
          position: 'right',
          grid: { display: false },
          ticks: { color: isDark ? '#fbbf24' : '#d97706', font: { family: "'Inter', sans-serif", size: 10 }, padding: 8 },
          border: { display: false }
        },
        x: {
          grid: { display: false },
          ticks: { color: textColor, font: { family: "'Inter', sans-serif", size: 9 }, maxRotation: 0, maxTicksLimit: 8, padding: 8 },
          border: { display: false }
        }
      }
    }
  });

  // F19-UX: Load chart annotations (event markers)
  var chartEl = document.querySelector('.chart-container') as HTMLElement;
  if (chartEl && historyChart) {
    // Pass raw timestamps for matching
    var rawTimestamps = snapshots.map(function(s: any) { return s.capturedAt; });
    loadChartAnnotations(chartEl, rawTimestamps, historyChart);
  }
}

export function populateChartAccountSelect(data: any): void {
  var accts = data.allAccounts || data.accounts;
  if (!accts) return;
  chartAccountsList = accts;
  filterChartAccounts();
}

export function filterChartAccounts(): void {
  var provSel = document.getElementById('chart-provider') as HTMLSelectElement;
  var accSel = document.getElementById('chart-account') as HTMLSelectElement;
  if (!accSel) return;

  var selectedProvider = provSel ? provSel.value : 'all';
  var currentlySelectedValue = accSel.value;

  // Keep "All Accounts" option, remove others
  while (accSel.options.length > 1) accSel.remove(1);

  for (var i = 0; i < chartAccountsList.length; i++) {
    var acc = chartAccountsList[i];
    // Filter by provider if a specific one is selected
    if (selectedProvider !== 'all' && acc.provider !== selectedProvider) {
      continue;
    }
    var opt = document.createElement('option');
    var id = acc.id !== undefined ? acc.id : acc.accountId;
    opt.value = id;
    opt.textContent = acc.email;
    accSel.appendChild(opt);
  }

  // Try to preserve previous selection if it's still available in the filtered list
  var optionExists = false;
  for (var j = 0; j < accSel.options.length; j++) {
    if (accSel.options[j].value === currentlySelectedValue) {
      accSel.selectedIndex = j;
      optionExists = true;
      break;
    }
  }
  if (!optionExists) {
    accSel.value = '0'; // Default to "All Accounts"
  }
}

export function updateKPINumbers(snapshots: any[]): void {
  var totalEl = document.getElementById('kpi-total-snapshots');
  var avgEl = document.getElementById('kpi-avg-remaining');
  var trendEl = document.getElementById('kpi-trend');

  if (!totalEl || !avgEl || !trendEl) return;

  if (snapshots.length === 0) {
    totalEl.textContent = '0';
    avgEl.textContent = '-%';
    trendEl.textContent = 'No Data';
    trendEl.className = 'kpi-value';
    return;
  }

  // 1. Total Snapshots
  totalEl.textContent = snapshots.length.toString();

  // 2. Average Remaining
  var sum = 0;
  var count = 0;
  for (var i = 0; i < snapshots.length; i++) {
    var groups = snapshots[i].groups || [];
    for (var j = 0; j < groups.length; j++) {
      sum += groups[j].remainingPercent;
      count++;
    }
  }
  var avgRemaining = count > 0 ? Math.round(sum / count) : 0;
  avgEl.textContent = avgRemaining + '%';

  // 3. Trend Calculation
  if (snapshots.length < 2) {
    trendEl.textContent = 'Stable';
    trendEl.className = 'kpi-value trend-stable';
    return;
  }

  // Note: snapshots array is sorted newest first. Let's reverse a copy to analyze chronologically
  var chronological = snapshots.slice().reverse();
  var midpoint = Math.floor(chronological.length / 2);
  
  var firstHalfSum = 0, firstHalfCount = 0;
  var secondHalfSum = 0, secondHalfCount = 0;

  for (var i = 0; i < chronological.length; i++) {
    var groups = chronological[i].groups || [];
    for (var j = 0; j < groups.length; j++) {
      if (i < midpoint) {
        firstHalfSum += groups[j].remainingPercent;
        firstHalfCount++;
      } else {
        secondHalfSum += groups[j].remainingPercent;
        secondHalfCount++;
      }
    }
  }

  var oldestAvg = firstHalfCount > 0 ? firstHalfSum / firstHalfCount : avgRemaining;
  var newestAvg = secondHalfCount > 0 ? secondHalfSum / secondHalfCount : avgRemaining;

  var diff = newestAvg - oldestAvg;
  var trendCard = trendEl.closest('.kpi-card');
  if (diff > 1.5) {
    trendEl.innerHTML = 'Improving <span class="trend-icon">▲</span>';
    trendEl.className = 'kpi-value trend-up';
    if (trendCard) {
      trendCard.className = 'kpi-card kpi-trend-improving';
    }
  } else if (diff < -1.5) {
    trendEl.innerHTML = 'Declining <span class="trend-icon">▼</span>';
    trendEl.className = 'kpi-value trend-down';
    if (trendCard) {
      trendCard.className = 'kpi-card kpi-trend-declining';
    }
  } else {
    trendEl.innerHTML = 'Stable <span class="trend-icon">●</span>';
    trendEl.className = 'kpi-value trend-stable';
    if (trendCard) {
      trendCard.className = 'kpi-card kpi-trend-stable';
    }
  }
}

// ════════════════════════════════════════════
// ════════════════════════════════════════════
// ════════════════════════════════════════════
// ════════════════════════════════════════════
