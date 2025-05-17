// utils.go
package hipoengine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type FileBlocks struct {
	Template string
	Script   string
	Style    string
}

// HTML dosyasındaki ana blokları ayırır
func SplitBlocks(content string) FileBlocks {
	reTemplate := regexp.MustCompile(`(?s)<template>(.*?)</template>`)
	reScript := regexp.MustCompile(`(?s)<script>(.*?)</script>`)
	reStyle := regexp.MustCompile(`(?s)<style>(.*?)</style>`)

	var blocks FileBlocks

	if match := reTemplate.FindStringSubmatch(content); len(match) > 1 {
		blocks.Template = match[1]
	}
	if match := reScript.FindStringSubmatch(content); len(match) > 1 {
		blocks.Script = match[1]
	}
	if match := reStyle.FindStringSubmatch(content); len(match) > 1 {
		blocks.Style = match[1]
	}
	return blocks
}

// Sadece <template> içeriğini çıkarır (block/extends için)
func extractTemplateBlock(content string) string {
	re := regexp.MustCompile(`(?s)<template>(.*?)</template>`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		// Hem baş hem son boşluk ve satır sonlarını sil!
		return strings.Trim(matches[1], "\r\n\t ")
	}
	return content
}

// Mantıksal karşılaştırmaları çalıştırır (if, elif vs)
func evalBool(expr string, ctx *Context) bool {
	expr = strings.TrimSpace(expr)
	if strings.Contains(expr, ">=") {
		parts := strings.Split(expr, ">=")
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		lval := ctx.Resolve(left)
		rval := ctx.Resolve(right)
		li, lok := lval.(int)
		ri, rok := rval.(int)
		if !lok {
			li, _ = strconv.Atoi(fmt.Sprintf("%v", lval))
		}
		if !rok {
			ri, _ = strconv.Atoi(fmt.Sprintf("%v", rval))
		}
		return li >= ri
	}
	if strings.Contains(expr, "<=") {
		parts := strings.Split(expr, "<=")
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		lval := ctx.Resolve(left)
		rval := ctx.Resolve(right)
		li, lok := lval.(int)
		ri, rok := rval.(int)
		if !lok {
			li, _ = strconv.Atoi(fmt.Sprintf("%v", lval))
		}
		if !rok {
			ri, _ = strconv.Atoi(fmt.Sprintf("%v", rval))
		}
		return li <= ri
	}
	if strings.Contains(expr, ">") {
		parts := strings.Split(expr, ">")
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		lval := ctx.Resolve(left)
		rval := ctx.Resolve(right)
		li, lok := lval.(int)
		ri, rok := rval.(int)
		if !lok {
			li, _ = strconv.Atoi(fmt.Sprintf("%v", lval))
		}
		if !rok {
			ri, _ = strconv.Atoi(fmt.Sprintf("%v", rval))
		}
		return li > ri
	}
	if strings.Contains(expr, "<") {
		parts := strings.Split(expr, "<")
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		lval := ctx.Resolve(left)
		rval := ctx.Resolve(right)
		li, lok := lval.(int)
		ri, rok := rval.(int)
		if !lok {
			li, _ = strconv.Atoi(fmt.Sprintf("%v", lval))
		}
		if !rok {
			ri, _ = strconv.Atoi(fmt.Sprintf("%v", rval))
		}
		return li < ri
	}
	if strings.Contains(expr, "==") {
		parts := strings.Split(expr, "==")
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		lval := ctx.Resolve(left)
		rval := ctx.Resolve(right)
		if rval == nil && strings.HasPrefix(right, "\"") && strings.HasSuffix(right, "\"") {
			rval = strings.Trim(right, "\"")
		}
		return fmt.Sprintf("%v", lval) == fmt.Sprintf("%v", rval)
	}
	if strings.Contains(expr, "!=") {
		parts := strings.Split(expr, "!=")
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		lval := ctx.Resolve(left)
		rval := ctx.Resolve(right)
		if rval == nil && strings.HasPrefix(right, "\"") && strings.HasSuffix(right, "\"") {
			rval = strings.Trim(right, "\"")
		}
		return fmt.Sprintf("%v", lval) != fmt.Sprintf("%v", rval)
	}
	val := ctx.Resolve(expr)
	if b, ok := val.(bool); ok {
		return b
	}
	if s, ok := val.(string); ok {
		return s != "" && s != "0"
	}
	if val != nil {
		return true
	}
	return false
}

// Her türlü map'i map[string]interface{}'ye çevirir
func toStringMap(val interface{}) map[string]interface{} {
	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	if m, ok := val.(map[interface{}]interface{}); ok {
		res := make(map[string]interface{})
		for k, v := range m {
			res[fmt.Sprintf("%v", k)] = v
		}
		return res
	}
	return nil
}

// Her türlü değeri float64'e çevirir
func toFloat(val interface{}) float64 {
	switch v := val.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	}
	return 0
}

// HTML çıktısını minify eder (gereksiz boşlukları temizler)
func MinifyHTML(s string) string {
	lines := strings.Split(s, "\n")
	var clean []string
	lastWasEmpty := false
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " \t\r")
		if strings.TrimSpace(trimmed) == "" {
			if !lastWasEmpty {
				clean = append(clean, "")
			}
			lastWasEmpty = true
			continue
		}
		clean = append(clean, trimmed)
		lastWasEmpty = false
	}
	return strings.Join(clean, "\n")
}
