package hipoengine

import "fmt"
import "strings"

// FilterFunc, bir filter fonksiyonunun imzası
// (ör: upper, lower, length, custom filters)
type FilterFunc func(interface{}) string

// DefaultFilters, engine'e otomatik eklenen filtreler
var DefaultFilters = map[string]FilterFunc{
	"upper": func(val interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%v", val))
	},
	"lower": func(val interface{}) string {
		return strings.ToLower(fmt.Sprintf("%v", val))
	},
	"length": func(val interface{}) string {
		switch v := val.(type) {
		case string:
			return fmt.Sprintf("%d", len(v))
		case []interface{}:
			return fmt.Sprintf("%d", len(v))
		}
		return "0"
	},
	"trim": func(val interface{}) string {
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	},
	"title": func(val interface{}) string {
		return strings.Title(fmt.Sprintf("%v", val))
	},
	"reverse": func(val interface{}) string {
		runes := []rune(fmt.Sprintf("%v", val))
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	},
}
