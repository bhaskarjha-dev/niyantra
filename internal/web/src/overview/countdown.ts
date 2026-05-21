// Niyantra Dashboard — Reset Countdown Timers (F6-UX)
// Shows countdown chips for providers whose quota resets within 24h.

export function renderCountdowns(quotaData: any): string {
  if (!quotaData) return '';

  var items: { provider: string; label: string; resetMs: number }[] = [];

  // Antigravity accounts: use resetTime from readiness data groups
  if (quotaData.accounts) {
    for (var i = 0; i < quotaData.accounts.length; i++) {
      var acc = quotaData.accounts[i];
      var soonestMs = Infinity;
      var hasReset = false;
      if (acc.groups) {
        for (var j = 0; j < acc.groups.length; j++) {
          var grp = acc.groups[j];
          if (grp.resetTime) {
            var ms = new Date(grp.resetTime).getTime() - Date.now();
            if (ms < soonestMs) {
              soonestMs = ms;
              hasReset = true;
            }
          }
        }
      }
      if (hasReset) {
        items.push({
          provider: '⚡ Antigravity',
          label: acc.email ? acc.email.split('@')[0] : 'account',
          resetMs: soonestMs,
        });
      }
    }
  }

  // Claude: 5h window reset
  if (quotaData.claudeSnapshot) {
    var cs = quotaData.claudeSnapshot;
    if (cs.capturedAt) {
      var fiveHReset = new Date(cs.capturedAt).getTime() + 5 * 3600000;
      var msLeft = fiveHReset - Date.now();
      items.push({ provider: '🔮 Claude', label: '5h window', resetMs: msLeft });
    }
  }

  // Codex: 7-day window
  if (quotaData.codexSnapshot) {
    var cx = quotaData.codexSnapshot;
    if (cx.capturedAt) {
      var sevenDReset = new Date(cx.capturedAt).getTime() + 7 * 86400000;
      var cxMs = sevenDReset - Date.now();
      items.push({ provider: '🤖 Codex', label: '7d window', resetMs: cxMs });
    }
  }

  if (items.length === 0) return '';

  // Sort by soonest reset
  items.sort(function(a, b) { return a.resetMs - b.resetMs; });

  var html = '<div class="countdown-strip">' +
    '<span class="countdown-title">⏱ Resets:</span>';
  for (var c = 0; c < Math.min(items.length, 6); c++) {
    var item = items[c];
    var timeStr = '';
    if (item.resetMs <= 0) {
      timeStr = 'ready';
    } else {
      var d = Math.floor(item.resetMs / 86400000);
      var h = Math.floor((item.resetMs % 86400000) / 3600000);
      var m = Math.floor((item.resetMs % 3600000) / 60000);
      timeStr = d > 0 ? d + 'd ' + h + 'h' : h > 0 ? h + 'h ' + m + 'm' : m + 'm';
    }
    html += '<div class="countdown-chip">' +
      '<span class="countdown-provider">' + item.provider + '</span>' +
      '<span class="countdown-time">' + timeStr + '</span>' +
      '</div>';
  }
  html += '</div>';
  return html;
}

// Live countdown refresh — call every 60s to update timers client-side
var countdownInterval: ReturnType<typeof setInterval> | null = null;

export function startCountdownRefresh(quotaData: any): void {
  if (countdownInterval) clearInterval(countdownInterval);
  countdownInterval = setInterval(function() {
    var container = document.getElementById('countdown-container');
    if (container) {
      container.innerHTML = renderCountdowns(quotaData);
    }
  }, 60000);
}
