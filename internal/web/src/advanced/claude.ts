// Niyantra Dashboard — Claude Code Bridge
import { esc, formatTimeAgo } from '../core/utils';
import { formatResetTime } from '../quotas/render';


export function loadClaudeBridgeStatus(): void {
  fetch('/api/claude/status').then(function(r) { return r.json(); })
  .then(function(data) {
    var statusEl = document.getElementById('claude-bridge-status');
    if (!statusEl) return;

    var bridgeOn = data.bridgeEnabled;
    var installed = data.installed;

    if (!bridgeOn) {
      statusEl.style.display = 'none';
      return;
    }

    var msg = '';
    if (!installed) {
      msg = '⚠️ Claude Code not detected (~/.claude/ not found)';
    } else if (data.bridgeFresh) {
      msg = '<span class="claude-bridge-dot"></span> Bridge active';
      if (data.snapshot) {
        msg += ' · 5h: ' + data.snapshot.fiveHourPct.toFixed(1) + '% used';
      }
    } else if (data.snapshot) {
      msg = '<span class="claude-bridge-dot stale"></span> Last data: ' + formatTimeAgo(data.snapshot.capturedAt);
    } else {
      msg = '<span class="claude-bridge-dot off"></span> Waiting for Claude Code statusline data...';
    }

    statusEl.innerHTML = msg;
    statusEl.style.display = '';
  }).catch(function() {});
}

// ════════════════════════════════════════════

