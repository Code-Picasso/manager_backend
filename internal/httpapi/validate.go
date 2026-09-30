package httpapi

import (
	"net/mail"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)

type validationErrors map[string][]string

func (e validationErrors) add(field, message string) {
	e[field] = append(e[field], message)
}

var hhmmPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

func validEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

// requiredString validates a required, non-empty string field.
func requiredString(m map[string]any, key string, e validationErrors) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		e.add(key, "The "+key+" field is required.")
		return "", false
	}
	s, isStr := v.(string)
	if !isStr {
		e.add(key, "The "+key+" field must be a string.")
		return "", false
	}
	if s == "" {
		e.add(key, "The "+key+" field is required.")
		return "", false
	}
	return s, true
}

// maxLength enforces Laravel's max rule on a validated string.
func maxLength(s, key string, max int, e validationErrors) {
	if utf8.RuneCountInString(s) > max {
		e.add(key, "The "+key+" field must not be greater than "+strconv.Itoa(max)+" characters.")
	}
}

// requiredEmail validates a required email string.
func requiredEmail(m map[string]any, key string, e validationErrors) (string, bool) {
	s, ok := requiredString(m, key, e)
	if !ok {
		return "", false
	}
	if !validEmail(s) {
		e.add(key, "The "+key+" field must be a valid email address.")
		return "", false
	}
	return s, true
}

// requiredEnum validates a required string against allowed values.
func requiredEnum(m map[string]any, key string, allowed []string, e validationErrors) string {
	v, ok := m[key]
	if !ok || v == nil {
		e.add(key, "The "+key+" field is required.")
		return ""
	}
	s, isStr := v.(string)
	if !isStr {
		e.add(key, "The selected "+key+" is invalid.")
		return ""
	}
	for _, a := range allowed {
		if s == a {
			return s
		}
	}
	e.add(key, "The selected "+key+" is invalid.")
	return ""
}

// requiredDate validates a required "YYYY-MM-DD" date string.
func requiredDate(m map[string]any, key string, e validationErrors) string {
	v, ok := m[key]
	if !ok || v == nil {
		e.add(key, "The "+key+" field is required.")
		return ""
	}
	s, isStr := v.(string)
	if !isStr || s == "" {
		e.add(key, "The "+key+" field is required.")
		return ""
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		e.add(key, "The "+key+" field must be a valid date.")
		return ""
	}
	return s
}

// requiredTime validates a required "HH:MM" time string.
func requiredTime(m map[string]any, key string, e validationErrors) string {
	v, ok := m[key]
	if !ok || v == nil {
		e.add(key, "The "+key+" field is required.")
		return ""
	}
	s, isStr := v.(string)
	if !isStr || s == "" {
		e.add(key, "The "+key+" field is required.")
		return ""
	}
	if !hhmmPattern.MatchString(s) {
		e.add(key, "The "+key+" field must match the format H:i.")
		return ""
	}
	return s
}

// optionalString returns an optional string field, treating absent and null as "".
func optionalString(m map[string]any, key string, e validationErrors) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", true
	}
	s, isStr := v.(string)
	if !isStr {
		e.add(key, "The "+key+" field must be a string.")
		return "", false
	}
	return s, true
}
