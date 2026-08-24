package ui

import (
	"strings"
	"testing"
)

func TestLauncherUISections(t *testing.T) {
	css, err := content.ReadFile("css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	js, err := content.ReadFile("js/app.js")
	if err != nil {
		t.Fatal(err)
	}

	for _, value := range []string{
		`id="app-shell"`,
		`id="sidebar-toggle"`,
		`href="/ui/css/app.css"`,
		`src="/ui/js/app.js"`,
		`src="/ui/sfsymbol/gearshape.png`,
		`src="/ui/sfsymbol/terminal.png`,
		`src="/ui/sfsymbol/sidebar.left.png`,
		`src="/ui/sfsymbol/plus.png`,
		`src="/ui/sfsymbol/arrow.right.png?size=14"`,
		`src="/assets/codex-app.png"`,
		`id="nav-launch"`,
		`id="nav-models"`,
		`id="nav-keys"`,
		`id="nav-settings"`,
		`id="view-keys"`,
		`id="toast-root"`,
		`class="keys-page-main"`,
		`id="setup-nudge"`,
		`id="launch-spend-bar"`,
		`id="launch-spend-fill"`,
		`class="launch-heading-row"`,
		`class="launch-heading-copy"`,
		`class="launch-spend-compact-meta"`,
		`id="setup-screen"`,
		`id="setup-subtitle"`,
		`id="setup-gateway-url"`,
		`id="setup-gateway-link"`,
		`id="setup-token"`,
		`id="setup-validation"`,
		`setup-key-section`,
		`https://gateway.example.com/ui/`,
		`new-virtual-key.png`,
		`<h2>Endpoint</h2>`,
		`id="gateway-url" class="endpoint-value"`,
		`id="token-state"`,
		`id="sidebar-update-btn"`,
		`id="nav-settings-item"`,
		`nav-settings-item`,
		`class="application-heading">Applications</div>`,
		`id="model-title"`,
		`id="model-meta"`,
		`id="model-provider-filter"`,
		`id="model-capability-filters"`,
		`id="account-panel"`,
		`id="account-content"`,
		`id="model-search"`,
		`class="model-sidebar"`,
		`class="model-page-main"`,
	} {
		if !strings.Contains(HTML, value) {
			t.Fatalf("launcher UI is missing %q", value)
		}
	}

	for _, value := range []string{
		`.shell {`,
		`.sidebar {`,
		`.sidebar-toggle {`,
		`.command {`,
		`.model-search {`,
		`.model-item-provider {`,
		`.model-capability-chip {`,
		`.account-grid {`,
		`.launch-heading-row {`,
		`.launch-spend-compact {`,
		`.launch-spend-compact-fill {`,
		`.model-meta {`,
		`.model-item-label {`,
		`.keys-page-main {`,
		`.key-model-badge {`,
		`.key-spend-amount {`,
		`keys-active`,
		`.key-head-status {`,
		`.key-badge.management {`,
		`.spinner {`,
		`.modal-action-buttons {`,
		`.modal-status {`,
		`.modal-backdrop.hidden {`,
		`.keys-table-body {`,
		`.toast-root {`,
		`.setup-key-section {`,
		`.setup-validation {`,
		`.setup-gateway-url {`,
		`.setup-screen {`,
		`.nav-settings-item {`,
		`.sidebar-update-btn {`,
		`.sidebar-update-btn.expanded {`,
		`.sidebar-update-label {`,
	} {
		if !strings.Contains(string(css), value) {
			t.Fatalf("launcher CSS is missing %q", value)
		}
	}

	for _, value := range []string{
		`type="password"`,
		`showModelDetail`,
		`updateModelCatalogSelection`,
		`initModelCatalogEvents`,
		`data-model-id`,
		`copyModelCommand`,
		`showModelsLoading`,
		`filterModelCatalog`,
		`fetchModelCatalog`,
		`prefetchModels`,
		`refreshLaunchSpendBar`,
		`formatResetInDays`,
		`fetchAccountSummary`,
		`formatCostPerMillion`,
		`capabilityChips`,
		`/api/models/catalog`,
		`/api/account`,
		`refreshModelsInBackground`,
		`loadKeysPage`,
		`ensureKeysPage`,
		`showToast`,
		`renderKeysListPreserveScroll`,
		`renderModelBadges`,
		`modalActionsHTML`,
		`setModalWorking`,
		`function sfIcon`,
		`function actionButton`,
		`/ui/sfsymbol/`,
		`handleDocumentLinkClick`,
		`openExternalHref`,
		`handleSetupTokenInput`,
		`validateSetupToken`,
		`/api/settings/validate`,
		`renderSetupValidation`,
		`renderSetupContinue`,
		`saveSetupToken`,
		`returnToSetup`,
		`isSetupRequired`,
		`renderSetupScreen`,
		`tokenValid`,
		`setupReason`,
		`refreshSettingsStatus`,
		`renderSetupContent`,
		`isGatewayAuthError`,
		`renderSetupGatewayLink`,
		`applySettingsState`,
		`slugifyKeyName`,
		`setDefaultKeyRequest`,
		`installPendingUpdate`,
		`refreshUpdateStatus`,
		`startUpdatePolling`,
		`keys-table-body`,
		`setAppSidebarCollapsed(true)`,
		`'claude'`,
		`'codex-desktop'`,
	} {
		if !strings.Contains(string(js), value) {
			t.Fatalf("launcher JS is missing %q", value)
		}
	}

	launchEnd := strings.Index(HTML, `<section id="view-models"`)
	if launchEnd < 0 {
		t.Fatal("launcher UI is missing the Models section")
	}
	launch := HTML[:launchEnd]
	if strings.Contains(launch, "model-inline") || strings.Contains(launch, "<select") {
		t.Fatal("home Launch section should not contain model selectors")
	}
	if strings.Contains(HTML, "model-card") {
		t.Fatal("Models section should use the compact application list layout")
	}
	if strings.Contains(string(js), "catalog.length+' models available'") {
		t.Fatal("Models section should not show a model count")
	}
	claude := strings.Index(string(js), `'claude'`)
	chatgpt := strings.Index(string(js), `'codex-desktop'`)
	if claude < 0 || chatgpt < 0 || claude > chatgpt {
		t.Fatal("Models applications should list Claude Code before ChatGPT")
	}
	for _, unwanted := range []string{"Hermes Agent", "OpenClaw", "default-codex", "default-claude"} {
		if strings.Contains(HTML, unwanted) || strings.Contains(string(js), unwanted) {
			t.Fatalf("launcher UI contains removed application or default mapping %q", unwanted)
		}
	}
}
