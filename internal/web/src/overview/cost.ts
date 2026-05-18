// Niyantra Dashboard - Heuristic Cost KPI
import { esc } from '../core/utils';

export function loadCostKPI(): void {
  var container = document.getElementById('cost-kpi-container');
  if (!container) return;

  fetch('/api/cost').then(function(res) { return res.json(); }).then(function(data) {
    if (!data || !data.accounts || data.accounts.length === 0) {
      container!.innerHTML = '';
      return;
    }

    var total = data.totalCost || 0;
    if (total < 0.01) {
      container!.innerHTML = '';
      return;
    }
    var totalLabel = data.totalLabel || '$0.00';

    var html = '<div class="cost-kpi-card overview-card">' +
      '<h3>Quota-Derived Cost Estimate</h3>' +
      '<div class="cost-kpi-amount">' + esc(totalLabel) + '</div>' +
      '<div class="cost-kpi-label">Estimated from quota consumption, configured token ceilings, and model pricing; not observed spend.</div>';

    var hasChips = false;
    var chipsHTML = '<div class="cost-kpi-breakdown">';
    if (data.accounts && data.accounts.length > 0) {
      for (var i = 0; i < data.accounts.length; i++) {
        var acct = data.accounts[i];
        if (acct.totalCost >= 0.01) {
          hasChips = true;
          var emailShort = acct.email;
          if (emailShort && emailShort.length > 20) {
            emailShort = emailShort.split('@')[0] + '@...';
          }
          chipsHTML += '<span class="cost-kpi-chip" title="' + esc(acct.email) + '">' +
            esc(emailShort) + ': ' + esc(acct.totalLabel) + '</span>';
        }
      }
    }
    chipsHTML += '</div>';
    if (hasChips) html += chipsHTML;

    html += '</div>';
    container!.innerHTML = html;
  }).catch(function(err) {
    console.error('Cost KPI fetch failed:', err);
    container!.innerHTML = '';
  });
}
