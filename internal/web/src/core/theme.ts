// Niyantra Dashboard — Theme & Navigation
// Dark mode toggle, tab switching, and navigation helpers.

import type { ThemeMode } from '../types/api';

export function initTheme(): void {
  var saved = localStorage.getItem('niyantra-theme') as ThemeMode | null;
  if (saved) {
    document.documentElement.setAttribute('data-theme', saved);
  } else if (window.matchMedia('(prefers-color-scheme: light)').matches) {
    document.documentElement.setAttribute('data-theme', 'light');
  }

  var themeBtn = document.getElementById('theme-btn');
  if (!themeBtn) return;
  themeBtn.addEventListener('click', function() {
    var current = document.documentElement.getAttribute('data-theme') as ThemeMode;
    var next: ThemeMode = current === 'light' ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('niyantra-theme', next);
    // M2: Update chart colors in-place to avoid flash
    // updateChartTheme is called from the chart module; we dispatch a custom event
    // so the chart module can listen without a circular dependency.
    document.dispatchEvent(new CustomEvent('niyantra:theme-change', { detail: { theme: next } }));
  });
}

export function initTabs(): void {
  var btns = document.querySelectorAll('.tab-btn');
  btns.forEach(function(btn) {
    btn.addEventListener('click', function() {
      var tab = (btn as HTMLElement).getAttribute('data-tab');
      if (tab) switchToTab(tab);
    });
  });

  // Restore saved tab or default to 'quotas'
  var savedTab = localStorage.getItem('niyantra-active-tab') || 'quotas';
  switchToTab(savedTab);
}

export function switchToTab(tabName: string): void {
  var btns = document.querySelectorAll('.tab-btn');
  btns.forEach(function(b) {
    b.classList.remove('active');
    b.setAttribute('aria-selected', 'false');
  });
  var target = document.querySelector('.tab-btn[data-tab="' + tabName + '"]');
  if (target) {
    target.classList.add('active');
    target.setAttribute('aria-selected', 'true');
  }

  document.querySelectorAll('.tab-panel').forEach(function(p) {
    p.classList.remove('active');
  });
  var panel = document.getElementById('panel-' + tabName);
  if (panel) panel.classList.add('active');

  // Save selected tab to localStorage
  localStorage.setItem('niyantra-active-tab', tabName);

  // Dispatch custom event so domain modules can react to tab activation
  document.dispatchEvent(new CustomEvent('niyantra:tab-change', { detail: { tab: tabName } }));
}
