package prime

import (
	"fmt"
	"strings"
)

func urlIteratorParams(url string, p *IteratorParams) string {

	appended := strings.Contains(url, "?")

	if len(p.Cursor) > 0 {
		url += fmt.Sprintf("%scursor=%s", urlParamSep(appended), p.Cursor)
		appended = true
	}

	if len(p.Limit) > 0 {
		url += fmt.Sprintf("%slimit=%s", urlParamSep(appended), p.Limit)
		appended = true
	}

	if len(p.SortDirection) > 0 {
		url += fmt.Sprintf("%sort_direction=%s", urlParamSep(appended), p.SortDirection)
		appended = true
	}

	return url
}

func urlParamSep(appended bool) string {
	if appended {
		return "&"
	}
	return "?"
}
