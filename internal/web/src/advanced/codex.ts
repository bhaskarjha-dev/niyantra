// Niyantra Dashboard — Codex & Sessions
import { esc, showToast, formatTimeAgo, formatDurationSec } from '../core/utils';
import { formatResetTime } from '../quotas/render';


// ── Codex Settings Status ──
export function loadCodexSettingsStatus(): void {
  var statusEl = document.getElementById('codex-status-settings');
  if (!statusEl) return;

  fetch('/api/codex/status').then(function(r) { return r.json(); })
  .then(function(data) {
    statusEl!.style.display = '';
    if (!data.installed) {
      statusEl!.innerHTML = '<span style="color:var(--text-muted)">⚠️ Codex CLI not detected. ' +
        'Install <a href="https://github.com/openai/codex" target="_blank" style="color:var(--accent)">Codex</a> ' +
        'and run <code>codex auth</code> to enable.</span>';
      return;
    }
    var tokenStatus = data.tokenExpired ?
      '<span style="color:var(--warning)">⚠️ Token expired — will auto-refresh on next poll</span>' :
      '<span style="color:var(--success)">✅ Token valid (expires ' + (data.tokenExpiresIn || '?') + ')</span>';
    var displayId = data.email || (data.accountId && data.accountId.length > 12 ? data.accountId.substring(0,6) + '…' + data.accountId.slice(-6) : (data.accountId || 'unknown'));
    statusEl!.innerHTML =
      '🤖 Codex detected · Account: <strong>' + esc(displayId) + '</strong><br>' +
      tokenStatus;
    if (data.snapshot) {
      statusEl!.innerHTML += '<br>Latest: <strong>' + data.snapshot.fiveHourPct.toFixed(1) + '%</strong> used (5h) · ' +
        '<span style="color:var(--text-muted)">' + formatTimeAgo(data.snapshot.capturedAt) + '</span>';
    }
  })
  .catch(function() {
    statusEl!.style.display = 'none';
  });
}

// ── Codex Manual Snap ──
export function handleCodexSnap(): void {
  showToast('🤖 Capturing Codex snapshot...', 'info');
  fetch('/api/codex/snap', { method: 'POST' })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.error) {
      showToast('❌ ' + data.error, 'error');
      return;
    }
    showToast('🤖 Codex snapshot captured! Plan: ' + (data.plan || 'unknown'), 'success');
    loadCodexSettingsStatus();
    document.dispatchEvent(new CustomEvent('niyantra:overview-refresh'));
  })
  .catch(function() { showToast('❌ Codex snap failed', 'error'); });
}

// ── Codex Status Card (for Overview tab) removed ──

// ── Sessions Timeline (for Overview tab) ──
export function renderSessionsTimeline(container: HTMLElement): void {
  fetch('/api/sessions?limit=10').then(function(r) { return r.json(); })
  .then(function(data) {
    if (!data.sessions || data.sessions.length === 0) return;

    var html = '<div class="overview-card sessions-card">';
    html += '<div class="card-header"><h3>⏱️ Usage Sessions</h3>';
    html += '<span class="card-count">' + data.count + ' sessions</span>';
    html += '</div>';
    html += '<div class="card-body">';
    html += '<div class="session-timeline">';

    for (var i = 0; i < data.sessions.length; i++) {
      var sess = data.sessions[i];
      var isActive = !sess.endedAt;
      var duration = isActive ?
        formatDurationSec(Math.floor((Date.now() - new Date(sess.startedAt).getTime()) / 1000)) :
        formatDurationSec(sess.durationSec);
      var providerIcon = sess.provider === 'codex' ? '🤖' :
                         sess.provider === 'claude' ? '🔮' : '⚡';

      html += '<div class="session-item' + (isActive ? ' active' : '') + '">';
      html += '<div class="session-dot' + (isActive ? ' pulse' : '') + '"></div>';
      html += '<div class="session-content">';
      html += '<div class="session-top">';
      html += '<span class="session-provider">' + providerIcon + ' ' + esc(sess.provider) + '</span>';
      html += '<span class="session-duration">' + duration + '</span>';
      html += '</div>';
      html += '<div class="session-bottom">';
      html += '<span class="session-time">' + formatTimeAgo(sess.startedAt) + '</span>';
      html += '<span class="session-snaps">' + sess.snapCount + ' snaps</span>';
      if (isActive) html += '<span class="session-active-badge">LIVE</span>';
      html += '</div>';
      html += '</div></div>';
    }

    html += '</div></div></div>';

    // Insert after codex card or at start
    var codexCard = container.querySelector('.codex-card');
    var existing = container.querySelector('.sessions-card');
    if (existing) {
      existing.outerHTML = html;
    } else if (codexCard) {
      codexCard.insertAdjacentHTML('afterend', html);
    } else {
      container.insertAdjacentHTML('afterbegin', html);
    }
  })
  .catch(function() {});
}

