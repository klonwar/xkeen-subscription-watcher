package main

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

// filterRules contains the filters that apply to one subscription scope.
// Empty include lists do not restrict the corresponding attribute.
type filterRules struct {
	includeNames      []string
	includeNameGlobs  []string
	excludeNames      []string
	excludeNameGlobs  []string
	includeProtocols  []string
	excludeProtocols  []string
	includeTransports []string
	excludeTransports []string
	limit             *int
}

// filterOptions contains raw repeatable CLI values before they are validated
// and assigned to global or per-tag rule sets.
type filterOptions struct {
	includeNames      []string
	includeNameGlobs  []string
	excludeNames      []string
	excludeNameGlobs  []string
	includeProtocols  []string
	excludeProtocols  []string
	includeTransports []string
	excludeTransports []string
	limits            []string
}

type filterConfig struct {
	global filterRules
	byTag  map[string]filterRules
}

func parseFilterConfig(options filterOptions, tags map[string]bool) (filterConfig, error) {
	filters := filterConfig{byTag: make(map[string]filterRules)}

	for _, value := range options.includeNames {
		if err := filters.addNameRule(value, tags, true, false); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --include-name: %w", err)
		}
	}
	for _, value := range options.includeNameGlobs {
		if err := filters.addNameRule(value, tags, true, true); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --include-name-glob: %w", err)
		}
	}
	for _, value := range options.excludeNames {
		if err := filters.addNameRule(value, tags, false, false); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --exclude-name: %w", err)
		}
	}
	for _, value := range options.excludeNameGlobs {
		if err := filters.addNameRule(value, tags, false, true); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --exclude-name-glob: %w", err)
		}
	}
	for _, value := range options.includeProtocols {
		if err := filters.addProtocolRule(value, tags, true); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --include-protocol: %w", err)
		}
	}
	for _, value := range options.excludeProtocols {
		if err := filters.addProtocolRule(value, tags, false); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --exclude-protocol: %w", err)
		}
	}
	for _, value := range options.includeTransports {
		if err := filters.addTransportRule(value, tags, true); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --include-transport: %w", err)
		}
	}
	for _, value := range options.excludeTransports {
		if err := filters.addTransportRule(value, tags, false); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --exclude-transport: %w", err)
		}
	}
	for _, value := range options.limits {
		if err := filters.addLimit(value, tags); err != nil {
			return filterConfig{}, fmt.Errorf("ошибка --limit: %w", err)
		}
	}

	return filters, nil
}

func (f *filterConfig) addNameRule(rawValue string, tags map[string]bool, include, glob bool) error {
	return f.addScopedValue(rawValue, tags, func(r *filterRules, value string) error {
		value = strings.ToLower(value)
		if glob {
			if _, err := path.Match(value, ""); err != nil {
				return fmt.Errorf("некорректный glob %q: %w", value, err)
			}
			if include {
				r.includeNameGlobs = append(r.includeNameGlobs, value)
			} else {
				r.excludeNameGlobs = append(r.excludeNameGlobs, value)
			}
			return nil
		}

		if include {
			r.includeNames = append(r.includeNames, value)
		} else {
			r.excludeNames = append(r.excludeNames, value)
		}
		return nil
	})
}

func (f *filterConfig) addProtocolRule(rawValue string, tags map[string]bool, include bool) error {
	return f.addScopedValue(rawValue, tags, func(r *filterRules, value string) error {
		value, err := normalizeFilterValue(value, filterProtocols, "протокол")
		if err != nil {
			return err
		}
		if include {
			r.includeProtocols = append(r.includeProtocols, value)
		} else {
			r.excludeProtocols = append(r.excludeProtocols, value)
		}
		return nil
	})
}

func (f *filterConfig) addTransportRule(rawValue string, tags map[string]bool, include bool) error {
	return f.addScopedValue(rawValue, tags, func(r *filterRules, value string) error {
		value, err := normalizeFilterValue(value, filterTransports, "транспорт")
		if err != nil {
			return err
		}
		if include {
			r.includeTransports = append(r.includeTransports, value)
		} else {
			r.excludeTransports = append(r.excludeTransports, value)
		}
		return nil
	})
}

func (f *filterConfig) addLimit(rawValue string, tags map[string]bool) error {
	scope, value, err := splitScopedFilterValue(rawValue, tags)
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("лимит %q должен быть целым числом", value)
	}
	if n < 0 {
		return fmt.Errorf("лимит не может быть отрицательным: %d", n)
	}

	if scope == "" {
		if f.global.limit != nil {
			return fmt.Errorf("лимит для общего правила указан несколько раз")
		}
		f.global.limit = &n
		return nil
	}

	rules := f.byTag[scope]
	if rules.limit != nil {
		return fmt.Errorf("лимит для тега %q указан несколько раз", scope)
	}
	rules.limit = &n
	f.byTag[scope] = rules
	return nil
}

func (f *filterConfig) addScopedValue(rawValue string, tags map[string]bool, add func(*filterRules, string) error) error {
	scope, value, err := splitScopedFilterValue(rawValue, tags)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("значение правила не может быть пустым")
	}

	if scope == "" {
		return add(&f.global, value)
	}

	rules := f.byTag[scope]
	if err := add(&rules, value); err != nil {
		return err
	}
	f.byTag[scope] = rules
	return nil
}

func splitScopedFilterValue(rawValue string, tags map[string]bool) (scope, value string, err error) {
	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return "", "", fmt.Errorf("значение правила не может быть пустым")
	}

	if idx := strings.IndexByte(rawValue, '='); idx >= 0 {
		scope = strings.TrimSpace(rawValue[:idx])
		value = strings.TrimSpace(rawValue[idx+1:])
		if scope == "" {
			return "", "", fmt.Errorf("не указан tag в значении %q", rawValue)
		}
		if !tags[scope] {
			return "", "", fmt.Errorf("неизвестный tag %q", scope)
		}
		if value == "" {
			return "", "", fmt.Errorf("значение правила для тега %q не может быть пустым", scope)
		}
		return scope, value, nil
	}

	return "", rawValue, nil
}

func normalizeFilterValue(value string, allowed map[string]struct{}, field string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("значение %s не может быть пустым", field)
	}
	if _, ok := allowed[value]; !ok {
		return "", fmt.Errorf("неподдерживаемый %s %q", field, value)
	}
	return value, nil
}

func (f filterConfig) rulesFor(tag string) filterRules {
	rules := f.global
	specific, ok := f.byTag[tag]
	if !ok {
		return rules
	}

	rules.includeNames = append(append([]string{}, f.global.includeNames...), specific.includeNames...)
	rules.includeNameGlobs = append(append([]string{}, f.global.includeNameGlobs...), specific.includeNameGlobs...)
	rules.excludeNames = append(append([]string{}, f.global.excludeNames...), specific.excludeNames...)
	rules.excludeNameGlobs = append(append([]string{}, f.global.excludeNameGlobs...), specific.excludeNameGlobs...)
	rules.includeProtocols = append(append([]string{}, f.global.includeProtocols...), specific.includeProtocols...)
	rules.excludeProtocols = append(append([]string{}, f.global.excludeProtocols...), specific.excludeProtocols...)
	rules.includeTransports = append(append([]string{}, f.global.includeTransports...), specific.includeTransports...)
	rules.excludeTransports = append(append([]string{}, f.global.excludeTransports...), specific.excludeTransports...)
	if specific.limit != nil {
		rules.limit = specific.limit
	}

	return rules
}

func (r filterRules) matches(name, protocol, transport string) bool {
	name = strings.ToLower(name)
	protocol = strings.ToLower(protocol)
	transport = strings.ToLower(transport)

	if len(r.includeNames) > 0 || len(r.includeNameGlobs) > 0 {
		matched := false
		for _, pattern := range r.includeNames {
			if strings.Contains(name, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			for _, pattern := range r.includeNameGlobs {
				if ok, _ := path.Match(pattern, name); ok {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	if len(r.includeProtocols) > 0 && !containsFilterValue(r.includeProtocols, protocol) {
		return false
	}
	if len(r.includeTransports) > 0 && !containsFilterValue(r.includeTransports, transport) {
		return false
	}

	for _, pattern := range r.excludeNames {
		if strings.Contains(name, pattern) {
			return false
		}
	}
	for _, pattern := range r.excludeNameGlobs {
		if ok, _ := path.Match(pattern, name); ok {
			return false
		}
	}
	if containsFilterValue(r.excludeProtocols, protocol) || containsFilterValue(r.excludeTransports, transport) {
		return false
	}

	return true
}

func containsFilterValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func proxyProtocolTransport(p proxy) (protocol, transport string) {
	switch p := p.(type) {
	case vlessProxy:
		return "vless", p.network
	case vmessProxy:
		return "vmess", p.network
	case trojanProxy:
		return "trojan", p.network
	case shadowSocksProxy:
		return "ss", ""
	case hysteria2Proxy:
		return "hysteria2", "hysteria"
	default:
		return "", ""
	}
}

var filterProtocols = map[string]struct{}{
	"vless":     {},
	"vmess":     {},
	"trojan":    {},
	"ss":        {},
	"hysteria2": {},
}

var filterTransports = map[string]struct{}{
	"tcp":      {},
	"raw":      {},
	"ws":       {},
	"grpc":     {},
	"xhttp":    {},
	"hysteria": {},
}
