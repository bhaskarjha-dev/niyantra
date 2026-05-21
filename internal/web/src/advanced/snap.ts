// Niyantra Dashboard — Snap Handler
import { snapInProgress, setSnapInProgress } from '../core/state';
import { showToast, updateTimestamp } from '../core/utils';
import { triggerSnap, fetchStatus } from '../core/api';
import { renderAccounts } from '../quotas/render';




// H3: Split-button snap — source-aware snapping
var snapDefault = localStorage.getItem('niyantra_snap_default') || 'antigravity';

export function initSnapDropdown(): void {
  var caret = document.getElementById('snap-caret');
  var dropdown = document.getElementById('snap-dropdown');
  if (!caret || !dropdown) return;

  // Toggle dropdown
  caret.addEventListener('click', function(e) {
    e.stopPropagation();
    dropdown!.classList.toggle('open');
  });

  // Close on outside click
  document.addEventListener('click', function() {
    dropdown!.classList.remove('open');
  });

  // Option clicks
  dropdown.querySelectorAll('.snap-option').forEach(function(opt) {
    opt.addEventListener('click', function(e) {
      e.stopPropagation();
      var source = (opt as HTMLElement).dataset.source;
      dropdown!.classList.remove('open');
      if (source === 'all') {
        snapSource('all');
      } else {
        // Set as new default + snap it
        snapDefault = source!;
        localStorage.setItem('niyantra_snap_default', source!);
        updateSnapDropdownIndicators();
        snapSource(source!);
      }
    });
  });

  updateSnapDropdownIndicators();
}

export function updateSnapDropdownIndicators(): void {
  var dropdown = document.getElementById('snap-dropdown');
  if (!dropdown) return;
  dropdown.querySelectorAll('.snap-option').forEach(function(opt) {
    if ((opt as HTMLElement).dataset.source === 'all') return; // divider option
    var isActive = (opt as HTMLElement).dataset.source === snapDefault;
    (opt as HTMLElement).textContent = (isActive ? '◉ ' : '○ ') + (opt as HTMLElement).textContent!.replace(/^[◉○] /, '');
    (opt as HTMLElement).classList.toggle('active', isActive);
  });
}

export function handleSnap(): void {
  snapSource(snapDefault);
}

export function snapSource(source: string): void {
  var btn = document.getElementById('snap-btn');
  if (!btn || (btn as HTMLButtonElement).disabled || snapInProgress) return;

  setSnapInProgress(true);
  (btn as HTMLButtonElement).disabled = true;
  btn!.classList.add('snapping');
  var orig = btn.innerHTML;
  btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/></svg> Capturing...';

  var promises = [];

  if (source === 'antigravity' || source === 'all') {
    promises.push(
      triggerSnap().then(function(data) {
        if (!data.captured || data.captured.length === 0) {
          return { source: 'Antigravity', error: 'No active session detected' };
        }
        var emails = data.captured.map(function(c: any) { return c.email; });
        var label = 'Antigravity · ' + emails.join(', ');
        return { source: 'Antigravity', data: data, label: label };
      }).catch(function(err) {
        return { source: 'Antigravity', error: err.message || 'Capture failed' };
      })
    );
  }

  if (source === 'claude' || source === 'all') {
    promises.push(
      fetch('/api/claude/snap', { method: 'POST' }).then(function(r) { return r.json(); })
      .then(function(d) {
        if (d.error) return { source: 'Claude Code', error: d.error };
        var label = 'Claude Code · 5h ' + (d.fiveHourPct || 0).toFixed(0) + '%';
        return { source: 'Claude Code', data: d, label: label };
      })
      .catch(function() { return { source: 'Claude Code', error: 'capture failed' }; })
    );
  }

  if (source === 'codex' || source === 'all') {
    promises.push(
      fetch('/api/codex/snap', { method: 'POST' }).then(function(r) { return r.json(); })
      .then(function(d) {
        var label = d.plan ? ('Codex · ' + d.plan) : 'Codex';
        return { source: 'Codex', data: d, label: label };
      })
      .catch(function() { return { source: 'Codex', error: 'capture failed' }; })
    );
  }

  if (source === 'claude' || source === 'all') {
    promises.push(
      fetch('/api/claude/snap', { method: 'POST' }).then(function(r) {
        if (!r.ok) return r.json().then(function(e) { throw new Error(e.error || 'capture failed'); });
        return r.json();
      })
      .then(function(d) {
        var label = 'Claude Code · ' + (d.fiveHourPct || 0).toFixed(0) + '%';
        return { source: 'Claude', data: d, label: label };
      })
      .catch(function(err) { return { source: 'Claude', error: err.message || 'capture failed' }; })
    );
  }

  if (source === 'cursor' || source === 'all') {
    promises.push(
      fetch('/api/cursor/snap', { method: 'POST' }).then(function(r) { return r.json(); })
      .then(function(d) {
        if (d.error) return { source: 'Cursor', error: d.error };
        var label = '';
        if (d.billingModel === 'usd_credit') {
          label = 'Cursor · $' + ((d.usedCents || 0) / 100).toFixed(2) + '/$' + ((d.limitCents || 0) / 100).toFixed(2);
        } else {
          label = 'Cursor · ' + (d.requestsUsed || 0) + '/' + (d.requestsMax || '?');
        }
        return { source: 'Cursor', data: d, label: label };
      })
      .catch(function() { return { source: 'Cursor', error: 'capture failed' }; })
    );
  }

  if (source === 'copilot' || source === 'all') {
    promises.push(
      fetch('/api/copilot/snap', { method: 'POST' }).then(function(r) { return r.json(); })
      .then(function(d) {
        if (d.error) return { source: 'Copilot', error: d.error };
        var label = 'Copilot · ' + (d.plan || 'unknown') + ' · ' + (d.premiumPct || 0).toFixed(0) + '%';
        return { source: 'Copilot', data: d, label: label };
      })
      .catch(function() { return { source: 'Copilot', error: 'capture failed' }; })
    );
  }

  if (promises.length === 0) {
    (btn as HTMLButtonElement).innerHTML = orig;
    (btn as HTMLButtonElement).disabled = false;
    setSnapInProgress(false);
    showToast('No snap source selected', 'warning');
    return;
  }

  Promise.all(promises).then(function(results: any[]) {
    var msgs = [];
    var success = false;
    for (var i = 0; i < results.length; i++) {
      var r = results[i];
      if (r.error) {
        msgs.push('❌ ' + r.source + ': ' + r.error);
      } else {
        msgs.push('✅ ' + r.label);
        success = true;
      }
    }
    showToast(msgs.join(' · '), msgs.some(function(m) { return m.startsWith('❌'); }) ? 'warning' : 'success');
    if (success) {
      fetchStatus().then(function(data) {
        document.dispatchEvent(new CustomEvent('niyantra:status-refreshed', { detail: { data: data } }));
      }).catch(function(err) {
        console.error('Failed to reload status after snap:', err);
      });
    }
  }).finally(function() {
    (btn as HTMLButtonElement).innerHTML = orig;
    (btn as HTMLButtonElement).disabled = false;
    (btn as HTMLButtonElement).classList.remove('snapping');
    setSnapInProgress(false);
  });
}



