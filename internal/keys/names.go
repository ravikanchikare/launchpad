package keys

import (
	"fmt"
	"regexp"
	"strings"
)

const ManagementSlug = "management-key"

// TestInvalidManagementKey bypasses gateway validation on save so developers can
// exercise invalid-key onboarding. It is always treated as invalid after save.
const TestInvalidManagementKey = "xxx"

func IsTestInvalidManagementKey(token string) bool {
	return strings.TrimSpace(token) == TestInvalidManagementKey
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func Slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '_' || r == '-':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return ""
	}
	return out
}

func ValidateSlug(slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return fmt.Errorf("key name is required")
	}
	if len(slug) > 64 {
		return fmt.Errorf("key name must be 64 characters or fewer")
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("key name must use lowercase letters, numbers, and hyphens only")
	}
	return nil
}

func SlugFromAlias(alias string) string {
	if err := ValidateSlug(alias); err == nil {
		return alias
	}
	return Slugify(alias)
}
