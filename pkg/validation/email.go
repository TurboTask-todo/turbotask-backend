package validation

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

// EmailValidator handles email validation
type EmailValidator struct {
	allowedDomains []string
	blockedDomains []string
}

// NewEmailValidator creates a new email validator
func NewEmailValidator() *EmailValidator {
	return &EmailValidator{
		allowedDomains: []string{}, // Empty means all domains allowed
		blockedDomains: []string{
			// Common temporary email domains to block
			"10minutemail.com",
			"guerrillamail.com",
			"mailinator.com",
			"tempmail.org",
			"temp-mail.org",
			"throwaway.email",
			"yopmail.com",
			"maildrop.cc",
		},
	}
}

// NewEmailValidatorWithDomains creates a new email validator with domain restrictions
func NewEmailValidatorWithDomains(allowedDomains, blockedDomains []string) *EmailValidator {
	return &EmailValidator{
		allowedDomains: allowedDomains,
		blockedDomains: blockedDomains,
	}
}

// ValidateEmail validates an email address
func (ev *EmailValidator) ValidateEmail(email string) []string {
	var errors []string

	// Basic email format validation using Go's mail package
	if err := ev.validateBasicFormat(email); err != nil {
		errors = append(errors, err.Error())
		return errors // Return early if basic format is invalid
	}

	// Length validation
	if len(email) > 320 { // RFC 5322 limit
		errors = append(errors, "Email address is too long (maximum 320 characters)")
	}

	// Extract domain for domain-specific validation
	domain := extractDomain(email)

	// Domain validation
	if domainErr := ev.validateDomain(domain); domainErr != nil {
		errors = append(errors, domainErr.Error())
	}

	// Additional format validation
	if formatErrs := ev.validateAdvancedFormat(email); len(formatErrs) > 0 {
		errors = append(errors, formatErrs...)
	}

	return errors
}

// validateBasicFormat validates basic email format using Go's mail package
func (ev *EmailValidator) validateBasicFormat(email string) error {
	if email == "" {
		return fmt.Errorf("email address is required")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// validateAdvancedFormat performs additional format validation
func (ev *EmailValidator) validateAdvancedFormat(email string) []string {
	var errors []string

	// Check for consecutive dots
	if strings.Contains(email, "..") {
		errors = append(errors, "Email cannot contain consecutive dots")
	}

	// Check for leading/trailing dots in local part
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		localPart := parts[0]
		if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
			errors = append(errors, "Email local part cannot start or end with a dot")
		}
	}

	// Check for valid characters (more restrictive than RFC)
	validEmailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !validEmailRegex.MatchString(email) {
		errors = append(errors, "Email contains invalid characters")
	}

	return errors
}

// validateDomain validates the domain part of the email
func (ev *EmailValidator) validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("email domain is required")
	}

	// Check if domain is blocked
	for _, blocked := range ev.blockedDomains {
		if strings.EqualFold(domain, blocked) {
			return fmt.Errorf("email domain '%s' is not allowed", domain)
		}
	}

	// Check if only specific domains are allowed
	if len(ev.allowedDomains) > 0 {
		allowed := false
		for _, allowedDomain := range ev.allowedDomains {
			if strings.EqualFold(domain, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("email domain '%s' is not in the allowed list", domain)
		}
	}

	// Basic domain format validation
	if !isValidDomain(domain) {
		return fmt.Errorf("invalid email domain format")
	}

	return nil
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// isValidDomain validates domain format
func isValidDomain(domain string) bool {
	// Basic domain validation regex
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

	if !domainRegex.MatchString(domain) {
		return false
	}

	// Check domain length
	if len(domain) > 253 {
		return false
	}

	// Check for valid TLD (at least 2 characters)
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	tld := parts[len(parts)-1]
	if len(tld) < 2 {
		return false
	}

	return true
}

// NormalizeEmail normalizes an email address for consistent storage
func NormalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	// Remove dots from Gmail addresses (gmail.com and googlemail.com)
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		localPart := parts[0]
		domain := parts[1]

		if domain == "gmail.com" || domain == "googlemail.com" {
			// Remove dots from local part
			localPart = strings.ReplaceAll(localPart, ".", "")

			// Remove everything after + (gmail alias)
			if plusIndex := strings.Index(localPart, "+"); plusIndex != -1 {
				localPart = localPart[:plusIndex]
			}

			// Always use gmail.com domain
			email = localPart + "@gmail.com"
		}
	}

	return email
}

// IsValidEmailFormat performs a quick email format check
func IsValidEmailFormat(email string) bool {
	if email == "" {
		return false
	}

	_, err := mail.ParseAddress(email)
	return err == nil
}

// GetEmailDomain extracts and returns the domain from an email address
func GetEmailDomain(email string) string {
	return extractDomain(email)
}

// IsBusinessEmail checks if an email appears to be from a business domain
func IsBusinessEmail(email string) bool {
	domain := extractDomain(email)

	// Common consumer email domains
	consumerDomains := []string{
		"gmail.com", "yahoo.com", "hotmail.com", "outlook.com",
		"aol.com", "icloud.com", "me.com", "mail.com",
		"protonmail.com", "zoho.com", "yandex.com",
	}

	for _, consumerDomain := range consumerDomains {
		if strings.EqualFold(domain, consumerDomain) {
			return false
		}
	}

	return true
}

// SuggestEmailCorrection suggests a corrected email for common typos
func SuggestEmailCorrection(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))

	// Common domain typo corrections
	corrections := map[string]string{
		"gmail.co":   "gmail.com",
		"gmail.cm":   "gmail.com",
		"gmial.com":  "gmail.com",
		"gmai.com":   "gmail.com",
		"yahoo.co":   "yahoo.com",
		"yahoo.cm":   "yahoo.com",
		"hotmail.co": "hotmail.com",
		"hotmail.cm": "hotmail.com",
		"outlook.co": "outlook.com",
		"outlook.cm": "outlook.com",
	}

	domain := extractDomain(email)
	if correctedDomain, exists := corrections[domain]; exists {
		parts := strings.Split(email, "@")
		if len(parts) == 2 {
			return parts[0] + "@" + correctedDomain
		}
	}

	return email
}
