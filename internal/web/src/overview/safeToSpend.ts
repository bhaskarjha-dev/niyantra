// Budget headroom card for recurring subscription commitments.
// This intentionally avoids pretending we have a true usage-based spend
// forecast before a trustworthy observed spend ledger exists.

type BudgetBaseline = {
  monthlyBudget: number;
  currentSpend: number;
  recurringMonthlySpend?: number;
  onTrack: boolean;
  dataMode?: string;
  observedSpendAvailable?: boolean;
};

export function renderSafeToSpend(bf: BudgetBaseline | null, currency: string): string {
  if (!bf || bf.monthlyBudget <= 0) {
    return '<div class="safe-to-spend-card no-budget">' +
      '<div class="sts-icon">Budget</div>' +
      '<div class="sts-label">Set a monthly AI budget to track recurring subscription headroom</div>' +
      '<button class="btn-add-sm" id="sts-set-budget-btn">Set Budget</button>' +
      '</div>';
  }

  var recurring = bf.recurringMonthlySpend != null ? bf.recurringMonthlySpend : bf.currentSpend;
  var headroom = Math.max(0, bf.monthlyBudget - recurring);
  var pct = bf.monthlyBudget > 0 ? Math.round((recurring / bf.monthlyBudget) * 100) : 0;

  var cls: string;
  var statusText: string;
  if (pct >= 100) {
    cls = 'over';
    statusText = 'Recurring subscriptions exceed budget by ' + sym(currency) + (recurring - bf.monthlyBudget).toFixed(2) + '/mo';
  } else if (pct >= 80) {
    cls = 'warning';
    statusText = 'Only ' + sym(currency) + headroom.toFixed(2) + '/mo of recurring budget headroom remains';
  } else {
    cls = 'healthy';
    statusText = sym(currency) + headroom.toFixed(2) + '/mo remains after recurring subscriptions';
  }

  return '<div class="safe-to-spend-card ' + cls + '">' +
    '<div class="sts-header">' +
      '<span class="sts-title">Budget Headroom</span>' +
      '<button class="sts-edit" id="sts-edit-budget-btn" title="Edit budget">Edit</button>' +
    '</div>' +
    '<div class="sts-amount">' + sym(currency) + headroom.toFixed(2) + '</div>' +
    '<div class="sts-bar-container">' +
      '<div class="sts-bar-fill ' + cls + '" style="width:' + Math.min(pct, 100) + '%"></div>' +
    '</div>' +
    '<div class="sts-details">' +
      '<span>' + statusText + '</span>' +
      '<span class="sts-budget">Budget: ' + sym(currency) + bf.monthlyBudget.toFixed(0) + '/mo</span>' +
    '</div>' +
    '<div class="sts-caption">Based on recurring subscription commitments only; observed usage spend is not available yet.</div>' +
    '</div>';
}

// Wire up click handlers after the HTML is in the DOM (CSP-safe).
export function wireSafeToSpendButtons(openBudgetFn: () => void): void {
  var editBtn = document.getElementById('sts-edit-budget-btn');
  if (editBtn) editBtn.addEventListener('click', openBudgetFn);

  var setBtn = document.getElementById('sts-set-budget-btn');
  if (setBtn) setBtn.addEventListener('click', openBudgetFn);
}

function sym(currency: string): string {
  if (currency === 'INR') return 'Rs ';
  if (currency === 'EUR') return 'EUR ';
  if (currency === 'GBP') return 'GBP ';
  return '$';
}
