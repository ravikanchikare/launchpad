// --- State ---
var settings = { gatewayUrl: '', tokenConfigured: false, tokenValid: false, setupReason: 'missing', defaultKeySlug: 'management-key' };
var catalog = [];
var modelCatalog = [];
var selectedModel = '';
var modelsFetchPromise = null;
var modelsCatalogFetchPromise = null;
var accountFetchPromise = null;
var accountSummary = null;
var keysPage = { supported: false, reason: '', keys: [], loaded: false, pendingSecret: '', openMenuId: '', confirmAction: null, editingKeyId: '' };
var toastTimer = null;

function sfIcon(name, size) {
  var px = size || 18;
  return '<img src="/ui/sfsymbol/' + encodeURIComponent(name) + '.png?size=' + px + '" class="ui-icon" alt="" aria-hidden="true">';
}

function actionButton(label, opts) {
  opts = opts || {};
  var cls = 'action' + (opts.primary ? ' primary' : '');
  var type = opts.submit ? 'submit' : 'button';
  var onclick = opts.onclick ? ' onclick="' + opts.onclick + '"' : '';
  var icon = opts.icon ? sfIcon(opts.icon, 14) : '';
  return '<button type="' + type + '" class="' + cls + '"' + onclick + '>' + icon + label + '</button>';
}

var editIconHTML = sfIcon('pencil');
var moreIconHTML = '<span class="key-menu-ellipsis" aria-hidden="true">···</span>';
var applications = [
  { id: 'claude', name: 'Claude Code', description: "Anthropic's coding tool with subagents", icon: '/assets/claude-code.svg', command: 'claude' },
  { id: 'codex-desktop', name: 'ChatGPT', description: 'Complete work with ChatGPT', icon: '/assets/codex.svg', command: 'chatgpt' },
  { id: 'codex-cli', name: 'Codex', description: "OpenAI's open-source coding agent", icon: '/assets/codex-app.png', command: 'codex' },
  { id: 'opencode', name: 'OpenCode', description: "Anomaly's open-source coding agent", icon: '/assets/opencode.svg', command: 'opencode' }
];
var copyIconHTML = sfIcon('doc.on.doc');
var downloadIconHTML = sfIcon('arrow.down.circle', 14);
var updateStatus = { currentVersion: '', available: false, downloaded: false, update: null, error: '' };
var updatePollTimer = null;
var updateInstalling = false;
var setupValidation = { token: '', valid: false, timer: null, requestId: 0 };

// --- Sidebar ---
function setAppSidebarCollapsed(collapsed) {
  var shell = document.getElementById('app-shell');
  shell.classList.toggle('sidebar-collapsed', collapsed);
  var button = document.getElementById('sidebar-toggle');
  if (!button) {
    return;
  }
  var label = collapsed ? 'Show Sidebar' : 'Hide Sidebar';
  button.title = label;
  button.setAttribute('aria-label', label);
  button.classList.toggle('is-collapsed', collapsed);
}

function toggleAppSidebar() {
  var shell = document.getElementById('app-shell');
  setAppSidebarCollapsed(!shell.classList.contains('sidebar-collapsed'));
}

// --- Navigation ---
// Setup is now optional – user is not forced to enter a management key.
// Keep the helper for status checks but never block navigation.
function isSetupRequired() {
  return false;
}
function isSetupRequiredStrict() {
  return !settings.tokenValid;
}

function applySettingsState(next) {
  settings = next;
  document.getElementById('gateway-url').textContent = settings.gatewayUrl;
  renderTokenState(false);
  document.getElementById('setup-nudge').classList.add('hidden');
  renderSetupGatewayLink();
  renderSetupContent();
  renderSetupScreen();
}

function gatewayDashboardURL() {
  return settings.gatewayUrl.replace(/\/+$/, '') + '/ui/';
}

function renderSetupGatewayLink() {
  var urlNode = document.getElementById('setup-gateway-url');
  var link = document.getElementById('setup-gateway-link');
  if (!urlNode || !link || !settings.gatewayUrl) {
    return;
  }
  var dashboard = gatewayDashboardURL();
  urlNode.textContent = dashboard;
  link.href = dashboard;
  link.title = 'Open Gateway dashboard (SSO sign-in)';
}

function resetSetupValidation() {
  clearTimeout(setupValidation.timer);
  setupValidation.token = '';
  setupValidation.valid = false;
  setupValidation.requestId += 1;
  renderSetupValidation('', '');
  renderSetupContinue();
}

function renderSetupValidation(kind, message) {
  var node = document.getElementById('setup-validation');
  if (!node) {
    return;
  }
  node.classList.remove('hidden', 'checking', 'good', 'error');
  if (!message) {
    node.classList.add('hidden');
    node.innerHTML = '';
    return;
  }
  if (kind === 'checking') {
    node.classList.add('checking');
    node.innerHTML = '<span class="spinner" aria-hidden="true"></span><span>' + escapeHTML(message) + '</span>';
    return;
  }
  node.textContent = message;
  node.classList.add(kind === 'good' ? 'good' : 'error');
}

function renderSetupContinue() {
  var input = document.getElementById('setup-token');
  var actions = document.getElementById('setup-actions');
  if (!input || !actions) {
    return;
  }
  var token = input.value.trim();
  actions.classList.toggle('hidden', !(setupValidation.valid && token === setupValidation.token));
}

function handleSetupTokenInput() {
  var token = document.getElementById('setup-token').value.trim();
  clearTimeout(setupValidation.timer);
  setupValidation.valid = false;
  setupValidation.token = '';
  renderSetupContinue();
  if (!token) {
    renderSetupValidation('', '');
    return;
  }
  renderSetupValidation('checking', 'Validating key…');
  setupValidation.timer = setTimeout(function () {
    validateSetupToken(token);
  }, 400);
}

async function validateSetupToken(token) {
  var requestId = ++setupValidation.requestId;
  try {
    await api('/api/settings/validate', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ token: token })
    });
    if (requestId !== setupValidation.requestId) {
      return;
    }
    setupValidation.token = token;
    setupValidation.valid = true;
    renderSetupValidation('good', 'Key verified');
    renderSetupContinue();
  } catch (e) {
    if (requestId !== setupValidation.requestId) {
      return;
    }
    setupValidation.valid = false;
    renderSetupValidation('error', e.message);
    renderSetupContinue();
  }
}

function renderSetupContent() {
  var title = document.getElementById('setup-title');
  var subtitle = document.getElementById('setup-subtitle');
  if (!title || !subtitle) {
    return;
  }
  if (settings.setupReason === 'expired') {
    title.textContent = 'Management key expired';
    subtitle.textContent = 'Create a replacement key in Virtual Keys, then paste it below.';
    return;
  }
  if (settings.setupReason === 'invalid') {
    title.textContent = 'Management key needs attention';
    subtitle.textContent = 'Your saved management key is no longer valid. Create a replacement and paste it below.';
    return;
  }
  title.textContent = 'Welcome to HarnezPad';
  subtitle.textContent = 'Connect your Gateway management key to get started.';
}

function isGatewayAuthError(message) {
  return /401|403|unauthorized|forbidden|expired|invalid/i.test(String(message || ''));
}

async function refreshSettingsStatus() {
  try {
    applySettingsState(await api('/api/settings'));
    if (isSetupRequired()) {
      showView('launch');
    }
  } catch (e) {
    // Keep the current UI state if settings cannot be refreshed.
  }
}

function renderSetupScreen() {
  var screen = document.getElementById('setup-screen');
  if (!screen) {
    return;
  }
  var show = isSetupRequired();
  var wasHidden = screen.classList.contains('hidden');
  screen.classList.toggle('hidden', !show);
  if (show && wasHidden) {
    resetSetupValidation();
    var input = document.getElementById('setup-token');
    if (input) {
      input.value = '';
    }
  }
}

function renderSetupHelpBanner(visible) {
  var banner = document.getElementById('setup-help-banner');
  if (banner) {
    banner.classList.toggle('hidden', !visible);
  }
}

function updateSetupActions() {
  handleSetupTokenInput();
}

function openSetupHelp() {
  showView('settings');
}

function returnToSetup() {
  renderSetupHelpBanner(false);
  renderSetupScreen();
}

function showView(name) {
  if (name !== 'keys') {
    closeKeysModal();
    closeKeyMenus();
  }
  ['launch', 'models', 'keys', 'settings'].forEach(function (view) {
    var viewEl = document.getElementById('view-' + view);
    if (viewEl) viewEl.classList.toggle('hidden', name !== view);
    if (view !== 'settings') {
      var navEl = document.getElementById('nav-' + view);
      if (navEl) navEl.classList.toggle('active', name === view);
    }
  });
  var settingsItem = document.getElementById('nav-settings-item');
  if (settingsItem) {
    settingsItem.classList.toggle('active', name === 'settings');
  }
  document.querySelector('.content').classList.toggle('models-active', name === 'models');
  document.querySelector('.content').classList.toggle('keys-active', name === 'keys');
  if (name === 'models') {
    setAppSidebarCollapsed(true);
    if (modelCatalog.length) {
      renderModelCatalog();
      var keepSelection = selectedModel && modelCatalog.some(function (model) { return model.id === selectedModel; });
      showModelDetail(keepSelection ? selectedModel : modelCatalog[0].id);
    } else {
      showModelsLoading();
    }
    refreshModelsInBackground();
  }
  if (name === 'settings') {
    refreshAccountPanel();
  }
  if (name === 'launch') {
    refreshLaunchSpendBar();
  }
  if (name === 'keys') {
    setAppSidebarCollapsed(true);
    ensureKeysPage();
  }
  if (isSetupRequired()) {
    renderSetupHelpBanner(false);
    renderSetupScreen();
  } else {
    renderSetupHelpBanner(false);
    renderSetupScreen();
  }
}

// --- Settings update ---
async function fetchUpdateStatus(check) {
  return api(check ? '/api/update?check=1' : '/api/update');
}

function renderUpdateStatus(status) {
  if (status) {
    updateStatus = status;
  }
  var button = document.getElementById('sidebar-update-btn');
  if (!button || updateInstalling) {
    return;
  }
  if (updateStatus.error) {
    button.classList.add('hidden');
    return;
  }
  if (updateStatus.downloaded && updateStatus.available && updateStatus.update) {
    button.classList.remove('hidden', 'downloading', 'installing', 'expanded');
    button.disabled = false;
    button.innerHTML = downloadIconHTML;
    button.title = 'Install HarnezPad ' + updateStatus.update.version;
    button.setAttribute('aria-label', 'Install HarnezPad ' + updateStatus.update.version);
    return;
  }
  button.classList.add('hidden');
  button.classList.remove('downloading', 'installing', 'expanded');
}

async function refreshUpdateStatus() {
  try {
    renderUpdateStatus(await fetchUpdateStatus(false));
  } catch (e) {
    var button = document.getElementById('sidebar-update-btn');
    if (button) {
      button.classList.add('hidden');
    }
  }
}

function startUpdatePolling() {
  refreshUpdateStatus();
  if (updatePollTimer) {
    clearInterval(updatePollTimer);
  }
  updatePollTimer = setInterval(refreshUpdateStatus, 10000);
}

async function installPendingUpdate() {
  var button = document.getElementById('sidebar-update-btn');
  if (!button || button.disabled || updateInstalling) {
    return;
  }
  updateInstalling = true;
  button.classList.remove('hidden');
  button.classList.add('installing', 'expanded');
  button.disabled = true;
  button.innerHTML =
    '<span class="spinner" aria-hidden="true"></span>' +
    '<span class="sidebar-update-label">Installing…</span>';
  button.title = 'Installing…';
  button.setAttribute('aria-label', 'Installing update');
  try {
    await api('/api/update/install', { method: 'POST' });
  } catch (e) {
    updateInstalling = false;
    showToast(e.message);
    await refreshUpdateStatus();
  }
}

// --- Toasts ---
function showToast(message) {
  var root = document.getElementById('toast-root');
  if (!root) {
    return;
  }
  if (toastTimer) {
    clearTimeout(toastTimer);
    toastTimer = null;
  }
  root.innerHTML = '<div class="toast">' + escapeHTML(message) + '</div>';
  root.classList.remove('hidden');
  toastTimer = setTimeout(function () {
    root.classList.add('hidden');
    root.innerHTML = '';
    toastTimer = null;
  }, 3200);
}

function invalidateKeysPage() {
  keysPage.loaded = false;
  keysPage.supported = false;
  keysPage.reason = '';
}

function userFacingError(message, status) {
  var msg = String(message || '').trim();
  if (!msg) {
    if (status === 502 || status === 503) {
      return 'Could not reach the Gateway. Try again in a moment.';
    }
    if (status >= 500) {
      return 'Something went wrong. Try again.';
    }
    return 'Request failed.';
  }
  if (/management key is expired/i.test(msg)) {
    return 'Management key is expired. Create a replacement in Virtual Keys.';
  }
  if (/management key is invalid/i.test(msg)) {
    return 'Management key is invalid. Create a replacement in Virtual Keys.';
  }
  if (/management key is not configured|gateway token/i.test(msg)) {
    return 'Management key is not configured. Save it in HarnezPad Settings before launching.';
  }
  if (/management key cannot be empty|api token cannot be empty/i.test(msg)) {
    return 'Management key cannot be empty.';
  }
  if (/save management key:|save token:/i.test(msg)) {
    return 'Could not save the management key to Keychain.';
  }
  if (/no models are available|gateway returned no models/i.test(msg)) {
    return 'No models are available for your management key.';
  }
  if (/401|403|unauthorized|forbidden|expired|invalid/i.test(msg)) {
    if (/expired/i.test(msg)) {
      return 'Management key is expired. Create a replacement in Virtual Keys.';
    }
    if (/management key/i.test(msg)) {
      return msg;
    }
    return 'Management key is invalid or expired.';
  }
  if (/gateway returned/i.test(msg)) {
    return 'Could not reach the Gateway. Check your connection and try again.';
  }
  if (/^HTTP \d+$/.test(msg)) {
    if (status === 502 || status === 503) {
      return 'Could not reach the Gateway. Try again in a moment.';
    }
    return 'Something went wrong. Try again.';
  }
  if (/self-update is unavailable|update checking is not available/i.test(msg)) {
    return 'Updates are not available in this build.';
  }
  if (/no verified update is ready/i.test(msg)) {
    return 'No update is ready to install yet.';
  }
  return msg;
}

async function api(path, options) {
  var response = await fetch(path, options);
  if (response.status === 204) {
    if (!response.ok) {
      throw new Error('HTTP ' + response.status);
    }
    return {};
  }
  var text = await response.text();
  var data = {};
  try {
    data = text ? JSON.parse(text) : {};
  } catch (e) {
    data = { error: text };
  }
  if (!response.ok) {
    throw new Error(userFacingError(data.error || text, response.status));
  }
  return data;
}

async function writeClipboard(value) {
  if (window.harnezpadClipboardWrite) {
    await window.harnezpadClipboardWrite(value);
    return;
  }
  await navigator.clipboard.writeText(value);
}

async function copyCommand(value, button) {
  try {
    await writeClipboard(value);
    showToast('Copied to clipboard', 'success');
    if (button) {
      button.style.color = '#248a3d';
      setTimeout(function () { button.style.color = ''; }, 900);
    }
  } catch (e) {
    showToast('Copy failed', 'error');
    if (button) {
      button.style.color = '#c4320a';
    }
  }
}

function shellArg(value) {
  return /^[A-Za-z0-9._:/@\[\]-]+$/.test(value) ? value : "'" + value.replace(/'/g, "'\\''") + "'";
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, function (character) {
    return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[character];
  });
}

function escapeAttr(value) {
  return String(value).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;');
}

function slugifyKeyName(value) {
  return String(value || '').toLowerCase().trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-+|-+$/g, '');
}

function isValidKeySlug(value) {
  return /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value);
}

function commandFor(app, model) {
  return model ? 'harnezpad launch ' + app.command + ' --model ' + shellArg(model) : '';
}

// --- Models page ---
function formatCostPerMillion(cost) {
  if (cost == null || !isFinite(cost) || cost <= 0) {
    return '';
  }
  var perMillion = cost * 1000000;
  if (perMillion >= 100) {
    return '$' + perMillion.toFixed(0) + ' / 1M';
  }
  if (perMillion >= 10) {
    return '$' + perMillion.toFixed(1) + ' / 1M';
  }
  if (perMillion >= 1) {
    return '$' + perMillion.toFixed(2) + ' / 1M';
  }
  return '$' + perMillion.toFixed(3) + ' / 1M';
}

function formatTokenCount(value) {
  if (value == null || !isFinite(value) || value <= 0) {
    return '';
  }
  if (value >= 1000000) {
    var millions = value / 1000000;
    return (millions % 1 === 0 ? millions.toFixed(0) : millions.toFixed(1)) + 'M';
  }
  if (value >= 1000) {
    var thousands = value / 1000;
    return (thousands % 1 === 0 ? thousands.toFixed(0) : thousands.toFixed(1)) + 'K';
  }
  return String(Math.round(value));
}

function providerLabel(providers) {
  if (!providers || !providers.length) {
    return '';
  }
  return providers.map(function (provider) {
    if (provider === 'bedrock') {
      return 'Bedrock';
    }
    if (provider === 'openai') {
      return 'OpenAI';
    }
    if (provider === 'baseten') {
      return 'Baseten';
    }
    return provider.charAt(0).toUpperCase() + provider.slice(1);
  }).join(', ');
}

function modelCostLine(model) {
  var input = formatCostPerMillion(model.inputCostPerToken);
  var output = formatCostPerMillion(model.outputCostPerToken);
  if (input && output) {
    return input.replace(' / 1M', '') + ' in · ' + output.replace(' / 1M', '') + ' out';
  }
  if (input) {
    return input + ' in';
  }
  if (output) {
    return output + ' out';
  }
  return '';
}

function capabilityChips(model, compact) {
  var chips = [];
  if (model.supportsVision) {
    chips.push('Vision');
  }
  if (model.supportsTools) {
    chips.push('Tools');
  }
  if (model.supportsReasoning) {
    chips.push('Reasoning');
  }
  if (model.supportsWebSearch) {
    chips.push('Web search');
  }
  if (!chips.length) {
    return '';
  }
  return chips.map(function (chip) {
    return '<span class="model-capability-chip' + (compact ? ' compact' : '') + '">' + chip + '</span>';
  }).join('');
}

function selectedCapabilityFilters() {
  return Array.prototype.slice.call(document.querySelectorAll('#model-capability-filters input:checked')).map(function (node) {
    return node.value;
  });
}

function modelMatchesCapabilities(model, filters) {
  if (!filters.length) {
    return true;
  }
  return filters.every(function (filter) {
    if (filter === 'vision') {
      return model.supportsVision;
    }
    if (filter === 'tools') {
      return model.supportsTools;
    }
    if (filter === 'reasoning') {
      return model.supportsReasoning;
    }
    if (filter === 'web') {
      return model.supportsWebSearch;
    }
    return true;
  });
}

function sortModelCatalog(models) {
  return models.slice().sort(function (a, b) {
    return String(a.id).localeCompare(String(b.id));
  });
}

function updateProviderFilterOptions() {
  var select = document.getElementById('model-provider-filter');
  if (!select) {
    return;
  }
  var current = select.value;
  var providers = {};
  modelCatalog.forEach(function (model) {
    (model.providers || []).forEach(function (provider) {
      providers[provider] = true;
    });
  });
  var options = ['<option value="">All providers</option>'];
  Object.keys(providers).sort().forEach(function (provider) {
    options.push('<option value="' + escapeHTML(provider) + '">' + escapeHTML(providerLabel([provider])) + '</option>');
  });
  select.innerHTML = options.join('');
  if (current && providers[current]) {
    select.value = current;
  }
}

function filteredCatalog() {
  var input = document.getElementById('model-search');
  var query = input ? input.value.trim().toLowerCase() : '';
  var provider = document.getElementById('model-provider-filter');
  var providerValue = provider ? provider.value : '';
  var capabilityFilters = selectedCapabilityFilters();
  var models = modelCatalog.filter(function (model) {
    if (providerValue && (!model.providers || model.providers.indexOf(providerValue) === -1)) {
      return false;
    }
    if (!modelMatchesCapabilities(model, capabilityFilters)) {
      return false;
    }
    if (query && String(model.id).toLowerCase().indexOf(query) === -1) {
      return false;
    }
    return true;
  });
  return sortModelCatalog(models);
}

function showModelsLoading() {
  document.getElementById('model-catalog').innerHTML = '<div class="models-loading"><span class="spinner" aria-hidden="true"></span><span>Loading models…</span></div>';
  document.getElementById('model-title').textContent = 'Loading…';
  document.getElementById('model-meta').classList.add('hidden');
  document.getElementById('model-meta').innerHTML = '';
  document.getElementById('model-applications').innerHTML =
    '<div class="models-loading models-loading-detail"><span class="spinner" aria-hidden="true"></span><span>Loading application commands…</span></div>';
}

function filterModelCatalog() {
  renderModelCatalog();
}

function renderModelCatalog() {
  var root = document.getElementById('model-catalog');
  var models = filteredCatalog();
  if (!modelCatalog.length) {
    root.innerHTML = '<div class="empty">No models available.</div>';
    return;
  }
  if (!models.length) {
    root.innerHTML = '<div class="empty">No matching models.</div>';
    return;
  }
  root.innerHTML = models.map(function (model) {
    var label = escapeHTML(model.id);
    var provider = providerLabel(model.providers);
    var chips = capabilityChips(model, true);
    return '<button type="button" class="model-item' + (model.id === selectedModel ? ' active' : '') +
      '" data-model-id="' + escapeAttr(model.id) + '" title="' + label + '">' +
      '<span class="model-item-header"><span class="model-item-label">' + label + '</span>' +
      (provider ? '<span class="model-item-provider">' + escapeHTML(provider) + '</span>' : '') +
      '</span>' +
      (chips ? '<span class="model-item-chips">' + chips + '</span>' : '') +
      '</button>';
  }).join('');
}

function initModelCatalogEvents() {
  var root = document.getElementById('model-catalog');
  if (!root || root.dataset.bound === '1') {
    return;
  }
  root.dataset.bound = '1';
  root.addEventListener('click', function (event) {
    var button = event.target.closest('.model-item[data-model-id]');
    if (!button) {
      return;
    }
    event.preventDefault();
    showModelDetail(button.getAttribute('data-model-id'));
  });
}

function renderModelApplications(model) {
  var root = document.getElementById('model-applications');
  root.innerHTML = applications.map(function (app) {
    var command = commandFor(app, model);
    return '<article class="application-row">' +
      '<div class="icon ' + (app.id === 'codex-cli' ? 'codex-app' : app.id === 'opencode' ? 'opencode' : '') + '"><img src="' + app.icon + '" alt=""></div>' +
      '<div class="application-body">' +
      '<h2>' + app.name + '</h2>' +
      '<p class="description">' + escapeHTML(app.description) + '</p>' +
      '<div class="command"><code>' + escapeHTML(command) + '</code>' +
      '<button class="copy" title="Copy command" onclick="copyModelCommand(\'' + app.id + '\', this)">' + copyIconHTML +
      '</button></div></div></article>';
  }).join('');
}

function copyModelCommand(id, button) {
  var app = applications.find(function (item) { return item.id === id; });
  copyCommand(commandFor(app, selectedModel), button);
}

function renderModelMeta(model) {
  var root = document.getElementById('model-meta');
  if (!model) {
    root.classList.add('hidden');
    root.innerHTML = '';
    return;
  }
  var provider = providerLabel(model.providers);
  var contextParts = [];
  if (model.maxInputTokens) {
    contextParts.push(formatTokenCount(model.maxInputTokens) + ' in');
  }
  if (model.maxOutputTokens) {
    contextParts.push(formatTokenCount(model.maxOutputTokens) + ' out');
  }
  var cost = modelCostLine(model);
  var chips = capabilityChips(model, false);
  var rows = [];
  if (provider) {
    rows.push('<div class="model-meta-row"><span class="model-meta-label">Provider</span><span>' + escapeHTML(provider) + '</span></div>');
  }
  if (contextParts.length) {
    rows.push('<div class="model-meta-row"><span class="model-meta-label">Context</span><span>' + escapeHTML(contextParts.join(' · ')) + '</span></div>');
  }
  if (cost) {
    rows.push('<div class="model-meta-row"><span class="model-meta-label">Cost</span><span>' + escapeHTML(cost) + '</span></div>');
  }
  if (chips) {
    rows.push('<div class="model-meta-row"><span class="model-meta-label">Capabilities</span><span class="model-meta-chips">' + chips + '</span></div>');
  }
  if (!rows.length) {
    root.classList.add('hidden');
    root.innerHTML = '';
    return;
  }
  root.classList.remove('hidden');
  root.innerHTML = rows.join('');
}

function findModelEntry(modelID) {
  return modelCatalog.find(function (model) { return model.id === modelID; }) || null;
}

function updateModelCatalogSelection() {
  var root = document.getElementById('model-catalog');
  if (!root) {
    return;
  }
  root.querySelectorAll('.model-item[data-model-id]').forEach(function (button) {
    button.classList.toggle('active', button.getAttribute('data-model-id') === selectedModel);
  });
}

function showModelDetail(model) {
  selectedModel = model;
  updateModelCatalogSelection();
  document.getElementById('model-title').textContent = model;
  renderModelMeta(findModelEntry(model));
  renderModelApplications(model);
}

function isModelsViewVisible() {
  return !document.getElementById('view-models').classList.contains('hidden');
}

function fetchModelsCatalog() {
  if (!modelsFetchPromise) {
    modelsFetchPromise = api('/api/models').finally(function () {
      modelsFetchPromise = null;
    });
  }
  return modelsFetchPromise;
}

function fetchModelCatalog() {
  if (!modelsCatalogFetchPromise) {
    modelsCatalogFetchPromise = api('/api/models/catalog').finally(function () {
      modelsCatalogFetchPromise = null;
    });
  }
  return modelsCatalogFetchPromise;
}

function applyModelsCatalog(models) {
  catalog = models;
}

function applyModelCatalog(models) {
  var previousSelection = selectedModel;
  modelCatalog = models;
  updateProviderFilterOptions();

  if (!isModelsViewVisible()) {
    return;
  }

  if (!modelCatalog.length) {
    selectedModel = '';
    renderModelCatalog();
    document.getElementById('model-title').textContent = '';
    document.getElementById('model-meta').classList.add('hidden');
    document.getElementById('model-meta').innerHTML = '';
    document.getElementById('model-applications').innerHTML = '';
    return;
  }

  renderModelCatalog();
  var keepSelection = previousSelection && modelCatalog.some(function (model) { return model.id === previousSelection; });
  showModelDetail(keepSelection ? previousSelection : modelCatalog[0].id);
}

function showModelsLoadError(message) {
  if (!isModelsViewVisible()) {
    return;
  }
  document.getElementById('model-catalog').innerHTML =
    '<div class="empty">Unable to load models: ' + escapeHTML(message) + '</div>';
  document.getElementById('model-title').textContent = '';
  document.getElementById('model-meta').classList.add('hidden');
  document.getElementById('model-meta').innerHTML = '';
  document.getElementById('model-applications').innerHTML = '';
}

async function refreshModelsInBackground() {
  try {
    var simpleModels = fetchModelsCatalog();
    applyModelCatalog(await fetchModelCatalog());
    applyModelsCatalog(await simpleModels);
  } catch (e) {
    if (isGatewayAuthError(e.message)) {
      await refreshSettingsStatus();
    }
    if (!modelCatalog.length) {
      showModelsLoadError(e.message);
    }
  }
}

function prefetchModels() {
  refreshModelsInBackground();
}

// --- Help ---
function helpHeadingID(value) {
  return value.toLowerCase()
    .replace(/[`*_]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

function helpLinkURL(value, image) {
  if (image && value.indexOf('images/') === 0) {
    return '/docs/' + value;
  }
  if (value.indexOf('https://') === 0 || value.indexOf('#') === 0 || value === 'harnezpad://settings') {
    return value;
  }
  return '#';
}

function renderHelpInline(value) {
  var tokens = [];
  function token(html) {
    tokens.push(html);
    return '@@HARNEZPAD_HELP_' + (tokens.length - 1) + '@@';
  }

  value = value.replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, function (_, alt, source) {
    return token('<img src="' + escapeHTML(helpLinkURL(source, true)) + '" alt="' + escapeHTML(alt) + '">');
  });
  value = value.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, function (_, label, target) {
    var href = helpLinkURL(target, false);
    return token('<a href="' + escapeHTML(href) + '">' + escapeHTML(label) + '</a>');
  });

  var html = escapeHTML(value)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  return html.replace(/@@HARNEZPAD_HELP_(\d+)@@/g, function (_, index) {
    return tokens[Number(index)];
  });
}

function renderHelpMarkdown(markdown) {
  var lines = markdown.replace(/\r/g, '').split('\n');
  var html = [];
  var list = '';
  var code = [];
  var inCode = false;
  var codeLanguage = '';

  function closeList() {
    if (list) {
      html.push('</' + list + '>');
      list = '';
    }
  }

  lines.forEach(function (line) {
    if (line.indexOf('```') === 0) {
      closeList();
      if (inCode) {
        html.push('<pre><code class="language-' + escapeHTML(codeLanguage) + '">' + escapeHTML(code.join('\n')) + '</code></pre>');
        code = [];
        codeLanguage = '';
        inCode = false;
      } else {
        inCode = true;
        codeLanguage = line.slice(3).trim();
      }
      return;
    }
    if (inCode) {
      code.push(line);
      return;
    }
    if (!line.trim()) {
      closeList();
      return;
    }

    var heading = line.match(/^(#{1,3})\s+(.+)$/);
    if (heading) {
      closeList();
      var level = heading[1].length;
      html.push('<h' + level + ' id="' + helpHeadingID(heading[2]) + '">' + renderHelpInline(heading[2]) + '</h' + level + '>');
      return;
    }
    if (/^---+$/.test(line)) {
      closeList();
      html.push('<hr>');
      return;
    }
    if (line.indexOf('> ') === 0) {
      closeList();
      html.push('<blockquote>' + renderHelpInline(line.slice(2)) + '</blockquote>');
      return;
    }

    var unordered = line.match(/^[-*]\s+(.+)$/);
    var ordered = line.match(/^\d+\.\s+(.+)$/);
    if (unordered || ordered) {
      var nextList = unordered ? 'ul' : 'ol';
      if (list !== nextList) {
        closeList();
        list = nextList;
        html.push('<' + list + '>');
      }
      html.push('<li>' + renderHelpInline((unordered || ordered)[1]) + '</li>');
      return;
    }

    closeList();
    html.push('<p>' + renderHelpInline(line) + '</p>');
  });
  closeList();
  return html.join('');
}

async function openExternalHref(href) {
  if (!href || href.indexOf('https://') !== 0) {
    return false;
  }
  if (window.harnezpadOpenExternal) {
    await window.harnezpadOpenExternal(href);
    return true;
  }
  window.open(href, '_blank', 'noopener,noreferrer');
  return true;
}

async function handleDocumentLinkClick(event) {
  var anchor = event.target.closest('a');
  if (!anchor) {
    return;
  }
  var href = anchor.getAttribute('href');
  if (!href || href.indexOf('#') === 0) {
    return;
  }
  if (href === 'harnezpad://settings') {
    event.preventDefault();
    showView('settings');
    return;
  }
  if (await openExternalHref(href)) {
    event.preventDefault();
  }
}

async function openHelpLink(event) {
  await handleDocumentLinkClick(event);
}

async function loadHelp() {
  var root = document.getElementById('help-content');
  try {
    var response = await fetch('/docs/HELP.md');
    if (!response.ok) {
      throw new Error('HTTP ' + response.status);
    }
    root.innerHTML = renderHelpMarkdown(await response.text());
    root.addEventListener('click', openHelpLink);
  } catch (e) {
    root.innerHTML = '<h1>HarnezPad Help</h1><p class="error">Help could not be loaded. ' + escapeHTML(e.message) + '</p>';
  }
}

// --- Settings ---
function renderTokenState(editing) {
  var root = document.getElementById('token-state');
  var resting = settings.tokenValid && !editing;
  root.innerHTML = '<input id="token" type="password" autocomplete="off" placeholder="' + (resting ? 'sk-••••••••' : 'Paste sk-… management key') + '" ' +
    (resting ? 'readonly onclick="beginTokenEdit(this)"' : 'oninput="updateTokenActions()"') + '>' +
    '<div id="token-actions" class="actions token-actions hidden">' +
    actionButton('Save', { primary: true, icon: 'checkmark', onclick: 'saveToken()' }) +
    actionButton('Cancel', { onclick: 'renderTokenState(false)' }) +
    '<span id="token-status" class="status"></span></div>';
}

function beginTokenEdit(input) {
  input.readOnly = false;
  input.placeholder = 'Paste sk-… management key';
  input.onclick = null;
  input.setAttribute('oninput', 'updateTokenActions()');
  input.focus();
}

function updateTokenActions() {
  var input = document.getElementById('token');
  document.getElementById('token-actions').classList.toggle('hidden', !input.value);
}

async function saveManagementKey(token, statusEl) {
  applySettingsState(await api('/api/settings', {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ token: token })
  }));
  document.getElementById('setup-nudge').classList.add('hidden');
  document.getElementById('settings-status').textContent = '';
  var setupInput = document.getElementById('setup-token');
  if (setupInput) {
    setupInput.value = '';
  }
  resetSetupValidation();
  renderSetupHelpBanner(false);
  invalidateKeysPage();
  invalidateAccountSummary();
  refreshAccountPanel();
  refreshLaunchSpendBar();
  showToast('Management key saved securely in Keychain');
  if (isKeysViewVisible()) {
    await loadKeysPage(true);
  }
  if (statusEl) {
    statusEl.textContent = '';
    statusEl.className = 'status';
  }
}

async function saveSetupToken() {
  var token = document.getElementById('setup-token').value.trim();
  if (!token || !setupValidation.valid || token !== setupValidation.token) {
    showToast('Enter a valid management key');
    renderSetupValidation('error', 'Enter a valid management key');
    return;
  }
  renderSetupValidation('checking', 'Saving…');
  try {
    await saveManagementKey(token, null);
    showView('launch');
  } catch (e) {
    showToast(e.message, 'error');
    setupValidation.valid = false;
    renderSetupValidation('error', e.message);
    renderSetupContinue();
  }
}

async function saveToken() {
  var token = document.getElementById('token').value.trim();
  var status = document.getElementById('token-status');
  if (!token) {
    showToast('Paste a management key');
    status.className = 'status error';
    status.textContent = 'Paste a management key';
    return;
  }
  try {
    await saveManagementKey(token, status);
  } catch (e) {
    showToast(e.message, 'error');
    status.className = 'status error';
    status.textContent = e.message;
  }
}

function formatAccountSpend(amount, maxBudget) {
  var spend = '$' + Number(amount || 0).toFixed(2);
  if (maxBudget != null && maxBudget > 0) {
    return spend + ' / $' + Number(maxBudget).toFixed(2);
  }
  return spend;
}

function formatBudgetReset(value) {
  if (!value) {
    return '';
  }
  var date = new Date(value);
  if (isNaN(date.getTime())) {
    return '';
  }
  return 'Resets ' + date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function daysUntilReset(value) {
  if (!value) {
    return null;
  }
  var reset = new Date(value);
  if (isNaN(reset.getTime())) {
    return null;
  }
  var now = new Date();
  var startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  var startOfReset = new Date(reset.getFullYear(), reset.getMonth(), reset.getDate());
  var diffMs = startOfReset.getTime() - startOfToday.getTime();
  return Math.max(0, Math.ceil(diffMs / 86400000));
}

function formatResetInDays(value) {
  var days = daysUntilReset(value);
  if (days == null) {
    return '';
  }
  if (days === 0) {
    return 'Resets today';
  }
  if (days === 1) {
    return 'Resets in 1 day';
  }
  return 'Resets in ' + days + ' days';
}

function spendProgressPercent(spend, maxBudget) {
  if (maxBudget == null || maxBudget <= 0) {
    return 0;
  }
  return Math.min(100, Math.max(0, (Number(spend || 0) / maxBudget) * 100));
}

function fetchAccountSummary(force) {
  if (!settings.tokenValid) {
    return Promise.reject(new Error('management key is not configured'));
  }
  if (!force && accountSummary) {
    return Promise.resolve(accountSummary);
  }
  if (!accountFetchPromise) {
    accountFetchPromise = api('/api/account').then(function (summary) {
      accountSummary = summary;
      return summary;
    }).finally(function () {
      accountFetchPromise = null;
    });
  }
  return accountFetchPromise;
}

function invalidateAccountSummary() {
  accountSummary = null;
}

function renderLaunchSpendBar(account) {
  var bar = document.getElementById('launch-spend-bar');
  var amount = document.getElementById('launch-spend-amount');
  var fill = document.getElementById('launch-spend-fill');
  var reset = document.getElementById('launch-spend-reset');
  if (!bar || !amount || !fill || !reset) {
    return;
  }
  if (!account || isSetupRequired() || !settings.tokenValid) {
    bar.classList.add('hidden');
    return;
  }
  var maxBudget = account.maxBudget;
  if (maxBudget == null || maxBudget <= 0) {
    bar.classList.add('hidden');
    return;
  }
  bar.classList.remove('hidden');
  amount.textContent = formatAccountSpend(account.spend, maxBudget);
  fill.style.width = spendProgressPercent(account.spend, maxBudget) + '%';
  reset.textContent = formatResetInDays(account.budgetResetAt) || formatBudgetReset(account.budgetResetAt);
}

async function refreshLaunchSpendBar() {
  if (!settings.tokenValid || isSetupRequired()) {
    renderLaunchSpendBar(null);
    return;
  }
  try {
    renderLaunchSpendBar(await fetchAccountSummary(false));
  } catch (e) {
    renderLaunchSpendBar(null);
  }
}

function renderAccountPanel(account, message, isError) {
  var root = document.getElementById('account-content');
  if (!root) {
    return;
  }
  if (message) {
    root.innerHTML = '<p class="' + (isError ? 'error' : 'muted-note') + '">' + escapeHTML(message) + '</p>';
    return;
  }
  var teamLabel = account.teamAlias || account.teamId || '—';
  var rows = [
    ['Team', teamLabel],
    ['Role', account.role || '—'],
    ['Spend', formatAccountSpend(account.spend, account.maxBudget)],
    ['Budget reset', formatBudgetReset(account.budgetResetAt) || '—']
  ];
  if (account.rpmLimit != null) {
    rows.push(['Rate limit', account.rpmLimit + ' req/min']);
  }
  root.innerHTML = '<div class="account-grid">' + rows.map(function (row) {
    return '<div class="account-stat"><span class="account-stat-label">' + escapeHTML(row[0]) +
      '</span><span class="account-stat-value">' + escapeHTML(row[1]) + '</span></div>';
  }).join('') + '</div>';
}

async function refreshAccountPanel() {
  var root = document.getElementById('account-content');
  if (!root) {
    return;
  }
  if (!settings.tokenValid) {
    renderAccountPanel(null, 'Save a valid management key to view account usage.', false);
    return;
  }
  renderAccountPanel(null, 'Loading account details…', false);
  try {
    renderAccountPanel(await fetchAccountSummary(true), null, false);
  } catch (e) {
    renderAccountPanel(null, e.message || 'Unable to load account details.', true);
  }
}

// --- Keys page ---
function modelsSummary(key) {
  if (key.allModels || !key.models || !key.models.length) {
    return 'All models';
  }
  return key.models.join(', ');
}

function spendColumn(key) {
  var amount = '$' + Number(key.spend || 0).toFixed(2);
  if (key.maxBudget != null) {
    var budget = '$' + Number(key.maxBudget).toFixed(2);
    return '<span class="key-spend-amount">' + amount + '</span><span class="key-spend-budget"> / ' + budget + '</span>';
  }
  return '<span class="key-spend-amount">' + amount + '</span>';
}

function renderModelBadges(key) {
  if (key.allModels || !key.models || !key.models.length) {
    return '<span class="key-model-badge">All models</span>';
  }
  return key.models.map(function (model) {
    return '<span class="key-model-badge">' + escapeHTML(model) + '</span>';
  }).join('');
}

function renderKeysStatus(message, className) {
  var node = document.getElementById('keys-status');
  node.className = 'keys-status ' + (className || '');
  node.innerHTML = message || '';
}

function sortKeysForDisplay(keys) {
  var management = keys.filter(function (key) { return key.management; });
  var regular = keys.filter(function (key) { return !key.management; });
  return management.concat(regular);
}

function renderKeysList() {
  var root = document.getElementById('keys-list');
  var createBtn = document.getElementById('keys-create-btn');
  if (isSetupRequired()) {
    createBtn.disabled = true;
    renderKeysStatus('Complete first-time setup or save a management key in <button class="inline-link" onclick="returnToSetup()">setup</button> before managing keys.', 'info-note');
    root.innerHTML = '';
    return;
  }
  if (!keysPage.supported) {
    createBtn.disabled = true;
    renderKeysStatus(escapeHTML(keysPage.reason || 'Key management is not available for your current key.') +
      ' <button class="inline-link" onclick="showView(\'settings\')">Open Settings</button>', 'info-note');
    root.innerHTML = '';
    return;
  }
  createBtn.disabled = false;
  renderKeysStatus('');
  if (!keysPage.keys.length) {
    root.innerHTML = '<div class="keys-panel keys-panel-empty"><div class="empty">No keys yet. Create a launch key for HarnezPad or your CLI tools.</div></div>';
    return;
  }
  var rows = sortKeysForDisplay(keysPage.keys).map(function (key) {
    var statusBadge = '<span class="key-badge' + (key.blocked ? ' blocked' : '') + '">' + (key.blocked ? 'Blocked' : 'Active') + '</span>';
    var activeBadge = key.active ? '<span class="key-badge active">Saved for launch</span>' : '';
    var defaultBadge = key.default ? '<span class="key-badge default">Default launch key</span>' : '';
    var managementBadge = key.management ? '<span class="key-badge management">Management</span>' : '';
    var menuOpen = keysPage.openMenuId === key.id;
    var menuPanelClass = menuOpen ? '' : ' hidden';
    var blockLabel = key.blocked ? 'Unblock' : 'Block';
    var blockAction = key.blocked ? 'unblock' : 'block';
    var toolsHTML = '';
    if (!key.management) {
      toolsHTML = '<button type="button" class="key-edit-btn" title="Edit key" aria-label="Edit key">' + editIconHTML + '</button>' +
        '<div class="key-menu">' +
        '<button type="button" class="key-menu-btn" title="More actions" aria-label="More actions"' + (menuOpen ? ' aria-expanded="true"' : '') + '>' + moreIconHTML + '</button>' +
        '<div class="key-menu-panel' + menuPanelClass + '">' +
        (key.default ? '' : '<button type="button" class="key-menu-item" data-action="default">Set as default</button>') +
        '<button type="button" class="key-menu-item" data-action="' + blockAction + '">' + blockLabel + '</button>' +
        '<button type="button" class="key-menu-item danger" data-action="delete">Delete</button>' +
        '</div></div>';
    }
    return '<article class="key-row' + (key.active ? ' active-key' : '') + (menuOpen ? ' menu-open' : '') + '" data-key-id="' + escapeHTML(key.id) + '">' +
      '<div class="key-row-grid">' +
      '<div class="key-row-main">' +
      '<div class="key-row-title-line">' +
      '<div class="key-row-title">' + escapeHTML(key.slug || key.alias) + '</div>' + managementBadge + defaultBadge + activeBadge + '</div>' +
      '<div class="key-model-badges">' + renderModelBadges(key) + '</div></div>' +
      '<div class="key-row-spend">' + spendColumn(key) + '</div>' +
      '<div class="key-badges">' + statusBadge + '</div>' +
      '<div class="key-row-tools">' + toolsHTML + '</div></div></article>';
  }).join('');
  root.innerHTML = '<div class="keys-panel"><div class="keys-table">' +
    '<div class="keys-table-head"><div>Key</div><div class="key-head-spend">Spend</div><div class="key-head-status">Status</div><div></div></div>' +
    '<div class="keys-table-body">' + rows + '</div></div></div>';
}

function findKeyByID(keyID) {
  return keysPage.keys.find(function (item) { return item.id === keyID; });
}

function isKeysViewVisible() {
  return !document.getElementById('view-keys').classList.contains('hidden');
}

function keysScrollContainer() {
  return document.querySelector('.keys-table-body');
}

function positionKeyMenuPanel(row) {
  if (!row) {
    return;
  }
  var panel = row.querySelector('.key-menu-panel');
  if (!panel) {
    return;
  }
  panel.classList.remove('open-up');
  var rect = panel.getBoundingClientRect();
  if (rect.bottom > window.innerHeight - 12) {
    panel.classList.add('open-up');
  }
}

function renderKeysListPreserveScroll() {
  var container = keysScrollContainer();
  var scrollTop = container ? container.scrollTop : 0;
  var openMenuId = keysPage.openMenuId;
  renderKeysList();
  keysPage.openMenuId = openMenuId;
  if (openMenuId) {
    var row = document.querySelector('.key-row[data-key-id="' + CSS.escape(openMenuId) + '"]');
    var panel = row ? row.querySelector('.key-menu-panel') : null;
    if (panel) {
      panel.classList.remove('hidden');
      positionKeyMenuPanel(row);
    }
  }
  if (container) {
    container.scrollTop = scrollTop;
  }
}

function patchKey(keyID, patch) {
  var key = findKeyByID(keyID);
  if (!key) {
    return null;
  }
  if (patch.alias !== undefined) {
    key.alias = patch.alias;
    key.slug = patch.alias;
  }
  if (patch.models !== undefined) {
    key.models = patch.models.slice();
    key.allModels = patch.models.length === 0;
  }
  if (patch.blocked !== undefined) {
    key.blocked = patch.blocked;
  }
  if (patch.active !== undefined) {
    key.active = patch.active;
  }
  return key;
}

function closeKeyMenus() {
  keysPage.openMenuId = '';
  document.querySelectorAll('.key-menu-panel').forEach(function (panel) {
    panel.classList.add('hidden');
  });
}

function handleKeysListClick(event) {
  var editButton = event.target.closest('.key-edit-btn');
  if (editButton) {
    var row = editButton.closest('.key-row');
    if (row) {
      openEditKeyModal(row.dataset.keyId);
    }
    return;
  }

  var menuButton = event.target.closest('.key-menu-btn');
  if (menuButton) {
    event.stopPropagation();
    var menuRow = menuButton.closest('.key-row');
    if (!menuRow) {
      return;
    }
    var keyID = menuRow.dataset.keyId;
    keysPage.openMenuId = keysPage.openMenuId === keyID ? '' : keyID;
    renderKeysListPreserveScroll();
    if (keysPage.openMenuId) {
      positionKeyMenuPanel(menuRow);
    }
    return;
  }

  var menuItem = event.target.closest('.key-menu-item');
  if (menuItem) {
    event.stopPropagation();
    var actionRow = menuItem.closest('.key-row');
    if (!actionRow) {
      return;
    }
    var actionID = actionRow.dataset.keyId;
    var key = findKeyByID(actionID);
    if (!key || key.management) {
      return;
    }
    closeKeyMenus();
    if (menuItem.dataset.action === 'default') {
      setDefaultKeyRequest(key);
      return;
    }
    if (menuItem.dataset.action === 'delete') {
      confirmKeyAction('Delete key', buildDeleteMessage(key), async function () {
        await deleteKeyRequest(key);
        var wasDefault = key.default;
        keysPage.keys = keysPage.keys.filter(function (item) { return item.id !== key.id; });
        if (wasDefault) {
          settings.defaultKeySlug = 'management-key';
          keysPage.keys.forEach(function (item) {
            item.default = !!item.management;
          });
        }
        renderKeysListPreserveScroll();
        showToast(wasDefault ? 'Key deleted; default launch key reset to management-key' : 'Key deleted', 'success');
      });
      return;
    }
    if (menuItem.dataset.action === 'block') {
      confirmKeyAction('Block key', 'Block "' + key.alias + '"? It will stop working immediately.', async function () {
        await setKeyBlockedRequest(key, true);
        patchKey(key.id, { blocked: true });
        renderKeysListPreserveScroll();
        showToast('Key blocked', 'success');
      });
      return;
    }
    if (menuItem.dataset.action === 'unblock') {
      confirmKeyAction('Unblock key', 'Unblock "' + key.alias + '"?', async function () {
        await setKeyBlockedRequest(key, false);
        patchKey(key.id, { blocked: false });
        renderKeysListPreserveScroll();
        showToast('Key unblocked', 'success');
      });
    }
  }
}

function buildDeleteMessage(key) {
  var message = 'Delete "' + key.alias + '"? This cannot be undone.';
  if (key.active) {
    message += ' This key is saved for launch in HarnezPad. Launches will stop working until you save a replacement in Settings.';
  }
  if (key.default) {
    message += ' Launches will fall back to management-key.';
  }
  return message;
}

function confirmKeyAction(title, message, action) {
  keysPage.confirmAction = action;
  openKeysModal(
    '<h2>' + escapeHTML(title) + '</h2>' +
    '<p class="subtitle">' + escapeHTML(message) + '</p>' +
    modalActionsHTML('keys-form-status',
      actionButton('Confirm', { primary: true, icon: 'checkmark', onclick: 'runKeysConfirmAction(this)' }) +
      actionButton('Cancel', { onclick: 'closeKeysModal()' }))
  );
}

async function runKeysConfirmAction(button) {
  if (!keysPage.confirmAction) {
    return;
  }
  var status = document.getElementById('keys-form-status');
  button.disabled = true;
  setModalWorking(status, 'Working…');
  try {
    await keysPage.confirmAction();
    keysPage.confirmAction = null;
    closeKeysModal();
  } catch (e) {
    button.disabled = false;
    showToast(e.message, 'error');
    setModalError(status, e.message);
  }
}

async function setDefaultKeyRequest(key) {
  try {
    await api('/api/keys/' + encodeURIComponent(key.id) + '/default', { method: 'POST' });
    settings.defaultKeySlug = key.slug || key.alias;
    keysPage.keys.forEach(function (item) {
      item.default = item.id === key.id;
    });
    renderKeysListPreserveScroll();
    showToast('Default launch key updated');
  } catch (e) {
    showToast(e.message);
  }
}

async function deleteKeyRequest(key) {
  await api('/api/keys/' + encodeURIComponent(key.id) + '/delete', { method: 'POST' });
}

async function setKeyBlockedRequest(key, blocked) {
  var path = blocked ? '/block' : '/unblock';
  await api('/api/keys/' + encodeURIComponent(key.id) + path, { method: 'POST' });
}

function initKeysListEvents() {
  var root = document.getElementById('keys-list');
  if (!root || root.dataset.bound === '1') {
    return;
  }
  root.dataset.bound = '1';
  root.addEventListener('click', handleKeysListClick);
  document.addEventListener('click', function (event) {
    if (!event.target.closest('.key-menu')) {
      closeKeyMenus();
    }
  });
}

async function ensureKeysPage() {
  if (keysPage.loaded) {
    renderKeysListPreserveScroll();
    return;
  }
  await loadKeysPage(true);
}

async function loadKeysPage(force) {
  if (force !== true && keysPage.loaded) {
    renderKeysListPreserveScroll();
    return;
  }
  var container = keysScrollContainer();
  var scrollTop = container ? container.scrollTop : 0;
  var showLoading = !keysPage.loaded || force === true;
  if (showLoading) {
    document.getElementById('keys-list').innerHTML = '<div class="models-loading"><span class="spinner" aria-hidden="true"></span><span>Loading keys…</span></div>';
    renderKeysStatus('');
  }
  if (isSetupRequired()) {
    keysPage.loaded = false;
    renderKeysList();
    return;
  }
  try {
    var caps = await api('/api/keys/capabilities');
    keysPage.supported = !!caps.supported;
    keysPage.reason = caps.reason || '';
    if (!keysPage.supported) {
      keysPage.loaded = true;
      renderKeysList();
      return;
    }
    var list = await api('/api/keys');
    keysPage.keys = sortKeysForDisplay(list.keys || []);
    keysPage.loaded = true;
    renderKeysList();
    if (container) {
      container.scrollTop = scrollTop;
    }
  } catch (e) {
    keysPage.loaded = false;
    if (isGatewayAuthError(e.message)) {
      await refreshSettingsStatus();
    }
    renderKeysStatus('Unable to load keys: ' + escapeHTML(e.message), 'error');
    document.getElementById('keys-list').innerHTML = '';
    showToast(e.message, 'error');
  }
}

function closeKeysModal() {
  document.getElementById('keys-modal-backdrop').classList.add('hidden');
  document.getElementById('keys-modal-content').innerHTML = '';
}

function closeKeysModalOnBackdrop(event) {
  if (event.target.id === 'keys-modal-backdrop') {
    closeKeysModal();
  }
}

document.addEventListener('keydown', function (event) {
  if (event.key === 'Escape' && !document.getElementById('keys-modal-backdrop').classList.contains('hidden')) {
    closeKeysModal();
  }
});

function openKeysModal(html) {
  document.getElementById('keys-modal-content').innerHTML = html;
  document.getElementById('keys-modal-backdrop').classList.remove('hidden');
}

function modalWorkingHTML(message) {
  return '<span class="spinner" aria-hidden="true"></span><span>' + escapeHTML(message || 'Working…') + '</span>';
}

function setModalWorking(statusNode, message) {
  if (!statusNode) {
    return;
  }
  statusNode.className = 'modal-status';
  statusNode.innerHTML = modalWorkingHTML(message);
}

function setModalError(statusNode, message) {
  if (!statusNode) {
    return;
  }
  statusNode.className = 'modal-status error';
  statusNode.textContent = message;
}

function modalActionsHTML(statusID, buttonsHTML) {
  return '<div class="modal-actions">' +
    '<span id="' + statusID + '" class="modal-status"></span>' +
    '<div class="modal-action-buttons">' + buttonsHTML + '</div></div>';
}

function renderModelPicker(selectedModels, inputName) {
  if (!catalog.length) {
    return '<p class="muted-note">All team models will be allowed. Model list loads from the Gateway when available.</p>';
  }
  var selected = {};
  (selectedModels || []).forEach(function (model) { selected[model] = true; });
  return '<div class="model-picker">' + catalog.map(function (model) {
    var checked = selected[model.id] ? ' checked' : '';
    return '<label><input type="checkbox" name="' + inputName + '" value="' + escapeHTML(model.id) + '"' + checked + '> ' +
      escapeHTML(model.id) + '</label>';
  }).join('') + '</div>';
}

function readSelectedModels(form) {
  var boxes = form.querySelectorAll('input[name="key-models"]:checked');
  var models = [];
  boxes.forEach(function (box) { models.push(box.value); });
  return models;
}

function openCreateKeyModal() {
  openKeysModal(
    '<h2>Create key</h2>' +
    '<p class="subtitle">Choose a short name for the CLI. Use lowercase letters, numbers, and hyphens only.</p>' +
    '<form id="keys-form" onsubmit="submitCreateKey(event)">' +
    '<div class="field"><label for="key-alias">Key name</label>' +
    '<input id="key-alias" required placeholder="harnezpad" autocomplete="off" autocapitalize="none" spellcheck="false" oninput="this.value = slugifyKeyName(this.value)"></div>' +
    '<div class="field"><label>Models (optional)</label>' + renderModelPicker([], 'key-models') + '</div>' +
    modalActionsHTML('keys-form-status',
      actionButton('Create', { primary: true, icon: 'plus', submit: true }) +
      actionButton('Cancel', { onclick: 'closeKeysModal()' })) +
    '</form>'
  );
  document.getElementById('key-alias').focus();
}

async function submitCreateKey(event) {
  event.preventDefault();
  var form = document.getElementById('keys-form');
  var status = document.getElementById('keys-form-status');
  var alias = slugifyKeyName(document.getElementById('key-alias').value);
  if (!isValidKeySlug(alias)) {
    showToast('Enter a valid key name');
    setModalError(status, 'Use lowercase letters, numbers, and hyphens only');
    return;
  }
  setModalWorking(status, 'Creating…');
  try {
    var result = await api('/api/keys', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ alias: alias, models: readSelectedModels(form) })
    });
    keysPage.pendingSecret = result.key;
    await api('/api/keys/register', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ slug: alias, token: result.key })
    });
    if (result.summary) {
      keysPage.keys = sortKeysForDisplay([result.summary].concat(keysPage.keys.filter(function (key) { return key.id !== result.summary.id; })));
      keysPage.loaded = true;
      renderKeysListPreserveScroll();
    }
    showToast('Key created');
    openSecretModal(result.key, result.summary && result.summary.alias ? result.summary.alias : alias);
  } catch (e) {
    showToast(e.message);
    setModalError(status, e.message);
  }
}

function openSecretModal(secret, alias) {
  keysPage.pendingSecret = secret;
  openKeysModal(
    '<h2>Key created</h2>' +
    '<p class="subtitle">Copy this secret now. It will not be shown again.</p>' +
    '<div class="secret-value">' + escapeHTML(secret) + '</div>' +
    modalActionsHTML('keys-form-status',
      actionButton('Copy key', { primary: true, icon: 'doc.on.doc', onclick: 'copyPendingSecret(this)' }) +
      actionButton('Done', { icon: 'checkmark', onclick: 'closeKeysModal()' }))
  );
}

async function copyPendingSecret(button) {
  try {
    await writeClipboard(keysPage.pendingSecret);
    showToast('Copied to clipboard');
    if (button) {
      button.innerHTML = sfIcon('checkmark', 14) + 'Copied';
      setTimeout(function () { button.innerHTML = sfIcon('doc.on.doc', 14) + 'Copy key'; }, 900);
    }
  } catch (e) {
    showToast('Copy failed');
    if (button) {
      button.innerHTML = sfIcon('doc.on.doc', 14) + 'Copy failed';
    }
  }
}

function openEditKeyModal(keyID) {
  var key = findKeyByID(keyID);
  if (!key || key.management) {
    return;
  }
  keysPage.editingKeyId = keyID;
  openKeysModal(
    '<h2>Edit key</h2>' +
    '<p class="subtitle">Update the name or allowed models.</p>' +
    '<form id="keys-form" onsubmit="submitEditKey(event)">' +
    '<div class="field"><label for="key-alias">Key name</label>' +
    '<input id="key-alias" required value="' + escapeHTML(key.slug || key.alias) + '" autocomplete="off" autocapitalize="none" spellcheck="false" oninput="this.value = slugifyKeyName(this.value)"></div>' +
    '<div class="field"><label>Models (optional)</label>' +
    renderModelPicker(key.allModels ? [] : key.models, 'key-models') + '</div>' +
    modalActionsHTML('keys-form-status',
      actionButton('Save', { primary: true, icon: 'checkmark', submit: true }) +
      actionButton('Cancel', { onclick: 'closeKeysModal()' })) +
    '</form>'
  );
  document.getElementById('key-alias').focus();
}

async function submitEditKey(event) {
  event.preventDefault();
  var form = document.getElementById('keys-form');
  var status = document.getElementById('keys-form-status');
  var alias = slugifyKeyName(document.getElementById('key-alias').value);
  if (!isValidKeySlug(alias)) {
    showToast('Enter a valid key name');
    setModalError(status, 'Use lowercase letters, numbers, and hyphens only');
    return;
  }
  setModalWorking(status, 'Saving…');
  try {
    var models = readSelectedModels(form);
    await api('/api/keys/' + encodeURIComponent(keysPage.editingKeyId), {
      method: 'PUT',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ alias: alias, models: models })
    });
    patchKey(keysPage.editingKeyId, { alias: alias, models: models });
    closeKeysModal();
    renderKeysListPreserveScroll();
    showToast('Key updated', 'success');
  } catch (e) {
    showToast(e.message, 'error');
    setModalError(status, e.message);
  }
}

// --- Launch page ---
async function loadCLIStatus() {
  var node = document.getElementById('cli-status');
  try {
    var state = await api('/api/cli-status');
    if (state.installed) {
      node.className = 'cli-note';
      node.innerHTML = 'HarnezPad CLI installed at <code>~/.local/bin/harnezpad</code>. Add that directory to your shell PATH if needed.';
    } else {
      node.className = 'cli-note error';
      node.textContent = state.error + ' Reinstall HarnezPad from the DMG.';
    }
  } catch (e) {
    node.className = 'cli-note error';
    node.textContent = 'Unable to verify HarnezPad CLI. Reinstall HarnezPad from the DMG.';
  }
}

// --- Init ---
document.addEventListener('click', handleDocumentLinkClick, true);

document.getElementById('view-models').addEventListener('wheel', function (event) {
  if (event.target.closest('.model-sidebar')) {
    return;
  }
  var main = document.querySelector('#view-models .model-page-main');
  if (main) {
    var canScrollDown = main.scrollTop + main.clientHeight < main.scrollHeight - 1;
    var canScrollUp = main.scrollTop > 0;
    if ((event.deltaY > 0 && canScrollDown) || (event.deltaY < 0 && canScrollUp)) {
      return;
    }
  }
  var list = document.getElementById('model-catalog');
  if (list) {
    list.scrollTop += event.deltaY;
  }
  event.preventDefault();
}, { passive: false });

async function init() {
  loadHelp();
  try {
    applySettingsState(await api('/api/settings'));
    if (isSetupRequired()) {
      showView('launch');
    }
    await loadCLIStatus();
  } catch (e) {
    document.getElementById('settings-status').className = 'status error';
    document.getElementById('settings-status').textContent = e.message;
  }
  prefetchModels();
  refreshLaunchSpendBar();
  initKeysListEvents();
  initModelCatalogEvents();
  startUpdatePolling();
}

init();
