// Package pages provides shared helpers for go-app page components.
package pages

import "regexp"

// UUIDPattern is the regex fragment matching a standard UUID. Use it when
// registering routes with app.RouteWithRegexp:
//
//	app.RouteWithRegexp(fmt.Sprintf("^/items/(%s)$", pages.UUIDPattern), ...)
const UUIDPattern = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

// IDExtractor is a function that extracts one or two IDs from a URL path.
type IDExtractor func(path string) (string, string)

// ExtractPathIDs returns the first and second UUIDs found in path.
// secondID is empty if fewer than two UUIDs are present.
func ExtractPathIDs(path string) (string, string) {
	uuidRegex := regexp.MustCompile(UUIDPattern)
	matches := uuidRegex.FindAllString(path, -1)
	if len(matches) == 0 {
		return "", ""
	}
	if len(matches) == 1 {
		return matches[0], ""
	}
	return matches[0], matches[1]
}
