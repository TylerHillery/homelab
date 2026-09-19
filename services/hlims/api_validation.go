package hlims

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	hostnameLabelPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	machineUserPattern   = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._-]{0,63}$`)
)

func decodeJSON(rContentType string, body io.Reader, value any) error {
	mediaType, _, err := mime.ParseMediaType(rContentType)
	if err != nil || mediaType != "application/json" {
		return errUnsupportedMediaType
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

var errUnsupportedMediaType = errors.New("Content-Type must be application/json")

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("name must not be empty")
	}
	return value, nil
}

func normalizeSlug(value string) (string, error) {
	var slug strings.Builder
	separator := false
	for _, char := range strings.TrimSpace(strings.ToLower(value)) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			if separator && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteRune(char)
			separator = false
		} else if unicode.IsSpace(char) || char == '-' || char == '_' {
			separator = true
		} else {
			return "", errors.New("slug may contain only ASCII letters, numbers, spaces, hyphens, or underscores")
		}
	}
	if slug.Len() == 0 {
		return "", errors.New("slug must not be empty")
	}
	return slug.String(), nil
}

func writeSlug(name string, supplied *string, existing string) (string, error) {
	if supplied != nil {
		return normalizeSlug(*supplied)
	}
	if existing != "" {
		return existing, nil
	}
	return normalizeSlug(name)
}

func normalizeHostname(value *string) (sql.NullString, error) {
	if value == nil {
		return sql.NullString{}, nil
	}
	hostname := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(*value)), ".")
	if hostname == "" || len(hostname) > 253 {
		return sql.NullString{}, errors.New("hostname is invalid")
	}
	for _, label := range strings.Split(hostname, ".") {
		if !hostnameLabelPattern.MatchString(label) {
			return sql.NullString{}, errors.New("hostname is invalid")
		}
	}
	return sql.NullString{String: hostname, Valid: true}, nil
}

func normalizeMachineUsername(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !machineUserPattern.MatchString(value) {
		return "", errors.New("username must be 1-64 ASCII letters, numbers, dots, underscores, or hyphens and must not start with a hyphen")
	}
	return value, nil
}

func normalizeIP(value string) (string, error) {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return "", errors.New("address must be a valid IP address")
	}
	return address.String(), nil
}

func normalizeCIDR(value *string) (sql.NullString, error) {
	if value == nil {
		return sql.NullString{}, nil
	}
	prefix, err := netip.ParsePrefix(strings.TrimSpace(*value))
	if err != nil {
		return sql.NullString{}, errors.New("cidr must be a valid IP prefix")
	}
	return sql.NullString{String: prefix.Masked().String(), Valid: true}, nil
}

func normalizeBasePath(value *string) (string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "", nil
	}
	path := strings.TrimSpace(*value)
	parsed, err := url.Parse(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("basePath must be a URL path without a query or fragment")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	for _, segment := range strings.Split(path, "/") {
		decoded, err := url.PathUnescape(segment)
		if err != nil || decoded == "." || decoded == ".." {
			return "", errors.New("basePath must not contain traversal segments")
		}
	}
	return path, nil
}

func nullString(value *string, trim bool) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	result := *value
	if trim {
		result = strings.TrimSpace(result)
	}
	return sql.NullString{String: result, Valid: true}
}

func stringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableTrimmed(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func nullInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func intPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func int64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func nullableEnum[T ~string](value *T) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(*value), Valid: true}
}

func boolInt(value *bool) int64 {
	if value != nil && *value {
		return 1
	}
	return 0
}
