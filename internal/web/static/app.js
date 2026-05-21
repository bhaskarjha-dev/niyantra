// GENERATED FILE — do not edit. Source: internal/web/src/
"use strict";
(() => {
  // internal/web/src/core/state.ts
  var GROUP_ORDER = ["claude_gpt", "gemini_unified", "unknown"];
  var GRID_COLUMNS = ["claude_gpt", "gemini_unified"];
  var GRID_LABELS = ["Claude + GPT", "Gemini Pool"];
  var GROUP_COLORS = { claude_gpt: "#D97757", gemini_unified: "#3B82F6", unknown: "#64748B" };
  var GROUP_NAMES = { claude_gpt: "Claude + GPT", gemini_unified: "Gemini Pool", unknown: "Other" };
  var expandedAccounts = /* @__PURE__ */ new Set();
  var savedCollapsed = typeof localStorage !== "undefined" ? localStorage.getItem("niyantra_collapsed_providers") : null;
  var collapsedProviders = new Set(savedCollapsed ? JSON.parse(savedCollapsed) : []);
  var savedSubCollapsed = typeof localStorage !== "undefined" ? localStorage.getItem("niyantra_collapsed_subs_providers") : null;
  var collapsedSubProviders = new Set(savedSubCollapsed ? JSON.parse(savedSubCollapsed) : []);
  var presetsData = [];
  function setPresetsData(data) {
    presetsData = data;
  }
  var activeTagFilter = null;
  function setActiveTagFilter(val) {
    activeTagFilter = val;
  }
  var usageDataCache = null;
  function setUsageDataCache(data) {
    usageDataCache = data;
  }
  var quotaSortState = { column: "account", direction: "asc" };
  var quotaSortStates = {
    antigravity: { column: "account", direction: "asc" },
    codex: { column: "account", direction: "asc" },
    cursor: { column: "account", direction: "asc" },
    copilot: { column: "account", direction: "asc" }
  };
  var latestQuotaData = null;
  function setLatestQuotaData(data) {
    latestQuotaData = data;
  }
  var serverConfig = {};
  var snapInProgress = false;
  function setSnapInProgress(val) {
    snapInProgress = val;
  }

  // internal/web/src/core/utils.ts
  function formatSeconds(seconds) {
    seconds = Math.floor(seconds);
    if (seconds <= 0) return "now";
    var h = Math.floor(seconds / 3600);
    var m = Math.floor(seconds % 3600 / 60);
    if (h >= 24) return Math.floor(h / 24) + "d " + h % 24 + "h";
    if (h > 0) return h + "h " + m + "m";
    if (m === 0) return "<1m";
    return m + "m";
  }
  function formatCredits(n) {
    if (n >= 1e3) return (n / 1e3).toFixed(n % 1e3 === 0 ? 0 : 1) + "k";
    return Math.round(n).toString();
  }
  function formatNumber(n) {
    if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
    if (n >= 1e3) return (n / 1e3).toFixed(n % 1e3 === 0 ? 0 : 1) + "k";
    return n.toString();
  }
  function currencySymbol(code) {
    var map = { USD: "$", EUR: "\u20AC", GBP: "\xA3", INR: "\u20B9", CAD: "C$", AUD: "A$" };
    return map[code] || code + " ";
  }
  function esc(s) {
    if (!s) return "";
    var d = document.createElement("div");
    d.textContent = s;
    return d.innerHTML.replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  function showToast(msg, type) {
    var el = document.getElementById("toast");
    if (!el) return;
    el.textContent = msg;
    el.className = "toast " + type + " visible";
    el.hidden = false;
    setTimeout(function() {
      el.classList.remove("visible");
      setTimeout(function() {
        el.hidden = true;
      }, 300);
    }, 3e3);
  }
  var lastUpdateTime = null;
  function updateTimestamp() {
    lastUpdateTime = /* @__PURE__ */ new Date();
    refreshTimestampDisplay();
  }
  function refreshTimestampDisplay() {
    var el = document.getElementById("last-updated");
    if (!el || !lastUpdateTime) return;
    var sec = Math.floor(((/* @__PURE__ */ new Date()).getTime() - lastUpdateTime.getTime()) / 1e3);
    var label;
    if (sec < 10) label = "just now";
    else if (sec < 60) label = sec + "s ago";
    else if (sec < 3600) label = Math.floor(sec / 60) + "m ago";
    else label = Math.floor(sec / 3600) + "h ago";
    el.textContent = "Updated " + label;
    el.title = lastUpdateTime.toLocaleTimeString();
  }
  function formatTimeAgo(isoStr) {
    if (!isoStr) return "never";
    var d = new Date(isoStr);
    var now = /* @__PURE__ */ new Date();
    var sec = Math.floor((now.getTime() - d.getTime()) / 1e3);
    if (sec < 60) return "just now";
    if (sec < 3600) return Math.floor(sec / 60) + "m ago";
    if (sec < 86400) return Math.floor(sec / 3600) + "h ago";
    return Math.floor(sec / 86400) + "d ago";
  }
  function formatPollInterval(seconds) {
    if (seconds >= 3600) return Math.floor(seconds / 3600) + "h";
    return Math.floor(seconds / 60) + "m";
  }
  function formatDurationSec(sec) {
    if (!sec || sec <= 0) return "0m";
    var h = Math.floor(sec / 3600);
    var m = Math.floor(sec % 3600 / 60);
    if (h > 0) return h + "h " + m + "m";
    return m + "m";
  }

  // internal/web/src/core/api.ts
  var TOKEN_STORAGE_KEY = "niyantra.dashboardToken";
  var DEFAULT_API_TIMEOUT_MS = 3e4;
  var originalFetch = null;
  function bootstrapDashboardToken() {
    var params = new URLSearchParams(window.location.search);
    var token = params.get("token") || params.get("auth_token");
    if (token) {
      sessionStorage.setItem(TOKEN_STORAGE_KEY, token);
      params.delete("token");
      params.delete("auth_token");
      var next = window.location.pathname + (params.toString() ? "?" + params.toString() : "") + window.location.hash;
      window.history.replaceState({}, document.title, next || "/");
      return token;
    }
    return sessionStorage.getItem(TOKEN_STORAGE_KEY);
  }
  function dashboardToken() {
    return sessionStorage.getItem(TOKEN_STORAGE_KEY);
  }
  function installAuthenticatedFetch() {
    if (originalFetch) return;
    bootstrapDashboardToken();
    originalFetch = window.fetch.bind(window);
    window.fetch = function(input, init) {
      return apiFetch(input, init);
    };
  }
  function apiFetch(input, init) {
    var fetchImpl = originalFetch || window.fetch.bind(window);
    var nextInit = Object.assign({}, init || {});
    var headers = new Headers(nextInit.headers || {});
    var url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    var protectedLocalURL = isProtectedLocalURL(url);
    var timeoutHandle;
    if (protectedLocalURL && !nextInit.signal && typeof AbortController !== "undefined") {
      var controller = new AbortController();
      nextInit.signal = controller.signal;
      timeoutHandle = window.setTimeout(function() {
        controller.abort();
      }, DEFAULT_API_TIMEOUT_MS);
    }
    if (protectedLocalURL && !headers.has("Authorization")) {
      var token = dashboardToken();
      if (token) headers.set("Authorization", "Bearer " + token);
    }
    nextInit.headers = headers;
    return fetchImpl(input, nextInit).then(function(res) {
      if (res.status === 401) {
        showAuthFailure();
      }
      return res;
    }).finally(function() {
      if (timeoutHandle !== void 0) window.clearTimeout(timeoutHandle);
    });
  }
  function isProtectedLocalURL(rawURL) {
    var url = new URL(rawURL, window.location.href);
    if (url.origin !== window.location.origin) return false;
    return url.pathname.indexOf("/api/") === 0 || url.pathname === "/mcp";
  }
  function showAuthFailure() {
    var existing = document.getElementById("auth-error-banner");
    if (existing) return;
    var banner = document.createElement("div");
    banner.id = "auth-error-banner";
    banner.className = "auth-error-banner";
    banner.textContent = "Dashboard API token is missing or invalid. Reopen the tokenized URL printed by niyantra serve.";
    document.body.prepend(banner);
  }
  function fetchStatus() {
    return apiFetch("/api/status").then(function(res) {
      if (!res.ok) throw new Error("Failed to fetch status");
      return res.json();
    });
  }
  function triggerSnap() {
    return apiFetch("/api/snap", { method: "POST" }).then(function(res) {
      return res.json().then(function(data) {
        if (!res.ok) throw new Error(data.error || "Snap failed");
        return data;
      });
    });
  }
  function fetchSubscriptions(status, category) {
    var params = new URLSearchParams();
    if (status) params.set("status", status);
    if (category) params.set("category", category);
    var url = "/api/subscriptions" + (params.toString() ? "?" + params : "");
    return apiFetch(url).then(function(res) {
      return res.json();
    });
  }
  function createSubscription(sub) {
    return apiFetch("/api/subscriptions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(sub)
    }).then(function(res) {
      return res.json().then(function(data) {
        if (!res.ok) throw new Error(data.error || "Create failed");
        return data;
      });
    });
  }
  function updateSubscription(id, sub) {
    return apiFetch("/api/subscriptions/" + id, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(sub)
    }).then(function(res) {
      return res.json().then(function(data) {
        if (!res.ok) throw new Error(data.error || "Update failed");
        return data;
      });
    });
  }
  function deleteSubscription(id) {
    return apiFetch("/api/subscriptions/" + id, { method: "DELETE" }).then(function(res) {
      return res.json().then(function(data) {
        if (!res.ok) throw new Error(data.error || "Delete failed");
        return data;
      });
    });
  }
  function fetchOverview() {
    return apiFetch("/api/overview").then(function(res) {
      return res.json();
    });
  }
  function fetchPresets() {
    return apiFetch("/api/presets").then(function(res) {
      return res.json();
    });
  }
  function fetchUsage(accountId) {
    var url = "/api/usage";
    if (accountId) url += "?account=" + accountId;
    return apiFetch(url).then(function(res) {
      return res.json();
    }).then(function(data) {
      setUsageDataCache(data);
      return data;
    });
  }
  function downloadBackup() {
    return apiFetch("/api/backup/create", { method: "POST" }).then(function(res) {
      if (!res.ok) {
        return res.json().catch(function() {
          return {};
        }).then(function(data) {
          throw new Error(data.error || "Backup failed");
        });
      }
      return res.blob().then(function(blob) {
        var filename = filenameFromDisposition(res.headers.get("Content-Disposition")) || "niyantra-backup.db";
        var url = URL.createObjectURL(blob);
        var a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        URL.revokeObjectURL(url);
      });
    });
  }
  function downloadAPIFile(url, fallbackName) {
    return apiFetch(url).then(function(res) {
      if (!res.ok) {
        return res.json().catch(function() {
          return {};
        }).then(function(data) {
          throw new Error(data.error || "Download failed");
        });
      }
      return res.blob().then(function(blob) {
        var filename = filenameFromDisposition(res.headers.get("Content-Disposition")) || fallbackName;
        var objectURL = URL.createObjectURL(blob);
        var a = document.createElement("a");
        a.href = objectURL;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        URL.revokeObjectURL(objectURL);
      });
    });
  }
  function filenameFromDisposition(header) {
    if (!header) return null;
    var match = /filename="([^"]+)"/.exec(header);
    return match ? match[1] : null;
  }
  function claimOverageBonus(accountId) {
    return apiFetch("/api/accounts/" + accountId + "/claim-bonus", { method: "POST" }).then(function(res) {
      return res.json().then(function(data) {
        if (!res.ok) throw new Error(data.error || "Claim failed");
        return data;
      });
    });
  }

  // internal/web/src/core/theme.ts
  function initTheme() {
    var saved = localStorage.getItem("niyantra-theme");
    if (saved) {
      document.documentElement.setAttribute("data-theme", saved);
    } else if (window.matchMedia("(prefers-color-scheme: light)").matches) {
      document.documentElement.setAttribute("data-theme", "light");
    }
    var themeBtn = document.getElementById("theme-btn");
    if (!themeBtn) return;
    themeBtn.addEventListener("click", function() {
      var current = document.documentElement.getAttribute("data-theme");
      var next = current === "light" ? "dark" : "light";
      document.documentElement.setAttribute("data-theme", next);
      localStorage.setItem("niyantra-theme", next);
      document.dispatchEvent(new CustomEvent("niyantra:theme-change", { detail: { theme: next } }));
    });
  }
  function initTabs() {
    var btns = document.querySelectorAll(".tab-btn");
    btns.forEach(function(btn) {
      btn.addEventListener("click", function() {
        var tab = btn.getAttribute("data-tab");
        if (tab) switchToTab(tab);
      });
    });
    var savedTab = localStorage.getItem("niyantra-active-tab") || "quotas";
    switchToTab(savedTab);
  }
  function switchToTab(tabName) {
    var btns = document.querySelectorAll(".tab-btn");
    btns.forEach(function(b) {
      b.classList.remove("active");
      b.setAttribute("aria-selected", "false");
    });
    var target = document.querySelector('.tab-btn[data-tab="' + tabName + '"]');
    if (target) {
      target.classList.add("active");
      target.setAttribute("aria-selected", "true");
    }
    document.querySelectorAll(".tab-panel").forEach(function(p) {
      p.classList.remove("active");
    });
    var panel = document.getElementById("panel-" + tabName);
    if (panel) panel.classList.add("active");
    localStorage.setItem("niyantra-active-tab", tabName);
    document.dispatchEvent(new CustomEvent("niyantra:tab-change", { detail: { tab: tabName } }));
  }

  // internal/web/src/quotas/features.ts
  var _renderAccounts = null;
  function setRenderAccounts(fn) {
    _renderAccounts = fn;
  }
  function refreshGrid() {
    if (_renderAccounts) fetchStatus().then(_renderAccounts);
  }
  function pinGroup(accountId, groupKey) {
    updateAccountMeta(accountId, { pinnedGroup: groupKey }).then(function() {
      showToast("\u2B50 Pinned " + (GROUP_NAMES[groupKey] || groupKey), "success");
      refreshGrid();
    });
  }
  function unpinGroup(accountId) {
    updateAccountMeta(accountId, { pinnedGroup: "" }).then(function() {
      showToast("\u2606 Unpinned \u2014 no group will be starred by default", "info");
      refreshGrid();
    });
  }
  function daysUntilRenewal(day) {
    if (!day || day < 1 || day > 31) return -1;
    var now = /* @__PURE__ */ new Date();
    var y = now.getFullYear();
    var m = now.getMonth();
    var today = now.getDate();
    var targetMonth = today < day ? m : m + 1;
    var target = new Date(y, targetMonth, day);
    var diff = Math.ceil((target.getTime() - now.getTime()) / (1e3 * 60 * 60 * 24));
    return diff < 0 ? 0 : diff;
  }
  function renderCreditRenewal(accountId, renewalDay) {
    if (!renewalDay || renewalDay < 1) {
      return '<span class="credit-renewal-set" data-renewal-edit="' + accountId + '" title="Set credit renewal day">\u21BB set</span>';
    }
    var days = daysUntilRenewal(renewalDay);
    var label = days === 0 ? "today" : days === 1 ? "1d" : days + "d";
    return '<span class="credit-renewal" data-renewal-edit="' + accountId + '" data-renewal-day="' + renewalDay + '" title="Credits renew on day ' + renewalDay + " (\u21BB " + label + ')">\u21BB ' + label + "</span>";
  }
  function openRenewalPicker(el) {
    var existing = document.querySelector(".renewal-picker");
    if (existing) existing.remove();
    var accountId = el.getAttribute("data-renewal-edit");
    var currentDay = parseInt(el.getAttribute("data-renewal-day")) || 0;
    var picker = document.createElement("div");
    picker.className = "renewal-picker";
    picker.innerHTML = '<div class="renewal-picker-label">Credit Renewal Day</div><input type="number" class="renewal-picker-input" min="1" max="31" value="' + (currentDay || "") + '" placeholder="1\u201331"><div class="renewal-picker-hint">Day of month when AI credits refresh.<br>Find at one.google.com/ai/activity</div>';
    el.closest(".credits-cell").appendChild(picker);
    var input = picker.querySelector("input");
    input.focus();
    input.select();
    function save() {
      var day = parseInt(input.value) || 0;
      if (day > 31) day = 31;
      if (day < 0) day = 0;
      picker.remove();
      updateAccountMeta(accountId, { creditRenewalDay: day }).then(function() {
        if (day > 0) {
          showToast("\u21BB Renewal day set to " + day, "success");
        } else {
          showToast("\u21BB Renewal day cleared", "info");
        }
        refreshGrid();
      });
    }
    input.addEventListener("keydown", function(e) {
      if (e.key === "Enter") {
        e.preventDefault();
        save();
      }
      if (e.key === "Escape") {
        e.preventDefault();
        picker.remove();
      }
    });
    input.addEventListener("blur", function() {
      setTimeout(function() {
        if (picker.parentNode) save();
      }, 150);
    });
  }
  var TAG_PRESETS = ["work", "personal", "primary", "backup", "shared", "test", "dev"];
  function renderAccountTags(acc) {
    var tags = (acc.tags || "").split(",").filter(function(t) {
      return t.trim();
    });
    var html = '<span class="account-tags" data-account-id="' + acc.accountId + '">';
    for (var i = 0; i < tags.length; i++) {
      html += '<span class="tag-chip" data-tag="' + esc(tags[i].trim()) + '">' + esc(tags[i].trim()) + '<span class="tag-remove" data-remove-tag="' + esc(tags[i].trim()) + '" data-account-id="' + acc.accountId + '" title="Remove tag">\u2715</span></span>';
    }
    html += "</span>";
    html += '<button class="tag-add-btn" data-tag-add="' + acc.accountId + '" title="Add tag">+</button>';
    return html;
  }
  function renderAccountNote(acc) {
    if (acc.notes) {
      return '<span class="account-note" data-note-edit="' + acc.accountId + '" data-current-note="' + esc(acc.notes) + '" title="' + esc(acc.notes) + ' \u2014 click to edit">\u{1F4DD} ' + esc(acc.notes) + "</span>";
    }
    return '<span class="account-note-empty" data-note-edit="' + acc.accountId + '" data-current-note="">+ note</span>';
  }
  function updateAccountMeta(accountId, patch) {
    return fetch("/api/accounts/" + accountId + "/meta", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(patch)
    }).then(function(r) {
      return r.json();
    });
  }
  function addTagToAccount(accountId, newTag) {
    newTag = newTag.trim().toLowerCase().replace(/[^a-z0-9_-]/g, "");
    if (!newTag) return;
    var container = document.querySelector('.account-tags[data-account-id="' + accountId + '"]');
    var existing = [];
    if (container) {
      container.querySelectorAll(".tag-chip").forEach(function(chip) {
        existing.push(chip.getAttribute("data-tag"));
      });
    }
    if (existing.indexOf(newTag) >= 0) return;
    existing.push(newTag);
    updateAccountMeta(accountId, { tags: existing.join(",") }).then(function() {
      showToast('\u{1F3F7}\uFE0F Tag "' + newTag + '" added', "success");
      refreshGrid();
    });
  }
  function removeTagFromAccount(accountId, tag) {
    var container = document.querySelector('.account-tags[data-account-id="' + accountId + '"]');
    var tags = [];
    if (container) {
      container.querySelectorAll(".tag-chip").forEach(function(chip) {
        var t = chip.getAttribute("data-tag");
        if (t !== tag) tags.push(t);
      });
    }
    updateAccountMeta(accountId, { tags: tags.join(",") }).then(function() {
      showToast("\u{1F3F7}\uFE0F Tag removed", "success");
      refreshGrid();
    });
  }
  function openTagPicker(btn) {
    closeTagPicker();
    var accountId = btn.getAttribute("data-tag-add");
    var meta = btn.closest(".account-meta");
    if (!meta) return;
    var existing = [];
    var container = meta.querySelector(".account-tags");
    if (container) {
      container.querySelectorAll(".tag-chip").forEach(function(chip) {
        existing.push(chip.getAttribute("data-tag"));
      });
    }
    var picker = document.createElement("div");
    picker.className = "tag-picker";
    picker.id = "active-tag-picker";
    picker.innerHTML = '<input type="text" class="tag-picker-input" placeholder="Type tag name..." autocomplete="off" maxlength="20"><div class="tag-picker-hint">Enter to add</div><div class="tag-picker-presets">' + TAG_PRESETS.map(function(p) {
      var active = existing.indexOf(p) >= 0 ? " active" : "";
      return '<button class="tag-preset' + active + '" data-preset-tag="' + p + '">' + p + "</button>";
    }).join("") + "</div>";
    meta.appendChild(picker);
    var input = picker.querySelector(".tag-picker-input");
    input.focus();
    picker.addEventListener("click", function(e) {
      e.stopPropagation();
    });
    input.addEventListener("keydown", function(e) {
      if (e.key === "Enter") {
        e.preventDefault();
        var val = input.value.trim();
        if (val) {
          addTagToAccount(accountId, val);
          closeTagPicker();
        }
      }
      if (e.key === "Escape") {
        closeTagPicker();
      }
    });
    picker.querySelectorAll(".tag-preset").forEach(function(btn2) {
      btn2.addEventListener("click", function(e) {
        e.stopPropagation();
        var tag = btn2.getAttribute("data-preset-tag");
        if (btn2.classList.contains("active")) {
          removeTagFromAccount(accountId, tag);
        } else {
          addTagToAccount(accountId, tag);
        }
        closeTagPicker();
      });
    });
    setTimeout(function() {
      document.addEventListener("click", closeTagPickerOnOutside);
    }, 10);
  }
  function closeTagPicker() {
    var picker = document.getElementById("active-tag-picker");
    if (picker) picker.remove();
    document.removeEventListener("click", closeTagPickerOnOutside);
  }
  function closeTagPickerOnOutside(e) {
    var picker = document.getElementById("active-tag-picker");
    if (picker && !picker.contains(e.target)) {
      closeTagPicker();
    }
  }
  function openNoteEditor(el) {
    var accountId = el.getAttribute("data-note-edit");
    var currentNote = el.getAttribute("data-current-note") || "";
    var editor = document.createElement("span");
    editor.className = "note-inline-editor";
    editor.innerHTML = '<textarea class="note-inline-input" placeholder="Add a note..." maxlength="500" rows="3">' + esc(currentNote) + "</textarea>";
    el.replaceWith(editor);
    var input = editor.querySelector(".note-inline-input");
    input.focus();
    editor.addEventListener("click", function(e) {
      e.stopPropagation();
    });
    function save() {
      var val = input.value.trim();
      updateAccountMeta(accountId, { notes: val }).then(function() {
        if (val) showToast("\u{1F4DD} Note saved", "success");
        refreshGrid();
      });
    }
    input.addEventListener("keydown", function(e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        save();
      }
      if (e.key === "Escape") {
        refreshGrid();
      }
    });
    input.addEventListener("blur", save);
  }
  function initAccountMetaHandlers() {
    var grid = document.getElementById("account-grid");
    if (!grid) return;
    grid.addEventListener("click", function(e) {
      var removeBtn = e.target.closest("[data-remove-tag]");
      if (removeBtn) {
        e.stopPropagation();
        e.preventDefault();
        removeTagFromAccount(removeBtn.getAttribute("data-account-id"), removeBtn.getAttribute("data-remove-tag"));
        return;
      }
      var addBtn = e.target.closest("[data-tag-add]");
      if (addBtn) {
        e.stopPropagation();
        e.preventDefault();
        openTagPicker(addBtn);
        return;
      }
      var noteEl = e.target.closest("[data-note-edit]");
      if (noteEl) {
        e.stopPropagation();
        e.preventDefault();
        openNoteEditor(noteEl);
        return;
      }
      var pinBtn = e.target.closest("[data-pin-group]");
      if (pinBtn) {
        e.stopPropagation();
        e.preventDefault();
        var pinAccountId = pinBtn.getAttribute("data-pin-account");
        var pinGroupKey = pinBtn.getAttribute("data-pin-group");
        if (pinBtn.classList.contains("pinned")) {
          unpinGroup(pinAccountId);
        } else {
          pinGroup(pinAccountId, pinGroupKey);
        }
        return;
      }
      var renewalEl = e.target.closest("[data-renewal-edit]");
      if (renewalEl) {
        e.stopPropagation();
        e.preventDefault();
        openRenewalPicker(renewalEl);
        return;
      }
    });
  }

  // internal/web/src/quotas/render.ts
  function getGroupPct(acc, groupKey) {
    if (!acc.groups) return -1;
    for (var i = 0; i < acc.groups.length; i++) {
      if (acc.groups[i].groupKey === groupKey) return acc.groups[i].remainingPercent;
    }
    return -1;
  }
  function getAICredits(acc) {
    if (acc.aiCredits && acc.aiCredits.length > 0) return acc.aiCredits[0].creditAmount;
    return -1;
  }
  function getSoonestResetSec(acc) {
    var groups = acc.groups || [];
    var soonest = Infinity;
    for (var i = 0; i < groups.length; i++) {
      var t = groups[i].timeUntilResetSec;
      if (t !== void 0 && t !== null && t < soonest) soonest = t;
    }
    return soonest === Infinity ? -1 : soonest;
  }
  function allExhausted(acc) {
    var grps = acc.groups || [];
    if (grps.length === 0) return false;
    for (var i = 0; i < grps.length; i++) {
      if (!grps[i].isExhausted && grps[i].remainingPercent > 0) return false;
    }
    return true;
  }
  function getCodexClaudeStatus(snap) {
    var fiveUsed = snap.fiveHourPct || 0;
    var sevenUsed = snap.sevenDayPct || 0;
    var fiveRem = Math.max(0, 100 - fiveUsed);
    var sevenRem = Math.max(0, 100 - sevenUsed);
    if (fiveRem === 0 && sevenRem === 0) return "empty";
    if (fiveUsed >= 80 || sevenUsed >= 80) return "low";
    return "ready";
  }
  function getCodexSoonestResetSec(cs) {
    var now = Date.now();
    var soonest = Infinity;
    if (cs.fiveHourReset) {
      var t5 = new Date(cs.fiveHourReset).getTime() - now;
      if (t5 > 0 && t5 < soonest) soonest = t5;
    }
    if (cs.sevenDayReset) {
      var t7 = new Date(cs.sevenDayReset).getTime() - now;
      if (t7 > 0 && t7 < soonest) soonest = t7;
    }
    return soonest === Infinity ? -1 : Math.round(soonest / 1e3);
  }
  function getCursorSoonestResetSec(cs) {
    if (!cs.cycleEnd) return -1;
    var ts = /^\d+$/.test(cs.cycleEnd) ? parseInt(cs.cycleEnd) : cs.cycleEnd;
    var t = new Date(ts).getTime() - Date.now();
    return t > 0 ? Math.round(t / 1e3) : -1;
  }
  function sortAccountsArray(accounts) {
    var state = quotaSortStates.antigravity || quotaSortState;
    var col = state.column;
    var dir = state.direction;
    return accounts.slice().sort(function(a, b) {
      var va, vb;
      switch (col) {
        case "account":
          va = a.email;
          vb = b.email;
          break;
        case "claude_gpt":
        case "gemini_unified":
        case "unknown":
          va = getGroupPct(a, col);
          vb = getGroupPct(b, col);
          break;
        case "credits":
          va = getAICredits(a);
          vb = getAICredits(b);
          break;
        case "lastsnap":
          va = a.lastSeen ? new Date(a.lastSeen).getTime() : 0;
          vb = b.lastSeen ? new Date(b.lastSeen).getTime() : 0;
          break;
        case "status":
          va = a.isReady ? 1 : 0;
          vb = b.isReady ? 1 : 0;
          break;
        case "resetsIn":
          va = getSoonestResetSec(a);
          vb = getSoonestResetSec(b);
          if (va < 0) va = 999999999;
          if (vb < 0) vb = 999999999;
          if (va <= 0) va = 0;
          if (vb <= 0) vb = 0;
          break;
        default:
          va = a.email;
          vb = b.email;
          break;
      }
      if (va === vb) return 0;
      var res = va > vb ? 1 : -1;
      return dir === "asc" ? res : -res;
    });
  }
  function sortProviderArray(array, provider) {
    var state = quotaSortStates[provider] || quotaSortState;
    var col = state.column;
    var dir = state.direction;
    return array.slice().sort(function(a, b) {
      var va, vb;
      if (provider === "codex") {
        switch (col) {
          case "account":
            va = a.email || a.accountId || "";
            vb = b.email || b.accountId || "";
            break;
          case "plan":
            va = a.planType || "";
            vb = b.planType || "";
            break;
          case "fiveHour":
            va = a.fiveHourPct || 0;
            vb = b.fiveHourPct || 0;
            break;
          case "sevenDay":
            va = a.sevenDayPct || 0;
            vb = b.sevenDayPct || 0;
            break;
          case "credits":
            va = a.creditsBalance || 0;
            vb = b.creditsBalance || 0;
            break;
          case "lastsnap":
            va = a.capturedAt ? new Date(a.capturedAt).getTime() : 0;
            vb = b.capturedAt ? new Date(b.capturedAt).getTime() : 0;
            break;
          case "status":
            var vaReset = getCodexSoonestResetSec(a);
            var vbReset = getCodexSoonestResetSec(b);
            if (vaReset < 0) vaReset = 999999999;
            if (vbReset < 0) vbReset = 999999999;
            if (vaReset <= 0) vaReset = 0;
            if (vbReset <= 0) vbReset = 0;
            if (vaReset === vbReset) {
              va = getCodexClaudeStatus(a) === "ready" ? 0 : getCodexClaudeStatus(a) === "low" ? 1 : 2;
              vb = getCodexClaudeStatus(b) === "ready" ? 0 : getCodexClaudeStatus(b) === "low" ? 1 : 2;
            } else {
              va = vaReset;
              vb = vbReset;
            }
            break;
          default:
            return 0;
        }
      } else if (provider === "cursor") {
        switch (col) {
          case "account":
            va = a.email || "";
            vb = b.email || "";
            break;
          case "plan":
            va = a.planTier || "";
            vb = b.planTier || "";
            break;
          case "premiumUsed":
            va = a.billingModel === "usd_credit" ? a.usedCents || 0 : a.requestsUsed || 0;
            vb = b.billingModel === "usd_credit" ? b.usedCents || 0 : b.requestsUsed || 0;
            break;
          case "usage":
            va = a.usagePct || 0;
            vb = b.usagePct || 0;
            break;
          case "lastsnap":
            va = a.capturedAt ? new Date(a.capturedAt).getTime() : 0;
            vb = b.capturedAt ? new Date(b.capturedAt).getTime() : 0;
            break;
          case "status":
            var vaReset = getCursorSoonestResetSec(a);
            var vbReset = getCursorSoonestResetSec(b);
            if (vaReset < 0) vaReset = 999999999;
            if (vbReset < 0) vbReset = 999999999;
            if (vaReset <= 0) vaReset = 0;
            if (vbReset <= 0) vbReset = 0;
            if (vaReset === vbReset) {
              va = getCursorStatus(a) === "ready" ? 0 : getCursorStatus(a) === "low" ? 1 : 2;
              vb = getCursorStatus(b) === "ready" ? 0 : getCursorStatus(b) === "low" ? 1 : 2;
            } else {
              va = vaReset;
              vb = vbReset;
            }
            break;
          default:
            return 0;
        }
      } else if (provider === "copilot") {
        switch (col) {
          case "account":
            va = a.username || a.email || "";
            vb = b.username || b.email || "";
            break;
          case "plan":
            va = a.plan || "";
            vb = b.plan || "";
            break;
          case "premium":
            va = a.premiumPct || 0;
            vb = b.premiumPct || 0;
            break;
          case "chat":
            va = a.chatPct || 0;
            vb = b.chatPct || 0;
            break;
          case "lastsnap":
            va = a.capturedAt ? new Date(a.capturedAt).getTime() : 0;
            vb = b.capturedAt ? new Date(b.capturedAt).getTime() : 0;
            break;
          case "status":
            va = getCopilotStatus(a) === "ready" ? 0 : getCopilotStatus(a) === "low" ? 1 : 2;
            vb = getCopilotStatus(b) === "ready" ? 0 : getCopilotStatus(b) === "low" ? 1 : 2;
            break;
          default:
            return 0;
        }
      } else {
        return 0;
      }
      if (va === vb) return 0;
      var res = va > vb ? 1 : -1;
      return dir === "asc" ? res : -res;
    });
  }
  function filterAccountsArray(accounts) {
    var searchInput = document.getElementById("quota-search");
    var statusFilter = document.getElementById("quota-filter-status");
    var query = searchInput ? searchInput.value.toLowerCase() : "";
    var status = statusFilter ? statusFilter.value : "all";
    return accounts.filter(function(acc) {
      var matchesSearch = !query || acc.email.toLowerCase().includes(query) || (acc.planName || "").toLowerCase().includes(query);
      var matchesStatus = true;
      if (status === "ready") matchesStatus = acc.isReady;
      else if (status === "low") matchesStatus = !acc.isReady && !allExhausted(acc);
      else if (status === "empty") matchesStatus = allExhausted(acc);
      var matchesTag = true;
      if (activeTagFilter) {
        var accTags = (acc.tags || "").split(",").map(function(t) {
          return t.trim().toLowerCase();
        });
        matchesTag = accTags.indexOf(activeTagFilter) >= 0;
      }
      return matchesSearch && matchesStatus && matchesTag;
    });
  }
  function updateSortHeaders() {
    document.querySelectorAll(".grid-header .sortable").forEach(function(el) {
      el.classList.remove("sort-active");
      var span = el.querySelector(".sort-indicator");
      if (span) span.textContent = "";
      var providerSection = el.closest(".provider-section");
      var provider = providerSection ? providerSection.dataset.provider : "antigravity";
      var state = quotaSortStates[provider || "antigravity"] || quotaSortState;
      if (el.dataset.sort === state.column) {
        el.classList.add("sort-active");
        if (span) span.textContent = state.direction === "asc" ? "\u25BE" : "\u25B4";
      }
    });
  }
  function getHumanReadableBasis(basis) {
    if (!basis) return "";
    switch (basis) {
      case "contains_post_reset_estimate":
        return "Estimated models restored following their reset window";
      case "sprint_reset_high":
        return "Optimistic 100% estimate (reset occurred <30m ago)";
      case "sprint_reset_medium":
        return "Optimistic 100% estimate (reset occurred <6h ago)";
      case "sprint_reset_low":
        return "Optimistic 100% estimate (reset occurred <24h ago)";
      case "sprint_reset_stale":
        return "Stale reset window (fallback to last observed value)";
      case "snapshot_too_stale":
        return "Snapshot is too old to confidently estimate";
      case "post_reset_estimate":
        return "Usage reset to 0% following provider reset boundary";
      default:
        return basis.replace(/_/g, " ");
    }
  }
  function getHumanReadableConfidence(confidence) {
    if (!confidence) return "";
    switch (confidence) {
      case "high":
        return "High (Recent reset or fresh capture)";
      case "medium":
        return "Medium (Less than 6h since reset/activity)";
      case "low":
        return "Low (Up to 24h since reset/activity)";
      case "very_low":
        return "Very Low (Stale data; needs refresh)";
      default:
        return confidence.charAt(0).toUpperCase() + confidence.slice(1);
    }
  }
  function renderQualityBadge(item) {
    if (!item) return "";
    var label = "";
    if (item.unavailableReason) label = "Unavailable";
    else if (item.isEstimated) {
      if (item.basis === "snapshot_too_stale") label = "Stale";
      else if (item.confidence === "very_low") label = "Stale";
      else return "";
    } else if (item.confidence && item.confidence !== "high") label = item.confidence + " confidence";
    if (!label) return "";
    var titleParts = [];
    if (item.basis) titleParts.push("Basis: " + getHumanReadableBasis(item.basis));
    if (item.confidence) titleParts.push("Confidence: " + getHumanReadableConfidence(item.confidence));
    if (item.unavailableReason) titleParts.push("Unavailable: " + item.unavailableReason);
    return '<span class="data-quality-badge" title="' + esc(titleParts.join(" | ")) + '">' + esc(label) + "</span>";
  }
  function getUniqueTagsFromData(data) {
    var tagCounts = {};
    var accounts = data.accounts || [];
    for (var i = 0; i < accounts.length; i++) {
      var tags = (accounts[i].tags || "").split(",");
      for (var j = 0; j < tags.length; j++) {
        var t = tags[j].trim().toLowerCase();
        if (t) {
          tagCounts[t] = (tagCounts[t] || 0) + 1;
        }
      }
    }
    return tagCounts;
  }
  function renderTagFilterStrip(data) {
    var strip = document.getElementById("tag-filter-strip");
    if (!strip) return;
    var tagCounts = getUniqueTagsFromData(data);
    var tagNames = Object.keys(tagCounts).sort();
    if (tagNames.length === 0) {
      if (activeTagFilter) {
        setActiveTagFilter(null);
      }
      strip.innerHTML = "";
      return;
    }
    if (activeTagFilter && tagNames.indexOf(activeTagFilter) < 0) {
      setActiveTagFilter(null);
    }
    var html = '<span class="tag-filter-label">\u{1F3F7}\uFE0F Filter:</span>';
    var allActive = !activeTagFilter ? " active" : "";
    var totalAccounts = (data.accounts || []).length;
    html += '<button class="tag-filter-chip' + allActive + '" data-tag-filter="">All <span class="tag-filter-count">' + totalAccounts + "</span></button>";
    for (var i = 0; i < tagNames.length; i++) {
      var tag = tagNames[i];
      var isActive = activeTagFilter === tag ? " active" : "";
      html += '<button class="tag-filter-chip' + isActive + '" data-tag-filter="' + esc(tag) + '">' + esc(tag) + ' <span class="tag-filter-count">' + tagCounts[tag] + "</span></button>";
    }
    strip.innerHTML = html;
  }
  function handleTagFilterClick(e) {
    var chip = e.target.closest(".tag-filter-chip");
    if (!chip) return;
    var tag = chip.getAttribute("data-tag-filter");
    setActiveTagFilter(tag || null);
    if (latestQuotaData) {
      renderTagFilterStrip(latestQuotaData);
      renderAccounts(latestQuotaData);
    }
  }
  function renderAccounts(data) {
    setLatestQuotaData(data);
    var grid = document.getElementById("account-grid");
    var countBadge = document.getElementById("account-count");
    var snapCount = document.getElementById("snap-count");
    if (!grid) return;
    renderTagFilterStrip(data);
    var statusFilterEl = document.getElementById("quota-filter-status");
    var providerFilterEl = document.getElementById("quota-filter-provider");
    if (statusFilterEl) statusFilterEl.classList.toggle("filter-active", statusFilterEl.value !== "all");
    if (providerFilterEl) providerFilterEl.classList.toggle("filter-active", providerFilterEl.value !== "all");
    var acctCount = (data.accounts || []).length;
    var parts = [];
    if (acctCount > 0) parts.push(acctCount + " Antigravity");
    if (data.codexSnapshots && data.codexSnapshots.length > 0) parts.push(data.codexSnapshots.length + " Codex");
    if (data.claudeSnapshot) parts.push("1 Claude");
    if (data.cursorSnapshots && data.cursorSnapshots.length > 0) parts.push(data.cursorSnapshots.length + " Cursor");
    if (data.copilotSnapshots && data.copilotSnapshots.length > 0) parts.push(data.copilotSnapshots.length + " Copilot");
    if (countBadge) countBadge.textContent = parts.join(" \xB7 ") || "0 accounts";
    if (snapCount) snapCount.textContent = data.snapshotCount ? data.snapshotCount + " snapshots" : "";
    if (acctCount === 0 && (!data.codexSnapshots || data.codexSnapshots.length === 0) && !data.claudeSnapshot && (!data.cursorSnapshots || data.cursorSnapshots.length === 0) && (!data.copilotSnapshots || data.copilotSnapshots.length === 0)) {
      grid.innerHTML = '<div class="empty-state"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/><path d="M12 2v4M12 18v4M2 12h4M18 12h4"/></svg><p>No accounts tracked yet</p><p class="empty-hint">Click <strong>Snap Now</strong> to capture your first snapshot</p></div>';
      return;
    }
    var providerFilter = document.getElementById("quota-filter-provider");
    var pf = providerFilter ? providerFilter.value : "all";
    var html = "";
    if (acctCount > 0 && (pf === "all" || pf === "antigravity")) {
      var filtered = filterAccountsArray(data.accounts);
      var sorted = sortAccountsArray(filtered);
      var agCollapseClass = collapsedProviders.has("section-antigravity") ? " collapsed" : "";
      var agChevron = collapsedProviders.has("section-antigravity") ? "\u25B8" : "\u25BE";
      var agReadyCount = 0;
      for (var i = 0; i < acctCount; i++) {
        if (data.accounts[i].isReady) agReadyCount++;
      }
      html += '<div class="provider-section" data-provider="antigravity"><div class="provider-header" data-toggle-provider="section-antigravity"><div class="provider-header-left"><span class="provider-chevron" id="pchev-section-antigravity">' + agChevron + '</span><span class="provider-name">Antigravity</span><span class="provider-count">' + acctCount + " account" + (acctCount !== 1 ? "s" : "") + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + agReadyCount + ' ready)</span></span></div></div><div class="provider-body' + agCollapseClass + '" id="section-antigravity">';
      html += '<div class="grid-header"><div class="grid-col-account sortable" data-sort="account">Account <span class="sort-indicator"></span></div>';
      for (var gh = 0; gh < GRID_COLUMNS.length; gh++) {
        html += '<div class="grid-col-group sortable" data-sort="' + GRID_COLUMNS[gh] + '">' + (GRID_LABELS[gh] || GRID_COLUMNS[gh]) + ' <span class="sort-indicator"></span></div>';
      }
      html += '<div class="grid-col-credits sortable" data-sort="credits">AI Credits <span class="sort-indicator"></span></div><div class="grid-col-snap sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div><div class="grid-col-status sortable" data-sort="resetsIn">Refresh In <span class="sort-indicator"></span></div></div>';
      for (var i = 0; i < sorted.length; i++) {
        var acc = sorted[i];
        var accId = "acc-" + acc.accountId;
        var isExpanded = expandedAccounts.has(accId);
        var groupCells = "";
        var modelIdsByGroup = {};
        var modelLabelsByGroup = {};
        if (acc.models) {
          for (var mi2 = 0; mi2 < acc.models.length; mi2++) {
            var mm = acc.models[mi2];
            var gk = mm.groupKey || "unknown";
            if (!modelIdsByGroup[gk]) modelIdsByGroup[gk] = [];
            if (!modelLabelsByGroup[gk]) modelLabelsByGroup[gk] = [];
            modelIdsByGroup[gk].push(mm.modelId || "");
            modelLabelsByGroup[gk].push(mm.label || mm.modelId);
          }
        }
        var pinnedKey = acc.pinnedGroup || "";
        var pinnedGroupData = null;
        if (pinnedKey) {
          var groups = acc.groups || [];
          for (var pg = 0; pg < groups.length; pg++) {
            if (groups[pg].groupKey === pinnedKey) {
              pinnedGroupData = groups[pg];
              break;
            }
          }
        }
        for (var gi = 0; gi < GRID_COLUMNS.length; gi++) {
          var key = GRID_COLUMNS[gi];
          var g = null;
          var groups = acc.groups || [];
          for (var gj = 0; gj < groups.length; gj++) {
            if (groups[gj].groupKey === key) {
              g = groups[gj];
              break;
            }
          }
          if (!g) {
            groupCells += '<div class="quota-cell"><span class="quota-pct">\u2014</span></div>';
            continue;
          }
          var pct = Math.round(g.remainingPercent);
          var cls = "good";
          if (g.isExhausted || pct === 0) cls = "exhausted";
          else if (pct < 20) cls = "warning";
          else if (pct < 50) cls = "ok";
          var barCls = cls;
          var groupModelIds = (modelIdsByGroup[key] || []).join("|||");
          var groupModelLabels = (modelLabelsByGroup[key] || []).join("|||");
          var groupAdjust = '<span class="group-adjust" data-snap-id="' + acc.latestSnapshotId + '" data-group-key="' + key + '" data-group-model-ids="' + esc(groupModelIds) + '" data-group-model-labels="' + esc(groupModelLabels) + '" data-current-pct="' + pct + '"><button class="gadj-btn" data-delta="-20" title="\u221220% all models in group">\u221220</button><button class="gadj-btn" data-delta="20" title="+20% all models in group">+20</button><button class="gadj-btn btn-custom" data-custom="true" title="Enter custom percentage for group">\u270F\uFE0F</button></span>';
          var tooltipParts = [];
          if (g.timeUntilResetSec > 0) {
            tooltipParts.push("Reset in: " + formatSeconds(g.timeUntilResetSec));
          }
          if (data.forecasts && data.forecasts[acc.accountId]) {
            var acctForecasts = data.forecasts[acc.accountId];
            for (var fi = 0; fi < acctForecasts.length; fi++) {
              if (acctForecasts[fi].groupKey === key && acctForecasts[fi].ttxLabel) {
                var ttxLabel = acctForecasts[fi].ttxLabel;
                if (ttxLabel && ttxLabel !== "") {
                  tooltipParts.push("TTX: " + ttxLabel);
                }
                break;
              }
            }
          }
          if (pct < 95 && data.estimatedCosts && data.estimatedCosts[acc.accountId]) {
            var acctCosts = data.estimatedCosts[acc.accountId];
            if (acctCosts.groups) {
              for (var ci = 0; ci < acctCosts.groups.length; ci++) {
                if (acctCosts.groups[ci].groupKey === key && acctCosts.groups[ci].hasData) {
                  var costVal = acctCosts.groups[ci].estimatedCost || 0;
                  if (costVal >= 0.01) {
                    var costLabel = acctCosts.groups[ci].costLabel || "\u2014";
                    var hourly = acctCosts.groups[ci].hourlyLabel ? " (" + acctCosts.groups[ci].hourlyLabel + ")" : "";
                    tooltipParts.push("Estimated Cost: " + costLabel + hourly);
                  }
                  break;
                }
              }
            }
          }
          var cellTitle = tooltipParts.join(" | ") || (GRID_LABELS[gi] || key);
          var pinnedStarHTML = key === pinnedKey ? '<span class="pinned-group-star" title="Pinned group" style="position: absolute; top: 4px; right: 4px; font-size: 12px; line-height: 1; z-index: 2;">\u2B50</span>' : "";
          groupCells += '<div class="quota-cell' + (key === "gemini_unified" ? " unified-pool-cell" : "") + '" title="' + esc(cellTitle) + '" style="display: flex; flex-direction: column; align-items: center; justify-content: center; position: relative;">' + pinnedStarHTML + '<span class="quota-pct ' + cls + '">' + (g.isEstimated ? "~" : "") + pct + "%</span>" + renderQualityBadge(g) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + barCls + '" style="width:' + pct + '%"></div></div>' + (g.timeUntilResetSec > 0 ? '<div class="reset-timer">\u21BB ' + formatSeconds(g.timeUntilResetSec) + "</div>" : "") + groupAdjust + "</div>";
        }
        var dotCls = "dot-ready";
        var badgeText = "Ready";
        if (allExhausted(acc)) {
          dotCls = "dot-empty";
          badgeText = "Empty";
        } else if (!acc.isReady) {
          dotCls = "dot-low";
          badgeText = "Low";
        }
        var creditsCell = '<div class="credits-cell" style="position:relative">';
        if (acc.aiCredits && acc.aiCredits.length > 0) {
          var credits = acc.aiCredits[0].creditAmount;
          var creditCls = credits > 500 ? "good" : credits > 100 ? "ok" : "warning";
          creditsCell += '<span class="credit-amount ' + creditCls + '" title="AI Credits">\u2726 ' + formatCredits(credits) + "</span>";
          creditsCell += renderCreditRenewal(acc.accountId, acc.creditRenewalDay);
        } else {
          creditsCell += '<span class="credit-amount muted">\u2014</span>';
        }
        creditsCell += "</div>";
        var modelsHTML = "";
        if (acc.models && acc.models.length > 0) {
          var groupedModels = {};
          for (var mi = 0; mi < acc.models.length; mi++) {
            var m = acc.models[mi];
            var gk2 = m.groupKey || "unknown";
            if (!groupedModels[gk2]) groupedModels[gk2] = [];
            groupedModels[gk2].push(m);
          }
          var modelRows = "";
          for (var goi = 0; goi < GROUP_ORDER.length; goi++) {
            var groupKey2 = GROUP_ORDER[goi];
            var groupModels = groupedModels[groupKey2];
            if (!groupModels || groupModels.length === 0) continue;
            var isPinned = pinnedKey === groupKey2;
            var starCls = isPinned ? "pin-star pinned" : "pin-star";
            var starTitle = isPinned ? "Pinned \u2014 click to unpin" : "Click to pin this group";
            var starChar = isPinned ? "\u2605" : "\u2606";
            var g2 = null;
            var groups2 = acc.groups || [];
            for (var gj2 = 0; gj2 < groups2.length; gj2++) {
              if (groups2[gj2].groupKey === groupKey2) {
                g2 = groups2[gj2];
                break;
              }
            }
            var expandedBadges = "";
            if (g2) {
              var pct2 = Math.round(g2.remainingPercent);
              if (g2.timeUntilResetSec > 0) {
                expandedBadges += ' <span class="quota-reset" style="margin-left:8px">\u21BB ' + formatSeconds(g2.timeUntilResetSec) + "</span>";
              }
              if (data.forecasts && data.forecasts[acc.accountId]) {
                var acctForecasts2 = data.forecasts[acc.accountId];
                for (var fi2 = 0; fi2 < acctForecasts2.length; fi2++) {
                  if (acctForecasts2[fi2].groupKey === groupKey2 && acctForecasts2[fi2].ttxLabel) {
                    var ttxSev2 = acctForecasts2[fi2].severity || "safe";
                    var ttxLabel2 = acctForecasts2[fi2].ttxLabel;
                    if (ttxLabel2 && ttxLabel2 !== "" && ttxSev2 !== "none") {
                      expandedBadges += ' <span class="ttx-badge ttx-' + ttxSev2 + '" style="margin-left:6px" title="Time to exhaustion at current burn rate">' + esc(ttxLabel2) + "</span>";
                    }
                    break;
                  }
                }
              }
              if (pct2 < 95 && data.estimatedCosts && data.estimatedCosts[acc.accountId]) {
                var acctCosts2 = data.estimatedCosts[acc.accountId];
                if (acctCosts2.groups) {
                  for (var ci2 = 0; ci2 < acctCosts2.groups.length; ci2++) {
                    if (acctCosts2.groups[ci2].groupKey === groupKey2 && acctCosts2.groups[ci2].hasData) {
                      var costVal2 = acctCosts2.groups[ci2].estimatedCost || 0;
                      if (costVal2 >= 0.01) {
                        var costLabel2 = acctCosts2.groups[ci2].costLabel || "\u2014";
                        var costCls2 = "cost-low";
                        if (costVal2 >= 10) costCls2 = "cost-high";
                        else if (costVal2 >= 3) costCls2 = "cost-medium";
                        var costTitle2 = "Estimated cost this cycle";
                        if (acctCosts2.groups[ci2].hourlyLabel) {
                          costTitle2 += " (" + acctCosts2.groups[ci2].hourlyLabel + ")";
                        }
                        expandedBadges += ' <span class="cost-badge ' + costCls2 + '" style="margin-left:6px" title="' + costTitle2 + '">' + esc(costLabel2) + "</span>";
                      }
                      break;
                    }
                  }
                }
              }
            }
            modelRows += '<div class="model-group-header"><button class="' + starCls + '" data-pin-group="' + groupKey2 + '" data-pin-account="' + acc.accountId + '" title="' + starTitle + '">' + starChar + '</button><span class="model-group-name" style="color:' + (GROUP_COLORS[groupKey2] || "var(--text-secondary)") + '">' + (GROUP_NAMES[groupKey2] || groupKey2) + "</span>" + expandedBadges + "</div>";
            for (var mi3 = 0; mi3 < groupModels.length; mi3++) {
              var m = groupModels[mi3];
              var mpct = Math.round(m.remainingPercent);
              var mcls = "good";
              if (m.isExhausted || mpct === 0) mcls = "exhausted";
              else if (mpct < 20) mcls = "warning";
              else if (mpct < 50) mcls = "ok";
              var color = GROUP_COLORS[m.groupKey] || "#94a3b8";
              var resetStr = m.resetSeconds > 0 ? "\u21BB " + formatSeconds(m.resetSeconds) : "";
              var intellBadges = "";
              var usageModels = usageDataCache ? usageDataCache.models : void 0;
              if (usageModels) {
                for (var ui = 0; ui < usageModels.length; ui++) {
                  var um = usageModels[ui];
                  if (um.modelId === m.modelId && um.accountId === acc.accountId && um.hasIntelligence) {
                    var rateStr = (um.currentRate * 100).toFixed(1) + "%/hr";
                    intellBadges += '<span class="rate-badge" title="Current consumption rate">' + rateStr + "</span>";
                    if (um.projectedUsage > 0) {
                      var projPct = Math.round(um.projectedUsage * 100);
                      var projCls = projPct > 95 ? "proj-danger" : projPct > 80 ? "proj-warn" : "proj-ok";
                      intellBadges += '<span class="proj-badge ' + projCls + '" title="Projected usage at reset">\u2192' + projPct + "%</span>";
                    }
                    if (um.projectedExhaustion) {
                      var exhaust = new Date(um.projectedExhaustion);
                      var minsLeft = Math.round((exhaust.getTime() - Date.now()) / 6e4);
                      if (minsLeft > 0) {
                        intellBadges += '<span class="exhaust-badge" title="Projected exhaustion time">\u26A0 ' + (minsLeft > 60 ? Math.round(minsLeft / 60) + "h" : minsLeft + "m") + "</span>";
                      }
                    }
                    break;
                  }
                }
              }
              var adjustBtns = '<span class="adjust-controls" data-snap-id="' + acc.latestSnapshotId + '" data-model-id="' + esc(m.modelId || "") + '" data-model-label="' + esc(m.label || m.modelId) + '" data-current-pct="' + mpct + '"><button class="adj-btn" data-delta="-20" title="\u221220%">\u221220</button><button class="adj-btn" data-delta="20" title="+20%">+20</button><button class="adj-btn btn-custom" data-custom="true" title="Enter custom percentage">\u270F\uFE0F</button></span>';
              modelRows += '<div class="model-row"><div class="model-indicator" style="background:' + color + '"></div><span class="model-label">' + esc(m.label || m.modelId) + '</span><div class="model-bar-track"><div class="model-bar-fill ' + mcls + '" style="width:' + mpct + '%"></div></div><span class="model-pct ' + mcls + '">' + (m.isEstimated ? "~" : "") + mpct + "%</span>" + renderQualityBadge(m) + adjustBtns + '<span class="model-reset">' + resetStr + "</span>" + intellBadges + "</div>";
            }
          }
          var expandedCls = isExpanded ? " is-expanded" : "";
          modelsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '">' + modelRows + '<div class="account-actions"><button class="btn-clear-snaps btn-warning" data-clear-account="' + acc.accountId + '" data-clear-email="' + esc(acc.email) + '" title="Delete all snapshots for this account">Clear Snapshots</button><button class="btn-delete-account btn-danger" data-delete-account="' + acc.accountId + '" data-delete-email="' + esc(acc.email) + '" title="Remove account and all its data">Remove Account</button></div></div>';
        }
        var chevronCls = isExpanded ? "chevron expanded" : "chevron";
        var statusClass = "";
        if (allExhausted(acc)) statusClass = " status-empty";
        else if (!acc.isReady) statusClass = " status-low";
        else statusClass = " status-ready";
        html += '<div class="account-card' + statusClass + '"><div class="account-row" data-toggle="' + accId + '"><div class="account-info"><div class="account-email"><span class="' + chevronCls + '" id="chev-' + accId + '">\u25B8</span> ' + esc(acc.email) + '</div><div class="account-meta" style="position:relative">' + (acc.planName ? '<span class="plan-badge">' + esc(acc.planName) + "</span>" : "") + renderAccountTags(acc) + renderAccountNote(acc) + "</div></div>" + groupCells + creditsCell + '<div class="snap-cell"><span class="snap-ago" title="' + esc(acc.lastSeen || "") + '">' + esc(acc.stalenessLabel) + '</span></div><div class="status-cell"><span class="health-dot ' + dotCls + '">\u25CF' + (function() {
          var rs = getSoonestResetSec(acc);
          if (rs <= 0) return "";
          return " \u21BB " + formatSeconds(rs);
        })() + "</span></div></div>" + modelsHTML + "</div>";
      }
      html += "</div></div>";
    }
    var sf = document.getElementById("quota-filter-status");
    var statusVal = sf ? sf.value : "all";
    if (data.codexSnapshots && data.codexSnapshots.length > 0 && (pf === "all" || pf === "codex")) {
      html += renderCodexProviderSection(data.codexSnapshots, statusVal, data.allAccounts || []);
    }
    if (data.claudeSnapshot && (pf === "all" || pf === "claude")) {
      var clStatus = getCodexClaudeStatus(data.claudeSnapshot);
      if (statusVal === "all" || clStatus === statusVal) {
        html += renderClaudeProviderSection(data.claudeSnapshot);
      }
    }
    if (data.cursorSnapshots && data.cursorSnapshots.length > 0 && (pf === "all" || pf === "cursor")) {
      html += renderCursorProviderSection(data.cursorSnapshots, statusVal, data.allAccounts || []);
    }
    if (data.copilotSnapshots && data.copilotSnapshots.length > 0 && (pf === "all" || pf === "copilot")) {
      html += renderCopilotProviderSection(data.copilotSnapshots, statusVal, data.allAccounts || []);
    }
    if (pf === "antigravity" && acctCount === 0) {
      html += '<div class="provider-empty-state" data-provider="antigravity"><span class="provider-empty-icon">\u26A1</span><p>No Antigravity accounts detected</p><p class="empty-hint">Open Windsurf and log in to start tracking quotas</p></div>';
    }
    if (pf === "codex" && (!data.codexSnapshots || data.codexSnapshots.length === 0)) {
      html += '<div class="provider-empty-state" data-provider="codex"><span class="provider-empty-icon">\u{1F916}</span><p>No Codex snapshots yet</p><p class="empty-hint">Install Codex CLI and click <strong>Snap Now</strong> to capture</p></div>';
    }
    if (pf === "claude" && !data.claudeSnapshot) {
      var claudeStatus = data.claudeStatus;
      var clInstalled = claudeStatus && claudeStatus.installed;
      var clBridge = claudeStatus && claudeStatus.bridgeEnabled;
      var clHint = "";
      if (!clInstalled) {
        clHint = '<p class="empty-hint">Install <a href="https://docs.anthropic.com/en/docs/claude-code/overview" target="_blank" style="color:var(--accent)">Claude Code</a> to start tracking quotas</p>';
      } else if (!clBridge) {
        clHint = `<p class="empty-hint">Claude Code detected but bridge is disabled</p><button class="snap-btn" onclick="fetch('/api/config',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({key:'claude_bridge',value:'true'})}).then(function(){location.reload()})" style="margin-top:8px">Enable Tracking</button>`;
      } else {
        clHint = '<p class="empty-hint">Bridge is active \u2014 start a Claude Code session, then click Snap</p><button class="snap-btn" id="claude-snap-btn" style="margin-top:8px">\u26A1 Snap Now</button>';
      }
      html += '<div class="provider-empty-state" data-provider="claude"><span class="provider-empty-icon">\u{1F517}</span><p>No Claude Code quota data yet</p>' + clHint + "</div>";
    }
    if (pf === "cursor" && (!data.cursorSnapshots || data.cursorSnapshots.length === 0)) {
      html += '<div class="provider-empty-state" data-provider="cursor"><span class="provider-empty-icon">\u{1F5B1}\uFE0F</span><p>No Cursor data yet</p><p class="empty-hint">Enable Cursor capture in <strong>Settings</strong> or click <strong>Snap Now</strong></p></div>';
    }
    if (pf === "copilot" && (!data.copilotSnapshots || data.copilotSnapshots.length === 0)) {
      html += '<div class="provider-empty-state" data-provider="copilot"><span class="provider-empty-icon">\u{1F419}</span><p>No GitHub Copilot data yet</p><p class="empty-hint">Add a PAT in <strong>Settings</strong> or click <strong>Snap Now</strong></p></div>';
    }
    var eligibleAccount = (data.accounts || []).find(function(acc2) {
      return acc2.planTier && acc2.planTier.toLowerCase() === "ultra" && acc2.hasClaimedBonus2026 === 0;
    });
    var bannerHTML = "";
    if (eligibleAccount) {
      bannerHTML = '<div class="io-alert-card" data-account-id="' + eligibleAccount.accountId + '"><div class="io-alert-content"><div class="io-alert-title">\u2728 Google I/O 2026 Promotional Bonus</div><div class="io-alert-desc">Exclusive for Ultra members: Claim your $100 Overage Credit Bonus before it expires on <strong>May 25, 2026</strong>.</div></div><button class="io-claim-btn" data-claim-account-id="' + eligibleAccount.accountId + '">Claim $100 Bonus</button></div>';
    }
    grid.innerHTML = bannerHTML + html;
    var claimBtn = grid.querySelector(".io-claim-btn");
    if (claimBtn) {
      claimBtn.addEventListener("click", function(e) {
        var btn = e.currentTarget;
        var accId2 = parseInt(btn.getAttribute("data-claim-account-id") || "0", 10);
        if (accId2 > 0) {
          btn.disabled = true;
          btn.textContent = "Claiming...";
          claimOverageBonus(accId2).then(function() {
            showToast("\u2728 $100 Overage Bonus credit added!", "success");
            fetchStatus().then(function(freshData) {
              document.dispatchEvent(new CustomEvent("niyantra:status-refreshed", { detail: { data: freshData } }));
              document.dispatchEvent(new CustomEvent("niyantra:overview-refresh"));
            }).catch(function() {
            });
          }).catch(function(err) {
            btn.disabled = false;
            btn.textContent = "Claim $100 Bonus";
            showToast("\u274C " + (err.message || "Claim failed"), "error");
          });
        }
      });
    }
    grid.querySelectorAll(".provider-header[data-toggle-provider]").forEach(function(hdr) {
      hdr.addEventListener("click", function() {
        var targetId = hdr.dataset.toggleProvider;
        var body = document.getElementById(targetId);
        var chev = document.getElementById("pchev-" + targetId);
        if (!body) return;
        var collapsed = body.classList.toggle("collapsed");
        if (chev) chev.textContent = collapsed ? "\u25B8" : "\u25BE";
        if (collapsed) {
          collapsedProviders.add(targetId);
        } else {
          collapsedProviders.delete(targetId);
        }
        localStorage.setItem("niyantra_collapsed_providers", JSON.stringify(Array.from(collapsedProviders)));
      });
    });
    updateSortHeaders();
    var claudeSnapBtn = document.getElementById("claude-snap-btn");
    if (claudeSnapBtn) {
      claudeSnapBtn.addEventListener("click", function() {
        var btn = claudeSnapBtn;
        btn.disabled = true;
        btn.textContent = "Snapping...";
        fetch("/api/claude/snap", { method: "POST" }).then(function(r) {
          if (!r.ok) return r.json().then(function(e) {
            throw new Error(e.error || "Snap failed");
          });
          return r.json();
        }).then(function() {
          showToast("\u2705 Claude Code snapshot captured", "success");
          fetchStatus().then(function(freshData) {
            document.dispatchEvent(new CustomEvent("niyantra:status-refreshed", { detail: { data: freshData } }));
          }).catch(function() {
          });
        }).catch(function(err) {
          btn.disabled = false;
          btn.textContent = "\u26A1 Snap Now";
          showToast("\u274C " + (err.message || "Snap failed"), "error");
        });
      });
    }
  }
  function renderCodexProviderSection(codexSnaps, statusFilter, allAccounts = []) {
    var cxCollapseClass = collapsedProviders.has("section-codex") ? " collapsed" : "";
    var cxChevron = collapsedProviders.has("section-codex") ? "\u25B8" : "\u25BE";
    var cxReadyCount = 0;
    for (var i = 0; i < codexSnaps.length; i++) {
      if (getCodexClaudeStatus(codexSnaps[i]) === "ready") cxReadyCount++;
    }
    var html = '<div class="provider-section" data-provider="codex"><div class="provider-header" data-toggle-provider="section-codex"><div class="provider-header-left"><span class="provider-chevron" id="pchev-section-codex">' + cxChevron + '</span><span class="provider-name">\u{1F916} Codex / ChatGPT</span><span class="provider-count">' + codexSnaps.length + " account" + (codexSnaps.length !== 1 ? "s" : "") + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + cxReadyCount + ' ready)</span></span></div></div><div class="provider-body' + cxCollapseClass + '" id="section-codex"><div class="grid-header grid-codex"><div class="sortable" data-sort="account">Account <span class="sort-indicator"></span></div><div class="sortable" data-sort="plan">Plan <span class="sort-indicator"></span></div><div class="sortable" data-sort="fiveHour">Short-Term <span class="sort-indicator"></span></div><div class="sortable" data-sort="sevenDay">Weekly <span class="sort-indicator"></span></div><div class="sortable" data-sort="credits">Credits <span class="sort-indicator"></span></div><div class="sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div><div class="sortable" data-sort="status">Refresh In <span class="sort-indicator"></span></div></div>';
    var sortedSnaps = sortProviderArray(codexSnaps, "codex");
    var renderedCount = 0;
    for (var i = 0; i < sortedSnaps.length; i++) {
      var cs = sortedSnaps[i];
      var cxStatus = getCodexClaudeStatus(cs);
      if (statusFilter !== "all" && cxStatus !== statusFilter) continue;
      renderedCount++;
      var fiveUsed = cs.fiveHourPct || 0;
      var fiveRem = Math.max(0, 100 - fiveUsed);
      var fiveCls = fiveRem > 50 ? "good" : fiveRem > 20 ? "ok" : fiveRem > 0 ? "warning" : "exhausted";
      var fiveReset = cs.fiveHourReset ? formatResetTime(cs.fiveHourReset) : "";
      var sevenUsed = cs.sevenDayPct ? cs.sevenDayPct : 0;
      var sevenRem = Math.max(0, 100 - sevenUsed);
      var sevenCls = sevenRem > 50 ? "good" : sevenRem > 20 ? "ok" : sevenRem > 0 ? "warning" : "exhausted";
      var sevenReset = cs.sevenDayReset ? formatResetTime(cs.sevenDayReset) : "";
      var capturedAgo = cs.capturedAt ? formatTimeAgo(cs.capturedAt) : "\u2014";
      var dotCls = fiveUsed >= 80 || sevenUsed >= 80 ? "dot-low" : "dot-ready";
      var dotText = dotCls === "dot-ready" ? "Ready" : "Low";
      var displayName = cs.email || (cs.accountId && cs.accountId.length > 12 ? cs.accountId.substring(0, 6) + ".." + cs.accountId.slice(-6) : cs.accountId || "Codex");
      var creditsStr = cs.creditsBalance !== null && cs.creditsBalance !== void 0 ? cs.creditsBalance.toFixed(2) : String.fromCharCode(8212);
      var localAccId = cs.ownerAccountId || 0;
      var accId = "acc-codex-" + cs.id;
      var isExpanded = expandedAccounts.has(accId);
      var chevronCls = isExpanded ? "chevron expanded" : "chevron";
      var chevronHTML = localAccId > 0 ? '<span class="' + chevronCls + '" id="chev-' + accId + '">\u25B8</span> ' : "";
      var emailHTML = '<div class="account-email">' + chevronHTML + esc(displayName) + "</div>";
      var accData = localAccId > 0 ? allAccounts.find(function(a) {
        return a.id === localAccId;
      }) : null;
      var metaHTML = accData ? '<div class="account-meta" style="position:relative">' + renderAccountTags(accData) + renderAccountNote(accData) + "</div>" : "";
      var actionsHTML = "";
      if (localAccId > 0) {
        var expandedCls = isExpanded ? " is-expanded" : "";
        actionsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '"><div class="account-actions" style="margin-top:0"><button class="btn-clear-snaps" data-clear-account="' + localAccId + '" data-clear-email="' + esc(displayName) + '" title="Delete all snapshots for this account">Clear Snapshots</button><button class="btn-delete-account" data-delete-account="' + localAccId + '" data-delete-email="' + esc(displayName) + '" title="Remove account and all its data">Remove Account</button></div></div>';
      }
      var toggleAttr = localAccId > 0 ? ' data-toggle="' + accId + '"' : "";
      var statusClass = "";
      if (cxStatus === "empty") statusClass = " status-empty";
      else if (cxStatus === "low") statusClass = " status-low";
      else statusClass = " status-ready";
      html += '<div class="account-card' + statusClass + '"><div class="account-row grid-codex"' + toggleAttr + '><div class="account-info">' + emailHTML + metaHTML + "</div><div>" + (cs.planType ? '<span class="plan-badge">' + esc(cs.planType) + "</span>" : String.fromCharCode(8212)) + '</div><div class="quota-cell"><span class="quota-pct ' + fiveCls + '">' + estPrefix(isResetElapsed(cs.fiveHourReset)) + fiveRem.toFixed(0) + "%</span>" + renderProviderEstBadge(isResetElapsed(cs.fiveHourReset)) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + fiveCls + '" style="width:' + fiveRem + '%"></div></div>' + (fiveReset ? '<span class="quota-reset">\u21BB ' + fiveReset + "</span>" : "") + '</div><div class="quota-cell"><span class="quota-pct ' + sevenCls + '">' + estPrefix(isResetElapsed(cs.sevenDayReset)) + sevenRem.toFixed(0) + "%</span>" + renderProviderEstBadge(isResetElapsed(cs.sevenDayReset)) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + sevenCls + '" style="width:' + sevenRem + '%"></div></div>' + (sevenReset ? '<span class="quota-reset">\u21BB ' + sevenReset + "</span>" : "") + '</div><div class="credits-cell"><span class="credit-amount">' + creditsStr + '</span></div><div class="snap-cell"><span class="snap-ago">' + capturedAgo + '</span></div><div class="status-cell"><span class="health-dot ' + dotCls + '">\u25CF' + (function() {
        var rs = getCodexSoonestResetSec(cs);
        return rs > 0 ? " \u21BB " + formatSeconds(rs) : "";
      })() + "</span></div></div>" + actionsHTML + "</div>";
    }
    if (renderedCount === 0) return "";
    html += "</div></div>";
    return html;
  }
  function renderClaudeProviderSection(cl) {
    var clFive = cl.fiveHourPct || 0;
    var clFiveRem = Math.max(0, 100 - clFive);
    var clFiveCls = clFiveRem > 50 ? "good" : clFiveRem > 20 ? "ok" : clFiveRem > 0 ? "warning" : "exhausted";
    var clSeven = cl.sevenDayPct ? cl.sevenDayPct : 0;
    var clSevenRem = Math.max(0, 100 - clSeven);
    var clSevenCls = clSevenRem > 50 ? "good" : clSevenRem > 20 ? "ok" : clSevenRem > 0 ? "warning" : "exhausted";
    var clAgo = cl.capturedAt ? formatTimeAgo(cl.capturedAt) : "\u2014";
    var dotCls = clFive >= 80 || clSeven >= 80 ? "dot-low" : "dot-ready";
    var dotText = dotCls === "dot-ready" ? "Ready" : "Low";
    var clCollapseClass = collapsedProviders.has("section-claude") ? " collapsed" : "";
    var clChevron = collapsedProviders.has("section-claude") ? "\u25B8" : "\u25BE";
    var clStatus = getCodexClaudeStatus(cl);
    var statusClass = "";
    if (clStatus === "empty") statusClass = " status-empty";
    else if (clStatus === "low") statusClass = " status-low";
    else statusClass = " status-ready";
    return '<div class="provider-section" data-provider="claude"><div class="provider-header" data-toggle-provider="section-claude"><div class="provider-header-left"><span class="provider-chevron" id="pchev-section-claude">' + clChevron + '</span><span class="provider-name">\u{1F517} Claude Code</span><span class="provider-count">1 account \xB7 Bridge <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + (getCodexClaudeStatus(cl) === "ready" ? "1 ready" : "0 ready") + ')</span></span></div></div><div class="provider-body' + clCollapseClass + '" id="section-claude"><div class="grid-header grid-claude"><div>Source</div><div>Short-Term</div><div>Weekly</div><div>Last Snap</div><div>Refresh In</div></div><div class="account-card' + statusClass + '"><div class="account-row grid-claude"><div class="account-info"><div class="account-email">' + esc(cl.source || "statusline") + '</div></div><div class="quota-cell"><span class="quota-pct ' + clFiveCls + '">' + estPrefix(isResetElapsed(cl.fiveHourReset)) + clFiveRem.toFixed(0) + "%</span>" + renderProviderEstBadge(isResetElapsed(cl.fiveHourReset)) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + clFiveCls + '" style="width:' + clFiveRem + '%"></div></div></div><div class="quota-cell"><span class="quota-pct ' + clSevenCls + '">' + estPrefix(isResetElapsed(cl.sevenDayReset)) + clSevenRem.toFixed(0) + "%</span>" + renderProviderEstBadge(isResetElapsed(cl.sevenDayReset)) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + clSevenCls + '" style="width:' + clSevenRem + '%"></div></div></div><div class="snap-cell"><span class="snap-ago">' + clAgo + '</span></div><div class="status-cell"><span class="health-dot ' + dotCls + '">\u25CF' + (function() {
      var rs = getCodexSoonestResetSec(cl);
      return rs > 0 ? " \u21BB " + formatSeconds(rs) : "";
    })() + "</span></div></div></div></div></div>";
  }
  function formatResetTime(isoString) {
    if (!isoString) return "";
    var ts = /^\d+$/.test(isoString) ? parseInt(isoString) : isoString;
    var reset = new Date(ts);
    var now = /* @__PURE__ */ new Date();
    var diffSec = (reset.getTime() - now.getTime()) / 1e3;
    if (diffSec <= 0) return "now";
    return formatSeconds(diffSec);
  }
  function isResetElapsed(isoString) {
    if (!isoString) return false;
    var ts = /^\d+$/.test(isoString) ? parseInt(isoString) : isoString;
    return new Date(ts).getTime() < Date.now();
  }
  function renderProviderEstBadge(resetElapsed) {
    return "";
  }
  function estPrefix(resetElapsed) {
    return resetElapsed ? "~" : "";
  }
  function getCursorStatus(snap) {
    var usagePct = snap.usagePct || 0;
    var rem = Math.max(0, 100 - usagePct);
    if (rem === 0) return "empty";
    if (usagePct >= 80) return "low";
    return "ready";
  }
  function renderCursorProviderSection(cursorSnaps, statusFilter, allAccounts = []) {
    var crCollapseClass = collapsedProviders.has("section-cursor") ? " collapsed" : "";
    var crChevron = collapsedProviders.has("section-cursor") ? "\u25B8" : "\u25BE";
    var crReadyCount = 0;
    for (var i = 0; i < cursorSnaps.length; i++) {
      if (getCursorStatus(cursorSnaps[i]) === "ready") crReadyCount++;
    }
    var html = '<div class="provider-section" data-provider="cursor"><div class="provider-header" data-toggle-provider="section-cursor"><div class="provider-header-left"><span class="provider-chevron" id="pchev-section-cursor">' + crChevron + '</span><span class="provider-name">\u{1F5B1}\uFE0F Cursor</span><span class="provider-count">' + cursorSnaps.length + " account" + (cursorSnaps.length !== 1 ? "s" : "") + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + crReadyCount + ' ready)</span></span></div></div><div class="provider-body' + crCollapseClass + '" id="section-cursor"><div class="grid-header grid-cursor"><div class="sortable" data-sort="account">Account <span class="sort-indicator"></span></div><div class="sortable" data-sort="plan">Plan <span class="sort-indicator"></span></div><div class="sortable" data-sort="premiumUsed">Premium Used <span class="sort-indicator"></span></div><div class="sortable" data-sort="usage">Usage <span class="sort-indicator"></span></div><div class="sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div><div class="sortable" data-sort="status">Refresh In <span class="sort-indicator"></span></div></div>';
    var sortedSnaps = sortProviderArray(cursorSnaps, "cursor");
    var renderedCount = 0;
    for (var i = 0; i < sortedSnaps.length; i++) {
      var cs = sortedSnaps[i];
      var crStatus = getCursorStatus(cs);
      if (statusFilter !== "all" && crStatus !== statusFilter) continue;
      renderedCount++;
      var usagePct = cs.usagePct || 0;
      var remaining = Math.max(0, 100 - usagePct);
      var cls = remaining > 50 ? "good" : remaining > 20 ? "ok" : remaining > 0 ? "warning" : "exhausted";
      var capturedAgo = cs.capturedAt ? formatTimeAgo(cs.capturedAt) : "\u2014";
      var dotCls = usagePct >= 80 ? "dot-low" : "dot-ready";
      var dotText = dotCls === "dot-ready" ? "Ready" : "Low";
      var displayName = cs.email || "Cursor";
      var usedStr = String.fromCharCode(8212);
      var limitStr = String.fromCharCode(8212);
      if (cs.billingModel === "usd_credit") {
        if (cs.usedCents !== void 0 && cs.limitCents !== void 0) {
          usedStr = "$" + (cs.usedCents / 100).toFixed(2);
          limitStr = "$" + (cs.limitCents / 100).toFixed(2);
        }
      } else {
        if (cs.requestsUsed !== void 0 && cs.requestsMax !== void 0) {
          usedStr = cs.requestsUsed.toString();
          limitStr = cs.requestsMax.toString();
        }
      }
      var modelRows = "";
      if (cs.modelsJson && cs.modelsJson !== "{}") {
        try {
          var models = typeof cs.modelsJson === "string" ? JSON.parse(cs.modelsJson) : cs.modelsJson;
          var modelKeys = Object.keys(models);
          if (modelKeys.length > 0) {
            modelRows = '<div class="cursor-model-breakdown">';
            for (var mi = 0; mi < modelKeys.length; mi++) {
              var mKey = modelKeys[mi];
              var mVal = models[mKey];
              var mUsed = mVal.numRequests || 0;
              var mLimit = mVal.maxRequestUsage || 0;
              var mPct = mLimit > 0 ? mUsed / mLimit * 100 : 0;
              var mRem = Math.max(0, 100 - mPct);
              var mCls = mRem > 50 ? "good" : mRem > 20 ? "ok" : mRem > 0 ? "warning" : "exhausted";
              modelRows += '<div class="cursor-model-row"><span class="cursor-model-name">' + esc(mKey) + '</span><div class="quota-minibar"><div class="quota-minibar-fill ' + mCls + '" style="width:' + mRem + '%"></div></div><span class="cursor-model-usage">' + mUsed + "/" + mLimit + "</span></div>";
            }
            modelRows += "</div>";
          }
        } catch (e) {
        }
      }
      var localAccId = cs.accountId || 0;
      var accId = "acc-cursor-" + cs.id;
      var isExpanded = expandedAccounts.has(accId);
      var chevronCls = isExpanded ? "chevron expanded" : "chevron";
      var chevronHTML = localAccId > 0 || modelRows ? '<span class="' + chevronCls + '" id="chev-' + accId + '">\u25B8</span> ' : "";
      var emailHTML = '<div class="account-email">' + chevronHTML + esc(displayName) + "</div>";
      var accData = localAccId > 0 ? allAccounts.find(function(a) {
        return a.id === localAccId;
      }) : null;
      var metaHTML = accData ? '<div class="account-meta" style="position:relative">' + renderAccountTags(accData) + renderAccountNote(accData) + "</div>" : "";
      var actionsHTML = "";
      if (localAccId > 0 || modelRows) {
        var expandedCls = isExpanded ? " is-expanded" : "";
        var accountActions = localAccId > 0 ? '<div class="account-actions"><button class="btn-clear-snaps" data-clear-account="' + localAccId + '" data-clear-email="' + esc(displayName) + '" title="Delete all snapshots for this account">Clear Snapshots</button><button class="btn-delete-account" data-delete-account="' + localAccId + '" data-delete-email="' + esc(displayName) + '" title="Remove account and all its data">Remove Account</button></div>' : "";
        actionsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '">' + modelRows + accountActions + "</div>";
      }
      var toggleAttr = localAccId > 0 || modelRows ? ' data-toggle="' + accId + '"' : "";
      var statusClass = "";
      if (crStatus === "empty") statusClass = " status-empty";
      else if (crStatus === "low") statusClass = " status-low";
      else statusClass = " status-ready";
      html += '<div class="account-card' + statusClass + '"><div class="account-row grid-cursor"' + toggleAttr + '><div class="account-info">' + emailHTML + metaHTML + "</div><div>" + (cs.planTier ? '<span class="plan-badge">' + esc(cs.planTier) + "</span>" : String.fromCharCode(8212)) + '</div><div class="quota-cell"><span class="quota-pct ' + cls + '">' + usedStr + " / " + limitStr + "</span>" + renderProviderEstBadge(isResetElapsed(cs.cycleEnd)) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + cls + '" style="width:' + remaining + '%"></div></div></div><div class="quota-cell"><span class="quota-pct ' + cls + '">' + remaining.toFixed(0) + '% left</span></div><div class="snap-cell"><span class="snap-ago">' + capturedAgo + '</span></div><div class="status-cell"><span class="health-dot ' + dotCls + '">\u25CF' + (function() {
        var rs = getCursorSoonestResetSec(cs);
        return rs > 0 ? " \u21BB " + formatSeconds(rs) : "";
      })() + "</span></div></div>" + actionsHTML + "</div>";
    }
    if (renderedCount === 0) return "";
    html += "</div></div>";
    return html;
  }
  function getCopilotStatus(snap) {
    var premiumPct = snap.premiumPct || 0;
    var rem = Math.max(0, 100 - premiumPct);
    if (rem === 0) return "empty";
    if (premiumPct >= 80) return "low";
    return "ready";
  }
  function renderCopilotProviderSection(copilotSnaps, statusFilter, allAccounts = []) {
    var cpCollapseClass = collapsedProviders.has("section-copilot") ? " collapsed" : "";
    var cpChevron = collapsedProviders.has("section-copilot") ? "\u25B8" : "\u25BE";
    var cpReadyCount = 0;
    for (var i = 0; i < copilotSnaps.length; i++) {
      if (getCopilotStatus(copilotSnaps[i]) === "ready") cpReadyCount++;
    }
    var html = '<div class="provider-section" data-provider="copilot"><div class="provider-header" data-toggle-provider="section-copilot"><div class="provider-header-left"><span class="provider-chevron" id="pchev-section-copilot">' + cpChevron + '</span><span class="provider-name">\u{1F419} GitHub Copilot</span><span class="provider-count">' + copilotSnaps.length + " account" + (copilotSnaps.length !== 1 ? "s" : "") + ' <span style="opacity:0.6; margin-left:6px; font-weight:normal; font-size:0.95em;">(' + cpReadyCount + ' ready)</span></span></div></div><div class="provider-body' + cpCollapseClass + '" id="section-copilot"><div class="grid-header grid-copilot"><div class="sortable" data-sort="account">Account <span class="sort-indicator"></span></div><div class="sortable" data-sort="plan">Plan <span class="sort-indicator"></span></div><div class="sortable" data-sort="premium">Premium <span class="sort-indicator"></span></div><div class="sortable" data-sort="chat">Chat <span class="sort-indicator"></span></div><div class="sortable" data-sort="lastsnap">Last Snap <span class="sort-indicator"></span></div><div class="sortable" data-sort="status">Refresh In <span class="sort-indicator"></span></div></div>';
    var sortedSnaps = sortProviderArray(copilotSnaps, "copilot");
    var renderedCount = 0;
    for (var i = 0; i < sortedSnaps.length; i++) {
      var cp = sortedSnaps[i];
      var cpStatus = getCopilotStatus(cp);
      if (statusFilter !== "all" && cpStatus !== statusFilter) continue;
      renderedCount++;
      var premiumPct = cp.premiumPct || 0;
      var premiumRem = Math.max(0, 100 - premiumPct);
      var premiumCls = premiumRem > 50 ? "good" : premiumRem > 20 ? "ok" : premiumRem > 0 ? "warning" : "exhausted";
      var chatPct = cp.chatPct || 0;
      var chatRem = Math.max(0, 100 - chatPct);
      var chatCls = chatRem > 50 ? "good" : chatRem > 20 ? "ok" : chatRem > 0 ? "warning" : "exhausted";
      var capturedAgo = cp.capturedAt ? formatTimeAgo(cp.capturedAt) : "\u2014";
      var dotCls = premiumPct >= 80 ? "dot-low" : "dot-ready";
      var dotText = dotCls === "dot-ready" ? "Ready" : "Low";
      var displayName = cp.username || cp.email || "Copilot";
      var sourceLabel = "";
      if (cp.captureSource === "github_cli") {
        sourceLabel = '<span class="source-badge" title="Auto-detected via GitHub CLI" style="font-size: 10px; opacity: 0.7; border: 1px solid var(--border-color); padding: 1px 4px; border-radius: 4px; margin-left: 6px; display: inline-block; vertical-align: middle;">gh cli</span>';
      } else if (cp.captureSource === "hosts.json") {
        sourceLabel = '<span class="source-badge" title="Auto-detected via IDE hosts.json" style="font-size: 10px; opacity: 0.7; border: 1px solid var(--border-color); padding: 1px 4px; border-radius: 4px; margin-left: 6px; display: inline-block; vertical-align: middle;">ide</span>';
      } else if (cp.captureSource === "manual_pat") {
        sourceLabel = '<span class="source-badge" title="Configured via Settings PAT" style="font-size: 10px; opacity: 0.7; border: 1px solid var(--border-color); padding: 1px 4px; border-radius: 4px; margin-left: 6px; display: inline-block; vertical-align: middle;">pat</span>';
      }
      var localAccId = cp.accountId || 0;
      var accId = "acc-copilot-" + cp.id;
      var isExpanded = expandedAccounts.has(accId);
      var chevronCls = isExpanded ? "chevron expanded" : "chevron";
      var chevronHTML = localAccId > 0 ? '<span class="' + chevronCls + '" id="chev-' + accId + '">\u25B8</span> ' : "";
      var emailHTML = '<div class="account-email">' + chevronHTML + esc(displayName) + sourceLabel + "</div>";
      var accData = localAccId > 0 ? allAccounts.find(function(a) {
        return a.id === localAccId;
      }) : null;
      var metaHTML = accData ? '<div class="account-meta" style="position:relative">' + renderAccountTags(accData) + renderAccountNote(accData) + "</div>" : "";
      var actionsHTML = "";
      if (localAccId > 0) {
        var expandedCls = isExpanded ? " is-expanded" : "";
        actionsHTML = '<div class="model-details' + expandedCls + '" id="' + accId + '"><div class="account-actions" style="margin-top:0"><button class="btn-clear-snaps" data-clear-account="' + localAccId + '" data-clear-email="' + esc(displayName) + '" title="Delete all snapshots for this account">Clear Snapshots</button><button class="btn-delete-account" data-delete-account="' + localAccId + '" data-delete-email="' + esc(displayName) + '" title="Remove account and all its data">Remove Account</button></div></div>';
      }
      var toggleAttr = localAccId > 0 ? ' data-toggle="' + accId + '"' : "";
      var statusClass = "";
      if (cpStatus === "empty") statusClass = " status-empty";
      else if (cpStatus === "low") statusClass = " status-low";
      else statusClass = " status-ready";
      html += '<div class="account-card' + statusClass + '"><div class="account-row grid-copilot"' + toggleAttr + '><div class="account-info">' + emailHTML + metaHTML + "</div><div>" + (cp.plan ? '<span class="plan-badge">' + esc(cp.plan) + "</span>" : String.fromCharCode(8212)) + '</div><div class="quota-cell"><span class="quota-pct ' + premiumCls + '">' + premiumRem.toFixed(0) + "% left</span>" + renderProviderEstBadge(cp.capturedAt && new Date(cp.capturedAt).getUTCMonth() !== (/* @__PURE__ */ new Date()).getUTCMonth()) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + premiumCls + '" style="width:' + premiumRem + '%"></div></div></div><div class="quota-cell"><span class="quota-pct ' + chatCls + '">' + chatRem.toFixed(0) + "% left</span>" + renderProviderEstBadge(cp.capturedAt && new Date(cp.capturedAt).getUTCMonth() !== (/* @__PURE__ */ new Date()).getUTCMonth()) + '<div class="quota-minibar"><div class="quota-minibar-fill ' + chatCls + '" style="width:' + chatRem + '%"></div></div></div><div class="snap-cell"><span class="snap-ago">' + capturedAgo + '</span></div><div class="status-cell"><span class="health-dot ' + dotCls + '">\u25CF</span></div></div>' + actionsHTML + "</div>";
    }
    if (renderedCount === 0) return "";
    html += "</div></div>";
    return html;
  }

  // internal/web/src/quotas/expand.ts
  function setupToggle() {
    var grid = document.getElementById("account-grid");
    if (!grid) return;
    grid.addEventListener("click", function(e) {
      var clearBtn = e.target.closest("[data-clear-account]");
      if (clearBtn) {
        e.stopPropagation();
        var accountId = clearBtn.getAttribute("data-clear-account");
        var email = clearBtn.getAttribute("data-clear-email");
        if (confirm("Clear all snapshots for " + email + "?\n\nThe account will remain but all quota history will be deleted. This cannot be undone.")) {
          fetch("/api/accounts/" + accountId + "/snapshots", { method: "DELETE" }).then(function(res) {
            return res.json();
          }).then(function(data) {
            showToast("\u2705 Cleared " + (data.snapshotsDeleted || 0) + " snapshots for " + email, "success");
            fetchStatus().then(renderAccounts);
            document.dispatchEvent(new CustomEvent("niyantra:chart-refresh"));
          }).catch(function(err) {
            showToast("\u274C " + err.message, "error");
          });
        }
        return;
      }
      var deleteBtn = e.target.closest("[data-delete-account]");
      if (deleteBtn) {
        e.stopPropagation();
        var accountId2 = deleteBtn.getAttribute("data-delete-account");
        var email2 = deleteBtn.getAttribute("data-delete-email");
        if (confirm("Remove account " + email2 + "?\n\nThis deletes the account record plus data explicitly tied to its local account ID, such as Antigravity snapshots and reset cycles. Provider-native records may require separate cleanup. This cannot be undone.")) {
          fetch("/api/accounts/" + accountId2, { method: "DELETE" }).then(function(res) {
            return res.json();
          }).then(function(data) {
            showToast("\u2705 Removed " + email2 + " (" + (data.totalDeleted || 0) + " records deleted)", "success");
            expandedAccounts.delete("acc-" + accountId2);
            fetchStatus().then(renderAccounts);
            document.dispatchEvent(new CustomEvent("niyantra:chart-refresh"));
          }).catch(function(err) {
            showToast("\u274C " + err.message, "error");
          });
        }
        return;
      }
      var gadjBtn = e.target.closest(".gadj-btn");
      if (gadjBtn) {
        e.stopPropagation();
        var gControls = gadjBtn.closest(".group-adjust");
        if (!gControls) return;
        var gSnapId = parseInt(gControls.getAttribute("data-snap-id"), 10);
        var gGroupKey = gControls.getAttribute("data-group-key");
        var gModelIdsStr = gControls.getAttribute("data-group-model-ids");
        var gModelLabelsStr = gControls.getAttribute("data-group-model-labels");
        var gCurrentPct = parseFloat(gControls.getAttribute("data-current-pct"));
        var gNewPct;
        if (gadjBtn.getAttribute("data-custom") === "true") {
          var input = prompt("Enter custom remaining quota percentage (0-100) for all models in this group:", Math.round(gCurrentPct).toString());
          if (input === null) return;
          var parsed = parseInt(input.trim(), 10);
          if (isNaN(parsed) || parsed < 0 || parsed > 100) {
            showToast("\u274C Please enter a valid percentage between 0 and 100.", "error");
            return;
          }
          gNewPct = parsed;
        } else {
          var gDelta = parseFloat(gadjBtn.getAttribute("data-delta"));
          gNewPct = Math.max(0, Math.min(100, gCurrentPct + gDelta));
        }
        var cell = gControls.closest(".quota-cell");
        if (cell) {
          var gPctSpan = cell.querySelector(".quota-pct");
          var gBarFill = cell.querySelector(".quota-minibar-fill");
          if (gPctSpan) {
            gPctSpan.textContent = Math.round(gNewPct) + "%";
            gPctSpan.className = "quota-pct " + (gNewPct <= 0 ? "exhausted" : gNewPct < 20 ? "warning" : gNewPct < 50 ? "ok" : "good");
          }
          if (gBarFill) {
            gBarFill.style.width = gNewPct + "%";
            gBarFill.className = "quota-minibar-fill " + (gNewPct <= 0 ? "exhausted" : gNewPct < 20 ? "warning" : gNewPct < 50 ? "ok" : "good");
          }
        }
        gControls.setAttribute("data-current-pct", String(gNewPct));
        var gModelIds = gModelIdsStr.split("|||");
        var gModelLabels = gModelLabelsStr.split("|||");
        var adjustments = [];
        for (var li = 0; li < gModelLabels.length; li++) {
          if (!gModelIds[li] && !gModelLabels[li]) continue;
          adjustments.push({ modelId: gModelIds[li] || "", label: gModelLabels[li] || "", remainingPercent: gNewPct });
        }
        if (adjustments.length === 0) return;
        fetch("/api/snap/adjust", {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ snapshotId: gSnapId, adjustments })
        }).then(function(res) {
          return res.json();
        }).then(function(data) {
          if (data.error) {
            showToast("\u274C " + data.error, "error");
            return;
          }
          var groupName = GROUP_NAMES[gGroupKey] || gGroupKey;
          showToast("\u270E " + groupName + " \u2192 " + Math.round(gNewPct) + "% (" + adjustments.length + " models)", "info");
          fetchStatus().then(renderAccounts);
        }).catch(function(err) {
          showToast("\u274C " + err.message, "error");
        });
        return;
      }
      var adjBtn = e.target.closest(".adj-btn");
      if (adjBtn) {
        e.stopPropagation();
        var controls = adjBtn.closest(".adjust-controls");
        if (!controls) return;
        var snapId = parseInt(controls.getAttribute("data-snap-id"), 10);
        var modelId = controls.getAttribute("data-model-id");
        var modelLabel = controls.getAttribute("data-model-label");
        var currentPct = parseFloat(controls.getAttribute("data-current-pct"));
        var newPct;
        if (adjBtn.getAttribute("data-custom") === "true") {
          var input = prompt("Enter custom remaining quota percentage (0-100) for " + (modelLabel || modelId) + ":", Math.round(currentPct).toString());
          if (input === null) return;
          var parsed = parseInt(input.trim(), 10);
          if (isNaN(parsed) || parsed < 0 || parsed > 100) {
            showToast("\u274C Please enter a valid percentage between 0 and 100.", "error");
            return;
          }
          newPct = parsed;
        } else {
          var delta = parseFloat(adjBtn.getAttribute("data-delta"));
          newPct = Math.max(0, Math.min(100, currentPct + delta));
        }
        var row = controls.closest(".model-row");
        if (row) {
          var pctSpan = row.querySelector(".model-pct");
          var barFill = row.querySelector(".model-bar-fill");
          if (pctSpan) {
            pctSpan.textContent = Math.round(newPct) + "%";
            pctSpan.className = "model-pct " + (newPct <= 0 ? "exhausted" : newPct < 20 ? "warning" : newPct < 50 ? "ok" : "good");
          }
          if (barFill) {
            barFill.style.width = newPct + "%";
            barFill.className = "model-bar-fill " + (newPct <= 0 ? "exhausted" : newPct < 20 ? "warning" : newPct < 50 ? "ok" : "good");
          }
        }
        controls.setAttribute("data-current-pct", String(newPct));
        fetch("/api/snap/adjust", {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            snapshotId: snapId,
            adjustments: [{ modelId, label: modelLabel, remainingPercent: newPct }]
          })
        }).then(function(res) {
          return res.json();
        }).then(function(data) {
          if (data.error) {
            showToast("\u274C " + data.error, "error");
            return;
          }
          showToast("\u270E Adjusted " + (modelLabel || modelId) + " \u2192 " + Math.round(newPct) + "%", "info");
          fetchStatus().then(renderAccounts);
        }).catch(function(err) {
          showToast("\u274C " + err.message, "error");
        });
        return;
      }
      if (e.target.closest("[data-tag-add]") || e.target.closest("[data-remove-tag]") || e.target.closest("[data-note-edit]") || e.target.closest("[data-pin-group]") || e.target.closest("[data-renewal-edit]") || e.target.closest(".tag-picker") || e.target.closest(".tag-chip")) {
        return;
      }
      var row = e.target.closest(".account-row[data-toggle]");
      if (!row) return;
      var id = row.getAttribute("data-toggle");
      var el = document.getElementById(id);
      var chev = document.getElementById("chev-" + id);
      if (!el) return;
      var willExpand = !el.classList.contains("is-expanded");
      el.classList.toggle("is-expanded", willExpand);
      if (willExpand) expandedAccounts.add(id);
      else expandedAccounts.delete(id);
      if (chev) chev.classList.toggle("expanded", willExpand);
    });
  }
  function initQuotas() {
    var qSearch = document.getElementById("quota-search");
    var qStatus = document.getElementById("quota-filter-status");
    if (qSearch) {
      qSearch.addEventListener("input", function() {
        if (latestQuotaData) renderAccounts(latestQuotaData);
      });
    }
    if (qStatus) {
      qStatus.addEventListener("change", function() {
        if (latestQuotaData) renderAccounts(latestQuotaData);
      });
    }
    var qProvider = document.getElementById("quota-filter-provider");
    if (qProvider) {
      qProvider.addEventListener("change", function() {
        if (latestQuotaData) renderAccounts(latestQuotaData);
      });
    }
    var gridEl = document.getElementById("account-grid");
    if (gridEl) {
      gridEl.addEventListener("click", function(e) {
        var el = e.target.closest(".sortable");
        if (!el) return;
        var col = el.dataset.sort;
        var providerSection = el.closest(".provider-section");
        var provider = providerSection ? providerSection.dataset.provider : "antigravity";
        var state = quotaSortStates[provider || "antigravity"];
        if (!state) {
          state = { column: "account", direction: "asc" };
          quotaSortStates[provider || "antigravity"] = state;
        }
        if (state.column === col) {
          state.direction = state.direction === "asc" ? "desc" : "asc";
        } else {
          state.column = col;
          state.direction = "asc";
        }
        quotaSortState.column = state.column;
        quotaSortState.direction = state.direction;
        if (latestQuotaData) renderAccounts(latestQuotaData);
      });
    }
    var tagStrip = document.getElementById("tag-filter-strip");
    if (tagStrip) {
      tagStrip.addEventListener("click", handleTagFilterClick);
    }
  }

  // internal/web/src/core/emptyStates.ts
  function emptyQuotas() {
    return '<div class="empty-state"><div class="empty-state-icon">\u{1F4CA}</div><h3 class="empty-state-title">Your AI Quota Dashboard</h3><p class="empty-state-desc">After your first snapshot, quota cards for each provider will appear here with real-time usage tracking across Antigravity, Claude, Codex, and more.</p><div class="empty-state-preview"><div class="empty-preview-row"><span class="empty-preview-label">Claude + GPT</span><span class="empty-bar"><span style="width:67%"></span></span><span class="empty-preview-pct">67%</span></div><div class="empty-preview-row"><span class="empty-preview-label">Gemini Pro</span><span class="empty-bar"><span style="width:89%"></span></span><span class="empty-preview-pct">89%</span></div><div class="empty-preview-row"><span class="empty-preview-label">Gemini Flash</span><span class="empty-bar"><span style="width:42%"></span></span><span class="empty-preview-pct">42%</span></div></div><button class="btn-add" id="empty-snap-btn">\u{1F4F8} Take First Snapshot</button></div>';
  }
  function emptySubscriptions() {
    return '<div class="empty-state"><div class="empty-state-icon">\u{1F4B3}</div><h3 class="empty-state-title">Track Your AI Subscriptions</h3><p class="empty-state-desc">Add your AI tool subscriptions to see monthly spend, renewal dates, budget tracking, and cost optimization insights.</p><div class="empty-state-preview"><div class="empty-preview-row"><span class="empty-preview-label">GitHub Copilot</span><span class="empty-preview-price">$19/mo</span></div><div class="empty-preview-row"><span class="empty-preview-label">Claude Pro</span><span class="empty-preview-price">$20/mo</span></div><div class="empty-preview-row"><span class="empty-preview-label">Cursor Pro</span><span class="empty-preview-price">$20/mo</span></div></div><button class="btn-add" id="empty-add-sub-btn">+ Add Subscription</button></div>';
  }

  // internal/web/src/subscriptions.ts
  function loadSubscriptions() {
    var status = document.getElementById("filter-status").value;
    var category = document.getElementById("filter-category").value;
    fetchSubscriptions(status, category).then(function(data) {
      renderSubscriptions(data);
    }).catch(function(err) {
      console.error("Failed to load subscriptions:", err);
    });
  }
  function renderSubscriptions(data) {
    var grid = document.getElementById("subs-grid");
    var summary = document.getElementById("subs-summary");
    if (!grid) return;
    var subs = data.subscriptions || [];
    summary.textContent = subs.length + " subscription" + (subs.length !== 1 ? "s" : "");
    if (subs.length === 0) {
      grid.innerHTML = emptySubscriptions();
      var emptyAddBtn = document.getElementById("empty-add-sub-btn");
      if (emptyAddBtn) emptyAddBtn.addEventListener("click", function() {
        openModal();
      });
      return;
    }
    var providerGroups = {};
    var manualSubs = [];
    var grandTotal = 0;
    for (var i = 0; i < subs.length; i++) {
      var s = subs[i];
      var monthly = 0;
      if (s.costAmount > 0) {
        if (s.billingCycle === "yearly") monthly = s.costAmount / 12;
        else monthly = s.costAmount;
      }
      grandTotal += monthly;
      if (s.autoTracked) {
        var pkey = s.platform || "Unknown";
        if (!providerGroups[pkey]) providerGroups[pkey] = { items: [], total: 0 };
        providerGroups[pkey].items.push(s);
        providerGroups[pkey].total += monthly;
      } else {
        manualSubs.push(s);
      }
    }
    var providerKeys = Object.keys(providerGroups);
    var autoCount = subs.length - manualSubs.length;
    var sym2 = currencySymbol(subs[0] ? subs[0].costCurrency : "USD");
    var html = '<div class="spend-summary-card"><div class="spend-hero"><div class="spend-amount">' + sym2 + grandTotal.toFixed(2) + '<span class="spend-period">/mo</span></div><div class="spend-label">Total Monthly Spend</div></div><div class="spend-breakdown">';
    var providerIcons = { "Antigravity": "\u26A1", "Codex": "\u{1F916}", "Claude": "\u{1F52E}" };
    for (var pk = 0; pk < providerKeys.length; pk++) {
      var pName = providerKeys[pk];
      var pIcon = providerIcons[pName] || "\u{1F4E6}";
      var pTotal = providerGroups[pName].total;
      html += '<span class="spend-chip">' + pIcon + " " + esc(pName) + " <strong>" + sym2 + pTotal.toFixed(2) + "</strong></span>";
    }
    if (manualSubs.length > 0) {
      var manualTotal = 0;
      for (var mi = 0; mi < manualSubs.length; mi++) {
        if (manualSubs[mi].costAmount > 0) {
          manualTotal += manualSubs[mi].billingCycle === "yearly" ? manualSubs[mi].costAmount / 12 : manualSubs[mi].costAmount;
        }
      }
      if (manualTotal > 0) {
        html += '<span class="spend-chip">\u{1F4CB} Manual <strong>' + sym2 + manualTotal.toFixed(2) + "</strong></span>";
      }
    }
    html += '</div><div class="spend-meta">' + autoCount + " auto-tracked \xB7 " + manualSubs.length + " manual</div></div>";
    for (var pi = 0; pi < providerKeys.length; pi++) {
      var provider = providerKeys[pi];
      var group = providerGroups[provider];
      var items = group.items;
      var icon = providerIcons[provider] || "\u{1F4E6}";
      var sectionId = "sub-provider-" + provider.replace(/\s+/g, "-").toLowerCase();
      var providerAttr = provider.toLowerCase().replace(/[^a-z]/g, "");
      var collapsed = collapsedSubProviders.has(sectionId);
      var subCollapseClass = collapsed ? " collapsed" : "";
      var subChevron = collapsed ? "\u25B8" : "\u25BE";
      html += '<div class="provider-section" data-provider="' + providerAttr + '"><div class="provider-header" data-toggle-provider="' + sectionId + '"><div class="provider-header-left"><span class="provider-chevron" id="pchev-' + sectionId + '">' + subChevron + '</span> <span class="provider-icon">' + icon + '</span><span class="provider-name">' + esc(provider) + '</span><span class="provider-count">' + items.length + " account" + (items.length !== 1 ? "s" : "") + '</span></div><span class="provider-spend">' + sym2 + group.total.toFixed(2) + '/mo</span></div><div class="provider-body' + subCollapseClass + '" id="' + sectionId + '"><div class="subs-card-grid">';
      for (var si = 0; si < items.length; si++) {
        html += renderSubCard(items[si]);
      }
      html += "</div></div></div>";
    }
    if (manualSubs.length > 0) {
      var grouped = {};
      for (var mi2 = 0; mi2 < manualSubs.length; mi2++) {
        var cat = manualSubs[mi2].category || "other";
        if (!grouped[cat]) grouped[cat] = [];
        grouped[cat].push(manualSubs[mi2]);
      }
      var catOrder = ["coding", "chat", "api", "image", "audio", "productivity", "other"];
      html += '<div class="sub-section-label">Manual Subscriptions (' + manualSubs.length + ")</div>";
      html += '<div class="subs-card-grid">';
      for (var ci = 0; ci < catOrder.length; ci++) {
        var catItems = grouped[catOrder[ci]];
        if (!catItems || catItems.length === 0) continue;
        for (var csi = 0; csi < catItems.length; csi++) {
          html += renderSubCard(catItems[csi]);
        }
      }
      html += "</div>";
    } else if (providerKeys.length > 0) {
      html += '<div class="sub-section-label">Manual Subscriptions</div><div class="manual-empty"><p>No manual subscriptions tracked.</p><p class="empty-hint">Click <strong>+ Add</strong> to track Claude Pro, Cursor, or other AI tools.</p></div>';
    }
    grid.innerHTML = html;
    grid.querySelectorAll(".provider-header[data-toggle-provider]").forEach(function(hdr) {
      hdr.addEventListener("click", function() {
        var targetId = hdr.dataset.toggleProvider;
        var body = document.getElementById(targetId);
        var chev = document.getElementById("pchev-" + targetId);
        if (!body) return;
        var collapsed2 = body.classList.toggle("collapsed");
        if (chev) chev.textContent = collapsed2 ? "\u25B8" : "\u25BE";
        if (collapsed2) {
          collapsedSubProviders.add(targetId);
        } else {
          collapsedSubProviders.delete(targetId);
        }
        localStorage.setItem("niyantra_collapsed_subs_providers", JSON.stringify(Array.from(collapsedSubProviders)));
      });
    });
  }
  function renderSubCard(sub) {
    var costHTML = "";
    if (sub.costAmount > 0) {
      var sym2 = currencySymbol(sub.costCurrency);
      costHTML = '<div class="sub-card-cost">' + sym2 + sub.costAmount.toFixed(2) + ' <span class="cycle">/' + esc(sub.billingCycle) + "</span></div>";
    } else if (sub.billingCycle === "payg") {
      costHTML = '<div class="sub-card-cost">Pay-as-you-go</div>';
    }
    var limitsHTML = "";
    var chips = [];
    if (sub.tokenLimit > 0) chips.push(formatNumber(sub.tokenLimit) + " tokens/" + esc(sub.limitPeriod));
    if (sub.creditLimit > 0) chips.push(formatNumber(sub.creditLimit) + " credits/" + esc(sub.limitPeriod));
    if (sub.requestLimit > 0) chips.push(formatNumber(sub.requestLimit) + " requests/" + esc(sub.limitPeriod));
    if (chips.length > 0) {
      limitsHTML = '<div class="sub-card-limits">';
      for (var c = 0; c < chips.length; c++) {
        limitsHTML += '<span class="sub-limit-chip">' + chips[c] + "</span>";
      }
      limitsHTML += "</div>";
    }
    var badgesHTML = '<span class="sub-status-badge ' + esc(sub.status) + '">' + esc(sub.status) + "</span>";
    badgesHTML += '<span class="sub-cat-badge">' + esc(sub.category) + "</span>";
    if (sub.autoTracked) badgesHTML += '<span class="sub-auto-badge">AUTO</span>';
    var trialHTML = "";
    if (sub.daysUntilTrialEnd !== void 0 && sub.daysUntilTrialEnd !== null) {
      if (sub.daysUntilTrialEnd <= 0) {
        trialHTML = '<span class="trial-countdown">Trial expired!</span>';
      } else if (sub.daysUntilTrialEnd <= 7) {
        trialHTML = '<span class="trial-countdown">Trial ends in ' + sub.daysUntilTrialEnd + "d</span>";
      }
    }
    var renewalHTML = "";
    if (sub.nextRenewal && sub.daysUntilRenewal !== void 0) {
      var rCls = sub.daysUntilRenewal <= 7 ? "soon" : "";
      if (sub.daysUntilRenewal < 0) rCls = "overdue";
      renewalHTML = '<span class="sub-renewal-tag ' + rCls + '">Renews: ' + sub.nextRenewal + " (" + sub.daysUntilRenewal + "d)</span>";
    }
    var linksHTML = "";
    if (sub.url || sub.statusPageUrl) {
      linksHTML = '<div class="sub-card-links">';
      if (sub.url) linksHTML += '<a href="' + esc(sub.url) + '" target="_blank" rel="noopener">\u{1F517} Dashboard</a>';
      if (sub.statusPageUrl) linksHTML += '<a href="' + esc(sub.statusPageUrl) + '" target="_blank" rel="noopener">\u{1F7E2} Status</a>';
      linksHTML += "</div>";
    }
    var notesHTML = "";
    if (sub.notes) {
      notesHTML = '<div class="sub-card-notes">' + esc(sub.notes) + "</div>";
    }
    var metaParts = [];
    if (sub.email) metaParts.push(esc(sub.email));
    if (sub.planName) metaParts.push(esc(sub.planName));
    var metaHTML = metaParts.length > 0 ? '<div class="sub-card-meta">' + metaParts.join(" \xB7 ") + "</div>" : "";
    var cardTitle, cardSubtitle;
    if (sub.autoTracked && sub.email) {
      cardTitle = esc(sub.email);
      cardSubtitle = '<span class="sub-card-platform-badge">' + esc(sub.platform) + (sub.planName ? " \xB7 " + esc(sub.planName) : "") + "</span>";
    } else {
      cardTitle = esc(sub.platform);
      cardSubtitle = "";
    }
    if (sub.autoTracked) {
      badgesHTML = badgesHTML.replace(/<span[^>]*>AUTO<\/span>/i, "");
    }
    var cardStatus = (sub.status || "active").toLowerCase();
    var expiringSoonClass = "";
    if (sub.daysUntilRenewal !== void 0 && sub.daysUntilRenewal <= 7 && sub.daysUntilRenewal >= 0) {
      expiringSoonClass = " expiring-soon";
    }
    if (sub.daysUntilTrialEnd !== void 0 && sub.daysUntilTrialEnd !== null && sub.daysUntilTrialEnd <= 7 && sub.daysUntilTrialEnd >= 0) {
      expiringSoonClass = " expiring-soon";
    }
    return '<div class="sub-card' + expiringSoonClass + '" data-sub-id="' + sub.id + '" data-status="' + esc(cardStatus) + '"><div class="sub-card-header"><div class="sub-card-title">' + cardTitle + '</div><div class="sub-card-badges">' + trialHTML + badgesHTML + "</div></div>" + (cardSubtitle ? '<div class="sub-card-subtitle">' + cardSubtitle + "</div>" : "") + metaHTML + costHTML + limitsHTML + notesHTML + linksHTML + renewalHTML + '<div class="sub-card-actions"><button class="btn-edit-card" data-edit-id="' + sub.id + '">Edit</button><button class="btn-delete-card btn-danger" data-delete-id="' + sub.id + '" data-delete-name="' + esc(sub.platform) + '">Delete</button></div></div>';
  }
  function initModal() {
    var overlay = document.getElementById("modal-overlay");
    var closeBtn = document.getElementById("modal-close");
    var cancelBtn = document.getElementById("modal-cancel");
    var saveBtn = document.getElementById("modal-save");
    document.getElementById("add-sub-btn").addEventListener("click", function() {
      openModal();
    });
    document.getElementById("add-sub-btn-2").addEventListener("click", function() {
      openModal();
    });
    closeBtn.addEventListener("click", closeModal);
    cancelBtn.addEventListener("click", closeModal);
    overlay.addEventListener("click", function(e) {
      if (e.target === overlay) closeModal();
    });
    saveBtn.addEventListener("click", handleSave);
    document.getElementById("f-platform").addEventListener("input", function() {
      var val = this.value;
      for (var i = 0; i < presetsData.length; i++) {
        if (presetsData[i].platform === val) {
          fillFromPreset(presetsData[i]);
          break;
        }
      }
    });
    document.getElementById("subs-grid").addEventListener("click", function(e) {
      var editBtn = e.target.closest("[data-edit-id]");
      if (editBtn) {
        var id = parseInt(editBtn.getAttribute("data-edit-id"));
        openEditModal(id);
        return;
      }
      var deleteBtn = e.target.closest("[data-delete-id]");
      if (deleteBtn) {
        var deleteId = parseInt(deleteBtn.getAttribute("data-delete-id"));
        var deleteName = deleteBtn.getAttribute("data-delete-name");
        openDeleteConfirm(deleteId, deleteName);
      }
    });
    document.getElementById("delete-close").addEventListener("click", closeDelete);
    document.getElementById("delete-cancel").addEventListener("click", closeDelete);
    document.getElementById("delete-overlay").addEventListener("click", function(e) {
      if (e.target.id === "delete-overlay") closeDelete();
    });
    document.getElementById("filter-status").addEventListener("change", loadSubscriptions);
    document.getElementById("filter-category").addEventListener("change", loadSubscriptions);
  }
  function openModal(sub) {
    var overlay = document.getElementById("modal-overlay");
    var title = document.getElementById("modal-title");
    if (sub) {
      title.textContent = "Edit Subscription";
      document.getElementById("f-id").value = sub.id || "";
      document.getElementById("f-platform").value = sub.platform || "";
      document.getElementById("f-category").value = sub.category || "other";
      document.getElementById("f-status").value = sub.status || "active";
      document.getElementById("f-email").value = sub.email || "";
      document.getElementById("f-plan").value = sub.planName || "";
      document.getElementById("f-cost").value = sub.costAmount || "";
      document.getElementById("f-currency").value = sub.costCurrency || "USD";
      document.getElementById("f-cycle").value = sub.billingCycle || "monthly";
      document.getElementById("f-token-limit").value = sub.tokenLimit || "";
      document.getElementById("f-credit-limit").value = sub.creditLimit || "";
      document.getElementById("f-request-limit").value = sub.requestLimit || "";
      document.getElementById("f-limit-period").value = sub.limitPeriod || "monthly";
      document.getElementById("f-renewal").value = sub.nextRenewal || "";
      document.getElementById("f-trial-ends").value = sub.trialEndsAt || "";
      document.getElementById("f-url").value = sub.url || "";
      document.getElementById("f-notes").value = sub.notes || "";
      document.getElementById("f-status-page-url").value = sub.statusPageUrl || "";
      document.getElementById("f-auto-tracked").value = sub.autoTracked ? "1" : "0";
      document.getElementById("f-account-id").value = sub.accountId || "0";
    } else {
      title.textContent = "Add Subscription";
      document.getElementById("sub-modal").querySelectorAll("input, select, textarea").forEach(function(el) {
        if (el.type === "hidden") {
          el.value = "";
          return;
        }
        if (el.tagName === "SELECT") {
          el.selectedIndex = 0;
          return;
        }
        el.value = "";
      });
      document.getElementById("f-currency").value = "USD";
      document.getElementById("f-cycle").value = "monthly";
      document.getElementById("f-category").value = "coding";
      document.getElementById("f-limit-period").value = "monthly";
    }
    overlay.hidden = false;
    document.getElementById("f-platform").focus();
  }
  function closeModal() {
    document.getElementById("modal-overlay").hidden = true;
  }
  function fillFromPreset(preset) {
    document.getElementById("f-category").value = preset.category || "other";
    document.getElementById("f-cost").value = preset.costAmount || "";
    document.getElementById("f-cycle").value = preset.billingCycle || "monthly";
    document.getElementById("f-token-limit").value = preset.tokenLimit || "";
    document.getElementById("f-credit-limit").value = preset.creditLimit || "";
    document.getElementById("f-request-limit").value = preset.requestLimit || "";
    document.getElementById("f-limit-period").value = preset.limitPeriod || "monthly";
    document.getElementById("f-url").value = preset.url || "";
    document.getElementById("f-notes").value = preset.notes || "";
    document.getElementById("f-status-page-url").value = preset.statusPageUrl || "";
  }
  function openEditModal(id) {
    fetch("/api/subscriptions/" + id).then(function(res) {
      return res.json();
    }).then(function(sub) {
      openModal(sub);
    }).catch(function(err) {
      showToast("\u274C " + err.message, "error");
    });
  }
  function handleSave() {
    var id = document.getElementById("f-id").value;
    var sub = {
      platform: document.getElementById("f-platform").value.trim(),
      category: document.getElementById("f-category").value,
      status: document.getElementById("f-status").value,
      email: document.getElementById("f-email").value.trim(),
      planName: document.getElementById("f-plan").value.trim(),
      costAmount: parseFloat(document.getElementById("f-cost").value) || 0,
      costCurrency: document.getElementById("f-currency").value,
      billingCycle: document.getElementById("f-cycle").value,
      tokenLimit: parseInt(document.getElementById("f-token-limit").value) || 0,
      creditLimit: parseInt(document.getElementById("f-credit-limit").value) || 0,
      requestLimit: parseInt(document.getElementById("f-request-limit").value) || 0,
      limitPeriod: document.getElementById("f-limit-period").value,
      nextRenewal: document.getElementById("f-renewal").value,
      trialEndsAt: document.getElementById("f-trial-ends").value,
      url: document.getElementById("f-url").value.trim(),
      notes: document.getElementById("f-notes").value.trim(),
      statusPageUrl: document.getElementById("f-status-page-url").value,
      autoTracked: document.getElementById("f-auto-tracked").value === "1",
      accountId: parseInt(document.getElementById("f-account-id").value) || 0
    };
    if (!sub.platform) {
      showToast("\u274C Platform name is required", "error");
      return;
    }
    var saveBtn = document.getElementById("modal-save");
    saveBtn.disabled = true;
    saveBtn.textContent = "Saving...";
    var promise = id ? updateSubscription(parseInt(id), sub) : createSubscription(sub);
    promise.then(function(data) {
      showToast("\u2705 " + (id ? "Updated" : "Created") + ": " + sub.platform, "success");
      closeModal();
      loadSubscriptions();
    }).catch(function(err) {
      showToast("\u274C " + err.message, "error");
    }).finally(function() {
      saveBtn.disabled = false;
      saveBtn.textContent = "Save Subscription";
    });
  }
  var pendingDeleteId = null;
  function openDeleteConfirm(id, name) {
    pendingDeleteId = id;
    document.getElementById("delete-name").textContent = name;
    document.getElementById("delete-overlay").hidden = false;
    document.getElementById("delete-confirm").onclick = function() {
      deleteSubscription(pendingDeleteId).then(function() {
        showToast("\u2705 Deleted: " + name, "success");
        closeDelete();
        loadSubscriptions();
      }).catch(function(err) {
        showToast("\u274C " + err.message, "error");
      });
    };
  }
  function closeDelete() {
    document.getElementById("delete-overlay").hidden = true;
    pendingDeleteId = null;
  }
  function initSearch() {
    var searchEl = document.getElementById("search-subs");
    if (!searchEl) return;
    searchEl.addEventListener("input", function() {
      var query = searchEl.value.toLowerCase().trim();
      var cards = document.querySelectorAll(".sub-card");
      var labels = document.querySelectorAll(".sub-category-label");
      cards.forEach(function(card) {
        var text = card.textContent.toLowerCase();
        card.style.display = text.indexOf(query) >= 0 ? "" : "none";
      });
      labels.forEach(function(label) {
        var next = label.nextElementSibling;
        var anyVisible = false;
        while (next && !next.classList.contains("sub-category-label")) {
          if (next.classList.contains("sub-card") && next.style.display !== "none") {
            anyVisible = true;
          }
          next = next.nextElementSibling;
        }
        label.style.display = anyVisible ? "" : "none";
      });
    });
  }

  // internal/web/src/overview/budget.ts
  function getBudget() {
    return parseFloat(serverConfig["budget_monthly"] || "0");
  }
  function setBudget(amount) {
    serverConfig["budget_monthly"] = amount.toString();
    updateConfig("budget_monthly", amount.toString());
  }
  function updateConfig(key, value) {
    return fetch("/api/config", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key, value })
    }).then(function(r) {
      if (!r.ok) {
        return r.json().then(function(e) {
          throw new Error(e.error || "Failed to update config");
        }).catch(function(err) {
          throw err;
        });
      }
      return r.json();
    }).then(function(data) {
      if (data.config) {
        data.config.forEach(function(c) {
          serverConfig[c.key] = c.value;
        });
      }
      return data;
    }).catch(function(err) {
      console.error("Config update failed:", err);
      showToast("\u274C " + (err.message || "Config update failed"), "error");
      throw err;
    });
  }
  function loadConfig() {
    return fetch("/api/config").then(function(r) {
      return r.json();
    }).then(function(data) {
      if (data.config) {
        data.config.forEach(function(c) {
          serverConfig[c.key] = c.value;
        });
      }
    });
  }
  function initBudget() {
    document.getElementById("budget-close").addEventListener("click", closeBudget);
    document.getElementById("budget-cancel").addEventListener("click", closeBudget);
    document.getElementById("budget-overlay").addEventListener("click", function(e) {
      if (e.target.id === "budget-overlay") closeBudget();
    });
    document.addEventListener("click", function(e) {
      var target = e.target;
      if (!target) return;
      var trigger = target.closest('[data-budget-edit="true"]');
      if (trigger) openBudgetModal();
    });
    document.getElementById("budget-save").addEventListener("click", function() {
      var val = parseFloat(document.getElementById("f-budget").value) || 0;
      setBudget(val);
      closeBudget();
      showToast("\u2705 Budget set to $" + val.toFixed(0) + "/mo", "success");
      var overviewPanel = document.getElementById("panel-overview");
      if (overviewPanel && overviewPanel.classList.contains("active")) document.dispatchEvent(new CustomEvent("niyantra:overview-refresh"));
    });
  }
  function openBudgetModal() {
    document.getElementById("f-budget").value = String(getBudget() || "");
    document.getElementById("budget-overlay").hidden = false;
  }
  function closeBudget() {
    document.getElementById("budget-overlay").hidden = true;
  }

  // internal/web/src/overview/insights.ts
  function renderServerInsights(insights) {
    if (!insights || insights.length === 0) return "";
    var html = '<div class="insight-panel"><h3>Intelligence Insights</h3><div class="insight-list">';
    var iconMap = {
      renewal_imminent: "Renewal",
      trial_expiring: "Trial",
      unused_subscription: "Unused",
      category_overlap: "Overlap",
      budget_exceeded: "Budget",
      renewal: "Renewal",
      trial: "Trial",
      unused: "Unused",
      overlap: "Overlap"
    };
    for (var i = 0; i < insights.length; i++) {
      var ins = insights[i];
      var icon = iconMap[ins.type] || "Info";
      var cls = ins.severity === "critical" ? "critical" : ins.severity === "warning" ? "warning" : "info";
      html += '<div class="insight-item ' + cls + '"><span class="insight-item-icon">' + esc(icon) + '</span><div class="insight-item-content"><div class="insight-item-title">' + esc(ins.title || ins.type.replace(/_/g, " ")) + '</div><div class="insight-item-msg">' + esc(ins.message) + "</div></div></div>";
    }
    html += "</div></div>";
    return html;
  }
  var advisorGroupPref = localStorage.getItem("niyantra_advisor_group") || "claude_gpt";
  if (advisorGroupPref === "gemini_pro" || advisorGroupPref === "gemini_flash") {
    advisorGroupPref = "gemini_unified";
  }
  function loadAdvisorCard() {
    var container = document.getElementById("advisor-card-container");
    if (!container) return;
    if (!latestQuotaData || !latestQuotaData.accounts || latestQuotaData.accounts.length < 2) {
      container.innerHTML = "";
      return;
    }
    renderAdvisorWithGroup(container, advisorGroupPref);
  }
  function renderAdvisorWithGroup(container, groupKey) {
    var accounts = latestQuotaData.accounts;
    var ranked = [];
    for (var i = 0; i < accounts.length; i++) {
      var acc = accounts[i];
      var groups = acc.groups || [];
      var pct = null;
      for (var g = 0; g < groups.length; g++) {
        if (groups[g].groupKey === groupKey) {
          pct = Math.round(groups[g].remainingPercent);
          break;
        }
      }
      if (pct === null) {
        if (groupKey === "all") {
          var total = 0;
          for (var gx = 0; gx < groups.length; gx++) total += groups[gx].remainingPercent;
          pct = groups.length > 0 ? Math.round(total / groups.length) : 0;
        } else {
          pct = 0;
        }
      }
      var isStale = false;
      if (acc.lastSeen) {
        var ageMs = Date.now() - new Date(acc.lastSeen).getTime();
        isStale = ageMs > 6 * 3600 * 1e3;
      }
      ranked.push({
        email: acc.email,
        pct,
        stale: isStale,
        label: acc.stalenessLabel || ""
      });
    }
    ranked.sort(function(a, b) {
      return b.pct - a.pct;
    });
    var groupNames = {
      "claude_gpt": "Claude + GPT",
      "gemini_unified": "Gemini Pool",
      "all": "All Models (avg)"
    };
    var best = ranked[0];
    var allHealthy = ranked.every(function(a) {
      return a.pct > 80;
    });
    var actionIcon = allHealthy ? "READY" : best.pct > 20 ? "SWITCH" : "WAIT";
    var actionLabel = allHealthy ? "ALL READY" : best.pct > 20 ? "SWITCH" : "WAIT";
    var html = '<div class="advisor-card"><h3>Antigravity Account Advisor</h3><div class="advisor-group-select"><label>Optimize for:</label><select id="advisor-group-filter" class="filter-select" style="margin-left:8px;font-size:12px"><option value="claude_gpt"' + (groupKey === "claude_gpt" ? " selected" : "") + '>Claude + GPT</option><option value="gemini_unified"' + (groupKey === "gemini_unified" ? " selected" : "") + '>Gemini Pool</option><option value="all"' + (groupKey === "all" ? " selected" : "") + ">All Models (avg)</option></select></div>";
    var actionCls = allHealthy ? "stay" : best.pct > 20 ? "switch" : "wait";
    html += '<div class="advisor-action ' + actionCls + '">' + actionIcon + " " + actionLabel + '</div><div class="advisor-reason">' + (allHealthy ? "All accounts have healthy quotas - no switch needed" : "Best: " + esc(best.email) + " (" + best.pct + "% " + esc(groupNames[groupKey] || groupKey) + " remaining)") + (best.stale ? " [stale data]" : "") + "</div>";
    html += '<div class="advisor-scores">';
    var initialShow = Math.min(ranked.length, 5);
    for (var s = 0; s < ranked.length; s++) {
      var acct = ranked[s];
      var isBest = s === 0;
      var barCls = acct.pct > 50 ? "good" : acct.pct > 20 ? "ok" : "low";
      var staleIcon = acct.stale ? ' <span class="stale-icon" title="Data ' + esc(acct.label) + '">STALE</span>' : "";
      var hidden = s >= initialShow ? ' style="display:none" data-advisor-extra' : "";
      html += '<div class="advisor-score-row' + (isBest ? " best" : "") + '"' + hidden + '><span class="advisor-score-email" title="' + esc(acct.email) + '">' + esc(acct.email) + '</span><div class="advisor-score-bar"><div class="advisor-score-fill ' + barCls + '" style="width:' + acct.pct + '%"></div></div><span class="advisor-score-val">' + acct.pct + "%" + staleIcon + "</span></div>";
    }
    if (ranked.length > initialShow) {
      html += '<button class="advisor-show-all" id="advisor-toggle-all">Show all ' + ranked.length + " accounts</button>";
    }
    html += "</div></div>";
    container.innerHTML = html;
    var sel = document.getElementById("advisor-group-filter");
    if (sel) {
      sel.addEventListener("change", function() {
        advisorGroupPref = sel.value;
        localStorage.setItem("niyantra_advisor_group", advisorGroupPref);
        renderAdvisorWithGroup(container, advisorGroupPref);
      });
    }
    var toggleBtn = document.getElementById("advisor-toggle-all");
    if (toggleBtn) {
      toggleBtn.addEventListener("click", function() {
        var extras = container.querySelectorAll("[data-advisor-extra]");
        var showing = toggleBtn.textContent.indexOf("Hide") >= 0;
        extras.forEach(function(el) {
          el.style.display = showing ? "none" : "";
        });
        toggleBtn.textContent = showing ? "Show all " + ranked.length + " accounts" : "Hide extras";
      });
    }
  }

  // internal/web/src/overview/cost.ts
  function loadCostKPI() {
    var container = document.getElementById("cost-kpi-container");
    if (!container) return;
    fetch("/api/cost").then(function(res) {
      return res.json();
    }).then(function(data) {
      if (!data || !data.accounts || data.accounts.length === 0) {
        container.innerHTML = "";
        return;
      }
      var total = data.totalCost || 0;
      if (total < 0.01) {
        container.innerHTML = "";
        return;
      }
      var totalLabel = data.totalLabel || "$0.00";
      var html = '<div class="cost-kpi-card overview-card"><h3>Quota-Derived Cost Estimate</h3><div class="cost-kpi-amount">' + esc(totalLabel) + '</div><div class="cost-kpi-label">Estimated from quota consumption, configured token ceilings, and model pricing; not observed spend.</div>';
      var hasChips = false;
      var chipsHTML = '<div class="cost-kpi-breakdown">';
      if (data.accounts && data.accounts.length > 0) {
        for (var i = 0; i < data.accounts.length; i++) {
          var acct = data.accounts[i];
          if (acct.totalCost >= 0.01) {
            hasChips = true;
            var emailShort = acct.email;
            if (emailShort && emailShort.length > 20) {
              emailShort = emailShort.split("@")[0] + "@...";
            }
            chipsHTML += '<span class="cost-kpi-chip" title="' + esc(acct.email) + '">' + esc(emailShort) + ": " + esc(acct.totalLabel) + "</span>";
          }
        }
      }
      chipsHTML += "</div>";
      if (hasChips) html += chipsHTML;
      html += "</div>";
      container.innerHTML = html;
    }).catch(function(err) {
      console.error("Cost KPI fetch failed:", err);
      container.innerHTML = "";
    });
  }

  // internal/web/src/overview/streaks.ts
  function renderStreakCard(data) {
    if (!data || data.totalSnapshots === 0) return "";
    var fireEmojis = data.streak >= 30 ? "\u{1F525}\u{1F525}\u{1F525}" : data.streak >= 14 ? "\u{1F525}\u{1F525}" : data.streak >= 7 ? "\u{1F525}" : data.streak >= 1 ? "\u{1F525}" : "";
    var streakLabel = data.streak === 0 ? "No active streak \u2014 take a snapshot today!" : data.streak + "-day streak " + fireEmojis;
    return '<div class="streak-card"><div class="streak-main"><div class="streak-number">' + (data.streak || 0) + '</div><div class="streak-label">' + streakLabel + '</div></div><div class="streak-stats"><div class="streak-stat"><span class="streak-stat-val">' + formatCount(data.totalSnapshots) + '</span><span class="streak-stat-label">Total Snaps</span></div><div class="streak-stat"><span class="streak-stat-val">' + data.activeDays + '</span><span class="streak-stat-label">Active Days</span></div><div class="streak-stat"><span class="streak-stat-val">' + data.longestStreak + '</span><span class="streak-stat-label">Best Streak</span></div></div></div>';
  }
  function formatCount(n) {
    if (n >= 1e4) return (n / 1e3).toFixed(1) + "k";
    if (n >= 1e3) return (n / 1e3).toFixed(1) + "k";
    return n.toString();
  }

  // internal/web/src/overview/heatmap.ts
  function loadHeatmap() {
    var container = document.getElementById("heatmap-container");
    if (!container) return;
    fetch("/api/history/heatmap?days=365").then(function(res) {
      return res.json();
    }).then(function(data) {
      if (!data) {
        container.innerHTML = "";
        return;
      }
      renderHeatmap(container, data);
    }).catch(function(err) {
      console.error("Heatmap fetch failed:", err);
      container.innerHTML = "";
    });
  }
  function renderHeatmap(container, data) {
    var days = data.days || [];
    var maxCount = data.maxCount || 1;
    var dayMap = {};
    for (var i = 0; i < days.length; i++) {
      dayMap[days[i].date] = days[i];
    }
    var today = /* @__PURE__ */ new Date();
    var startDate = new Date(today);
    startDate.setDate(startDate.getDate() - 364);
    var dayOfWeek = startDate.getDay();
    startDate.setDate(startDate.getDate() - dayOfWeek);
    var totalDays = Math.ceil((today.getTime() - startDate.getTime()) / (1e3 * 60 * 60 * 24)) + 1;
    var totalWeeks = Math.ceil(totalDays / 7);
    var monthLabels = [];
    var lastMonth = -1;
    var monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
    var cellsHTML = "";
    for (var w = 0; w < totalWeeks; w++) {
      for (var d = 0; d < 7; d++) {
        var cellDate = new Date(startDate);
        cellDate.setDate(startDate.getDate() + w * 7 + d);
        if (cellDate > today) {
          cellsHTML += '<div class="heatmap-cell heatmap-empty"></div>';
          continue;
        }
        var dateStr = formatDateISO(cellDate);
        var entry = dayMap[dateStr];
        var count = entry ? entry.count : 0;
        var level = getIntensityLevel(count, maxCount);
        var month = cellDate.getMonth();
        if (month !== lastMonth && d === 0) {
          monthLabels.push({ label: monthNames[month], col: w });
          lastMonth = month;
        }
        var tooltip = formatDateHuman(cellDate) + ": ";
        if (count === 0) {
          tooltip += "No activity";
        } else {
          tooltip += count + " snapshot" + (count !== 1 ? "s" : "");
          if (entry) {
            var parts = [];
            if (entry.antigravity > 0) parts.push(entry.antigravity + " AG");
            if (entry.claude > 0) parts.push(entry.claude + " Claude");
            if (entry.codex > 0) parts.push(entry.codex + " Codex");
            if (entry.cursor > 0) parts.push(entry.cursor + " Cursor");
            if (entry.gemini > 0) parts.push(entry.gemini + " Gemini");
            if (entry.copilot > 0) parts.push(entry.copilot + " Copilot");
            if (entry.plugin > 0) parts.push(entry.plugin + " Plugin");
            if (parts.length > 0) tooltip += " (" + parts.join(", ") + ")";
          }
        }
        cellsHTML += '<div class="heatmap-cell heatmap-level-' + level + '" data-date="' + dateStr + '" data-count="' + count + '" aria-label="' + tooltip + '" title="' + tooltip + '"></div>';
      }
    }
    var monthLabelHTML = '<div class="heatmap-month-labels" style="grid-template-columns: 28px repeat(' + totalWeeks + ', 1fr)">';
    monthLabelHTML += "<div></div>";
    var lastCol = -2;
    for (var m = 0; m < monthLabels.length; m++) {
      if (monthLabels[m].col > lastCol + 2) {
        monthLabelHTML += '<div class="heatmap-month" style="grid-column: ' + (monthLabels[m].col + 2) + '">' + monthLabels[m].label + "</div>";
        lastCol = monthLabels[m].col;
      }
    }
    monthLabelHTML += "</div>";
    var statsHTML = '<div class="heatmap-stats"><span class="heatmap-stat"><span class="heatmap-stat-value">' + data.totalSnapshots + '</span><span class="heatmap-stat-label">snapshots</span></span><span class="heatmap-stat"><span class="heatmap-stat-value">' + data.activeDays + '</span><span class="heatmap-stat-label">active days</span></span><span class="heatmap-stat"><span class="heatmap-stat-value">' + data.streak + 'd</span><span class="heatmap-stat-label">current streak</span></span><span class="heatmap-stat"><span class="heatmap-stat-value">' + data.longestStreak + 'd</span><span class="heatmap-stat-label">longest streak</span></span></div>';
    var legendHTML = '<div class="heatmap-legend"><span class="heatmap-legend-label">Less</span><div class="heatmap-cell heatmap-level-0 heatmap-legend-cell"></div><div class="heatmap-cell heatmap-level-1 heatmap-legend-cell"></div><div class="heatmap-cell heatmap-level-2 heatmap-legend-cell"></div><div class="heatmap-cell heatmap-level-3 heatmap-legend-cell"></div><div class="heatmap-cell heatmap-level-4 heatmap-legend-cell"></div><span class="heatmap-legend-label">More</span></div>';
    var dayLabels = '<div class="heatmap-day-labels"><div></div><div class="heatmap-day-label">Mon</div><div></div><div class="heatmap-day-label">Wed</div><div></div><div class="heatmap-day-label">Fri</div><div></div></div>';
    var gridHTML = '<div class="heatmap-scroll"><div class="heatmap-body">' + dayLabels + '<div class="heatmap-grid" style="grid-template-columns: repeat(' + totalWeeks + ', 1fr)">' + cellsHTML + "</div></div></div>";
    var streakHTML = renderStreakCard(data);
    container.innerHTML = streakHTML + "<h3>Activity</h3>" + monthLabelHTML + gridHTML + '<div class="heatmap-footer">' + legendHTML + "</div>";
  }
  function getIntensityLevel(count, max) {
    if (count === 0) return 0;
    if (max <= 1) return 4;
    var ratio = count / max;
    if (ratio <= 0.25) return 1;
    if (ratio <= 0.5) return 2;
    if (ratio <= 0.75) return 3;
    return 4;
  }
  function formatDateISO(d) {
    var y = d.getFullYear();
    var m = (d.getMonth() + 1).toString().padStart(2, "0");
    var day = d.getDate().toString().padStart(2, "0");
    return y + "-" + m + "-" + day;
  }
  function formatDateHuman(d) {
    var months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
    var days = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
    return days[d.getDay()] + ", " + months[d.getMonth()] + " " + d.getDate() + ", " + d.getFullYear();
  }

  // internal/web/src/overview/calendar.ts
  var calendarViewDate = /* @__PURE__ */ new Date();
  function renderRenewalCalendar(renewals, subs) {
    var container = document.getElementById("renewal-calendar-container");
    if (!container) return;
    var renewalMap = {};
    if (renewals) {
      for (var i = 0; i < renewals.length; i++) {
        var r = renewals[i];
        var dateKey = r.nextRenewal;
        if (!renewalMap[dateKey]) renewalMap[dateKey] = [];
        var cat = "other";
        if (subs) {
          for (var s = 0; s < subs.length; s++) {
            if (subs[s].platform === r.platform && subs[s].category) {
              cat = subs[s].category;
              break;
            }
          }
        }
        renewalMap[dateKey].push({ platform: r.platform, category: cat, daysUntil: r.daysUntil });
      }
    }
    var year = calendarViewDate.getFullYear();
    var month = calendarViewDate.getMonth();
    var today = /* @__PURE__ */ new Date();
    var todayKey = today.getFullYear() + "-" + String(today.getMonth() + 1).padStart(2, "0") + "-" + String(today.getDate()).padStart(2, "0");
    var monthNames = [
      "January",
      "February",
      "March",
      "April",
      "May",
      "June",
      "July",
      "August",
      "September",
      "October",
      "November",
      "December"
    ];
    var dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
    var firstDay = new Date(year, month, 1).getDay();
    var daysInMonth = new Date(year, month + 1, 0).getDate();
    var prevDays = new Date(year, month, 0).getDate();
    var html = '<div class="calendar-container"><div class="calendar-header"><h3>\u{1F4C5} Renewal Calendar</h3><div class="calendar-nav"><button class="calendar-nav-btn" data-calendar-nav="-1">\u2039</button><span class="calendar-month-label">' + monthNames[month] + " " + year + '</span><button class="calendar-nav-btn" data-calendar-nav="1">\u203A</button></div></div>';
    html += '<div class="calendar-weekdays">';
    for (var d = 0; d < 7; d++) {
      html += '<div class="calendar-weekday">' + dayNames[d] + "</div>";
    }
    html += "</div>";
    html += '<div class="calendar-grid">';
    for (var p = firstDay - 1; p >= 0; p--) {
      html += '<div class="calendar-day other-month"><span class="calendar-day-num">' + (prevDays - p) + "</span></div>";
    }
    for (var day = 1; day <= daysInMonth; day++) {
      dateKey = year + "-" + String(month + 1).padStart(2, "0") + "-" + String(day).padStart(2, "0");
      var isToday = dateKey === todayKey;
      var dayClass = isToday ? "calendar-day today" : "calendar-day";
      var events = renewalMap[dateKey];
      html += '<div class="' + dayClass + '"';
      if (events && events.length > 0) {
        var tooltipText = events.map(function(e2) {
          return e2.platform;
        }).join(", ");
        html += ' title="' + esc(tooltipText) + '"';
      }
      html += ">";
      html += '<span class="calendar-day-num">' + day + "</span>";
      if (events && events.length > 0) {
        html += '<div class="calendar-pins">';
        for (var e = 0; e < Math.min(events.length, 4); e++) {
          html += '<span class="calendar-pin ' + esc(events[e].category) + '"></span>';
        }
        html += "</div>";
      }
      html += "</div>";
    }
    var totalCells = firstDay + daysInMonth;
    var remaining = 7 - totalCells % 7;
    if (remaining < 7) {
      for (var n = 1; n <= remaining; n++) {
        html += '<div class="calendar-day other-month"><span class="calendar-day-num">' + n + "</span></div>";
      }
    }
    html += "</div>";
    var categories = {};
    for (var key in renewalMap) {
      for (var ci = 0; ci < renewalMap[key].length; ci++) {
        categories[renewalMap[key][ci].category] = true;
      }
    }
    var catKeys = Object.keys(categories);
    if (catKeys.length > 0) {
      html += '<div class="calendar-legend">';
      for (var cl = 0; cl < catKeys.length; cl++) {
        html += '<div class="calendar-legend-item"><span class="calendar-legend-dot ' + esc(catKeys[cl]) + '"></span>' + esc(catKeys[cl]) + "</div>";
      }
      html += "</div>";
    }
    html += "</div>";
    container.innerHTML = html;
    container.querySelectorAll("[data-calendar-nav]").forEach(function(el) {
      el.addEventListener("click", function() {
        var btn = el;
        calendarNav(parseInt(btn.dataset.calendarNav || "0", 10));
      });
    });
  }
  function calendarNav(delta) {
    calendarViewDate.setMonth(calendarViewDate.getMonth() + delta);
    var el = document.getElementById("renewal-calendar-container");
    if (el) {
      document.dispatchEvent(new CustomEvent("niyantra:overview-refresh"));
    }
  }

  // internal/web/src/advanced/codex.ts
  function loadCodexSettingsStatus() {
    var statusEl = document.getElementById("codex-status-settings");
    if (!statusEl) return;
    fetch("/api/codex/status").then(function(r) {
      return r.json();
    }).then(function(data) {
      statusEl.style.display = "";
      if (!data.installed) {
        statusEl.innerHTML = '<span style="color:var(--text-muted)">\u26A0\uFE0F Codex CLI not detected. Install <a href="https://github.com/openai/codex" target="_blank" style="color:var(--accent)">Codex</a> and run <code>codex auth</code> to enable.</span>';
        return;
      }
      var tokenStatus = data.tokenExpired ? '<span style="color:var(--warning)">\u26A0\uFE0F Token expired \u2014 will auto-refresh on next poll</span>' : '<span style="color:var(--success)">\u2705 Token valid (expires ' + (data.tokenExpiresIn || "?") + ")</span>";
      var displayId = data.email || (data.accountId && data.accountId.length > 12 ? data.accountId.substring(0, 6) + "\u2026" + data.accountId.slice(-6) : data.accountId || "unknown");
      statusEl.innerHTML = "\u{1F916} Codex detected \xB7 Account: <strong>" + esc(displayId) + "</strong><br>" + tokenStatus;
      if (data.snapshot) {
        statusEl.innerHTML += "<br>Latest: <strong>" + data.snapshot.fiveHourPct.toFixed(1) + '%</strong> used (5h) \xB7 <span style="color:var(--text-muted)">' + formatTimeAgo(data.snapshot.capturedAt) + "</span>";
      }
    }).catch(function() {
      statusEl.style.display = "none";
    });
  }
  function handleCodexSnap() {
    showToast("\u{1F916} Capturing Codex snapshot...", "info");
    fetch("/api/codex/snap", { method: "POST" }).then(function(r) {
      return r.json();
    }).then(function(data) {
      if (data.error) {
        showToast("\u274C " + data.error, "error");
        return;
      }
      showToast("\u{1F916} Codex snapshot captured! Plan: " + (data.plan || "unknown"), "success");
      loadCodexSettingsStatus();
      document.dispatchEvent(new CustomEvent("niyantra:overview-refresh"));
    }).catch(function() {
      showToast("\u274C Codex snap failed", "error");
    });
  }
  function renderSessionsTimeline(container) {
    fetch("/api/sessions?limit=10").then(function(r) {
      return r.json();
    }).then(function(data) {
      if (!data.sessions || data.sessions.length === 0) return;
      var html = '<div class="overview-card sessions-card">';
      html += '<div class="card-header"><h3>\u23F1\uFE0F Usage Sessions</h3>';
      html += '<span class="card-count">' + data.count + " sessions</span>";
      html += "</div>";
      html += '<div class="card-body">';
      html += '<div class="session-timeline">';
      for (var i = 0; i < data.sessions.length; i++) {
        var sess = data.sessions[i];
        var isActive = !sess.endedAt;
        var duration = isActive ? formatDurationSec(Math.floor((Date.now() - new Date(sess.startedAt).getTime()) / 1e3)) : formatDurationSec(sess.durationSec);
        var providerIcon = sess.provider === "codex" ? "\u{1F916}" : sess.provider === "claude" ? "\u{1F52E}" : "\u26A1";
        html += '<div class="session-item' + (isActive ? " active" : "") + '">';
        html += '<div class="session-dot' + (isActive ? " pulse" : "") + '"></div>';
        html += '<div class="session-content">';
        html += '<div class="session-top">';
        html += '<span class="session-provider">' + providerIcon + " " + esc(sess.provider) + "</span>";
        html += '<span class="session-duration">' + duration + "</span>";
        html += "</div>";
        html += '<div class="session-bottom">';
        html += '<span class="session-time">' + formatTimeAgo(sess.startedAt) + "</span>";
        html += '<span class="session-snaps">' + sess.snapCount + " snaps</span>";
        if (isActive) html += '<span class="session-active-badge">LIVE</span>';
        html += "</div>";
        html += "</div></div>";
      }
      html += "</div></div></div>";
      var codexCard = container.querySelector(".codex-card");
      var existing = container.querySelector(".sessions-card");
      if (existing) {
        existing.outerHTML = html;
      } else if (codexCard) {
        codexCard.insertAdjacentHTML("afterend", html);
      } else {
        container.insertAdjacentHTML("afterbegin", html);
      }
    }).catch(function() {
    });
  }

  // internal/web/src/charts/sparkline.ts
  function sparkline(data, opts) {
    var w = opts && opts.width || 60;
    var h = opts && opts.height || 20;
    var color = opts && opts.color || "var(--accent)";
    var dir = opts && opts.direction || "flat";
    if (!data || data.length < 2) {
      return '<span class="sparkline-container"><svg width="' + w + '" height="' + h + '"></svg></span>';
    }
    var min = Math.min.apply(null, data);
    var max = Math.max.apply(null, data);
    var range = max - min || 1;
    var pad = 2;
    var points = "";
    var lastX = 0;
    var lastY = 0;
    for (var i = 0; i < data.length; i++) {
      var x = pad + i / (data.length - 1) * (w - 2 * pad);
      var y = h - pad - (data[i] - min) / range * (h - 2 * pad);
      points += x.toFixed(1) + "," + y.toFixed(1) + " ";
      lastX = x;
      lastY = y;
    }
    var arrowColor = dir === "up" ? "var(--green)" : dir === "down" ? "var(--red)" : "var(--text-muted)";
    var arrow = dir === "up" ? "\u2191" : dir === "down" ? "\u2193" : "\u2192";
    return '<span class="sparkline-container"><svg width="' + w + '" height="' + h + '" class="sparkline-svg"><polyline points="' + points.trim() + '" fill="none" stroke="' + color + '" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" opacity="0.85"/><circle cx="' + lastX.toFixed(1) + '" cy="' + lastY.toFixed(1) + '" r="2" fill="' + color + '"/></svg><span class="sparkline-arrow" style="color:' + arrowColor + '">' + arrow + "</span></span>";
  }
  function trendDirection(data) {
    if (!data || data.length < 2) return "flat";
    var first = data[0];
    var last = data[data.length - 1];
    if (last > first * 1.05) return "up";
    if (last < first * 0.95) return "down";
    return "flat";
  }

  // internal/web/src/overview/tokenAnalytics.ts
  function loadTokenAnalytics() {
    var container = document.getElementById("token-analytics-container");
    if (!container) return;
    var rangeSelector = document.getElementById("token-range-selector");
    var days = 30;
    if (rangeSelector) {
      days = parseInt(rangeSelector.value) || 30;
    }
    fetch("/api/token-usage?days=" + days).then(function(res) {
      return res.json();
    }).then(function(data) {
      renderTokenAnalytics(container, data, days);
    }).catch(function(err) {
      console.error("Token analytics fetch failed:", err);
      container.innerHTML = '<div class="token-analytics-empty">Failed to load token analytics</div>';
    });
  }
  function renderTokenAnalytics(container, data, days) {
    if (!data || !data.totals || data.totals.totalTokens === 0) {
      container.innerHTML = '<div class="overview-card full-width token-analytics-card"><h3>Observed Token Usage</h3><div class="token-analytics-empty"><p>No observed token usage data is available yet.</p><p style="font-size:12px;color:var(--text-secondary)">Current observed sources are Claude Code session files in <code>~/.claude/projects/</code> plus any provider rows already persisted into Niyantra&apos;s <code>token_usage</code> table.</p></div></div>';
      return;
    }
    var totals = data.totals;
    var kpis = data.kpis || {};
    var models = data.byModel || [];
    var dailyData = data.byDay || [];
    var rangeOptions = [
      { value: "7", label: "7d" },
      { value: "30", label: "30d" },
      { value: "90", label: "90d" },
      { value: "365", label: "1y" }
    ];
    var rangeHTML = '<div class="token-range-bar">';
    for (var i = 0; i < rangeOptions.length; i++) {
      var opt = rangeOptions[i];
      var activeClass = String(days) === opt.value ? " token-range-active" : "";
      rangeHTML += '<button class="token-range-btn' + activeClass + '" data-days="' + opt.value + '">' + opt.label + "</button>";
    }
    rangeHTML += "</div>";
    var tokenSparkData = [];
    var costSparkData = [];
    if (dailyData.length >= 3) {
      var sparkSlice = dailyData.slice(-7);
      for (var si = 0; si < sparkSlice.length; si++) {
        tokenSparkData.push(sparkSlice[si].totalTokens || 0);
        costSparkData.push(sparkSlice[si].costUSD || 0);
      }
    }
    var tokenSpark = tokenSparkData.length >= 3 ? sparkline(tokenSparkData, { width: 50, height: 18, color: "#6366f1", direction: trendDirection(tokenSparkData) }) : "";
    var costSpark = costSparkData.length >= 3 ? sparkline(costSparkData, { width: 50, height: 18, color: "#f59e0b", direction: trendDirection(costSparkData) }) : "";
    var kpiHTML = '<div class="token-kpi-row">';
    kpiHTML += buildKpiCard("Total Tokens", formatTokens(totals.totalTokens), "Usage", tokenSpark);
    kpiHTML += buildKpiCard("Heuristic Cost", "$" + (totals.estimatedCostUSD || 0).toFixed(2), "Cost", costSpark);
    kpiHTML += buildKpiCard("Active Days", String(kpis.daysActive || 0), "Days");
    kpiHTML += buildKpiCard("Avg/Day", formatTokens(kpis.avgTokensPerDay || 0), "Rate");
    kpiHTML += buildKpiCard("Cache Rate", Math.round((kpis.cacheHitRate || 0) * 100) + "%", "Cache");
    kpiHTML += "</div>";
    var chipsHTML = '<div class="token-breakdown-chips">';
    chipsHTML += '<span class="token-chip token-chip-input">Input: ' + formatTokens(totals.inputTokens) + "</span>";
    chipsHTML += '<span class="token-chip token-chip-output">Output: ' + formatTokens(totals.outputTokens) + "</span>";
    chipsHTML += '<span class="token-chip token-chip-cache">Cache: ' + formatTokens(totals.cacheTokens) + "</span>";
    if (totals.sessions > 0) {
      chipsHTML += '<span class="token-chip token-chip-sessions">Sessions: ' + totals.sessions + "</span>";
    }
    chipsHTML += "</div>";
    var modelHTML = "";
    if (models.length > 0) {
      modelHTML = '<div class="token-section">';
      modelHTML += "<h4>Model Distribution</h4>";
      modelHTML += '<div class="token-model-bars">';
      var colors = ["#6366f1", "#8b5cf6", "#ec4899", "#f59e0b", "#10b981", "#3b82f6", "#ef4444"];
      var topModels = models.slice(0, 7);
      for (var mi = 0; mi < topModels.length; mi++) {
        var model = topModels[mi];
        var color = colors[mi % colors.length];
        var pct = model.percentage || 0;
        var costLabel = model.costUSD > 0 ? " | $" + model.costUSD.toFixed(2) : "";
        modelHTML += '<div class="token-model-row"><div class="token-model-header"><span class="token-model-name" style="color:' + color + '">' + escapeHtml(model.model) + '</span><span class="token-model-stats">' + formatTokens(model.totalTokens) + " (" + pct.toFixed(1) + "%)" + costLabel + '</span></div><div class="token-model-bar-track"><div class="token-model-bar-fill" style="width:' + pct + "%;background:" + color + '"></div></div></div>';
      }
      modelHTML += "</div></div>";
    }
    var chartHTML = "";
    if (dailyData.length > 0) {
      chartHTML = '<div class="token-section">';
      chartHTML += "<h4>Daily Token Burn</h4>";
      chartHTML += '<div class="token-daily-chart">';
      var maxTokens = 0;
      for (var di = 0; di < dailyData.length; di++) {
        if (dailyData[di].totalTokens > maxTokens) maxTokens = dailyData[di].totalTokens;
      }
      var displayDays = dailyData;
      if (displayDays.length > 60) {
        displayDays = displayDays.slice(displayDays.length - 60);
      }
      for (var dj = 0; dj < displayDays.length; dj++) {
        var day = displayDays[dj];
        var barHeight = maxTokens > 0 ? Math.max(2, day.totalTokens / maxTokens * 100) : 2;
        var inputPct = day.totalTokens > 0 ? day.inputTokens / day.totalTokens * barHeight : 0;
        var outputPct = barHeight - inputPct;
        var dayLabel = day.date.substring(5);
        chartHTML += '<div class="token-bar-col" title="' + day.date + ": " + formatTokens(day.totalTokens) + " tokens, $" + (day.costUSD || 0).toFixed(2) + '"><div class="token-bar-stack" style="height:' + barHeight + '%"><div class="token-bar-output" style="height:' + outputPct + '%"></div><div class="token-bar-input" style="height:' + inputPct + '%"></div></div><span class="token-bar-label">' + dayLabel + "</span></div>";
      }
      chartHTML += "</div>";
      chartHTML += '<div class="token-chart-legend"><span class="token-legend-item"><span class="token-legend-dot" style="background:var(--token-input-color)"></span>Input</span><span class="token-legend-item"><span class="token-legend-dot" style="background:var(--token-output-color)"></span>Output</span></div>';
      chartHTML += "</div>";
    }
    var peakHTML = "";
    if (kpis.peakDay) {
      peakHTML = '<div class="token-peak-badge">Peak: ' + kpis.peakDay + " | " + formatTokens(kpis.peakDayTokens) + " tokens</div>";
    }
    container.innerHTML = '<div class="overview-card full-width token-analytics-card"><div class="token-analytics-header"><h3>Observed Token Usage</h3>' + rangeHTML + '</div><p style="font-size:12px;color:var(--text-muted);margin:0 0 12px">Observed sources are Claude session files plus any provider rows already persisted into <code>token_usage</code>.</p>' + kpiHTML + chipsHTML + peakHTML + modelHTML + chartHTML + "</div>";
    var rangeBtns = container.querySelectorAll(".token-range-btn");
    for (var bi = 0; bi < rangeBtns.length; bi++) {
      rangeBtns[bi].addEventListener("click", function() {
        var newDays = this.getAttribute("data-days") || "30";
        var allBtns = container.querySelectorAll(".token-range-btn");
        for (var k = 0; k < allBtns.length; k++) allBtns[k].classList.remove("token-range-active");
        this.classList.add("token-range-active");
        fetch("/api/token-usage?days=" + newDays).then(function(res) {
          return res.json();
        }).then(function(d) {
          renderTokenAnalytics(container, d, parseInt(newDays));
        });
      });
    }
  }
  function buildKpiCard(label, value, icon, spark) {
    return '<div class="token-kpi-card"><div class="token-kpi-icon">' + icon + '</div><div class="token-kpi-value">' + value + "</div>" + (spark ? '<div class="token-kpi-spark">' + spark + "</div>" : "") + '<div class="token-kpi-label">' + label + "</div></div>";
  }
  function formatTokens(n) {
    if (n >= 1e9) return (n / 1e9).toFixed(1) + "B";
    if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
    if (n >= 1e3) return (n / 1e3).toFixed(1) + "K";
    return String(n);
  }
  function escapeHtml(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }

  // internal/web/src/overview/gitCosts.ts
  function loadGitCosts() {
    var container = document.getElementById("git-costs-container");
    if (!container) return;
    fetch("/api/git-costs?days=30").then(function(res) {
      return res.json().then(function(data) {
        if (!res.ok) {
          throw new Error(data.error || "Failed to load git costs");
        }
        return data;
      });
    }).then(function(data) {
      renderGitCosts(container, data);
    }).catch(function(err) {
      console.error("Git costs fetch failed:", err);
      renderGitCostsError(container, err instanceof Error ? err.message : "Failed to load git costs");
    });
  }
  function renderGitCostsError(container, message) {
    container.innerHTML = '<div class="overview-card full-width git-costs-card"><h3>Git Activity Attribution</h3><div class="git-costs-empty"><p>Unable to analyze git activity attribution.</p><p style="font-size:12px;color:var(--text-secondary)">' + escapeHtml2(message) + "</p></div></div>";
  }
  function renderGitCosts(container, data) {
    if (!data || !data.commits || data.commits.length === 0) {
      container.innerHTML = '<div class="overview-card full-width git-costs-card"><h3>Git Activity Attribution</h3><div class="git-costs-empty"><p>No git commit data available.</p><p style="font-size:12px;color:var(--text-secondary)">Ensure you are running Niyantra from within a git repository, or pass <code>?repo=/path</code> to the API.</p></div></div>';
      return;
    }
    var totals = data.totals || {};
    var commits = data.commits || [];
    var branches = data.branches || [];
    var hasAICosts = totals.totalTokens > 0;
    var kpiHTML = '<div class="git-kpi-row">';
    kpiHTML += buildKpi("Commits", String(totals.commitCount || 0), "Commits");
    kpiHTML += buildKpi("Nearby AI Estimate", "$" + (totals.costUSD || 0).toFixed(2), "Estimate");
    kpiHTML += buildKpi("Avg/Commit Est.", "$" + (totals.avgPerCommit || 0).toFixed(2), "Avg");
    kpiHTML += buildKpi("Top Branch", truncate(totals.topBranch || "-", 18), "Branch");
    kpiHTML += "</div>";
    kpiHTML += '<p style="font-size:12px;color:var(--text-muted);margin:0 0 12px">' + escapeHtml2(data.notAccountingGradeReason || "Claude token events are heuristically assigned to nearby commits. This is guidance, not ground truth.") + "</p>";
    if (!hasAICosts) {
      kpiHTML += '<div class="git-no-ai-banner">No nearby Claude Code session data was attributable inside the commit lookback windows.</div>';
    }
    var chartHTML = "";
    if (commits.length > 0 && hasAICosts) {
      chartHTML = '<div class="git-section">';
      chartHTML += "<h4>Estimated Nearby Cost per Commit</h4>";
      chartHTML += '<div class="git-commit-chart">';
      var maxCost = 0;
      for (var ci = 0; ci < commits.length; ci++) {
        if (commits[ci].costUSD > maxCost) maxCost = commits[ci].costUSD;
      }
      var displayCommits = commits.length > 40 ? commits.slice(0, 40) : commits;
      for (var di = 0; di < displayCommits.length; di++) {
        var c = displayCommits[di];
        var barH = maxCost > 0 ? Math.max(3, c.costUSD / maxCost * 100) : 3;
        var barColor = c.costUSD > 0 ? "var(--accent)" : "var(--border)";
        chartHTML += '<div class="git-bar-col" title="' + escapeAttr(c.shortHash) + ": " + escapeAttr(c.message) + "\n$" + c.costUSD.toFixed(2) + " | " + formatTokens2(c.totalTokens) + ' tokens"><div class="git-bar" style="height:' + barH + "%;background:" + barColor + '"></div><span class="git-bar-hash">' + c.shortHash + "</span></div>";
      }
      chartHTML += "</div></div>";
    }
    var branchHTML = "";
    if (branches.length > 0 && hasAICosts) {
      branchHTML = '<div class="git-section">';
      branchHTML += "<h4>Branch Estimates</h4>";
      branchHTML += '<div class="git-branch-table">';
      branchHTML += '<div class="git-branch-header"><span>Branch</span><span>Commits</span><span>Tokens</span><span>Cost</span><span>Avg</span></div>';
      var displayBranches = branches.slice(0, 10);
      for (var bi = 0; bi < displayBranches.length; bi++) {
        var b = displayBranches[bi];
        if (b.costUSD === 0 && b.totalTokens === 0) continue;
        branchHTML += '<div class="git-branch-row"><span class="git-branch-name">' + escapeHtml2(truncate(b.name, 30)) + '</span><span class="git-branch-val">' + b.commits + '</span><span class="git-branch-val">' + formatTokens2(b.totalTokens) + '</span><span class="git-branch-cost">$' + b.costUSD.toFixed(2) + '</span><span class="git-branch-val">$' + b.avgPerCommit.toFixed(2) + "</span></div>";
      }
      branchHTML += "</div></div>";
    }
    var commitsHTML = '<div class="git-section">';
    commitsHTML += "<h4>Recent Commits</h4>";
    commitsHTML += '<div class="git-commits-list">';
    var showCommits = commits.slice(0, 15);
    for (var ri = 0; ri < showCommits.length; ri++) {
      var rc = showCommits[ri];
      var costBadge = rc.costUSD > 0 ? '<span class="git-cost-badge">$' + rc.costUSD.toFixed(2) + "</span>" : '<span class="git-cost-badge git-cost-zero">-</span>';
      var tokenBadge = rc.totalTokens > 0 ? '<span class="git-token-badge">' + formatTokens2(rc.totalTokens) + "</span>" : "";
      commitsHTML += '<div class="git-commit-item"><span class="git-commit-hash">' + rc.shortHash + '</span><span class="git-commit-msg">' + escapeHtml2(rc.message) + '</span><div class="git-commit-meta">' + tokenBadge + costBadge + "</div></div>";
    }
    commitsHTML += "</div></div>";
    container.innerHTML = '<div class="overview-card full-width git-costs-card"><div class="git-costs-header"><h3>Git Activity Attribution</h3><span class="git-repo-path" title="' + escapeAttr(data.repoPath || "") + '">' + escapeHtml2(shortenPath(data.repoPath || "")) + "</span></div>" + kpiHTML + chartHTML + branchHTML + commitsHTML + "</div>";
  }
  function buildKpi(label, value, icon) {
    return '<div class="git-kpi-card"><div class="git-kpi-icon">' + icon + '</div><div class="git-kpi-value">' + value + '</div><div class="git-kpi-label">' + label + "</div></div>";
  }
  function formatTokens2(n) {
    if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
    if (n >= 1e3) return (n / 1e3).toFixed(1) + "K";
    return String(n);
  }
  function truncate(s, max) {
    return s.length > max ? s.substring(0, max - 1) + "..." : s;
  }
  function shortenPath(p) {
    var parts = p.replace(/\\/g, "/").split("/");
    return parts.length > 2 ? ".../" + parts.slice(-2).join("/") : p;
  }
  function escapeHtml2(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }
  function escapeAttr(s) {
    return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  // internal/web/src/overview/safeToSpend.ts
  function renderSafeToSpend(bf, currency) {
    if (!bf || bf.monthlyBudget <= 0) {
      return '<div class="safe-to-spend-card no-budget"><div class="sts-icon">Budget</div><div class="sts-label">Set a monthly AI budget to track recurring subscription headroom</div><button class="btn-add-sm" id="sts-set-budget-btn">Set Budget</button></div>';
    }
    var recurring = bf.recurringMonthlySpend != null ? bf.recurringMonthlySpend : bf.currentSpend;
    var headroom = Math.max(0, bf.monthlyBudget - recurring);
    var pct = bf.monthlyBudget > 0 ? Math.round(recurring / bf.monthlyBudget * 100) : 0;
    var cls;
    var statusText;
    if (pct >= 100) {
      cls = "over";
      statusText = "Recurring subscriptions exceed budget by " + sym(currency) + (recurring - bf.monthlyBudget).toFixed(2) + "/mo";
    } else if (pct >= 80) {
      cls = "warning";
      statusText = "Only " + sym(currency) + headroom.toFixed(2) + "/mo of recurring budget headroom remains";
    } else {
      cls = "healthy";
      statusText = sym(currency) + headroom.toFixed(2) + "/mo remains after recurring subscriptions";
    }
    return '<div class="safe-to-spend-card ' + cls + '"><div class="sts-header"><span class="sts-title">Budget Headroom</span><button class="sts-edit" id="sts-edit-budget-btn" title="Edit budget">Edit</button></div><div class="sts-amount">' + sym(currency) + headroom.toFixed(2) + '</div><div class="sts-bar-container"><div class="sts-bar-fill ' + cls + '" style="width:' + Math.min(pct, 100) + '%"></div></div><div class="sts-details"><span>' + statusText + '</span><span class="sts-budget">Budget: ' + sym(currency) + bf.monthlyBudget.toFixed(0) + '/mo</span></div><div class="sts-caption">Based on recurring subscription commitments only; observed usage spend is not available yet.</div></div>';
  }
  function wireSafeToSpendButtons(openBudgetFn) {
    var editBtn = document.getElementById("sts-edit-budget-btn");
    if (editBtn) editBtn.addEventListener("click", openBudgetFn);
    var setBtn = document.getElementById("sts-set-budget-btn");
    if (setBtn) setBtn.addEventListener("click", openBudgetFn);
  }
  function sym(currency) {
    if (currency === "INR") return "Rs ";
    if (currency === "EUR") return "EUR ";
    if (currency === "GBP") return "GBP ";
    return "$";
  }

  // internal/web/src/advanced/report.ts
  async function assembleReportData() {
    var results = await Promise.all([
      fetch("/api/overview").then(function(r) {
        return r.json();
      }),
      fetch("/api/history/heatmap").then(function(r) {
        return r.json();
      })
    ]);
    var overview = results[0];
    var heatmap = results[1];
    var stats = overview.stats || {};
    var byCat = stats.byCategory || {};
    var catKeys = Object.keys(byCat);
    var topCat = null;
    if (catKeys.length > 0) {
      catKeys.sort(function(a, b) {
        return (byCat[b].monthlySpend || 0) - (byCat[a].monthlySpend || 0);
      });
      var topName = catKeys[0];
      var topSpend = byCat[topName].monthlySpend || 0;
      var totalSpend = stats.totalMonthlySpend || 0;
      topCat = {
        name: topName,
        pct: totalSpend > 0 ? Math.round(topSpend / totalSpend * 100) : 0,
        spend: topSpend
      };
    }
    var dailyBurn = [];
    if (heatmap && heatmap.days) {
      var dayKeys = Object.keys(heatmap.days).sort().slice(-30);
      for (var i = 0; i < dayKeys.length; i++) {
        dailyBurn.push(heatmap.days[dayKeys[i]] || 0);
      }
    }
    var now = /* @__PURE__ */ new Date();
    var period = now.toLocaleString("default", { month: "long", year: "numeric" });
    return {
      totalSpend: stats.totalMonthlySpend || 0,
      providerCount: overview.providerCount || (overview.quotaSummary ? Object.keys(overview.quotaSummary.byProvider || {}).length : 0),
      accountCount: overview.accountCount || 0,
      topCategory: topCat,
      activeDays: heatmap ? heatmap.activeDays || 0 : 0,
      totalSnaps: heatmap ? heatmap.totalSnaps || 0 : 0,
      currentStreak: heatmap ? heatmap.currentStreak || 0 : 0,
      longestStreak: heatmap ? heatmap.longestStreak || 0 : 0,
      period,
      dailyBurn
    };
  }
  async function generateReport() {
    var data = await assembleReportData();
    var canvas = document.createElement("canvas");
    var dpr = window.devicePixelRatio || 1;
    var W = 1200;
    var H = 630;
    canvas.width = W * dpr;
    canvas.height = H * dpr;
    canvas.style.width = W + "px";
    canvas.style.height = H + "px";
    var ctx = canvas.getContext("2d");
    ctx.scale(dpr, dpr);
    var bg = ctx.createLinearGradient(0, 0, 0, H);
    bg.addColorStop(0, "#0a0f1a");
    bg.addColorStop(1, "#0f172a");
    ctx.fillStyle = bg;
    ctx.fillRect(0, 0, W, H);
    ctx.strokeStyle = "rgba(255,255,255,0.02)";
    ctx.lineWidth = 1;
    for (var gy = 0; gy < H; gy += 40) {
      ctx.beginPath();
      ctx.moveTo(0, gy);
      ctx.lineTo(W, gy);
      ctx.stroke();
    }
    drawHeader(ctx, data, W);
    drawHeroMetrics(ctx, data);
    drawStatCards(ctx, data, W);
    drawTrendBars(ctx, data.dailyBurn, W);
    drawFooter(ctx, data, W, H);
    return new Promise(function(resolve) {
      canvas.toBlob(function(blob) {
        resolve(blob);
      }, "image/png");
    });
  }
  function drawHeader(ctx, data, W) {
    ctx.font = "700 20px Inter, system-ui, -apple-system, sans-serif";
    ctx.fillStyle = "#6ee7b7";
    ctx.fillText("\u26A1 Niyantra", 40, 42);
    ctx.font = "400 14px Inter, system-ui, sans-serif";
    ctx.fillStyle = "#64748b";
    ctx.textAlign = "right";
    ctx.fillText(data.period + " Report", W - 40, 42);
    ctx.textAlign = "left";
    ctx.strokeStyle = "rgba(255,255,255,0.06)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(40, 60);
    ctx.lineTo(W - 40, 60);
    ctx.stroke();
  }
  function drawHeroMetrics(ctx, data) {
    ctx.font = "400 11px Inter, system-ui, sans-serif";
    ctx.fillStyle = "#64748b";
    ctx.letterSpacing = "1px";
    ctx.fillText("TOTAL AI SPEND", 50, 98);
    ctx.font = "700 48px Inter, system-ui, sans-serif";
    ctx.fillStyle = "#f1f5f9";
    ctx.fillText("$" + data.totalSpend.toFixed(2), 50, 155);
    if (data.topCategory) {
      ctx.font = "400 11px Inter, system-ui, sans-serif";
      ctx.fillStyle = "#64748b";
      ctx.fillText("TOP CATEGORY", 700, 98);
      ctx.font = "700 28px Inter, system-ui, sans-serif";
      ctx.fillStyle = "#f1f5f9";
      ctx.fillText(data.topCategory.name + " (" + data.topCategory.pct + "%)", 700, 135);
      ctx.font = "400 16px Inter, system-ui, sans-serif";
      ctx.fillStyle = "#94a3b8";
      ctx.fillText("$" + data.topCategory.spend.toFixed(2) + "/mo", 700, 160);
    }
  }
  function drawStatCards(ctx, data, W) {
    var cards = [
      { label: "PROVIDERS", value: String(data.providerCount || "\u2014"), color: "#6366f1" },
      { label: "SNAPSHOTS", value: String(data.totalSnaps || "\u2014"), color: "#10b981" },
      { label: "ACTIVE DAYS", value: String(data.activeDays || "\u2014"), color: "#3b82f6" },
      { label: "BEST STREAK", value: data.longestStreak > 0 ? data.longestStreak + "d" : "\u2014", color: "#f59e0b" }
    ];
    var startX = 50;
    var cardW = (W - 100 - 60) / 4;
    var gap = 20;
    var y = 200;
    var cardH = 80;
    for (var i = 0; i < cards.length; i++) {
      var x = startX + i * (cardW + gap);
      var c = cards[i];
      ctx.fillStyle = "#131b2e";
      roundRect(ctx, x, y, cardW, cardH, 8);
      ctx.fill();
      ctx.fillStyle = c.color;
      roundRect(ctx, x, y, 3, cardH, 2);
      ctx.fill();
      ctx.font = "500 10px Inter, system-ui, sans-serif";
      ctx.fillStyle = "#64748b";
      ctx.fillText(c.label, x + 16, y + 24);
      ctx.font = "700 28px Inter, system-ui, sans-serif";
      ctx.fillStyle = "#f1f5f9";
      ctx.fillText(c.value, x + 16, y + 60);
    }
  }
  function drawTrendBars(ctx, daily, W) {
    var chartX = 50;
    var chartY = 320;
    var chartW = W - 100;
    var chartH = 200;
    ctx.fillStyle = "#131b2e";
    roundRect(ctx, chartX, chartY, chartW, chartH, 8);
    ctx.fill();
    if (daily.length === 0) {
      ctx.font = "400 13px Inter, system-ui, sans-serif";
      ctx.fillStyle = "#475569";
      ctx.textAlign = "center";
      ctx.fillText("No activity data available", chartX + chartW / 2, chartY + chartH / 2);
      ctx.textAlign = "left";
      return;
    }
    var max = Math.max.apply(null, daily);
    if (max === 0) max = 1;
    var barCount = daily.length;
    var barArea = chartW - 40;
    var barW = Math.max(4, Math.floor(barArea / barCount) - 2);
    var barStartX = chartX + 20;
    for (var i = 0; i < barCount; i++) {
      var barH = Math.max(2, daily[i] / max * (chartH - 50));
      var bx = barStartX + i * (barW + 2);
      var by = chartY + chartH - 20 - barH;
      var alpha = 0.4 + daily[i] / max * 0.6;
      ctx.fillStyle = "rgba(110, 231, 183, " + alpha + ")";
      roundRect(ctx, bx, by, barW, barH, 2);
      ctx.fill();
    }
    ctx.font = "400 11px Inter, system-ui, sans-serif";
    ctx.fillStyle = "#64748b";
    ctx.fillText("Activity Trend (last " + daily.length + " days)", chartX + 20, chartY + chartH - 4);
  }
  function drawFooter(ctx, data, W, H) {
    ctx.strokeStyle = "rgba(255,255,255,0.06)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(40, H - 40);
    ctx.lineTo(W - 40, H - 40);
    ctx.stroke();
    ctx.font = "400 11px Inter, system-ui, sans-serif";
    ctx.fillStyle = "#475569";
    ctx.fillText("Tracked with Niyantra \xB7 niyantra.bhaskarjha.dev", 50, H - 18);
    ctx.textAlign = "right";
    ctx.fillText((/* @__PURE__ */ new Date()).toLocaleDateString(), W - 50, H - 18);
    ctx.textAlign = "left";
  }
  function roundRect(ctx, x, y, w, h, r) {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.lineTo(x + w - r, y);
    ctx.arcTo(x + w, y, x + w, y + r, r);
    ctx.lineTo(x + w, y + h - r);
    ctx.arcTo(x + w, y + h, x + w - r, y + h, r);
    ctx.lineTo(x + r, y + h);
    ctx.arcTo(x, y + h, x, y + h - r, r);
    ctx.lineTo(x, y + r);
    ctx.arcTo(x, y, x + r, y, r);
    ctx.closePath();
  }
  function downloadReport() {
    var btn = document.getElementById("generate-report-btn");
    if (btn) {
      btn.textContent = "\u23F3 Generating...";
      btn.setAttribute("disabled", "true");
    }
    generateReport().then(function(blob) {
      if (btn) {
        btn.textContent = "\u{1F4CA} Monthly Report";
        btn.removeAttribute("disabled");
      }
      if (!blob) return;
      var url = URL.createObjectURL(blob);
      var a = document.createElement("a");
      a.href = url;
      a.download = "niyantra-report-" + (/* @__PURE__ */ new Date()).toISOString().slice(0, 7) + ".png";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }).catch(function(err) {
      console.error("Report generation failed:", err);
      if (btn) {
        btn.textContent = "\u{1F4CA} Monthly Report";
        btn.removeAttribute("disabled");
      }
    });
  }

  // internal/web/src/overview/overview.ts
  function loadOverview() {
    Promise.all([fetchOverview(), fetchSubscriptions("", ""), fetchUsage()]).then(function(results) {
      var data = results[0];
      var subsData = results[1];
      var usageData = results[2];
      renderOverviewEnhanced(data, subsData.subscriptions || subsData || [], usageData);
    }).catch(function(err) {
      console.error("Failed to load overview:", err);
    });
  }
  function renderOverviewEnhanced(data, subs, usageData) {
    var el = document.getElementById("overview-content");
    if (!el) return;
    var stats = data.stats || { totalMonthlySpend: 0, totalAnnualSpend: 0, byCategory: {}, byStatus: {} };
    var renewals = data.renewals || [];
    var links = data.quickLinks || [];
    var quotas = data.quotaSummary;
    var serverInsights = data.insights || [];
    var advisorHTML = '<div id="advisor-card-container"></div>';
    var insightsHTML = renderServerInsights(serverInsights);
    var safeToSpendHTML = renderSafeToSpend(
      usageData && usageData.budgetForecast ? usageData.budgetForecast : null,
      serverConfig["currency"] || "USD"
    );
    var cats = Object.keys(stats.byCategory);
    var spendHTML = '<div class="overview-card"><h3>Monthly Recurring Spend</h3><div class="kpi-with-sparkline"><div class="overview-big-number">$' + stats.totalMonthlySpend.toFixed(2) + "</div></div>";
    if (cats.length > 1) {
      cats.sort(function(a, b) {
        return (stats.byCategory[b].monthlySpend || 0) - (stats.byCategory[a].monthlySpend || 0);
      });
      for (var i = 0; i < cats.length; i++) {
        var c = stats.byCategory[cats[i]];
        spendHTML += '<div class="overview-category-row"><span class="overview-category-name">' + esc(cats[i]) + '<span class="overview-category-count">' + c.count + ' subs</span></span><span class="overview-category-spend">$' + c.monthlySpend.toFixed(2) + "/mo</span></div>";
      }
    } else if (cats.length === 1) {
      var onlyCat = stats.byCategory[cats[0]];
      spendHTML += '<div class="overview-big-label">' + onlyCat.count + " " + cats[0] + " subscription" + (onlyCat.count !== 1 ? "s" : "") + "</div>";
    }
    spendHTML += "</div>";
    var calendarHTML = "";
    if (renewals.length > 0) {
      calendarHTML = '<div id="renewal-calendar-container" class="overview-card full-width"></div>';
    }
    var linksHTML = "";
    if (links.length > 0) {
      if (links.length > 1 || links.length === 1 && links[0].platform !== "Antigravity") {
        linksHTML = '<div class="overview-card full-width"><h3>Quick Links</h3><div class="quick-links-grid">';
        for (var pk = 0; pk < links.length; pk++) {
          var pl = links[pk];
          linksHTML += '<a class="quick-link" href="' + esc(pl.url) + '" target="_blank" rel="noopener">\u{1F517} ' + esc(pl.platform) + "</a>";
        }
        linksHTML += "</div></div>";
      }
    }
    var exportHTML = '<div class="overview-card full-width"><h3>Export</h3><p style="font-size:13px;color:var(--text-secondary);margin-bottom:12px">Download a redacted JSON report or a full database backup.</p><div style="display:flex;gap:8px;flex-wrap:wrap"><button class="btn-add" id="download-csv-btn" style="padding:6px 12px;font-size:12px">\u{1F4E5} CSV</button><button class="btn-add" id="download-json-btn" style="padding:6px 12px;font-size:12px">\u{1F4E6} Redacted JSON</button><button class="btn-add" id="download-backup-btn" style="padding:6px 12px;font-size:12px">\u{1F4BE} DB Backup</button><button class="btn-add" id="generate-report-btn" style="padding:6px 12px;font-size:12px">\u{1F4CA} Monthly Report</button></div></div>';
    var costKPIHTML = '<div id="cost-kpi-container"></div>';
    var tokenAnalyticsHTML = '<div id="token-analytics-container" class="overview-card full-width"></div>';
    var gitCostsHTML = '<div id="git-costs-container" class="overview-card full-width"></div>';
    var heatmapHTML = '<div id="heatmap-container" class="overview-card full-width"></div>';
    var eligibleAccount = null;
    if (latestQuotaData && latestQuotaData.accounts) {
      eligibleAccount = latestQuotaData.accounts.find(function(acc) {
        return acc.planTier && acc.planTier.toLowerCase() === "ultra" && acc.hasClaimedBonus2026 === 0;
      });
    }
    var bannerHTML = "";
    if (eligibleAccount) {
      bannerHTML = '<div class="io-alert-card" data-account-id="' + eligibleAccount.accountId + '"><div class="io-alert-content"><div class="io-alert-title">\u2728 Google I/O 2026 Promotional Bonus</div><div class="io-alert-desc">Exclusive for Ultra members: Claim your $100 Overage Credit Bonus before it expires on <strong>May 25, 2026</strong>.</div></div><button class="io-claim-btn" data-claim-account-id="' + eligibleAccount.accountId + '">Claim $100 Bonus</button></div>';
    }
    el.innerHTML = bannerHTML + safeToSpendHTML + advisorHTML + costKPIHTML + tokenAnalyticsHTML + gitCostsHTML + heatmapHTML + insightsHTML + spendHTML + calendarHTML + linksHTML + exportHTML;
    wireSafeToSpendButtons(openBudgetModal);
    var claimBtn = el.querySelector(".io-claim-btn");
    if (claimBtn) {
      claimBtn.addEventListener("click", function(e) {
        var btn = e.currentTarget;
        var accId = parseInt(btn.getAttribute("data-claim-account-id") || "0", 10);
        if (accId > 0) {
          btn.disabled = true;
          btn.textContent = "Claiming...";
          claimOverageBonus(accId).then(function() {
            showToast("\u2728 $100 Overage Bonus credit added!", "success");
            fetchStatus().then(function(freshData) {
              document.dispatchEvent(new CustomEvent("niyantra:status-refreshed", { detail: { data: freshData } }));
              document.dispatchEvent(new CustomEvent("niyantra:overview-refresh"));
            }).catch(function() {
            });
          }).catch(function(err) {
            btn.disabled = false;
            btn.textContent = "Claim $100 Bonus";
            showToast("\u274C " + (err.message || "Claim failed"), "error");
          });
        }
      });
    }
    var reportBtn = document.getElementById("generate-report-btn");
    if (reportBtn) {
      reportBtn.addEventListener("click", function() {
        downloadReport();
      });
    }
    var backupBtn = document.getElementById("download-backup-btn");
    if (backupBtn) {
      backupBtn.addEventListener("click", function() {
        downloadBackup().catch(function(err) {
          alert(err.message || "Backup failed");
        });
      });
    }
    var csvBtn = document.getElementById("download-csv-btn");
    if (csvBtn) {
      csvBtn.addEventListener("click", function() {
        downloadAPIFile("/api/export/csv", "niyantra-export.csv").catch(function(err) {
          alert(err.message || "CSV export failed");
        });
      });
    }
    var jsonBtn = document.getElementById("download-json-btn");
    if (jsonBtn) {
      jsonBtn.addEventListener("click", function() {
        downloadAPIFile("/api/export/json", "niyantra-export.json").catch(function(err) {
          alert(err.message || "JSON export failed");
        });
      });
    }
    loadAdvisorCard();
    loadCostKPI();
    loadHeatmap();
    loadTokenAnalytics();
    loadGitCosts();
    if (renewals.length > 0) {
      renderRenewalCalendar(renewals, subs);
    }
    renderSessionsTimeline(el);
  }

  // internal/web/src/advanced/snap.ts
  var snapDefault = localStorage.getItem("niyantra_snap_default") || "antigravity";
  function initSnapDropdown() {
    var caret = document.getElementById("snap-caret");
    var dropdown = document.getElementById("snap-dropdown");
    if (!caret || !dropdown) return;
    caret.addEventListener("click", function(e) {
      e.stopPropagation();
      dropdown.classList.toggle("open");
    });
    document.addEventListener("click", function() {
      dropdown.classList.remove("open");
    });
    dropdown.querySelectorAll(".snap-option").forEach(function(opt) {
      opt.addEventListener("click", function(e) {
        e.stopPropagation();
        var source = opt.dataset.source;
        dropdown.classList.remove("open");
        if (source === "all") {
          snapSource("all");
        } else {
          snapDefault = source;
          localStorage.setItem("niyantra_snap_default", source);
          updateSnapDropdownIndicators();
          snapSource(source);
        }
      });
    });
    updateSnapDropdownIndicators();
  }
  function updateSnapDropdownIndicators() {
    var dropdown = document.getElementById("snap-dropdown");
    if (!dropdown) return;
    dropdown.querySelectorAll(".snap-option").forEach(function(opt) {
      if (opt.dataset.source === "all") return;
      var isActive = opt.dataset.source === snapDefault;
      opt.textContent = (isActive ? "\u25C9 " : "\u25CB ") + opt.textContent.replace(/^[◉○] /, "");
      opt.classList.toggle("active", isActive);
    });
  }
  function handleSnap() {
    snapSource(snapDefault);
  }
  function snapSource(source) {
    var btn = document.getElementById("snap-btn");
    if (!btn || btn.disabled || snapInProgress) return;
    setSnapInProgress(true);
    btn.disabled = true;
    btn.classList.add("snapping");
    var orig = btn.innerHTML;
    btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/></svg> Capturing...';
    var promises = [];
    if (source === "antigravity" || source === "all") {
      promises.push(
        triggerSnap().then(function(data) {
          if (!data.captured || data.captured.length === 0) {
            return { source: "Antigravity", error: "No active session detected" };
          }
          var emails = data.captured.map(function(c) {
            return c.email;
          });
          var label = "Antigravity \xB7 " + emails.join(", ");
          return { source: "Antigravity", data, label };
        }).catch(function(err) {
          return { source: "Antigravity", error: err.message || "Capture failed" };
        })
      );
    }
    if (source === "claude" || source === "all") {
      promises.push(
        fetch("/api/claude/snap", { method: "POST" }).then(function(r) {
          if (!r.ok) {
            return r.json().then(
              function(e) {
                throw new Error(e.error || "capture failed");
              },
              function() {
                throw new Error("capture failed");
              }
            );
          }
          return r.json();
        }).then(function(d) {
          var label = "Claude Code \xB7 " + (d.fiveHourPct || 0).toFixed(0) + "%";
          return { source: "Claude Code", data: d, label };
        }).catch(function(err) {
          return { source: "Claude Code", error: err.message || "capture failed" };
        })
      );
    }
    if (source === "codex" || source === "all") {
      promises.push(
        fetch("/api/codex/snap", { method: "POST" }).then(function(r) {
          if (!r.ok) {
            return r.json().then(
              function(e) {
                throw new Error(e.error || "capture failed");
              },
              function() {
                throw new Error("capture failed");
              }
            );
          }
          return r.json();
        }).then(function(d) {
          var label = d.plan ? "Codex \xB7 " + d.plan : "Codex";
          return { source: "Codex", data: d, label };
        }).catch(function(err) {
          return { source: "Codex", error: err.message || "capture failed" };
        })
      );
    }
    if (source === "cursor" || source === "all") {
      promises.push(
        fetch("/api/cursor/snap", { method: "POST" }).then(function(r) {
          if (!r.ok) {
            return r.json().then(
              function(e) {
                throw new Error(e.error || "capture failed");
              },
              function() {
                throw new Error("capture failed");
              }
            );
          }
          return r.json();
        }).then(function(d) {
          var label = "";
          if (d.billingModel === "usd_credit") {
            label = "Cursor \xB7 $" + ((d.usedCents || 0) / 100).toFixed(2) + "/$" + ((d.limitCents || 0) / 100).toFixed(2);
          } else {
            label = "Cursor \xB7 " + (d.requestsUsed || 0) + "/" + (d.requestsMax || "?");
          }
          return { source: "Cursor", data: d, label };
        }).catch(function(err) {
          return { source: "Cursor", error: err.message || "capture failed" };
        })
      );
    }
    if (source === "copilot" || source === "all") {
      promises.push(
        fetch("/api/copilot/snap", { method: "POST" }).then(function(r) {
          if (!r.ok) {
            return r.json().then(
              function(e) {
                throw new Error(e.error || "capture failed");
              },
              function() {
                throw new Error("capture failed");
              }
            );
          }
          return r.json();
        }).then(function(d) {
          var label = "Copilot \xB7 " + (d.plan || "unknown") + " \xB7 " + (d.premiumPct || 0).toFixed(0) + "%";
          return { source: "Copilot", data: d, label };
        }).catch(function(err) {
          return { source: "Copilot", error: err.message || "capture failed" };
        })
      );
    }
    if (promises.length === 0) {
      btn.innerHTML = orig;
      btn.disabled = false;
      setSnapInProgress(false);
      showToast("No snap source selected", "warning");
      return;
    }
    Promise.all(promises).then(function(results) {
      var msgs = [];
      var success = false;
      for (var i = 0; i < results.length; i++) {
        var r = results[i];
        if (r.error) {
          msgs.push("\u274C " + r.source + ": " + r.error);
        } else {
          msgs.push("\u2705 " + r.label);
          success = true;
        }
      }
      showToast(msgs.join(" \xB7 "), msgs.some(function(m) {
        return m.startsWith("\u274C");
      }) ? "warning" : "success");
      if (success) {
        fetchStatus().then(function(data) {
          document.dispatchEvent(new CustomEvent("niyantra:status-refreshed", { detail: { data } }));
        }).catch(function(err) {
          console.error("Failed to reload status after snap:", err);
        });
      }
    }).finally(function() {
      btn.innerHTML = orig;
      btn.disabled = false;
      btn.classList.remove("snapping");
      setSnapInProgress(false);
    });
  }

  // internal/web/src/charts/annotations.ts
  function getAnnotationMeta(type) {
    switch (type) {
      case "config_change":
        return { icon: "\u2699\uFE0F", color: "#6366f1" };
      case "account_added":
        return { icon: "\u2795", color: "#10b981" };
      case "account_removed":
        return { icon: "\u2796", color: "#ef4444" };
      case "subscription_created":
        return { icon: "\u{1F4B3}", color: "#f59e0b" };
      case "subscription_updated":
        return { icon: "\u270F\uFE0F", color: "#8b5cf6" };
      case "subscription_deleted":
        return { icon: "\u274C", color: "#ef4444" };
      case "budget_changed":
        return { icon: "\u{1F4B0}", color: "#f59e0b" };
      case "notification":
        return { icon: "\u{1F514}", color: "#ec4899" };
      case "quota_alert":
        return { icon: "\u26A0\uFE0F", color: "#ef4444" };
      default:
        return { icon: "\u{1F4CC}", color: "#94a3b8" };
    }
  }
  var SKIP_EVENTS = {
    "snapshot": true,
    "auto_capture": true,
    "snap": true,
    "poll_cycle": true
  };
  function parseAnnotations(entries) {
    var annotations = [];
    for (var i = 0; i < entries.length; i++) {
      var e = entries[i];
      if (SKIP_EVENTS[e.eventType]) continue;
      var meta = getAnnotationMeta(e.eventType);
      var label = e.eventType.replace(/_/g, " ");
      var details = {};
      try {
        details = JSON.parse(e.details || "{}");
      } catch (ex) {
      }
      if (details.key) {
        label = details.key + " \u2192 " + (details.value || "");
      } else if (e.accountEmail) {
        label += ": " + e.accountEmail;
      }
      if (label.length > 40) label = label.substring(0, 37) + "\u2026";
      annotations.push({
        date: e.timestamp,
        type: e.eventType,
        icon: meta.icon,
        color: meta.color,
        label,
        tooltip: meta.icon + " " + label + "\n" + new Date(e.timestamp).toLocaleString()
      });
    }
    return annotations;
  }
  function renderChartAnnotations(chartContainer, annotations, chartLabels, chartInstance) {
    var existing = chartContainer.querySelectorAll(".chart-annotation");
    for (var i = 0; i < existing.length; i++) existing[i].remove();
    if (!annotations.length || !chartInstance) return;
    var visible = annotations.slice(0, 10);
    var chartArea = chartInstance.chartArea;
    if (!chartArea) return;
    for (var j = 0; j < visible.length; j++) {
      var ann = visible[j];
      var annDate = new Date(ann.date);
      var bestIdx = -1;
      var bestDiff = Infinity;
      for (var k = 0; k < chartLabels.length; k++) {
        var labelDate = new Date(chartLabels[k]);
        if (isNaN(labelDate.getTime())) continue;
        var diff = Math.abs(labelDate.getTime() - annDate.getTime());
        if (diff < bestDiff) {
          bestDiff = diff;
          bestIdx = k;
        }
      }
      if (bestIdx < 0) continue;
      var x = chartInstance.scales.x.getPixelForValue(bestIdx);
      if (x < chartArea.left || x > chartArea.right) continue;
      var marker = document.createElement("div");
      marker.className = "chart-annotation";
      marker.style.left = x + "px";
      marker.style.top = chartArea.top + "px";
      marker.style.height = chartArea.bottom - chartArea.top + "px";
      marker.style.borderLeftColor = ann.color;
      var dot = document.createElement("span");
      dot.className = "chart-annotation-dot";
      dot.textContent = ann.icon;
      marker.appendChild(dot);
      var tip = document.createElement("div");
      tip.className = "chart-annotation-tooltip";
      tip.textContent = ann.label;
      marker.appendChild(tip);
      chartContainer.appendChild(marker);
    }
  }
  function loadChartAnnotations(chartContainer, chartLabels, chartInstance) {
    fetch("/api/activity?limit=100").then(function(r) {
      return r.json();
    }).then(function(data) {
      if (!data || !data.entries || data.entries.length === 0) return;
      var annotations = parseAnnotations(data.entries);
      if (annotations.length > 0) {
        renderChartAnnotations(chartContainer, annotations, chartLabels, chartInstance);
      }
    }).catch(function(err) {
      console.error("Chart annotations failed:", err);
    });
  }

  // internal/web/src/charts/history.ts
  var Chart = window.Chart;
  var historyChart = null;
  function updateChartTheme(theme) {
    if (!historyChart) return;
    var isDark = theme !== "light";
    var gridColor = isDark ? "rgba(255,255,255,0.06)" : "rgba(0,0,0,0.06)";
    var textColor = isDark ? "#94a3b8" : "#64748b";
    if (historyChart.options.scales && historyChart.options.scales.y) {
      historyChart.options.scales.y.grid.color = gridColor;
      historyChart.options.scales.y.ticks.color = textColor;
    }
    if (historyChart.options.scales && historyChart.options.scales.x) {
      historyChart.options.scales.x.grid.color = gridColor;
      historyChart.options.scales.x.ticks.color = textColor;
    }
    historyChart.update("none");
  }
  var chartAccountsList = [];
  function loadHistoryChart() {
    if (typeof Chart === "undefined") return;
    var provSel = document.getElementById("chart-provider");
    var accSel = document.getElementById("chart-account");
    var rangeSel = document.getElementById("chart-range");
    var provider = provSel ? provSel.value : "all";
    var accountId = accSel ? parseInt(accSel.value) || 0 : 0;
    var range = rangeSel ? rangeSel.value : "7d";
    var customContainer = document.getElementById("chart-custom-range");
    if (customContainer) {
      if (range === "custom") {
        customContainer.style.display = "flex";
      } else {
        customContainer.style.display = "none";
      }
    }
    var since = "";
    var until = "";
    var now = /* @__PURE__ */ new Date();
    if (range === "24h") {
      since = new Date(now.getTime() - 24 * 60 * 60 * 1e3).toISOString();
    } else if (range === "7d") {
      since = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1e3).toISOString();
    } else if (range === "30d") {
      since = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1e3).toISOString();
    } else if (range === "90d") {
      since = new Date(now.getTime() - 90 * 24 * 60 * 60 * 1e3).toISOString();
    } else if (range === "custom") {
      var startInput = document.getElementById("chart-start-date");
      var endInput = document.getElementById("chart-end-date");
      if (startInput && !startInput.value) {
        var dStart = new Date(Date.now() - 7 * 24 * 60 * 60 * 1e3);
        startInput.value = dStart.toISOString().split("T")[0];
      }
      if (endInput && !endInput.value) {
        endInput.value = (/* @__PURE__ */ new Date()).toISOString().split("T")[0];
      }
      if (startInput && startInput.value) {
        since = (/* @__PURE__ */ new Date(startInput.value + "T00:00:00")).toISOString();
      }
      if (endInput && endInput.value) {
        until = (/* @__PURE__ */ new Date(endInput.value + "T23:59:59")).toISOString();
      }
    }
    var url = "/api/history?limit=1000";
    if (provider !== "all") url += "&provider=" + provider;
    if (accountId > 0) url += "&account=" + accountId;
    if (since) url += "&since=" + encodeURIComponent(since);
    if (until) url += "&until=" + encodeURIComponent(until);
    fetch(url).then(function(res) {
      return res.json();
    }).then(function(data) {
      var snapshots = data.snapshots || [];
      updateKPINumbers(snapshots);
      renderHistoryChart(snapshots);
    }).catch(function(err) {
      console.error("Failed to load history:", err);
    });
  }
  function renderHistoryChart(snapshots) {
    var container = document.querySelector(".chart-container");
    if (!container || typeof Chart === "undefined") return;
    if (snapshots.length === 0) {
      container.innerHTML = '<div class="chart-empty"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" opacity="0.4"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 17v-5M15 17V7M12 17v-3"/></svg>No snapshot history found matching current filters.</div>';
      return;
    }
    container.innerHTML = '<canvas id="history-chart"></canvas>';
    snapshots = snapshots.slice().reverse();
    var labels = snapshots.map(function(s) {
      var d = new Date(s.capturedAt);
      return d.toLocaleDateString(void 0, { month: "short", day: "numeric" }) + " " + d.toLocaleTimeString(void 0, { hour: "2-digit", minute: "2-digit" });
    });
    var groupData = {};
    var groupNames = {
      claude_gpt: "Claude + GPT",
      gemini_unified: "Gemini Pool",
      cursor: "Cursor Quota",
      codex_5h: "Codex 5-Hour",
      codex_7d: "Codex 7-Day",
      copilot: "Copilot Premium",
      unknown: "Unknown"
    };
    var groupColors = {
      claude_gpt: "#D97757",
      gemini_unified: "#3B82F6",
      cursor: "#00E6FF",
      codex_5h: "#9B51E0",
      codex_7d: "#BB6BD9",
      copilot: "#2EA44F",
      unknown: "#64748B"
    };
    for (var i = 0; i < snapshots.length; i++) {
      var groups = snapshots[i].groups || [];
      for (var j = 0; j < groups.length; j++) {
        var g = groups[j];
        if (!groupData[g.groupKey]) groupData[g.groupKey] = [];
      }
    }
    var aiCreditsData = [];
    var hasAICredits = false;
    for (var i = 0; i < snapshots.length; i++) {
      var snap = snapshots[i];
      var groups = snap.groups || [];
      var seen = {};
      for (var j = 0; j < groups.length; j++) {
        var g = groups[j];
        if (!groupData[g.groupKey]) groupData[g.groupKey] = [];
        groupData[g.groupKey].push(Math.round(g.remainingPercent || 0));
        seen[g.groupKey] = true;
      }
      var keys = Object.keys(groupData);
      for (var k = 0; k < keys.length; k++) {
        if (!seen[keys[k]]) groupData[keys[k]].push(null);
      }
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
      if (!key || !groupNames[key]) continue;
      datasets.push({
        label: groupNames[key],
        data: groupData[key],
        borderColor: groupColors[key] || "#94a3b8",
        backgroundColor: (groupColors[key] || "#94a3b8") + "15",
        yAxisID: "y",
        fill: true,
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 6,
        pointHoverBorderWidth: 2,
        pointHoverBackgroundColor: groupColors[key] || "#94a3b8",
        pointHoverBorderColor: "#ffffff",
        borderWidth: 2.5
      });
    }
    if (hasAICredits) {
      datasets.push({
        label: "AI Credits",
        data: aiCreditsData,
        borderColor: "#fbbf24",
        // Amber
        backgroundColor: "transparent",
        yAxisID: "yCredits",
        borderDash: [6, 4],
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 6,
        pointHoverBorderWidth: 2,
        pointHoverBackgroundColor: "#fbbf24",
        pointHoverBorderColor: "#ffffff",
        borderWidth: 2.5
      });
    }
    var isDark = document.documentElement.getAttribute("data-theme") !== "light";
    var gridColor = isDark ? "rgba(255,255,255,0.06)" : "rgba(0,0,0,0.06)";
    var textColor = isDark ? "#94a3b8" : "#64748b";
    if (historyChart) historyChart.destroy();
    var ctx = document.getElementById("history-chart");
    if (!ctx) return;
    historyChart = new Chart(ctx, {
      type: "line",
      data: { labels, datasets },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: "index", intersect: false },
        plugins: {
          legend: {
            position: "bottom",
            labels: {
              color: textColor,
              font: { family: "'Inter', sans-serif", size: 11, weight: "500" },
              boxWidth: 8,
              boxHeight: 8,
              usePointStyle: true,
              pointStyle: "circle",
              padding: 20
            }
          },
          tooltip: {
            backgroundColor: isDark ? "rgba(15, 23, 42, 0.95)" : "rgba(255, 255, 255, 0.95)",
            titleColor: isDark ? "#f8fafc" : "#0f172a",
            bodyColor: isDark ? "#cbd5e1" : "#475569",
            borderColor: isDark ? "rgba(255, 255, 255, 0.1)" : "rgba(0, 0, 0, 0.08)",
            borderWidth: 1,
            padding: 12,
            cornerRadius: 8,
            titleFont: { family: "'Inter', sans-serif", weight: "700", size: 12 },
            bodyFont: { family: "'Inter', sans-serif", size: 11 },
            multiKeyBackground: "transparent",
            usePointStyle: true,
            boxWidth: 6,
            boxHeight: 6,
            boxPadding: 6,
            callbacks: {
              label: function(ctx2) {
                if (ctx2.dataset.yAxisID === "yCredits") return ctx2.dataset.label + ": " + ctx2.parsed.y.toLocaleString();
                return ctx2.dataset.label + ": " + ctx2.parsed.y + "%";
              }
            }
          }
        },
        scales: {
          y: {
            type: "linear",
            display: true,
            position: "left",
            min: 0,
            max: 100,
            grid: { color: gridColor, drawTicks: false },
            ticks: { color: textColor, font: { family: "'Inter', sans-serif", size: 10 }, padding: 8, callback: function(v) {
              return v + "%";
            } },
            border: { display: false }
          },
          yCredits: {
            type: "linear",
            display: hasAICredits,
            position: "right",
            grid: { display: false },
            ticks: { color: isDark ? "#fbbf24" : "#d97706", font: { family: "'Inter', sans-serif", size: 10 }, padding: 8 },
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
    var chartEl = document.querySelector(".chart-container");
    if (chartEl && historyChart) {
      var rawTimestamps = snapshots.map(function(s) {
        return s.capturedAt;
      });
      loadChartAnnotations(chartEl, rawTimestamps, historyChart);
    }
  }
  function populateChartAccountSelect(data) {
    var accts = data.allAccounts || data.accounts;
    if (!accts) return;
    chartAccountsList = accts;
    filterChartAccounts();
  }
  function filterChartAccounts() {
    var provSel = document.getElementById("chart-provider");
    var accSel = document.getElementById("chart-account");
    if (!accSel) return;
    var selectedProvider = provSel ? provSel.value : "all";
    var currentlySelectedValue = accSel.value;
    while (accSel.options.length > 1) accSel.remove(1);
    for (var i = 0; i < chartAccountsList.length; i++) {
      var acc = chartAccountsList[i];
      if (selectedProvider !== "all" && acc.provider !== selectedProvider) {
        continue;
      }
      var opt = document.createElement("option");
      var id = acc.id !== void 0 ? acc.id : acc.accountId;
      opt.value = id;
      opt.textContent = acc.email;
      accSel.appendChild(opt);
    }
    var optionExists = false;
    for (var j = 0; j < accSel.options.length; j++) {
      if (accSel.options[j].value === currentlySelectedValue) {
        accSel.selectedIndex = j;
        optionExists = true;
        break;
      }
    }
    if (!optionExists) {
      accSel.value = "0";
    }
  }
  function updateKPINumbers(snapshots) {
    var totalEl = document.getElementById("kpi-total-snapshots");
    var avgEl = document.getElementById("kpi-avg-remaining");
    var trendEl = document.getElementById("kpi-trend");
    if (!totalEl || !avgEl || !trendEl) return;
    if (snapshots.length === 0) {
      totalEl.textContent = "0";
      avgEl.textContent = "-%";
      trendEl.textContent = "No Data";
      trendEl.className = "kpi-value";
      return;
    }
    totalEl.textContent = snapshots.length.toString();
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
    avgEl.textContent = avgRemaining + "%";
    if (snapshots.length < 2) {
      trendEl.textContent = "Stable";
      trendEl.className = "kpi-value trend-stable";
      return;
    }
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
    var trendCard = trendEl.closest(".kpi-card");
    if (diff > 1.5) {
      trendEl.innerHTML = 'Improving <span class="trend-icon">\u25B2</span>';
      trendEl.className = "kpi-value trend-up";
      if (trendCard) {
        trendCard.className = "kpi-card kpi-trend-improving";
      }
    } else if (diff < -1.5) {
      trendEl.innerHTML = 'Declining <span class="trend-icon">\u25BC</span>';
      trendEl.className = "kpi-value trend-down";
      if (trendCard) {
        trendCard.className = "kpi-card kpi-trend-declining";
      }
    } else {
      trendEl.innerHTML = 'Stable <span class="trend-icon">\u25CF</span>';
      trendEl.className = "kpi-value trend-stable";
      if (trendCard) {
        trendCard.className = "kpi-card kpi-trend-stable";
      }
    }
  }

  // internal/web/src/advanced/alerts.ts
  function loadSystemAlerts() {
    fetch("/api/alerts").then(function(r) {
      return r.json();
    }).then(function(data) {
      var container = document.getElementById("alert-banner-container");
      if (!container) return;
      var alerts = data.alerts || [];
      if (alerts.length === 0) {
        container.innerHTML = "";
        return;
      }
      var html = "";
      var shown = Math.min(alerts.length, 3);
      for (var i = 0; i < shown; i++) {
        var a = alerts[i];
        var icon = a.severity === "critical" ? "\u{1F6A8}" : a.severity === "warning" ? "\u26A0\uFE0F" : "\u2139\uFE0F";
        html += '<div class="alert-banner ' + esc(a.severity) + '"><span class="alert-banner-icon">' + icon + '</span><div class="alert-banner-content"><div class="alert-banner-title">' + esc(a.category) + '</div><div class="alert-banner-msg">' + esc(a.message) + '</div></div><button class="alert-banner-dismiss" data-alert-dismiss="' + a.id + '" title="Dismiss">&times;</button></div>';
      }
      if (alerts.length > 3) {
        html += '<div class="alert-more-link" data-alert-nav="overview">+ ' + (alerts.length - 3) + " more alert(s)</div>";
      }
      container.innerHTML = html;
      container.querySelectorAll("[data-alert-dismiss]").forEach(function(el) {
        el.addEventListener("click", function() {
          var button = el;
          dismissAlert(button.dataset.alertDismiss || "");
        });
      });
      var moreLink = container.querySelector('[data-alert-nav="overview"]');
      if (moreLink) {
        moreLink.addEventListener("click", function() {
          switchToTab("overview");
        });
      }
    }).catch(function() {
    });
  }
  function dismissAlert(id) {
    fetch("/api/alerts/dismiss", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id })
    }).then(function() {
      loadSystemAlerts();
      showToast("Alert dismissed", "success");
    }).catch(function() {
      showToast("Failed to dismiss alert", "error");
    });
  }

  // internal/web/src/settings/activity.ts
  function loadActivityLog() {
    var filter = document.getElementById("activity-filter").value;
    var url = "/api/activity?limit=50";
    if (filter) url += "&type=" + filter;
    fetch(url).then(function(r) {
      return r.json();
    }).then(function(data) {
      var container = document.getElementById("activity-log");
      if (!data.entries || data.entries.length === 0) {
        container.innerHTML = '<div class="activity-empty">No activity' + (filter ? ' for "' + filter + '"' : "") + " yet</div>";
        return;
      }
      var html = "";
      data.entries.forEach(function(entry) {
        var time = entry.timestamp ? entry.timestamp.replace("T", " ").substring(5, 16) : "";
        var detail = formatActivityDetail(entry);
        html += '<div class="activity-entry"><span class="activity-time">' + time + '</span><span class="activity-type ' + esc(entry.eventType) + '">' + esc(entry.eventType.replace(/_/g, " ")) + '</span><span class="activity-detail">' + detail + "</span></div>";
      });
      container.innerHTML = html;
    }).catch(function() {
      document.getElementById("activity-log").innerHTML = '<div class="activity-empty">Failed to load activity log</div>';
    });
  }
  function formatActivityDetail(entry) {
    try {
      var d = JSON.parse(entry.details || "{}");
      switch (entry.eventType) {
        case "snap":
          return esc(entry.accountEmail || "") + (d.method ? " \xB7 " + d.method : "") + (d.source ? " via " + d.source : "");
        case "snap_failed":
          return esc(d.error || "Unknown error");
        case "config_change":
          return esc(d.key || "") + ": " + esc(d.from || '""') + " \u2192 " + esc(d.to || '""');
        case "server_start":
          return "Port " + (d.port || "?") + " \xB7 " + esc(d.mode || "manual") + " mode";
        case "sub_created":
        case "sub_deleted":
          return esc(d.platform || "");
        case "auto_link":
          return esc(entry.accountEmail || "") + " \u2192 " + esc(d.platform || "");
        case "codex_snap":
          var acctId = entry.accountEmail || "";
          if (acctId.length > 20) acctId = acctId.substring(0, 6) + ".." + acctId.slice(-6);
          return esc(acctId) + (d.plan ? " (" + esc(d.plan) + ")" : "");
        case "model_reset":
          return esc(entry.accountEmail || "");
        case "quota_alert":
          return "\u{1F514} " + esc(d.model || "") + " \u2014 " + (d.remainingPct != null ? d.remainingPct.toFixed(1) + "% remaining" : "");
        default:
          return entry.accountEmail ? esc(entry.accountEmail) : "";
      }
    } catch (e) {
      return "";
    }
  }

  // internal/web/src/settings/mode.ts
  var modeRefreshTimer = null;
  function loadMode() {
    fetch("/api/mode").then(function(r) {
      return r.json();
    }).then(function(data) {
      var badge = document.getElementById("mode-badge");
      var label = document.getElementById("mode-label");
      if (data.mode === "auto") {
        badge.className = "mode-badge mode-auto";
        label.textContent = "Auto";
      } else {
        badge.className = "mode-badge mode-manual";
        label.textContent = "Manual";
      }
      var statusEl = document.getElementById("polling-status");
      if (statusEl) {
        if (data.isPolling) {
          var lastMsg = "";
          if (data.lastPoll) {
            lastMsg = "Last: " + formatTimeAgo(data.lastPoll);
            if (data.lastPollOK === false) lastMsg += " (failed)";
          } else {
            lastMsg = "Starting...";
          }
          statusEl.innerHTML = '<span class="polling-dot"></span> Polling every ' + formatPollInterval(data.pollInterval) + " \xB7 " + lastMsg;
          statusEl.style.display = "";
        } else {
          statusEl.style.display = "none";
        }
      }
      var aboutEl = document.getElementById("s-about-info");
      if (aboutEl) {
        var srcCount = (data.sources || []).filter(function(s) {
          return s.enabled;
        }).length;
        var schemaV = data.schemaVersion ? "Schema v" + data.schemaVersion : "Schema";
        var presetCount = presetsData.length || 0;
        aboutEl.textContent = schemaV + " \xB7 " + presetCount + " presets \xB7 Mode: " + (data.mode === "auto" ? "Auto" : "Manual") + (data.isPolling ? " (polling)" : "") + " \xB7 " + srcCount + " active source" + (srcCount !== 1 ? "s" : "");
      }
      if (modeRefreshTimer) {
        clearInterval(modeRefreshTimer);
        modeRefreshTimer = null;
      }
      if (data.isPolling) {
        modeRefreshTimer = setInterval(function() {
          loadMode();
          loadSystemAlerts();
          var activeTab = document.querySelector(".tab-btn.active");
          if (activeTab && activeTab.getAttribute("data-tab") === "settings") {
            loadActivityLog();
          }
        }, 3e4);
      }
    }).catch(function() {
    });
  }

  // internal/web/src/settings/data.ts
  function loadDataSources() {
    fetch("/api/mode").then(function(r) {
      return r.json();
    }).then(function(data) {
      var container = document.getElementById("data-sources-list");
      if (!data.sources || data.sources.length === 0) {
        container.innerHTML = "";
        return;
      }
      var html = '<div style="font-size:12px;font-weight:600;color:var(--text-secondary);margin-bottom:4px;margin-top:4px">Data Sources</div>';
      data.sources.forEach(function(src) {
        var meta = src.captureCount + " captures";
        if (src.lastCapture) {
          meta += " \xB7 Last: " + formatTimeAgo(src.lastCapture);
        }
        html += '<div class="data-source-item"><div class="data-source-info"><span class="data-source-name">' + esc(src.name) + '</span><span class="data-source-meta">' + esc(src.sourceType) + " \xB7 " + meta + '</span></div><span class="data-source-status ' + (src.enabled ? "enabled" : "disabled") + '">' + (src.enabled ? "\u25CF Active" : "\u25CB Disabled") + "</span></div>";
      });
      container.innerHTML = html;
    }).catch(function() {
    });
  }

  // internal/web/src/settings/pricing.ts
  var pricingDataCache = null;
  function loadModelPricing() {
    fetch("/api/config/pricing").then(function(res) {
      return res.json();
    }).then(function(data) {
      pricingDataCache = data.pricing || [];
      renderPricingTable(pricingDataCache);
    }).catch(function(err) {
      console.error("Failed to load model pricing:", err);
    });
  }
  function renderPricingTable(pricing) {
    var tbody = document.getElementById("pricing-tbody");
    if (!tbody) return;
    var providerIcons = { anthropic: "\u{1F7E4}", openai: "\u{1F7E2}", google: "\u{1F535}" };
    var html = "";
    for (var i = 0; i < pricing.length; i++) {
      var p = pricing[i];
      var providerCls = p.provider || "custom";
      var providerLabel = p.provider ? p.provider.charAt(0).toUpperCase() + p.provider.slice(1) : "Custom";
      var icon = providerIcons[p.provider] || "\u26AA";
      html += '<tr data-pricing-idx="' + i + '"><td><span class="pricing-model-name">' + esc(p.displayName) + '</span></td><td><span class="pricing-provider ' + esc(providerCls) + '">' + icon + " " + esc(providerLabel) + '</span></td><td style="text-align:right"><input type="number" class="pricing-input" data-field="inputPer1M" step="0.01" min="0" value="' + p.inputPer1M + '"></td><td style="text-align:right"><input type="number" class="pricing-input" data-field="outputPer1M" step="0.01" min="0" value="' + p.outputPer1M + '"></td><td style="text-align:right"><input type="number" class="pricing-input" data-field="cachePer1M" step="0.001" min="0" value="' + p.cachePer1M + '"></td><td><button class="pricing-delete-btn" data-pricing-del="' + i + '" title="Remove this model">\u2715</button></td></tr>';
    }
    tbody.innerHTML = html;
    tbody.querySelectorAll(".pricing-input").forEach(function(input) {
      input.addEventListener("change", function() {
        var tr = input.closest("tr");
        var idx = parseInt(tr.dataset.pricingIdx);
        var field = input.dataset.field;
        var val = parseFloat(input.value) || 0;
        if (val < 0) val = 0;
        input.value = String(val);
        if (pricingDataCache && pricingDataCache[idx]) {
          pricingDataCache[idx][field] = val;
          savePricingFromTable();
        }
      });
    });
    tbody.querySelectorAll(".pricing-delete-btn").forEach(function(btn) {
      btn.addEventListener("click", function() {
        var idx = parseInt(btn.dataset.pricingDel);
        deletePricingRow(idx);
      });
    });
  }
  function savePricingFromTable() {
    if (!pricingDataCache || pricingDataCache.length === 0) return;
    fetch("/api/config/pricing", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pricing: pricingDataCache })
    }).then(function(res) {
      return res.json();
    }).then(function(data) {
      if (data.error) {
        showToast("\u274C " + data.error, "error");
        return;
      }
      showToast("\u{1F4B0} Pricing saved", "success");
    }).catch(function() {
      showToast("\u274C Failed to save pricing", "error");
    });
  }
  function addPricingRow() {
    if (!pricingDataCache) pricingDataCache = [];
    var newModel = {
      modelId: "custom-" + Date.now(),
      displayName: "New Model",
      provider: "custom",
      inputPer1M: 1,
      outputPer1M: 5,
      cachePer1M: 0.1
    };
    pricingDataCache.push(newModel);
    renderPricingTable(pricingDataCache);
    var tbody = document.getElementById("pricing-tbody");
    var lastRow = tbody.lastElementChild;
    if (lastRow) {
      var nameCell = lastRow.querySelector(".pricing-model-name");
      if (nameCell) {
        nameCell.contentEditable = "true";
        nameCell.focus();
        var range = document.createRange();
        range.selectNodeContents(nameCell);
        var sel = window.getSelection();
        sel.removeAllRanges();
        sel.addRange(range);
        nameCell.addEventListener("blur", function() {
          nameCell.contentEditable = "false";
          var idx = parseInt(lastRow.dataset.pricingIdx);
          var newName = nameCell.textContent.trim();
          if (newName && pricingDataCache[idx]) {
            pricingDataCache[idx].displayName = newName;
            pricingDataCache[idx].modelId = newName.toLowerCase().replace(/[^a-z0-9]+/g, "-");
            savePricingFromTable();
          }
        }, { once: true });
        nameCell.addEventListener("keydown", function(e) {
          if (e.key === "Enter") {
            e.preventDefault();
            nameCell.blur();
          }
        });
      }
    }
    showToast("\u{1F4B0} New model added \u2014 edit the name and prices", "info");
  }
  function deletePricingRow(idx) {
    if (!pricingDataCache || idx < 0 || idx >= pricingDataCache.length) return;
    var name = pricingDataCache[idx].displayName;
    if (!confirm('Remove pricing for "' + name + '"?')) return;
    pricingDataCache.splice(idx, 1);
    renderPricingTable(pricingDataCache);
    savePricingFromTable();
    showToast("\u{1F5D1}\uFE0F Removed " + name, "success");
  }
  function resetPricingDefaults() {
    if (!confirm("Reset all model pricing to current market defaults? This will overwrite your custom prices.")) return;
    var defaults = [
      { modelId: "claude-opus-4.6", displayName: "Claude Opus 4.6", provider: "anthropic", inputPer1M: 5, outputPer1M: 25, cachePer1M: 0.5 },
      { modelId: "claude-sonnet-4.6", displayName: "Claude Sonnet 4.6", provider: "anthropic", inputPer1M: 3, outputPer1M: 15, cachePer1M: 0.3 },
      { modelId: "claude-haiku-4.5", displayName: "Claude Haiku 4.5", provider: "anthropic", inputPer1M: 1, outputPer1M: 5, cachePer1M: 0.1 },
      { modelId: "gpt-4o", displayName: "GPT-4o", provider: "openai", inputPer1M: 2.5, outputPer1M: 10, cachePer1M: 1.25 },
      { modelId: "gemini-3.1-pro", displayName: "Gemini 3.1 Pro", provider: "google", inputPer1M: 2, outputPer1M: 12, cachePer1M: 0.5 },
      { modelId: "gemini-2.5-flash", displayName: "Gemini 2.5 Flash", provider: "google", inputPer1M: 0.3, outputPer1M: 2.5, cachePer1M: 0.075 },
      // Contemporary Google Gemini Models
      { modelId: "MODEL_PLACEHOLDER_M133", displayName: "Gemini 3.5 Flash (High)", provider: "google", inputPer1M: 0.3, outputPer1M: 2.5, cachePer1M: 0.075 },
      { modelId: "MODEL_PLACEHOLDER_M20", displayName: "Gemini 3.5 Flash (Medium)", provider: "google", inputPer1M: 0.3, outputPer1M: 2.5, cachePer1M: 0.075 },
      { modelId: "MODEL_PLACEHOLDER_M16", displayName: "Gemini 3.1 Pro (High)", provider: "google", inputPer1M: 2, outputPer1M: 12, cachePer1M: 0.5 },
      { modelId: "MODEL_PLACEHOLDER_M36", displayName: "Gemini 3.1 Pro (Low)", provider: "google", inputPer1M: 2, outputPer1M: 12, cachePer1M: 0.5 }
    ];
    pricingDataCache = defaults;
    renderPricingTable(pricingDataCache);
    fetch("/api/config/pricing", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pricing: defaults })
    }).then(function(res) {
      return res.json();
    }).then(function(data) {
      if (data.error) {
        showToast("\u274C " + data.error, "error");
        return;
      }
      showToast("\u21BB Pricing reset to defaults", "success");
    }).catch(function() {
      showToast("\u274C Failed to reset pricing", "error");
    });
  }

  // internal/web/src/advanced/claude.ts
  function loadClaudeBridgeStatus() {
    fetch("/api/claude/status").then(function(r) {
      return r.json();
    }).then(function(data) {
      var statusEl = document.getElementById("claude-bridge-status");
      if (!statusEl) return;
      var bridgeOn = data.bridgeEnabled;
      var installed = data.installed;
      if (!bridgeOn) {
        statusEl.style.display = "none";
        return;
      }
      var msg = "";
      if (!installed) {
        msg = "\u26A0\uFE0F Claude Code not detected (~/.claude/ not found)";
      } else if (data.bridgeFresh) {
        msg = '<span class="claude-bridge-dot"></span> Bridge active';
        if (data.snapshot) {
          msg += " \xB7 5h: " + data.snapshot.fiveHourPct.toFixed(1) + "% used";
        }
      } else if (data.snapshot) {
        msg = '<span class="claude-bridge-dot stale"></span> Last data: ' + formatTimeAgo(data.snapshot.capturedAt);
      } else {
        msg = '<span class="claude-bridge-dot off"></span> Waiting for Claude Code statusline data...';
      }
      statusEl.innerHTML = msg;
      statusEl.style.display = "";
    }).catch(function() {
    });
  }

  // internal/web/src/settings/plugins.ts
  function loadPlugins() {
    fetch("/api/plugins").then(function(r) {
      return r.json();
    }).then(function(data) {
      var container = document.getElementById("plugins-list");
      if (!container) return;
      var plugins = data.plugins || [];
      var pluginsDir = data.pluginsDir || "";
      var errors = data.errors || [];
      if (plugins.length === 0 && errors.length === 0) {
        container.innerHTML = '<div class="plugin-empty"><div class="plugin-empty-icon">\u{1F9E9}</div><div class="plugin-empty-title">No plugins installed</div><div class="plugin-empty-hint">Add plugins to <code>' + esc(pluginsDir) + "</code><br>Each plugin needs a <code>plugin.json</code> manifest and an executable entry point.</div></div>";
        return;
      }
      var html = '<div class="plugin-warning"><strong>Trusted local code only.</strong> Enabled plugins execute only through the local polling agent with your current user permissions. Manual HTTP plugin execution is disabled.</div>';
      if (errors.length > 0) {
        html += '<div class="plugin-errors">';
        errors.forEach(function(e) {
          html += '<div class="plugin-error">\u26A0\uFE0F ' + esc(e) + "</div>";
        });
        html += "</div>";
      }
      plugins.forEach(function(p) {
        var meta = p.captureCount + " captures";
        if (p.lastCapture) {
          meta += " \xB7 Last: " + formatTimeAgo(p.lastCapture);
        }
        html += '<div class="plugin-card" data-plugin-id="' + esc(p.manifest.id) + '">';
        html += '<div class="plugin-header">';
        html += '<div class="plugin-info">';
        html += '<div class="plugin-name">' + esc(p.manifest.name) + '<span class="plugin-version">v' + esc(p.manifest.version) + "</span></div>";
        html += '<div class="plugin-meta">' + esc(p.manifest.description || "No description") + "</div>";
        if (p.manifest.author) {
          html += '<div class="plugin-meta">By ' + esc(p.manifest.author) + " \xB7 " + meta + "</div>";
        }
        html += "</div>";
        html += '<div class="plugin-actions">';
        html += '<label class="toggle-label">';
        html += '<input type="checkbox" class="plugin-toggle" data-plugin="' + esc(p.manifest.id) + '"' + (p.enabled ? " checked" : "") + ">";
        html += '<span class="toggle-slider"></span>';
        html += "</label>";
        html += "</div>";
        html += "</div>";
        var configKeys = Object.keys(p.manifest.config || {});
        if (configKeys.length > 0) {
          html += '<div class="plugin-config" style="' + (p.enabled ? "" : "display:none") + '">';
          configKeys.forEach(function(key) {
            var field = p.manifest.config[key];
            var val = p.config[key] || field.default || "";
            html += '<div class="plugin-config-row">';
            html += '<label class="plugin-config-label">' + esc(field.label || key);
            if (field.required) html += ' <span style="color:var(--accent)">*</span>';
            html += "</label>";
            if (field.secret) {
              html += '<input type="password" class="plugin-config-input" data-plugin="' + esc(p.manifest.id) + '" data-key="' + esc(key) + '" placeholder="' + (val === "configured" ? "configured" : "Enter " + esc(field.label || key)) + '">';
            } else {
              html += '<input type="text" class="plugin-config-input" data-plugin="' + esc(p.manifest.id) + '" data-key="' + esc(key) + '" value="' + esc(val) + '" placeholder="Enter ' + esc(field.label || key) + '">';
            }
            html += "</div>";
          });
          html += "</div>";
        }
        html += '<div class="plugin-footer" style="' + (p.enabled ? "" : "display:none") + '">';
        html += '<span class="plugin-test-result" id="plugin-result-' + esc(p.manifest.id) + '">Manual HTTP run disabled</span>';
        html += "</div>";
        html += "</div>";
      });
      container.innerHTML = html;
      container.querySelectorAll(".plugin-toggle").forEach(function(el) {
        el.addEventListener("change", function() {
          var input = el;
          var pluginId = input.dataset.plugin;
          var enabled = input.checked;
          fetch("/api/plugins/" + pluginId + "/config", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ enabled: enabled ? "true" : "false" })
          }).then(function(r) {
            return r.json().then(function(data2) {
              if (!r.ok || data2.error) {
                throw new Error(data2.error || "Failed to update plugin");
              }
              return data2;
            });
          }).then(function() {
            showToast(enabled ? "\u{1F9E9} Plugin enabled: " + pluginId : "\u{1F9E9} Plugin disabled: " + pluginId, "success");
            var card = input.closest(".plugin-card");
            var config = card.querySelector(".plugin-config");
            var footer = card.querySelector(".plugin-footer");
            if (config) config.style.display = enabled ? "" : "none";
            if (footer) footer.style.display = enabled ? "" : "none";
            loadDataSources();
          }).catch(function(err) {
            input.checked = !enabled;
            showToast("\u274C " + (err instanceof Error ? err.message : "Failed to update plugin"), "error");
          });
        });
      });
      container.querySelectorAll(".plugin-config-input").forEach(function(el) {
        el.addEventListener("change", function() {
          var input = el;
          var pluginId = input.dataset.plugin;
          var key = input.dataset.key;
          var val = input.value.trim();
          if (!val) return;
          var body = {};
          body[key] = val;
          fetch("/api/plugins/" + pluginId + "/config", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body)
          }).then(function(r) {
            return r.json().then(function(data2) {
              if (!r.ok || data2.error) {
                throw new Error(data2.error || "Failed to save config");
              }
              return data2;
            });
          }).then(function() {
            showToast("\u{1F9E9} " + key + " saved", "success");
            if (input.type === "password") {
              input.value = "";
              input.placeholder = "configured";
            }
          }).catch(function(err) {
            showToast("\u274C " + (err instanceof Error ? err.message : "Failed to save config"), "error");
          });
        });
      });
    }).catch(function() {
    });
  }

  // internal/web/src/settings/settings.ts
  function initSettings() {
    var themeEl = document.getElementById("s-theme");
    var savedTheme = localStorage.getItem("niyantra-theme") || "dark";
    themeEl.value = savedTheme;
    if (!themeEl) return;
    themeEl.addEventListener("change", function() {
      var val = themeEl.value;
      if (val === "system") {
        localStorage.removeItem("niyantra-theme");
        var prefer = window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
        document.documentElement.setAttribute("data-theme", prefer);
      } else {
        localStorage.setItem("niyantra-theme", val);
        document.documentElement.setAttribute("data-theme", val);
      }
      var applied = document.documentElement.getAttribute("data-theme");
      updateChartTheme(applied);
    });
    loadConfig().then(function() {
      var cfg = serverConfig;
      migrateLocalStorage(cfg);
      var budgetEl = document.getElementById("s-budget");
      var currencyEl = document.getElementById("s-currency");
      var autoCaptureEl = document.getElementById("s-auto-capture");
      var autoLinkEl = document.getElementById("s-auto-link");
      var pollEl = document.getElementById("s-poll-interval");
      var retentionEl = document.getElementById("s-retention");
      budgetEl.value = String(parseFloat(cfg["budget_monthly"] || "0") || "");
      currencyEl.value = cfg["currency"] || "USD";
      autoCaptureEl.checked = cfg["auto_capture"] === "true";
      autoLinkEl.checked = cfg["auto_link_subs"] !== "false";
      pollEl.value = cfg["poll_interval"] || "300";
      retentionEl.value = cfg["retention_days"] || "365";
      document.getElementById("poll-interval-row").style.display = autoCaptureEl.checked ? "" : "none";
      budgetEl.addEventListener("change", function() {
        var val = parseFloat(budgetEl.value) || 0;
        setBudget(val);
        if (val > 0) showToast("\u2705 Budget: $" + val.toFixed(0) + "/mo", "success");
      });
      currencyEl.addEventListener("change", function() {
        updateConfig("currency", currencyEl.value);
        showToast("\u2705 Currency: " + currencyEl.value, "success");
      });
      autoCaptureEl.addEventListener("change", function() {
        var val = autoCaptureEl.checked ? "true" : "false";
        updateConfig("auto_capture", val).then(function() {
          loadMode();
          showToast(autoCaptureEl.checked ? "\u{1F7E2} Auto-capture started" : "\u23F8\uFE0F Auto-capture stopped", "success");
        });
        document.getElementById("poll-interval-row").style.display = autoCaptureEl.checked ? "" : "none";
      });
      autoLinkEl.addEventListener("change", function() {
        updateConfig("auto_link_subs", autoLinkEl.checked ? "true" : "false");
      });
      pollEl.addEventListener("change", function() {
        var v = pollEl.value;
        updateConfig("poll_interval", v).then(function() {
          var label = pollEl.options[pollEl.selectedIndex].text;
          showToast("\u23F1\uFE0F Interval updated to " + label + " \u2014 takes effect on next cycle.", "success");
          loadMode();
        });
      });
      retentionEl.addEventListener("change", function() {
        var v = parseInt(retentionEl.value);
        if (v >= 30 && v <= 3650) updateConfig("retention_days", v.toString());
      });
      var claudeBridgeEl = document.getElementById("s-claude-bridge");
      if (claudeBridgeEl) {
        claudeBridgeEl.checked = cfg["claude_bridge"] === "true";
        claudeBridgeEl.addEventListener("change", function() {
          var val = claudeBridgeEl.checked ? "true" : "false";
          updateConfig("claude_bridge", val).then(function() {
            showToast(claudeBridgeEl.checked ? "\u{1F517} Claude Code bridge enabled" : "\u{1F517} Bridge disabled", "success");
            loadClaudeBridgeStatus();
          });
        });
        loadClaudeBridgeStatus();
      }
      var notifyEl = document.getElementById("s-notify-enabled");
      var thresholdEl = document.getElementById("s-notify-threshold");
      var thresholdRow = document.getElementById("notify-threshold-row");
      var testRow = document.getElementById("notify-test-row");
      if (notifyEl) {
        notifyEl.checked = cfg["notify_enabled"] === "true";
        thresholdEl.value = cfg["notify_threshold"] || "10";
        thresholdRow.style.display = notifyEl.checked ? "" : "none";
        testRow.style.display = notifyEl.checked ? "" : "none";
        notifyEl.addEventListener("change", function() {
          var val = notifyEl.checked ? "true" : "false";
          updateConfig("notify_enabled", val).then(function() {
            showToast(notifyEl.checked ? "\u{1F514} Notifications enabled" : "\u{1F515} Notifications disabled", "success");
          });
          thresholdRow.style.display = notifyEl.checked ? "" : "none";
          testRow.style.display = notifyEl.checked ? "" : "none";
        });
        thresholdEl.addEventListener("change", function() {
          var v = parseInt(thresholdEl.value);
          if (v >= 5 && v <= 50) {
            updateConfig("notify_threshold", v.toString());
            showToast("\u{1F514} Threshold: " + v + "%", "success");
          }
        });
        document.getElementById("notify-test-btn").addEventListener("click", function() {
          fetch("/api/notify/test", { method: "POST" }).then(function(r) {
            return r.json();
          }).then(function(data) {
            if (data.error) showToast("\u274C " + data.error, "error");
            else showToast("\u{1F514} Test notification sent!", "success");
          }).catch(function() {
            showToast("\u274C Failed to send test", "error");
          });
        });
      }
      var smtpEnabledEl = document.getElementById("s-smtp-enabled");
      var smtpConfigRows = document.getElementById("smtp-config-rows");
      if (smtpEnabledEl) {
        smtpEnabledEl.checked = cfg["smtp_enabled"] === "true";
        smtpConfigRows.style.display = smtpEnabledEl.checked ? "" : "none";
        var smtpHostEl = document.getElementById("s-smtp-host");
        var smtpPortEl = document.getElementById("s-smtp-port");
        var smtpTlsEl = document.getElementById("s-smtp-tls");
        var smtpUserEl = document.getElementById("s-smtp-user");
        var smtpPassEl = document.getElementById("s-smtp-pass");
        var smtpFromEl = document.getElementById("s-smtp-from");
        var smtpToEl = document.getElementById("s-smtp-to");
        smtpHostEl.value = cfg["smtp_host"] || "";
        smtpPortEl.value = cfg["smtp_port"] || "587";
        smtpTlsEl.value = cfg["smtp_tls"] || "starttls";
        smtpUserEl.value = cfg["smtp_user"] || "";
        smtpFromEl.value = cfg["smtp_from"] || "";
        smtpToEl.value = cfg["smtp_to"] || "";
        if (cfg["smtp_pass"]) {
          smtpPassEl.value = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
        }
        smtpPassEl.addEventListener("focus", function() {
          if (smtpPassEl.value === "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022") {
            smtpPassEl.select();
          }
        });
        smtpEnabledEl.addEventListener("change", function() {
          var val = smtpEnabledEl.checked ? "true" : "false";
          updateConfig("smtp_enabled", val).then(function() {
            showToast(smtpEnabledEl.checked ? "\u{1F4E7} Email notifications enabled" : "\u{1F4E7} Email notifications disabled", "success");
          });
          smtpConfigRows.style.display = smtpEnabledEl.checked ? "" : "none";
        });
        smtpHostEl.addEventListener("change", function() {
          updateConfig("smtp_host", smtpHostEl.value.trim());
          showToast("\u{1F4E7} SMTP host saved", "success");
        });
        smtpPortEl.addEventListener("change", function() {
          updateConfig("smtp_port", smtpPortEl.value);
          showToast("\u{1F4E7} SMTP port saved", "success");
        });
        smtpTlsEl.addEventListener("change", function() {
          updateConfig("smtp_tls", smtpTlsEl.value);
          showToast("\u{1F4E7} Encryption mode saved", "success");
        });
        smtpUserEl.addEventListener("change", function() {
          updateConfig("smtp_user", smtpUserEl.value.trim());
          showToast("\u{1F4E7} SMTP username saved", "success");
        });
        smtpPassEl.addEventListener("change", function() {
          var val = smtpPassEl.value.trim();
          if (val === "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022") return;
          updateConfig("smtp_pass", val).then(function() {
            if (val) {
              showToast("\u{1F4E7} SMTP password saved", "success");
              smtpPassEl.value = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
            } else {
              showToast("\u{1F4E7} SMTP password cleared", "success");
              smtpPassEl.value = "";
            }
          });
        });
        smtpFromEl.addEventListener("change", function() {
          updateConfig("smtp_from", smtpFromEl.value.trim());
          showToast("\u{1F4E7} From address saved", "success");
        });
        smtpToEl.addEventListener("change", function() {
          updateConfig("smtp_to", smtpToEl.value.trim());
          showToast("\u{1F4E7} To address saved", "success");
        });
        document.getElementById("smtp-test-btn").addEventListener("click", function() {
          var btn = document.getElementById("smtp-test-btn");
          btn.disabled = true;
          btn.textContent = "\u{1F4E7} Sending...";
          fetch("/api/notify/test-email", { method: "POST" }).then(function(r) {
            return r.json();
          }).then(function(data) {
            if (data.error) showToast("\u274C " + data.error, "error");
            else showToast("\u{1F4E7} Test email sent!", "success");
          }).catch(function() {
            showToast("\u274C Failed to send test email", "error");
          }).finally(function() {
            btn.disabled = false;
            btn.textContent = "\u{1F4E7} Send Test";
          });
        });
      }
      var webhookEnabledEl = document.getElementById("s-webhook-enabled");
      var webhookConfigRows = document.getElementById("webhook-config-rows");
      if (webhookEnabledEl) {
        let updateWebhookLabels2 = function() {
          var urlLabel = document.getElementById("webhook-url-label");
          var urlHint = document.getElementById("webhook-url-hint");
          var secretLabel = document.getElementById("webhook-secret-label");
          var secretHint = document.getElementById("webhook-secret-hint");
          var secretRow = document.getElementById("webhook-secret-row");
          var urlInput = document.getElementById("s-webhook-url");
          switch (webhookTypeEl.value) {
            case "discord":
              urlLabel.textContent = "Webhook URL";
              urlHint.textContent = "Discord channel webhook URL";
              urlInput.placeholder = "https://discord.com/api/webhooks/...";
              secretRow.style.display = "none";
              break;
            case "telegram":
              urlLabel.textContent = "Chat ID";
              urlHint.textContent = "Telegram chat/group ID (numeric)";
              urlInput.placeholder = "123456789";
              secretRow.style.display = "";
              secretLabel.textContent = "Bot Token";
              secretHint.textContent = "Telegram bot token from @BotFather";
              webhookSecretEl.placeholder = "123456:ABC-DEF...";
              break;
            case "slack":
              urlLabel.textContent = "Webhook URL";
              urlHint.textContent = "Slack incoming webhook URL";
              urlInput.placeholder = "https://hooks.slack.com/services/...";
              secretRow.style.display = "none";
              break;
            case "generic":
              urlLabel.textContent = "Endpoint URL";
              urlHint.textContent = "ntfy/Gotify/custom POST URL";
              urlInput.placeholder = "https://ntfy.sh/mytopic";
              secretRow.style.display = "";
              secretLabel.textContent = "Auth Header";
              secretHint.textContent = "Optional: Bearer token or Basic auth";
              webhookSecretEl.placeholder = "Bearer your-token";
              break;
          }
        };
        var updateWebhookLabels = updateWebhookLabels2;
        webhookEnabledEl.checked = cfg["webhook_enabled"] === "true";
        webhookConfigRows.style.display = webhookEnabledEl.checked ? "" : "none";
        var webhookTypeEl = document.getElementById("s-webhook-type");
        var webhookUrlEl = document.getElementById("s-webhook-url");
        var webhookSecretEl = document.getElementById("s-webhook-secret");
        webhookTypeEl.value = cfg["webhook_type"] || "discord";
        webhookUrlEl.value = cfg["webhook_url"] || "";
        if (cfg["webhook_secret"]) {
          webhookSecretEl.value = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
        }
        webhookSecretEl.addEventListener("focus", function() {
          if (webhookSecretEl.value === "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022") {
            webhookSecretEl.select();
          }
        });
        updateWebhookLabels2();
        webhookEnabledEl.addEventListener("change", function() {
          var val = webhookEnabledEl.checked ? "true" : "false";
          updateConfig("webhook_enabled", val).then(function() {
            showToast(webhookEnabledEl.checked ? "\u{1F517} Webhook enabled" : "\u{1F517} Webhook disabled", "success");
          });
          webhookConfigRows.style.display = webhookEnabledEl.checked ? "" : "none";
        });
        webhookTypeEl.addEventListener("change", function() {
          updateConfig("webhook_type", webhookTypeEl.value);
          showToast("\u{1F517} Webhook service updated", "success");
          updateWebhookLabels2();
        });
        webhookUrlEl.addEventListener("change", function() {
          updateConfig("webhook_url", webhookUrlEl.value.trim());
          showToast("\u{1F517} Webhook URL saved", "success");
        });
        webhookSecretEl.addEventListener("change", function() {
          var val = webhookSecretEl.value.trim();
          if (val === "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022") return;
          updateConfig("webhook_secret", val).then(function() {
            if (val) {
              showToast("\u{1F517} Webhook secret saved", "success");
              webhookSecretEl.value = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
            } else {
              showToast("\u{1F517} Webhook secret cleared", "success");
              webhookSecretEl.value = "";
            }
          });
        });
        document.getElementById("webhook-test-btn").addEventListener("click", function() {
          var btn = document.getElementById("webhook-test-btn");
          btn.disabled = true;
          btn.textContent = "\u{1F517} Sending...";
          fetch("/api/notify/test-webhook", { method: "POST" }).then(function(r) {
            return r.json();
          }).then(function(data) {
            if (data.error) showToast("\u274C " + data.error, "error");
            else showToast("\u{1F517} Test webhook sent!", "success");
          }).catch(function() {
            showToast("\u274C Failed to send test webhook", "error");
          }).finally(function() {
            btn.disabled = false;
            btn.textContent = "\u{1F517} Send Test";
          });
        });
      }
      var webpushEnabledEl = document.getElementById("s-webpush-enabled");
      var webpushConfigRows = document.getElementById("webpush-config-rows");
      if (webpushEnabledEl && "serviceWorker" in navigator && "PushManager" in window) {
        let urlBase64ToUint8Array2 = function(base64String) {
          var padding = "=".repeat((4 - base64String.length % 4) % 4);
          var base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
          var rawData = window.atob(base64);
          var outputArray = new Uint8Array(rawData.length);
          for (var i = 0; i < rawData.length; ++i) {
            outputArray[i] = rawData.charCodeAt(i);
          }
          return outputArray;
        }, updateWebPushStatus2 = function() {
          var badge = document.getElementById("webpush-status-badge");
          var btn = document.getElementById("webpush-subscribe-btn");
          navigator.serviceWorker.getRegistration("/sw.js").then(function(reg) {
            if (!reg) {
              badge.textContent = "\u26AA Not registered";
              badge.style.color = "var(--text-secondary)";
              btn.textContent = "\u{1F514} Subscribe";
              return;
            }
            reg.pushManager.getSubscription().then(function(sub) {
              if (sub) {
                badge.textContent = "\u{1F7E2} Subscribed";
                badge.style.color = "#22c55e";
                btn.textContent = "\u{1F515} Unsubscribe";
              } else {
                badge.textContent = "\u26AA Not subscribed";
                badge.style.color = "var(--text-secondary)";
                btn.textContent = "\u{1F514} Subscribe";
              }
            });
          });
        };
        var urlBase64ToUint8Array = urlBase64ToUint8Array2, updateWebPushStatus = updateWebPushStatus2;
        webpushEnabledEl.checked = cfg["webpush_enabled"] === "true";
        webpushConfigRows.style.display = webpushEnabledEl.checked ? "" : "none";
        updateWebPushStatus2();
        webpushEnabledEl.addEventListener("change", function() {
          var val = webpushEnabledEl.checked ? "true" : "false";
          updateConfig("webpush_enabled", val).then(function() {
            showToast(webpushEnabledEl.checked ? "\u{1F514} WebPush enabled" : "\u{1F514} WebPush disabled", "success");
          });
          webpushConfigRows.style.display = webpushEnabledEl.checked ? "" : "none";
        });
        document.getElementById("webpush-subscribe-btn").addEventListener("click", function() {
          var btn = document.getElementById("webpush-subscribe-btn");
          btn.disabled = true;
          navigator.serviceWorker.getRegistration("/sw.js").then(function(reg) {
            if (!reg) {
              return navigator.serviceWorker.register("/sw.js");
            }
            return reg;
          }).then(function(reg) {
            return reg.pushManager.getSubscription().then(function(existingSub) {
              if (existingSub) {
                return existingSub.unsubscribe().then(function() {
                  return fetch("/api/webpush/unsubscribe", {
                    method: "DELETE",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ endpoint: existingSub.endpoint })
                  });
                }).then(function() {
                  showToast("\u{1F515} Unsubscribed from push notifications", "success");
                  updateWebPushStatus2();
                });
              } else {
                return fetch("/api/webpush/vapid-key").then(function(r) {
                  return r.json();
                }).then(function(data) {
                  var applicationServerKey = urlBase64ToUint8Array2(data.publicKey);
                  return reg.pushManager.subscribe({
                    userVisibleOnly: true,
                    applicationServerKey
                  });
                }).then(function(sub) {
                  return fetch("/api/webpush/subscribe", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(sub.toJSON())
                  });
                }).then(function() {
                  showToast("\u{1F514} Subscribed to push notifications!", "success");
                  updateWebPushStatus2();
                });
              }
            });
          }).catch(function(err) {
            showToast("\u274C Push subscription failed: " + err.message, "error");
          }).finally(function() {
            btn.disabled = false;
          });
        });
        document.getElementById("webpush-test-btn").addEventListener("click", function() {
          var btn = document.getElementById("webpush-test-btn");
          btn.disabled = true;
          btn.textContent = "\u{1F514} Sending...";
          fetch("/api/notify/test-webpush", { method: "POST" }).then(function(r) {
            return r.json();
          }).then(function(data) {
            if (data.error) showToast("\u274C " + data.error, "error");
            else showToast("\u{1F514} Test push sent!", "success");
          }).catch(function() {
            showToast("\u274C Failed to send test push", "error");
          }).finally(function() {
            btn.disabled = false;
            btn.textContent = "\u{1F514} Send Test";
          });
        });
      } else if (webpushEnabledEl) {
        webpushEnabledEl.disabled = true;
        var hint = document.getElementById("webpush-status-hint");
        if (hint) hint.textContent = "Not supported in this browser";
      }
      var codexCaptureEl = document.getElementById("s-codex-capture");
      if (codexCaptureEl) {
        codexCaptureEl.checked = cfg["codex_capture"] === "true";
        codexCaptureEl.addEventListener("change", function() {
          var val = codexCaptureEl.checked ? "true" : "false";
          updateConfig("codex_capture", val).then(function() {
            showToast(codexCaptureEl.checked ? "\u{1F916} Codex capture enabled" : "\u{1F916} Codex capture disabled", "success");
            loadCodexSettingsStatus();
            loadDataSources();
          });
        });
        loadCodexSettingsStatus();
      }
      var cursorCaptureEl = document.getElementById("s-cursor-capture");
      if (cursorCaptureEl) {
        cursorCaptureEl.checked = cfg["cursor_capture"] === "true";
        cursorCaptureEl.addEventListener("change", function() {
          var val = cursorCaptureEl.checked ? "true" : "false";
          updateConfig("cursor_capture", val).then(function() {
            showToast(cursorCaptureEl.checked ? "\u{1F5B1}\uFE0F Cursor capture enabled" : "\u{1F5B1}\uFE0F Cursor capture disabled", "success");
            loadDataSources();
          });
        });
      }
      var copilotCaptureEl = document.getElementById("s-copilot-capture");
      var copilotPatEl = document.getElementById("s-copilot-pat");
      if (copilotCaptureEl) {
        copilotCaptureEl.checked = cfg["copilot_capture"] === "true";
        copilotCaptureEl.addEventListener("change", function() {
          var val = copilotCaptureEl.checked ? "true" : "false";
          updateConfig("copilot_capture", val).then(function() {
            showToast(copilotCaptureEl.checked ? "\u{1F419} Copilot capture enabled" : "\u{1F419} Copilot capture disabled", "success");
            loadDataSources();
          });
        });
      }
      if (copilotPatEl) {
        if (cfg["copilot_pat"]) {
          copilotPatEl.value = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
        }
        copilotPatEl.addEventListener("focus", function() {
          if (copilotPatEl.value === "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022") {
            copilotPatEl.select();
          }
        });
        copilotPatEl.addEventListener("change", function() {
          var val = copilotPatEl.value.trim();
          if (val === "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022") return;
          updateConfig("copilot_pat", val).then(function() {
            if (val) {
              showToast("\u{1F419} Copilot PAT saved", "success");
              copilotPatEl.value = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
            } else {
              showToast("\u{1F419} Copilot PAT cleared (falling back to auto-detection)", "success");
              copilotPatEl.value = "";
              copilotPatEl.placeholder = "Enter GitHub Personal Access Token...";
            }
            loadDataSources();
          });
        });
      }
      var importBtn = document.getElementById("import-json-btn");
      var importFile = document.getElementById("import-file");
      if (importBtn && importFile) {
        importBtn.addEventListener("click", function() {
          importFile.click();
        });
        importFile.addEventListener("change", function() {
          if (!importFile.files || !importFile.files[0]) return;
          var file = importFile.files[0];
          showToast("\u{1F4E5} Importing " + file.name + "...", "info");
          var reader = new FileReader();
          reader.onload = function(e) {
            fetch("/api/import/json", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: e.target.result
            }).then(function(r) {
              return r.json();
            }).then(function(data) {
              if (data.error) {
                showToast("\u274C Import failed: " + data.error, "error");
                return;
              }
              var msg = "\u2705 Imported: " + (data.accountsCreated || 0) + " accounts, " + (data.subsCreated || 0) + " subs, " + (data.snapshotsImported || 0) + " snapshots";
              showToast(msg, "success");
              var resultEl = document.getElementById("import-result");
              if (resultEl) {
                resultEl.style.display = "";
                resultEl.innerHTML = '<span style="color:var(--accent)">' + msg + "</span>" + (data.accountsSkipped ? "<br>Accounts skipped (existing): " + data.accountsSkipped : "") + (data.subsSkipped ? "<br>Subs skipped (existing): " + data.subsSkipped : "") + (data.snapshotsDuped ? "<br>Snapshots deduped: " + data.snapshotsDuped : "") + (data.errors && data.errors.length ? "<br>\u26A0\uFE0F Errors: " + data.errors.length : "");
              }
              fetchStatus().then(renderAccounts);
              loadSubscriptions();
            }).catch(function() {
              showToast("\u274C Import failed", "error");
            });
          };
          reader.readAsText(file);
          importFile.value = "";
        });
      }
    });
    loadModelPricing();
    document.getElementById("pricing-add-btn").addEventListener("click", addPricingRow);
    document.getElementById("pricing-reset-btn").addEventListener("click", resetPricingDefaults);
    loadMode();
    loadDataSources();
    document.getElementById("activity-refresh").addEventListener("click", loadActivityLog);
    document.getElementById("activity-filter").addEventListener("change", loadActivityLog);
    loadActivityLog();
    loadPlugins();
    initSettingsNav();
  }
  function migrateLocalStorage(cfg) {
    var lsBudget = localStorage.getItem("niyantra-budget");
    var lsCurrency = localStorage.getItem("niyantra-currency");
    if (lsBudget && (!cfg["budget_monthly"] || cfg["budget_monthly"] === "0")) {
      updateConfig("budget_monthly", lsBudget);
      serverConfig["budget_monthly"] = lsBudget;
      localStorage.removeItem("niyantra-budget");
    }
    if (lsCurrency && cfg["currency"] === "USD") {
      updateConfig("currency", lsCurrency);
      serverConfig["currency"] = lsCurrency;
      localStorage.removeItem("niyantra-currency");
    }
  }
  function initSettingsNav() {
    var nav = document.querySelector(".settings-nav");
    if (!nav) return;
    var links = nav.querySelectorAll(".settings-nav-link");
    var sections = [];
    links.forEach(function(link) {
      var id = link.getAttribute("data-section");
      if (id) {
        var section = document.getElementById(id);
        if (section) sections.push(section);
      }
    });
    links.forEach(function(link) {
      link.addEventListener("click", function() {
        var id = link.getAttribute("data-section");
        if (!id) return;
        var target = document.getElementById(id);
        if (target) {
          target.scrollIntoView({ behavior: "smooth", block: "start" });
          links.forEach(function(l) {
            l.classList.remove("active");
          });
          link.classList.add("active");
        }
      });
    });
    if ("IntersectionObserver" in window) {
      var observer = new IntersectionObserver(function(entries) {
        entries.forEach(function(entry) {
          if (entry.isIntersecting) {
            var id = entry.target.id;
            links.forEach(function(link) {
              if (link.getAttribute("data-section") === id) {
                link.classList.add("active");
              } else {
                link.classList.remove("active");
              }
            });
          }
        });
      }, {
        rootMargin: "-20% 0px -60% 0px",
        threshold: 0
      });
      sections.forEach(function(section) {
        observer.observe(section);
      });
    }
  }

  // internal/web/src/advanced/palette.ts
  var PALETTE_COMMANDS = [
    { name: "Snap Now", key: "S", icon: "\u{1F4F8}", action: function() {
      handleSnap();
    } },
    { name: "Show Quotas", key: "1", icon: "\u{1F4CA}", action: function() {
      switchToTab("quotas");
    } },
    { name: "Show Subscriptions", key: "2", icon: "\u{1F4B3}", action: function() {
      switchToTab("subscriptions");
    } },
    { name: "Show Overview", key: "3", icon: "\u{1F4CB}", action: function() {
      switchToTab("overview");
    } },
    { name: "Show Settings", key: "4", icon: "\u2699\uFE0F", action: function() {
      switchToTab("settings");
    } },
    { name: "New Subscription", key: "N", icon: "\u2795", action: function() {
      openModal();
    } },
    { name: "Toggle Auto-Capture", icon: "\u{1F504}", action: function() {
      var el = document.getElementById("s-auto-capture");
      if (el) {
        el.checked = !el.checked;
        el.dispatchEvent(new Event("change"));
      }
    } },
    { name: "Export CSV", icon: "\u{1F4E5}", action: function() {
      downloadAPIFile("/api/export/csv", "niyantra-export.csv").catch(function(err) {
        alert(err.message || "CSV export failed");
      });
    } },
    { name: "Export JSON", icon: "\u{1F4E6}", action: function() {
      downloadAPIFile("/api/export/json", "niyantra-export.json").catch(function(err) {
        alert(err.message || "JSON export failed");
      });
    } },
    { name: "Download Backup", icon: "\u{1F4BE}", action: function() {
      downloadBackup().catch(function(err) {
        alert(err.message || "Backup failed");
      });
    } },
    { name: "Search Subscriptions", key: "/", icon: "\u{1F50D}", action: function() {
      switchToTab("subscriptions");
      setTimeout(function() {
        var s = document.getElementById("search-subs");
        if (s) s.focus();
      }, 100);
    } },
    { name: "Set Budget", icon: "\u{1F4B0}", action: function() {
      openBudgetModal();
    } },
    { name: "Toggle Theme", icon: "\u{1F313}", action: function() {
      var cur = document.documentElement.getAttribute("data-theme");
      var next = cur === "dark" ? "light" : "dark";
      document.documentElement.setAttribute("data-theme", next);
      localStorage.setItem("niyantra-theme", next);
      var themeEl = document.getElementById("s-theme");
      if (themeEl) themeEl.value = next;
      updateChartTheme(next);
    } },
    { name: "Codex Snap", icon: "\u{1F916}", action: function() {
      handleCodexSnap();
    } },
    { name: "Import JSON", icon: "\u{1F4E5}", action: function() {
      var f = document.getElementById("import-file");
      if (f) f.click();
    } }
  ];
  var paletteSelectedIndex = 0;
  var paletteFilteredCommands = PALETTE_COMMANDS;
  function initCommandPalette() {
    var overlay = document.getElementById("command-palette-overlay");
    var search = document.getElementById("command-palette-search");
    if (!overlay || !search) return;
    overlay.addEventListener("click", function(e) {
      if (e.target === overlay) closeCommandPalette();
    });
    search.addEventListener("input", function() {
      var query = search.value.toLowerCase().trim();
      paletteFilteredCommands = PALETTE_COMMANDS.filter(function(cmd) {
        return cmd.name.toLowerCase().indexOf(query) >= 0;
      });
      paletteSelectedIndex = 0;
      renderPaletteList();
    });
    search.addEventListener("keydown", function(e) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        paletteSelectedIndex = Math.min(paletteSelectedIndex + 1, paletteFilteredCommands.length - 1);
        renderPaletteList();
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        paletteSelectedIndex = Math.max(paletteSelectedIndex - 1, 0);
        renderPaletteList();
      } else if (e.key === "Enter") {
        e.preventDefault();
        if (paletteFilteredCommands[paletteSelectedIndex]) {
          closeCommandPalette();
          paletteFilteredCommands[paletteSelectedIndex].action();
        }
      } else if (e.key === "Escape") {
        closeCommandPalette();
      }
    });
  }
  function toggleCommandPalette() {
    var overlay = document.getElementById("command-palette-overlay");
    if (overlay.hidden) {
      openCommandPalette();
    } else {
      closeCommandPalette();
    }
  }
  function openCommandPalette() {
    var overlay = document.getElementById("command-palette-overlay");
    var search = document.getElementById("command-palette-search");
    overlay.hidden = false;
    search.value = "";
    paletteFilteredCommands = PALETTE_COMMANDS;
    paletteSelectedIndex = 0;
    renderPaletteList();
    setTimeout(function() {
      search.focus();
    }, 50);
  }
  function closeCommandPalette() {
    document.getElementById("command-palette-overlay").hidden = true;
  }
  function renderPaletteList() {
    var list = document.getElementById("command-palette-list");
    if (paletteFilteredCommands.length === 0) {
      list.innerHTML = '<div class="command-palette-empty">No matching commands</div>';
      return;
    }
    var html = "";
    for (var i = 0; i < paletteFilteredCommands.length; i++) {
      var cmd = paletteFilteredCommands[i];
      var sel = i === paletteSelectedIndex ? " selected" : "";
      html += '<div class="command-palette-item' + sel + '" data-idx="' + i + '"><span class="cp-icon">' + cmd.icon + '</span><span class="cp-name">' + esc(cmd.name) + "</span>" + (cmd.key ? '<span class="cp-shortcut">' + cmd.key + "</span>" : "") + "</div>";
    }
    list.innerHTML = html;
    list.querySelectorAll(".command-palette-item").forEach(function(el) {
      el.addEventListener("click", function() {
        var idx = parseInt(el.getAttribute("data-idx"));
        closeCommandPalette();
        paletteFilteredCommands[idx].action();
      });
    });
    var selected = list.querySelector(".selected");
    if (selected) selected.scrollIntoView({ block: "nearest" });
  }

  // internal/web/src/advanced/keyboard.ts
  function initKeyboardShortcuts() {
    document.addEventListener("keydown", function(e) {
      var tag = document.activeElement?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") {
        if (e.key === "Escape") {
          document.activeElement?.blur();
          closeModal();
          closeDelete();
          closeBudget();
        }
        return;
      }
      var anyModal = !document.getElementById("modal-overlay").hidden || !document.getElementById("delete-overlay").hidden || !document.getElementById("budget-overlay").hidden;
      if (e.key === "Escape") {
        closeModal();
        closeDelete();
        closeBudget();
        return;
      }
      if (anyModal) return;
      switch (e.key) {
        case "1":
          switchToTab("quotas");
          break;
        case "2":
          switchToTab("subscriptions");
          break;
        case "3":
          switchToTab("overview");
          break;
        case "4":
          switchToTab("settings");
          break;
        case "n":
        case "N":
          openModal();
          e.preventDefault();
          break;
        case "s":
        case "S":
          handleSnap();
          e.preventDefault();
          break;
        case "/":
          e.preventDefault();
          switchToTab("subscriptions");
          setTimeout(function() {
            var search = document.getElementById("search-subs");
            if (search) search.focus();
          }, 100);
          break;
      }
    });
    document.addEventListener("keydown", function(e) {
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        toggleCommandPalette();
      }
    });
  }

  // internal/web/src/main.ts
  document.addEventListener("DOMContentLoaded", function() {
    installAuthenticatedFetch();
    initTheme();
    var isInitialTabChange = true;
    document.addEventListener("niyantra:tab-change", function(e) {
      var tab = e.detail.tab;
      if (tab === "overview") {
        loadOverview();
      }
      if (tab === "quotas") {
        if (!isInitialTabChange) {
          fetchStatus().then(function(data) {
            document.dispatchEvent(new CustomEvent("niyantra:status-refreshed", { detail: { data } }));
          }).catch(function() {
          });
        }
      }
      if (tab === "subscriptions") {
        if (!isInitialTabChange) {
          loadSubscriptions();
        }
      }
      if (tab === "settings") {
        loadActivityLog();
        loadMode();
        loadDataSources();
      }
    });
    initTabs();
    isInitialTabChange = false;
    setRenderAccounts(renderAccounts);
    initQuotas();
    setupToggle();
    initModal();
    initBudget();
    initSettings();
    initSearch();
    initKeyboardShortcuts();
    initAccountMetaHandlers();
    document.addEventListener("visibilitychange", function() {
      if (document.visibilityState === "visible") {
        var activeTab = document.querySelector(".tab-btn.active")?.getAttribute("data-tab");
        if (activeTab === "quotas") {
          fetchStatus().then(function(data) {
            document.dispatchEvent(new CustomEvent("niyantra:status-refreshed", { detail: { data } }));
          }).catch(function() {
          });
        }
      }
    });
    document.addEventListener("niyantra:status-refreshed", function(e) {
      var data = e.detail.data;
      renderAccounts(data);
      populateChartAccountSelect(data);
      loadHistoryChart();
      updateTimestamp();
      if (localStorage.getItem("niyantra-active-tab") === "overview") {
        document.dispatchEvent(new CustomEvent("niyantra:overview-refresh"));
      }
    });
    document.addEventListener("niyantra:theme-change", function(e) {
      updateChartTheme(e.detail.theme);
    });
    document.addEventListener("niyantra:chart-refresh", function() {
      loadHistoryChart();
    });
    document.addEventListener("niyantra:overview-refresh", function() {
      loadOverview();
    });
    document.getElementById("snap-btn").addEventListener("click", handleSnap);
    var settingsBackupBtn = document.getElementById("settings-backup-btn");
    if (settingsBackupBtn) {
      settingsBackupBtn.addEventListener("click", function() {
        downloadBackup().catch(function(err) {
          alert(err.message || "Backup failed");
        });
      });
    }
    initSnapDropdown();
    var chartProvider = document.getElementById("chart-provider");
    if (chartProvider) {
      chartProvider.addEventListener("change", function() {
        filterChartAccounts();
        loadHistoryChart();
      });
    }
    var chartAccount = document.getElementById("chart-account");
    if (chartAccount) {
      chartAccount.addEventListener("change", loadHistoryChart);
    }
    var chartRange = document.getElementById("chart-range");
    if (chartRange) {
      chartRange.addEventListener("change", loadHistoryChart);
    }
    var chartStart = document.getElementById("chart-start-date");
    if (chartStart) {
      chartStart.addEventListener("change", loadHistoryChart);
    }
    var chartEnd = document.getElementById("chart-end-date");
    if (chartEnd) {
      chartEnd.addEventListener("change", loadHistoryChart);
    }
    Promise.all([fetchStatus(), fetchUsage()]).then(function(results) {
      var data = results[0];
      renderAccounts(data);
      updateTimestamp();
      populateChartAccountSelect(data);
      loadHistoryChart();
      if (!data.accounts || data.accounts.length === 0) {
        var grid = document.getElementById("account-grid");
        if (grid) grid.innerHTML = emptyQuotas();
        var emptySnapBtn = document.getElementById("empty-snap-btn");
        if (emptySnapBtn) emptySnapBtn.addEventListener("click", handleSnap);
      }
      if (!data.codexSnapshot || !data.claudeSnapshot) {
        setTimeout(function() {
          fetchStatus().then(function(data2) {
            if (data2.codexSnapshot || data2.claudeSnapshot) {
              renderAccounts(data2);
            }
          }).catch(function() {
          });
        }, 3e3);
      }
    }).catch(function(err) {
      console.error("Failed to load status:", err);
    });
    loadSubscriptions();
    fetchPresets().then(function(data) {
      setPresetsData(data.presets || []);
      var list = document.getElementById("preset-list");
      for (var i = 0; i < presetsData.length; i++) {
        var opt = document.createElement("option");
        opt.value = presetsData[i].platform;
        list.appendChild(opt);
      }
    });
    loadMode();
    initCommandPalette();
    loadSystemAlerts();
    setInterval(refreshTimestampDisplay, 3e4);
  });
})();
