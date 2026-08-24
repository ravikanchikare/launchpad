package app

import (
	"strings"
)

func UserFacingError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "management key is expired"):
		return "Management key is expired. Create a replacement in Virtual Keys."
	case strings.Contains(lower, "management key is invalid"):
		return "Management key is invalid. Create a replacement in Virtual Keys."
	case strings.Contains(lower, "management key is not configured"):
		return "Management key is not configured. Save it in HarnezPad Settings before launching."
	case strings.Contains(lower, "management key cannot be empty"), strings.Contains(lower, "api token cannot be empty"):
		return "Management key cannot be empty."
	case strings.Contains(lower, "gateway token"):
		return "Management key is not configured. Save it in HarnezPad Settings before launching."
	case strings.HasPrefix(lower, "save management key:") || strings.HasPrefix(lower, "save token:"):
		return "Could not save the management key to Keychain."
	case strings.Contains(lower, "gateway returned no models"), strings.Contains(lower, "no models are available"):
		return "No models are available for your management key."
	case strings.HasPrefix(lower, "gateway returned"):
		if strings.Contains(msg, "401") || strings.Contains(lower, "unauthorized") {
			if strings.Contains(lower, "expir") {
				return "Management key is expired. Create a replacement in Virtual Keys."
			}
			return "Management key is invalid or expired."
		}
		if strings.Contains(msg, "403") || strings.Contains(lower, "forbidden") {
			return "Management key does not have access to this Gateway resource."
		}
		return "Could not reach the Gateway. Check your connection and try again."
	case strings.Contains(lower, "unable to discover models"):
		return "Could not load models from the Gateway. Check your management key and try again."
	case strings.Contains(lower, "self-update is unavailable"):
		return "Updates are not available in this build."
	case strings.Contains(lower, "no verified update is ready"):
		return "No update is ready to install yet."
	case strings.Contains(lower, "update checking is not available"):
		return "Update checking is not available yet."
	case strings.Contains(lower, "downloaded update sha-256"):
		return "The downloaded update could not be verified. Try checking for updates again."
	case strings.Contains(lower, "previous update is awaiting cleanup"):
		return "Restart HarnezPad before installing another update."
	case strings.Contains(lower, "could not be downloaded"), strings.Contains(lower, "check for update:"), strings.Contains(lower, "download update:"):
		return "Could not download the update. Check your connection and try again."
	case strings.Contains(lower, "key not found"):
		return "Key not found."
	}

	return msg
}
