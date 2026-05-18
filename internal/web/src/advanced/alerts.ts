// Niyantra Dashboard — System Alerts
import { esc, showToast } from '../core/utils';
import { switchToTab } from '../core/theme';


export function loadSystemAlerts(): void {
  fetch('/api/alerts').then(function(r) { return r.json(); })
  .then(function(data) {
    var container = document.getElementById('alert-banner-container');
    if (!container) return;
    var alerts = data.alerts || [];
    if (alerts.length === 0) {
      container.innerHTML = '';
      return;
    }
    var html = '';
    var shown = Math.min(alerts.length, 3);
    for (var i = 0; i < shown; i++) {
      var a = alerts[i];
      var icon = a.severity === 'critical' ? '🚨' : (a.severity === 'warning' ? '⚠️' : 'ℹ️');
      html += '<div class="alert-banner ' + esc(a.severity) + '">' +
        '<span class="alert-banner-icon">' + icon + '</span>' +
        '<div class="alert-banner-content">' +
        '<div class="alert-banner-title">' + esc(a.category) + '</div>' +
        '<div class="alert-banner-msg">' + esc(a.message) + '</div>' +
        '</div>' +
        '<button class="alert-banner-dismiss" data-alert-dismiss="' + a.id + '" title="Dismiss">&times;</button>' +
        '</div>';
    }
    if (alerts.length > 3) {
      html += '<div class="alert-more-link" data-alert-nav="overview">' +
        '+ ' + (alerts.length - 3) + ' more alert(s)</div>';
    }
    container.innerHTML = html;

    container.querySelectorAll('[data-alert-dismiss]').forEach(function(el) {
      el.addEventListener('click', function() {
        var button = el as HTMLElement;
        dismissAlert(button.dataset.alertDismiss || '');
      });
    });

    var moreLink = container.querySelector('[data-alert-nav="overview"]');
    if (moreLink) {
      moreLink.addEventListener('click', function() {
        switchToTab('overview');
      });
    }
  }).catch(function() {});
}

export function dismissAlert(id: string | number): void {
  fetch('/api/alerts/dismiss', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: id })
  }).then(function() {
    loadSystemAlerts();
    showToast('Alert dismissed', 'success');
  }).catch(function() {
    showToast('Failed to dismiss alert', 'error');
  });
}

// ════════════════════════════════════════════
