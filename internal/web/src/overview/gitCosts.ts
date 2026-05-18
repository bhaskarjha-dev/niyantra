// Heuristic git-to-AI attribution.

export function loadGitCosts(): void {
  var container = document.getElementById('git-costs-container');
  if (!container) return;

  fetch('/api/git-costs?days=30').then(function(res) {
    return res.json().then(function(data) {
      if (!res.ok) {
        throw new Error(data.error || 'Failed to load git costs');
      }
      return data;
    });
  }).then(function(data: any) {
    renderGitCosts(container!, data);
  }).catch(function(err) {
    console.error('Git costs fetch failed:', err);
    renderGitCostsError(container!, err instanceof Error ? err.message : 'Failed to load git costs');
  });
}

function renderGitCostsError(container: HTMLElement, message: string): void {
  container.innerHTML = '<div class="overview-card full-width git-costs-card">' +
    '<h3>Git Activity Attribution</h3>' +
    '<div class="git-costs-empty">' +
    '<p>Unable to analyze git activity attribution.</p>' +
    '<p style="font-size:12px;color:var(--text-secondary)">' + escapeHtml(message) + '</p>' +
    '</div></div>';
}

function renderGitCosts(container: HTMLElement, data: any): void {
  if (!data || !data.commits || data.commits.length === 0) {
    container.innerHTML = '<div class="overview-card full-width git-costs-card">' +
      '<h3>Git Activity Attribution</h3>' +
      '<div class="git-costs-empty">' +
      '<p>No git commit data available.</p>' +
      '<p style="font-size:12px;color:var(--text-secondary)">Ensure you are running Niyantra from within a git repository, ' +
      'or pass <code>?repo=/path</code> to the API.</p>' +
      '</div></div>';
    return;
  }

  var totals = data.totals || {};
  var commits = data.commits || [];
  var branches = data.branches || [];
  var hasAICosts = totals.totalTokens > 0;

  var kpiHTML = '<div class="git-kpi-row">';
  kpiHTML += buildKpi('Commits', String(totals.commitCount || 0), 'Commits');
  kpiHTML += buildKpi('Nearby AI Estimate', '$' + (totals.costUSD || 0).toFixed(2), 'Estimate');
  kpiHTML += buildKpi('Avg/Commit Est.', '$' + (totals.avgPerCommit || 0).toFixed(2), 'Avg');
  kpiHTML += buildKpi('Top Branch', truncate(totals.topBranch || '-', 18), 'Branch');
  kpiHTML += '</div>';
  kpiHTML += '<p style="font-size:12px;color:var(--text-muted);margin:0 0 12px">' +
    escapeHtml(data.notAccountingGradeReason || 'Claude token events are heuristically assigned to nearby commits. This is guidance, not ground truth.') +
    '</p>';

  if (!hasAICosts) {
    kpiHTML += '<div class="git-no-ai-banner">No nearby Claude Code session data was attributable inside the commit lookback windows.</div>';
  }

  var chartHTML = '';
  if (commits.length > 0 && hasAICosts) {
    chartHTML = '<div class="git-section">';
    chartHTML += '<h4>Estimated Nearby Cost per Commit</h4>';
    chartHTML += '<div class="git-commit-chart">';

    var maxCost = 0;
    for (var ci = 0; ci < commits.length; ci++) {
      if (commits[ci].costUSD > maxCost) maxCost = commits[ci].costUSD;
    }

    var displayCommits = commits.length > 40 ? commits.slice(0, 40) : commits;
    for (var di = 0; di < displayCommits.length; di++) {
      var c = displayCommits[di];
      var barH = maxCost > 0 ? Math.max(3, (c.costUSD / maxCost) * 100) : 3;
      var barColor = c.costUSD > 0 ? 'var(--accent)' : 'var(--border)';
      chartHTML += '<div class="git-bar-col" title="' + escapeAttr(c.shortHash) + ': ' + escapeAttr(c.message) + '\n$' + c.costUSD.toFixed(2) + ' | ' + formatTokens(c.totalTokens) + ' tokens">' +
        '<div class="git-bar" style="height:' + barH + '%;background:' + barColor + '"></div>' +
        '<span class="git-bar-hash">' + c.shortHash + '</span>' +
        '</div>';
    }

    chartHTML += '</div></div>';
  }

  var branchHTML = '';
  if (branches.length > 0 && hasAICosts) {
    branchHTML = '<div class="git-section">';
    branchHTML += '<h4>Branch Estimates</h4>';
    branchHTML += '<div class="git-branch-table">';
    branchHTML += '<div class="git-branch-header">' +
      '<span>Branch</span><span>Commits</span><span>Tokens</span><span>Cost</span><span>Avg</span>' +
      '</div>';

    var displayBranches = branches.slice(0, 10);
    for (var bi = 0; bi < displayBranches.length; bi++) {
      var b = displayBranches[bi];
      if (b.costUSD === 0 && b.totalTokens === 0) continue;
      branchHTML += '<div class="git-branch-row">' +
        '<span class="git-branch-name">' + escapeHtml(truncate(b.name, 30)) + '</span>' +
        '<span class="git-branch-val">' + b.commits + '</span>' +
        '<span class="git-branch-val">' + formatTokens(b.totalTokens) + '</span>' +
        '<span class="git-branch-cost">$' + b.costUSD.toFixed(2) + '</span>' +
        '<span class="git-branch-val">$' + b.avgPerCommit.toFixed(2) + '</span>' +
        '</div>';
    }
    branchHTML += '</div></div>';
  }

  var commitsHTML = '<div class="git-section">';
  commitsHTML += '<h4>Recent Commits</h4>';
  commitsHTML += '<div class="git-commits-list">';

  var showCommits = commits.slice(0, 15);
  for (var ri = 0; ri < showCommits.length; ri++) {
    var rc = showCommits[ri];
    var costBadge = rc.costUSD > 0
      ? '<span class="git-cost-badge">$' + rc.costUSD.toFixed(2) + '</span>'
      : '<span class="git-cost-badge git-cost-zero">-</span>';
    var tokenBadge = rc.totalTokens > 0
      ? '<span class="git-token-badge">' + formatTokens(rc.totalTokens) + '</span>'
      : '';

    commitsHTML += '<div class="git-commit-item">' +
      '<span class="git-commit-hash">' + rc.shortHash + '</span>' +
      '<span class="git-commit-msg">' + escapeHtml(rc.message) + '</span>' +
      '<div class="git-commit-meta">' + tokenBadge + costBadge + '</div>' +
      '</div>';
  }

  commitsHTML += '</div></div>';

  container.innerHTML = '<div class="overview-card full-width git-costs-card">' +
    '<div class="git-costs-header">' +
    '<h3>Git Activity Attribution</h3>' +
    '<span class="git-repo-path" title="' + escapeAttr(data.repoPath || '') + '">' +
    escapeHtml(shortenPath(data.repoPath || '')) + '</span>' +
    '</div>' +
    kpiHTML + chartHTML + branchHTML + commitsHTML +
    '</div>';
}

function buildKpi(label: string, value: string, icon: string): string {
  return '<div class="git-kpi-card">' +
    '<div class="git-kpi-icon">' + icon + '</div>' +
    '<div class="git-kpi-value">' + value + '</div>' +
    '<div class="git-kpi-label">' + label + '</div>' +
    '</div>';
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return String(n);
}

function truncate(s: string, max: number): string {
  return s.length > max ? s.substring(0, max - 1) + '...' : s;
}

function shortenPath(p: string): string {
  var parts = p.replace(/\\/g, '/').split('/');
  return parts.length > 2 ? '.../' + parts.slice(-2).join('/') : p;
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}
