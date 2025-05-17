package hipoengine

import (
	"reflect"
	"strconv"
	"strings"
)

// Context, template çalıştırılırken değişkenlerin, fonksiyonların ve filtrelerin tutulduğu yapıdır. Scope zinciri için parent referansı içerir.
type Context struct {
	data    map[string]interface{} // template verileri
	funcs   map[string]Function    // fonksiyonlar
	filters map[string]FilterFunc  // filtreler
	parent  *Context               // opsiyonel ebeveyn context (scoping için)
	engine  *Engine                // engine referansı (include vb için)

	CurrentLocale  string
	StrictMode     bool
	SafeMode       bool
	AllowedFilters map[string]bool
	AllowedFuncs   map[string]bool
	AllowedVars    map[string]bool
	DebugMode      bool
	DebugLogger    func(msg string)
}

// NewContext, yeni bir context oluşturur.
func NewContext(data map[string]interface{}, funcs map[string]Function, filters map[string]FilterFunc, engine *Engine) *Context {
	return &Context{
		data:           data,
		funcs:          funcs,
		filters:        filters,
		parent:         nil,
		engine:         engine,
		CurrentLocale:  "",
		StrictMode:     false,
		SafeMode:       false,
		AllowedFilters: nil,
		AllowedFuncs:   nil,
		AllowedVars:    nil,
		DebugMode:      false,
		DebugLogger:    nil,
	}
}

// NewChild, mevcut context'in parent'ı olarak zincirli yeni bir context oluşturur.
func (ctx *Context) NewChild(data map[string]interface{}) *Context {
	return &Context{
		data:           data,
		funcs:          ctx.funcs,
		filters:        ctx.filters,
		parent:         ctx,
		engine:         ctx.engine,
		CurrentLocale:  ctx.CurrentLocale,
		StrictMode:     ctx.StrictMode,
		SafeMode:       ctx.SafeMode,
		AllowedFilters: ctx.AllowedFilters,
		AllowedFuncs:   ctx.AllowedFuncs,
		AllowedVars:    ctx.AllowedVars,
		DebugMode:      ctx.DebugMode,
		DebugLogger:    ctx.DebugLogger,
	}
}

// Resolve, verilen path'e göre context zincirinde değişken/fonksiyon/filtre arar ve döndürür.
func (ctx *Context) Resolve(path string) interface{} {
	if path == "" {
		return nil
	}
	parts := splitPathWithBrackets(path)
	return ctx.resolveParts(parts)
}

// resolveParts, path parçalarını recursive olarak çözer.
func (ctx *Context) resolveParts(parts []string) interface{} {
	if len(parts) == 0 {
		return nil
	}
	// Fonksiyon çağrısı
	if strings.HasSuffix(parts[0], ")") {
		openIdx := strings.Index(parts[0], "(")
		if openIdx != -1 {
			funcName := parts[0][:openIdx]
			argsStr := parts[0][openIdx+1 : len(parts[0])-1]
			var args []interface{}
			if argsStr != "" {
				for _, arg := range strings.Split(argsStr, ",") {
					a := strings.TrimSpace(arg)
					if a == "ctx" || a == "." {
						args = append(args, ctx.data)
					} else if i, err := strconv.Atoi(a); err == nil {
						args = append(args, i)
					} else if f, err := strconv.ParseFloat(a, 64); err == nil {
						args = append(args, f)
					} else if (strings.HasPrefix(a, "\"") && strings.HasSuffix(a, "\"")) || (strings.HasPrefix(a, "'") && strings.HasSuffix(a, "'")) {
						args = append(args, a[1:len(a)-1])
					} else if val, ok := ctx.data[a]; ok {
						args = append(args, val)
					} else {
						// Değişken adı yoksa string olarak ekle
						args = append(args, a)
					}
				}
			}
			current := ctx
			for current != nil {
				if current.funcs != nil {
					if fn, ok := current.funcs[funcName]; ok {
						val := fn(args...)
						if len(parts) == 1 {
							return val
						}
						return resolveValue(val, parts[1:])
					}
				}
				current = current.parent
			}
			return nil
		}
	}
	// Map veya context zinciri
	current := ctx
	for current != nil {
		if val, ok := current.data[parts[0]]; ok {
			if len(parts) == 1 {
				return val
			}
			return resolveValue(val, parts[1:])
		}
		current = current.parent
	}
	return ""
}

// resolveValue, map/slice/struct üzerinde zincirli erişim sağlar.
func resolveValue(val interface{}, parts []string) interface{} {
	if len(parts) == 0 || val == nil {
		return val
	}
	switch v := val.(type) {
	case map[string]interface{}:
		return resolveValue(v[parts[0]], parts[1:])
	case []interface{}:
		idx, err := strconv.Atoi(parts[0])
		if err == nil {
			if idx < 0 {
				idx = len(v) + idx // negatif index desteği
			}
			if idx >= 0 && idx < len(v) {
				return resolveValue(v[idx], parts[1:])
			}
		}
	case string:
		// string üzerinde index erişimi
		idx, err := strconv.Atoi(parts[0])
		if err == nil {
			if idx < 0 {
				idx = len(v) + idx
			}
			if idx >= 0 && idx < len(v) {
				return resolveValue(string(v[idx]), parts[1:])
			}
		}
	default:
		// struct alanı erişimi
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			field := rv.FieldByName(parts[0])
			if field.IsValid() {
				return resolveValue(field.Interface(), parts[1:])
			}
		}
	}
	return nil
}

// splitPathWithBrackets, path'i products[0].name veya user["first_name"] gibi parçalara ayırır.
func splitPathWithBrackets(path string) []string {
	var parts []string
	var buf strings.Builder
	inBracket := false
	inQuote := false
	quoteChar := byte(0)
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch c {
		case '.':
			if !inBracket && !inQuote {
				if buf.Len() > 0 {
					parts = append(parts, buf.String())
					buf.Reset()
				}
				continue
			}
		case '[':
			if !inQuote {
				if buf.Len() > 0 {
					parts = append(parts, buf.String())
					buf.Reset()
				}
				inBracket = true
				continue
			}
		case ']':
			if inBracket && !inQuote {
				if buf.Len() > 0 {
					// Eğer tırnaklı ise tırnakları kaldır
					s := buf.String()
					if len(s) > 1 && (s[0] == '"' || s[0] == '\'') && s[0] == s[len(s)-1] {
						s = s[1 : len(s)-1]
					}
					parts = append(parts, s)
					buf.Reset()
				}
				inBracket = false
				continue
			}
		case '"', '\'':
			if inBracket {
				if !inQuote {
					inQuote = true
					quoteChar = c
				} else if c == quoteChar {
					inQuote = false
				}
				continue
			}
		}
		buf.WriteByte(c)
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	return parts
}

// Copy, context verilerini kopyalar (ör: for döngüsünde yeni scope için)
func (ctx *Context) Copy() *Context {
	newData := make(map[string]interface{})
	for k, v := range ctx.data {
		newData[k] = v
	}
	return &Context{
		data:           newData,
		funcs:          ctx.funcs,
		filters:        ctx.filters,
		parent:         ctx.parent,
		engine:         ctx.engine,
		CurrentLocale:  ctx.CurrentLocale,
		StrictMode:     ctx.StrictMode,
		SafeMode:       ctx.SafeMode,
		AllowedFilters: ctx.AllowedFilters,
		AllowedFuncs:   ctx.AllowedFuncs,
		AllowedVars:    ctx.AllowedVars,
		DebugMode:      ctx.DebugMode,
		DebugLogger:    ctx.DebugLogger,
	}
}
