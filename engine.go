// engine.go
package hipoengine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Engine, template engine'in ana yapısıdır. Filtre, fonksiyon, cache ve context yönetimini içerir.
type Engine struct {
	filters   map[string]FilterFunc
	funcs     map[string]Function
	cache     map[string]ASTNode        // template cache (ana template için)
	fileCache map[string]fileCacheEntry // dosya içeriği cache
	cacheMu   sync.RWMutex              // cache için mutex

	templatePaths   []string
	templateAliases map[string]string

	contextProcessors []func(map[string]interface{})
	globalContext     map[string]interface{}

	translations map[string]interface{} // i18n: lang -> key -> value (string veya map)
	lang         string                 // aktif dil
	fallbackLang string                 // fallback dil

	lastContext map[string]interface{} // lastContext, son render edilen context'i tutar (internal)

	// Yeni alanlar
	StrictMode     bool
	SafeMode       bool
	AllowedFilters map[string]bool
	AllowedFuncs   map[string]bool
	AllowedVars    map[string]bool
	DebugMode      bool
	DebugLogger    func(msg string)
	currentLocale  string // dinamik dil için

	Profiler    *Profiler
	LastTrace   *RenderTrace
	AuditLogger AuditLogFunc
}

type fileCacheEntry struct {
	content string
	modTime int64
}

func NewEngine() *Engine {
	filters := make(map[string]FilterFunc)
	for k, v := range DefaultFilters {
		filters[k] = v
	}
	e := &Engine{
		filters:           filters,
		funcs:             make(map[string]Function),
		cache:             make(map[string]ASTNode),
		fileCache:         make(map[string]fileCacheEntry),
		templatePaths:     []string{"."},
		templateAliases:   make(map[string]string),
		contextProcessors: nil,
		globalContext:     make(map[string]interface{}),
		translations:      nil,
		lang:              "en",
		StrictMode:        false,
		SafeMode:          false,
		AllowedFilters:    nil,
		AllowedFuncs:      nil,
		AllowedVars:       nil,
		DebugMode:         false,
		DebugLogger:       nil,
		currentLocale:     "",
		Profiler:          NewProfiler(),
		LastTrace:         nil,
		AuditLogger:       nil,
	}
	// trans fonksiyonunu closure olarak kaydet
	e.RegisterFunction("trans", func(args ...interface{}) interface{} {
		var ctx map[string]interface{}
		if len(args) > 1 {
			if m, ok := args[len(args)-1].(map[string]interface{}); ok {
				ctx = m
				args = args[:len(args)-1]
			}
		}
		return e.transFuncWithContext(ctx, args...)
	})
	return e
}

// RegisterFilter, yeni bir filtre fonksiyonu kaydeder.
func (e *Engine) RegisterFilter(name string, filter FilterFunc) {
	if _, exists := e.filters[name]; exists {
		_, err := fmt.Fprintf(os.Stderr, "[hipoengine] Uyarı: '%s' isimli filtre zaten kayıtlı, üzerine yazılıyor.\n", name)
		if err != nil {
			return
		}
	}
	e.filters[name] = filter
}

// RegisterFunction, yeni bir fonksiyon kaydeder.
func (e *Engine) RegisterFunction(name string, fn Function) {
	if _, exists := e.funcs[name]; exists {
		fmt.Fprintf(os.Stderr, "[hipoengine] Uyarı: '%s' isimli fonksiyon zaten kayıtlı, üzerine yazılıyor.\n", name)
	}
	e.funcs[name] = fn
}

func (e *Engine) ParseFile(filename string) (ASTNode, error) {
	resolved, err := e.resolveTemplatePath(filename)
	if err != nil {
		return nil, err
	}
	e.cacheMu.RLock()
	if ast, ok := e.cache[resolved]; ok {
		e.cacheMu.RUnlock()
		return ast, nil
	}
	e.cacheMu.RUnlock()
	// Dosyayı oku ve parse et
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	parser := NewParser(string(data))
	ast, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	e.cacheMu.Lock()
	e.cache[resolved] = ast
	e.cacheMu.Unlock()
	return ast, nil
}

// Dosya içeriğini thread-safe cache'le
func (e *Engine) ReadFileCached(filename string) (string, error) {
	resolved, err := e.resolveTemplatePath(filename)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	modTime := stat.ModTime().UnixNano()
	e.cacheMu.RLock()
	if entry, ok := e.fileCache[resolved]; ok {
		e.cacheMu.RUnlock()
		if entry.modTime == modTime {
			return entry.content, nil
		}
		// Modifiye olmuş, cache'i temizle
		e.cacheMu.Lock()
		delete(e.fileCache, resolved)
		e.cacheMu.Unlock()
	} else {
		e.cacheMu.RUnlock()
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	e.cacheMu.Lock()
	e.fileCache[resolved] = fileCacheEntry{content: string(data), modTime: modTime}
	e.cacheMu.Unlock()
	return string(data), nil
}

func (e *Engine) mergeContext(ctx map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range e.globalContext {
		merged[k] = v
	}
	for k, v := range ctx {
		merged[k] = v
	}
	for _, proc := range e.contextProcessors {
		proc(merged)
	}
	return merged
}

// Render, verilen template stringini ve context'i render eder.
func (e *Engine) Render(template string, ctx map[string]interface{}) (string, error) {
	e.lastContext = ctx
	ctx = e.mergeContext(ctx)
	parser := NewParser(template)
	ast, err := parser.Parse()
	if err != nil {
		return "", err
	}
	context := NewContext(ctx, e.funcs, e.filters, e)

	start := time.Now()
	result, err := ast.Execute(context)
	dur := time.Since(start)
	if e.Profiler != nil {
		e.Profiler.Add("Render", "template", dur)
	}
	// Trace
	user := "anonymous"
	if u, ok := ctx["user"].(string); ok {
		user = u
	}
	trace := &RenderTrace{
		Templates:      []string{"inline"},
		ContextSummary: fmt.Sprintf("%#v", ctx),
		StartTime:      start,
		EndTime:        time.Now(),
	}
	e.LastTrace = trace
	if e.AuditLogger != nil {
		e.AuditLogger(user, "inline", trace.ContextSummary, dur, err == nil, err)
	}
	if err != nil {
		return "", err
	}
	return MinifyHTML(result), nil // otomatik minify!
}

// RenderWithLayout, view ve layout dosyalarını birleştirerek render eder.
func (e *Engine) RenderWithLayout(viewFile string, layoutFile string, ctx map[string]interface{}) (string, error) {
	ctx = e.mergeContext(ctx)
	viewContent, err := e.ReadFileCached(viewFile)
	if err != nil {
		return "", fmt.Errorf("View dosyası okunamadı: %w", err)
	}
	viewBlocks := SplitBlocks(viewContent)
	viewTpl := viewBlocks.Template
	viewBlockMap, err := ParseBlocks(viewTpl)
	if err != nil {
		return "", fmt.Errorf("View block parse hatası: %w", err)
	}
	layoutContent, err := e.ReadFileCached(layoutFile)
	if err != nil {
		return "", fmt.Errorf("Layout dosyası okunamadı: %w", err)
	}
	layoutBlocks := SplitBlocks(layoutContent)
	layoutTpl := layoutBlocks.Template
	layoutTpl = strings.Replace(layoutTpl, "{{ embed }}", viewTpl, 1)
	ast, err := NewParser(layoutTpl).ParseWithBlocks(viewBlockMap)
	if err != nil {
		return "", fmt.Errorf("Layout parse hatası: %w", err)
	}
	context := NewContext(ctx, e.funcs, e.filters, e)

	start := time.Now()
	html, err := ast.Execute(context)
	dur := time.Since(start)
	if e.Profiler != nil {
		e.Profiler.Add(layoutFile, "template", dur)
	}
	user := "anonymous"
	if u, ok := ctx["user"].(string); ok {
		user = u
	}
	trace := &RenderTrace{
		Templates:      []string{layoutFile, viewFile},
		ContextSummary: fmt.Sprintf("%#v", ctx),
		StartTime:      start,
		EndTime:        time.Now(),
	}
	e.LastTrace = trace
	if e.AuditLogger != nil {
		e.AuditLogger(user, layoutFile+"|"+viewFile, trace.ContextSummary, dur, err == nil, err)
	}
	if err != nil {
		return "", fmt.Errorf("Layout render hatası: %w", err)
	}
	finalScript := strings.TrimSpace(layoutBlocks.Script + "\n" + viewBlocks.Script)
	finalStyle := strings.TrimSpace(layoutBlocks.Style + "\n" + viewBlocks.Style)
	output := ""
	if finalScript != "" {
		output += "<script>\n" + finalScript + "\n</script>\n"
	}
	output += html
	if finalStyle != "" {
		output += "\n<style>\n" + finalStyle + "\n</style>"
	}
	return output, nil
}

// RenderFile, verilen dosya adını ve context'i render eder.
func (e *Engine) RenderFile(filename string, ctx map[string]interface{}) (string, error) {
	start := time.Now()
	content, err := e.ReadFileCached(filename)
	if err != nil {
		return "", err
	}
	blocks := SplitBlocks(content)
	html := ""
	if blocks.Template != "" {
		html, err = e.Render(blocks.Template, ctx)
		if err != nil {
			return "", err
		}
	}
	out := ""
	if blocks.Script != "" {
		out += "<script>\n" + blocks.Script + "\n</script>\n"
	}
	out += html
	if blocks.Style != "" {
		out += "\n<style>\n" + blocks.Style + "\n</style>"
	}
	dur := time.Since(start)
	if e.Profiler != nil {
		e.Profiler.Add(filename, "template", dur)
	}
	user := "anonymous"
	if u, ok := ctx["user"].(string); ok {
		user = u
	}
	trace := &RenderTrace{
		Templates:      []string{filename},
		ContextSummary: fmt.Sprintf("%#v", ctx),
		StartTime:      start,
		EndTime:        time.Now(),
	}
	e.LastTrace = trace
	if e.AuditLogger != nil {
		e.AuditLogger(user, filename, trace.ContextSummary, dur, err == nil, err)
	}
	return MinifyHTML(out), nil
}

// RenderFileContext, zincirli context ile dosya render eder.
func (e *Engine) RenderFileContext(filename string, ctx *Context) (string, error) {
	start := time.Now()
	content, err := e.ReadFileCached(filename)
	if err != nil {
		return "", err
	}
	blocks := SplitBlocks(content)
	html := ""
	if blocks.Template != "" {
		parser := NewParser(blocks.Template)
		ast, err := parser.Parse()
		if err != nil {
			return "", err
		}
		result, err := ast.Execute(ctx)
		if err != nil {
			return "", err
		}
		html = result
	}
	out := ""
	if blocks.Script != "" {
		out += "<script>\n" + blocks.Script + "\n</script>\n"
	}
	out += html
	if blocks.Style != "" {
		out += "\n<style>\n" + blocks.Style + "\n</style>"
	}
	dur := time.Since(start)
	if e.Profiler != nil {
		e.Profiler.Add(filename, "template", dur)
	}
	user := "anonymous"
	if ctx != nil && ctx.data != nil {
		if u, ok := ctx.data["user"].(string); ok {
			user = u
		}
	}
	trace := &RenderTrace{
		Templates:      []string{filename},
		ContextSummary: fmt.Sprintf("%#v", ctx.data),
		StartTime:      start,
		EndTime:        time.Now(),
	}
	if ctx != nil && ctx.engine != nil {
		ctx.engine.LastTrace = trace
		if ctx.engine.AuditLogger != nil {
			ctx.engine.AuditLogger(user, filename, trace.ContextSummary, dur, err == nil, err)
		}
	}
	return MinifyHTML(out), nil
}

// Template dosya yolunu alias ve arama yollarına göre çözer
func (e *Engine) resolveTemplatePath(name string) (string, error) {
	// Alias kontrolü
	if real, ok := e.templateAliases[name]; ok {
		return real, nil
	}
	// Doğrudan dosya mevcutsa
	if _, err := os.Stat(name); err == nil {
		return name, nil
	}
	// Arama yollarında sırayla dene
	for _, dir := range e.templatePaths {
		full := dir + string(os.PathSeparator) + name
		if _, err := os.Stat(full); err == nil {
			return full, nil
		}
	}
	return "", fmt.Errorf("Template bulunamadı: %s", name)
}

// Template arama yolu ekle
func (e *Engine) AddTemplatePath(path string) {
	e.templatePaths = append(e.templatePaths, path)
}

// Template alias ekle
func (e *Engine) SetTemplateAlias(alias, realPath string) {
	e.templateAliases[alias] = realPath
}

// Context processor ekle
func (e *Engine) AddContextProcessor(proc func(map[string]interface{})) {
	e.contextProcessors = append(e.contextProcessors, proc)
}

// Global context ayarla
func (e *Engine) SetGlobalContext(ctx map[string]interface{}) {
	e.globalContext = ctx
}

// SetFallbackLang, fallback dili ayarlar.
func (e *Engine) SetFallbackLang(lang string) {
	e.fallbackLang = lang
}

// SetTranslations, engine'e çeviri map'ini ekler. Artık value interface{} (string veya map).
func (e *Engine) SetTranslations(trans map[string]interface{}) {
	e.translations = trans
}

// SetTranslationsFromDir, verilen klasördeki tüm .json dosyalarını okuyup translations map'ine ekler.
func (e *Engine) SetTranslationsFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	if e.translations == nil {
		e.translations = make(map[string]interface{})
	}
	for _, file := range files {
		lang := filepath.Base(file)
		lang = lang[:len(lang)-len(filepath.Ext(lang))] // dosya adından .json'ı çıkar
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		e.translations[lang] = m
	}
	return nil
}

// lookupNamespace: sadece iç içe map olarak (noktalı anahtarları parçalayıp) arar
func lookupNamespace(m map[string]interface{}, key string) (interface{}, bool) {
	parts := strings.Split(key, ".")
	cur := m
	for i, part := range parts {
		v, ok := cur[part]
		if !ok {
			return nil, false
		}
		if i == len(parts)-1 {
			return v, true
		}
		if next, ok := v.(map[string]interface{}); ok {
			cur = next
		} else {
			return nil, false
		}
	}
	return nil, false
}

// lookupWithFallback: verilen dillerde, map[string]interface{} içinde anahtarı arar
func lookupWithFallback(trans map[string]interface{}, langs []string, key string) (interface{}, bool) {
	for _, lang := range langs {
		if m, ok := trans[lang].(map[string]interface{}); ok {
			if val, found := lookupNamespace(m, key); found {
				return val, true
			}
		}
	}
	return nil, false
}
func selectPluralForm(v map[string]interface{}, c int) string {
	if c == 0 {
		if _, ok := v["zero"]; ok {
			return "zero"
		}
	}
	if c == 1 {
		if _, ok := v["one"]; ok {
			return "one"
		}
	}
	// Genişletmek istersen buraya diğer formları ekle
	if _, ok := v["other"]; ok {
		return "other"
	}
	// Hiçbiri yoksa, ilk bulduğunu döndür (hata önleme)
	for k := range v {
		return k
	}
	return "other"
}

func isLangCode(s string) bool {
	// 2 harf veya tr-TR gibi 5 harf (xx-XX)
	return len(s) == 2 || (len(s) == 5 && s[2] == '-')
}

var EnableI18nDebug = false

// trans fonksiyonu: çeviri anahtarını aktif dile göre döndürür, context opsiyoneldir, pluralization ve fallback destekler
func (e *Engine) transFuncWithContext(ctx map[string]interface{}, args ...interface{}) interface{} {
	if len(args) == 0 {
		return ""
	}
	key := fmt.Sprintf("%v", args[0])

	// Locale önceliği: context > engine.currentLocale > engine.lang
	lang := e.lang
	if e.currentLocale != "" {
		lang = e.currentLocale
	}
	if ctx != nil {
		if l, ok := ctx["locale"].(string); ok && l != "" {
			lang = l
		}
	}

	var ctxArg map[string]interface{} = ctx
	var count interface{} = nil

	for _, arg := range args[1:] {
		if s, ok := arg.(string); ok && isLangCode(s) {
			lang = s
		} else if m, ok := arg.(map[string]interface{}); ok {
			ctxArg = m
		} else if count == nil && isNumeric(fmt.Sprintf("%v", arg)) {
			count = arg
		}
	}

	langsToTry := []string{lang}
	if e.fallbackLang != "" && e.fallbackLang != lang {
		langsToTry = append(langsToTry, e.fallbackLang)
	}

	if e.translations != nil {
		if EnableI18nDebug {
			fmt.Fprintf(os.Stderr, "[i18n-debug] langsToTry=%#v key=%s translations=%#v\n", langsToTry, key, e.translations)
		}
		val, found := lookupWithFallback(e.translations, langsToTry, key)
		if EnableI18nDebug {
			fmt.Fprintf(os.Stderr, "[i18n-debug] key=%s lang=%s count=%v found=%v val=%#v\n", key, lang, count, found, val)
		}
		if found {
			switch v := val.(type) {
			case string:
				if strings.Contains(v, "{{") {
					if ctxArg == nil {
						ctxArg = e.lastContext
					}
					if ctxArg == nil {
						ctxArg = map[string]interface{}{}
					}
					merged := e.mergeContext(ctxArg)
					out, err := e.Render(v, merged)
					if EnableI18nDebug {
						fmt.Fprintf(os.Stderr, "[i18n-debug] render string: %s => %s\n", v, out)
					}
					if err == nil {
						return out
					}
				}
				return v
			case map[string]interface{}:
				c := 1
				if count != nil {
					switch ct := count.(type) {
					case int:
						c = ct
					case float64:
						c = int(ct)
					case string:
						c, _ = strconv.Atoi(ct)
					}
				}
				form := selectPluralForm(v, c)
				msg, ok := v[form]
				if EnableI18nDebug {
					fmt.Fprintf(os.Stderr, "[i18n-debug] plural form=%s msg=%#v\n", form, msg)
				}
				if ok {
					msgStr := fmt.Sprintf("%v", msg)
					if strings.Contains(msgStr, "{{") {
						if ctxArg == nil {
							ctxArg = e.lastContext
						}
						if ctxArg == nil {
							ctxArg = map[string]interface{}{}
						}
						merged := e.mergeContext(ctxArg)
						merged["count"] = c
						out, err := e.Render(msgStr, merged)
						if EnableI18nDebug {
							fmt.Fprintf(os.Stderr, "[i18n-debug] render plural: %s => %s\n", msgStr, out)
						}
						if err == nil {
							return out
						}
					}
					return msgStr
				}
				return "" // form yoksa boş string dön
			}
		}
	}
	if EnableI18nDebug {
		fmt.Fprintf(os.Stderr, "[i18n-debug] fallback: key=%s\n", key)
	}
	return key // fallback: anahtarın kendisi
}

// SetLang, aktif dili ayarlar.
func (e *Engine) SetLang(lang string) {
	e.lang = lang
}

// Dinamik dil değiştirme
func (e *Engine) SetLocale(locale string) {
	e.currentLocale = locale
}

func (ctx *Context) SetLocale(locale string) {
	ctx.CurrentLocale = locale
}

// Strict mode
func (e *Engine) SetStrictMode(strict bool) {
	e.StrictMode = strict
}
func (ctx *Context) SetStrictMode(strict bool) {
	ctx.StrictMode = strict
}

// Safe mode
func (e *Engine) SetSafeMode(safe bool) {
	e.SafeMode = safe
}
func (ctx *Context) SetSafeMode(safe bool) {
	ctx.SafeMode = safe
}

// Debug mode
func (e *Engine) SetDebugMode(debug bool) {
	e.DebugMode = debug
}
func (ctx *Context) SetDebugMode(debug bool) {
	ctx.DebugMode = debug
}

// Debug logger
func (e *Engine) SetDebugLogger(logger func(msg string)) {
	e.DebugLogger = logger
}
func (ctx *Context) SetDebugLogger(logger func(msg string)) {
	ctx.DebugLogger = logger
}

// Allowed filters
func (e *Engine) SetAllowedFilters(filters []string) {
	m := make(map[string]bool)
	for _, f := range filters {
		m[f] = true
	}
	e.AllowedFilters = m
}
func (ctx *Context) SetAllowedFilters(filters []string) {
	m := make(map[string]bool)
	for _, f := range filters {
		m[f] = true
	}
	ctx.AllowedFilters = m
}

// Allowed funcs
func (e *Engine) SetAllowedFuncs(funcs []string) {
	m := make(map[string]bool)
	for _, f := range funcs {
		m[f] = true
	}
	e.AllowedFuncs = m
}
func (ctx *Context) SetAllowedFuncs(funcs []string) {
	m := make(map[string]bool)
	for _, f := range funcs {
		m[f] = true
	}
	ctx.AllowedFuncs = m
}

// Allowed vars
func (e *Engine) SetAllowedVars(vars []string) {
	m := make(map[string]bool)
	for _, v := range vars {
		m[v] = true
	}
	e.AllowedVars = m
}
func (ctx *Context) SetAllowedVars(vars []string) {
	m := make(map[string]bool)
	for _, v := range vars {
		m[v] = true
	}
	ctx.AllowedVars = m
}
