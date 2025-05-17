// filters.go
// Built-in filtreler ve filtre yönetimi
package hipoengine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type FilterFunc func(val interface{}, args ...interface{}) interface{}

// DefaultFilters: built-in filtrelerin listesi
var DefaultFilters = map[string]FilterFunc{
	"upper": func(val interface{}, args ...interface{}) interface{} {
		return strings.ToUpper(fmt.Sprintf("%v", val))
	},
	"lower": func(val interface{}, args ...interface{}) interface{} {
		return strings.ToLower(fmt.Sprintf("%v", val))
	},
	"length": func(val interface{}, args ...interface{}) interface{} {
		switch v := val.(type) {
		case string:
			return len(v)
		case []interface{}:
			return len(v)
		}
		return 0
	},
	"trim": func(val interface{}, args ...interface{}) interface{} {
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	},
	"title": func(val interface{}, args ...interface{}) interface{} {
		return strings.Title(fmt.Sprintf("%v", val))
	},
	"reverse": func(val interface{}, args ...interface{}) interface{} {
		runes := []rune(fmt.Sprintf("%v", val))
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	},
	"default": func(val interface{}, args ...interface{}) interface{} {
		if val == nil || fmt.Sprintf("%v", val) == "" {
			if len(args) > 0 {
				return args[0]
			}
		}
		return val
	},
	"safe": func(val interface{}, args ...interface{}) interface{} {
		return val // HTML escape'i engellemek için
	},
	"date": func(val interface{}, args ...interface{}) interface{} {
		format := "2006-01-02"
		if len(args) > 0 {
			format = fmt.Sprintf("%v", args[0])
		}
		if t, ok := val.(time.Time); ok {
			return t.Format(format)
		}
		return val
	},
	"join": func(val interface{}, args ...interface{}) interface{} {
		sep := ","
		if len(args) > 0 {
			sep = fmt.Sprintf("%v", args[0])
		}
		if arr, ok := val.([]interface{}); ok {
			strs := make([]string, len(arr))
			for i, v := range arr {
				strs[i] = fmt.Sprintf("%v", v)
			}
			return strings.Join(strs, sep)
		}
		return val
	},
	"add": func(val interface{}, args ...interface{}) interface{} {
		if len(args) == 0 {
			return val
		}
		f1 := toFloat(val)
		f2 := toFloat(args[0])
		return f1 + f2
	},
	"money": func(val interface{}, args ...interface{}) interface{} {
		if f, ok := val.(float64); ok {
			return fmt.Sprintf("%.2f", f)
		}
		if i, ok := val.(int); ok {
			return fmt.Sprintf("%d", i)
		}
		return val
	},
	"truncate": func(val interface{}, args ...interface{}) interface{} {
		limit := 10
		if len(args) > 0 {
			limit = int(toFloat(args[0]))
		}
		s := fmt.Sprintf("%v", val)
		if len(s) > limit {
			return s[:limit] + "..."
		}
		return s
	},
	"slice": func(val interface{}, args ...interface{}) interface{} {
		start, end := 0, 0
		if len(args) > 0 {
			start = int(toFloat(args[0]))
		}
		if len(args) > 1 {
			end = int(toFloat(args[1]))
		}
		if arr, ok := val.([]interface{}); ok {
			if end == 0 || end > len(arr) {
				end = len(arr)
			}
			if start < 0 {
				start = 0
			}
			if start < end && end <= len(arr) {
				return arr[start:end]
			}
		}
		s := fmt.Sprintf("%v", val)
		if end == 0 || end > len(s) {
			end = len(s)
		}
		if start < 0 {
			start = 0
		}
		if start < end && end <= len(s) {
			return s[start:end]
		}
		return val
	},
	"replace": func(val interface{}, args ...interface{}) interface{} {
		if len(args) < 2 {
			return val
		}
		old := fmt.Sprintf("%v", args[0])
		value := fmt.Sprintf("%v", args[1])
		s := fmt.Sprintf("%v", val)
		return strings.ReplaceAll(s, old, value)
	},
	"abs": func(val interface{}, args ...interface{}) interface{} {
		f := toFloat(val)
		if f < 0 {
			return -f
		}
		return f
	},
	"yesno": func(val interface{}, args ...interface{}) interface{} {
		yes, no := "evet", "hayır"
		if len(args) > 0 {
			yes = fmt.Sprintf("%v", args[0])
		}
		if len(args) > 1 {
			no = fmt.Sprintf("%v", args[1])
		}
		b := false
		if val == nil {
			b = false
		} else if v, ok := val.(bool); ok {
			b = v
		} else if s, ok := val.(string); ok {
			b = s != "" && s != "0"
		} else if i, ok := val.(int); ok {
			b = i != 0
		}
		if b {
			return yes
		}
		return no
	},
	"sort": func(val interface{}, args ...interface{}) interface{} {
		arr, ok := val.([]interface{})
		if !ok {
			return val
		}
		if len(arr) == 0 {
			return arr
		}
		if _, ok := arr[0].(string); ok {
			sorted := make([]string, len(arr))
			for i, v := range arr {
				sorted[i] = fmt.Sprintf("%v", v)
			}
			sort.Strings(sorted)
			res := make([]interface{}, len(sorted))
			for i, v := range sorted {
				res[i] = v
			}
			return res
		}
		if _, ok := arr[0].(int); ok {
			sorted := make([]int, len(arr))
			for i, v := range arr {
				sorted[i] = int(toFloat(v))
			}
			sort.Ints(sorted)
			res := make([]interface{}, len(sorted))
			for i, v := range sorted {
				res[i] = v
			}
			return res
		}
		return arr
	},
	"uniq": func(val interface{}, args ...interface{}) interface{} {
		arr, ok := val.([]interface{})
		if !ok {
			return val
		}
		seen := make(map[interface{}]bool)
		res := make([]interface{}, 0, len(arr))
		for _, v := range arr {
			if !seen[v] {
				seen[v] = true
				res = append(res, v)
			}
		}
		return res
	},
	"split": func(val interface{}, args ...interface{}) interface{} {
		s := fmt.Sprintf("%v", val)
		sep := ","
		if len(args) > 0 {
			sep = fmt.Sprintf("%v", args[0])
		}
		parts := strings.Split(s, sep)
		res := make([]interface{}, len(parts))
		for i, v := range parts {
			res[i] = v
		}
		return res
	},
	"slugify": func(val interface{}, args ...interface{}) interface{} {
		s := strings.ToLower(fmt.Sprintf("%v", val))
		turkish := map[string]string{"ç": "c", "ğ": "g", "ı": "i", "ö": "o", "ş": "s", "ü": "u"}
		for k, v := range turkish {
			s = strings.ReplaceAll(s, k, v)
		}
		s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
		s = strings.Trim(s, "-")
		return s
	},
	"startswith": func(val interface{}, args ...interface{}) interface{} {
		s := fmt.Sprintf("%v", val)
		if len(args) == 0 {
			return false
		}
		prefix := fmt.Sprintf("%v", args[0])
		return strings.HasPrefix(s, prefix)
	},
	"endswith": func(val interface{}, args ...interface{}) interface{} {
		s := fmt.Sprintf("%v", val)
		if len(args) == 0 {
			return false
		}
		suffix := fmt.Sprintf("%v", args[0])
		return strings.HasSuffix(s, suffix)
	},
	"pad": func(val interface{}, args ...interface{}) interface{} {
		s := fmt.Sprintf("%v", val)
		width := 0
		if len(args) > 0 {
			width = int(toFloat(args[0]))
		}
		if len(s) >= width {
			return s
		}
		return s + strings.Repeat(" ", width-len(s))
	},
	"ljust": func(val interface{}, args ...interface{}) interface{} {
		s := fmt.Sprintf("%v", val)
		width := 0
		if len(args) > 0 {
			width = int(toFloat(args[0]))
		}
		if len(s) >= width {
			return s
		}
		return s + strings.Repeat(" ", width-len(s))
	},
	"rjust": func(val interface{}, args ...interface{}) interface{} {
		s := fmt.Sprintf("%v", val)
		width := 0
		if len(args) > 0 {
			width = int(toFloat(args[0]))
		}
		if len(s) >= width {
			return s
		}
		return strings.Repeat(" ", width-len(s)) + s
	},
	"humanize": func(val interface{}, args ...interface{}) interface{} {
		t, ok := val.(time.Time)
		if !ok {
			return val
		}
		delta := time.Since(t)
		if delta < time.Minute {
			return "az önce"
		}
		if delta < time.Hour {
			return fmt.Sprintf("%d dakika önce", int(delta.Minutes()))
		}
		if delta < 24*time.Hour {
			return fmt.Sprintf("%d saat önce", int(delta.Hours()))
		}
		if delta < 7*24*time.Hour {
			return fmt.Sprintf("%d gün önce", int(delta.Hours()/24))
		}
		return t.Format("2006-01-02")
	},
	"regex_replace": func(val interface{}, args ...interface{}) interface{} {
		s := fmt.Sprintf("%v", val)
		if len(args) < 2 {
			return s
		}
		pattern := fmt.Sprintf("%v", args[0])
		repl := fmt.Sprintf("%v", args[1])
		re := regexp.MustCompile(pattern)
		return re.ReplaceAllString(s, repl)
	},
}
