package utils

import (
	"strings"
)

// EscapeHTMLEntities returns a string in which HTML entities are escaped as required by Telegram: <>&
func EscapeHTMLEntities(s string) string {
	r := strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;")
	return r.Replace(s)
}
