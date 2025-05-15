package hipoengine

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Package hipoengine provides a modern, fast and extensible template engine for Go.

// Engine is the main struct for the template engine.
// It manages filters, hooks, template cache and rendering.
type Engine struct {
	filters map[string]FilterFunc
	hooks   HookFiles

	tplCache   map[string]string
	cacheMutex sync.RWMutex
}

// New creates and returns a new Engine instance.
func New() *Engine {
	return &Engine{
		filters:  DefaultFilters,
		hooks:    HookFiles{},
		tplCache: map[string]string{},
	}
}

// RegisterFilter adds a custom filter to the engine.
func (e *Engine) RegisterFilter(name string, fn FilterFunc) {
	e.filters[name] = fn
}

// RegisterHook registers a hook with a list of template files.
func (e *Engine) RegisterHook(name string, files []string) {
	e.hooks[name] = files
}

// ParseFile reads and caches a template file by filename.
func (e *Engine) ParseFile(filename string) (string, error) {
	e.cacheMutex.RLock()
	if tpl, ok := e.tplCache[filename]; ok {
		e.cacheMutex.RUnlock()
		return tpl, nil
	}
	e.cacheMutex.RUnlock()
	b, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	e.cacheMutex.Lock()
	e.tplCache[filename] = string(b)
	e.cacheMutex.Unlock()
	return string(b), nil
}

// Render renders a template string with the given context.
func (e *Engine) Render(template string, context map[string]interface{}) (string, error) {
	return e.renderInternal(template, context, nil)
}

// --- Internal ---
var (
	includeRe = regexp.MustCompile(`{%\s*include\s+"([^"]+)"(\s+with\s+\{([^}]*)\})?\s*%}`)
	varRe     = regexp.MustCompile(`{{\s*([\w\.]+)(?:\s*\|\s*(\w+))?\s*}}`)
	ifRe      = regexp.MustCompile(`{%\s*if\s+([\w\s><=!\'\"]+)\s*%}([\s\S]*?)(?:{%\s*else\s*%}([\s\S]*?))?{%\s*endif\s*%}`)
	forRe     = regexp.MustCompile(`{%\s*for\s+(\w+)(?:,\s*(\w+))?\s+in\s+(\w+)\s*%}([\s\S]*?){%\s*endfor\s*%}`)
	hookRe    = regexp.MustCompile(`{%\s*hook\s+"([^"]+)"\s*%}`)
	setRe     = regexp.MustCompile(`{%\s*set\s+(\w+)\s*=\s*['"]?([^%]+?)['"]?\s*%}`)
	commentRe = regexp.MustCompile(`{#.*?#}`)
	rawRe     = regexp.MustCompile(`{%-?\s*raw\s*-?%}([\s\S]*?){%-?\s*endraw\s*-?%}`)
	blockRe   = regexp.MustCompile(`{%-?\s*block\s+(\w+)-?%}([\s\S]*?){%-?\s*endblock\s*-?%}`)
	extendsRe = regexp.MustCompile(`{%-?\s*extends\s+"([^"]+)"\s*-?%}`)
)

func (e *Engine) renderInternal(template string, context map[string]interface{}, parentRawBlocks map[string]string) (string, error) {
	template = commentRe.ReplaceAllString(template, "")

	rawBlocks := map[string]string{}
	if parentRawBlocks != nil {
		for k, v := range parentRawBlocks {
			rawBlocks[k] = v
		}
	}
	template = rawRe.ReplaceAllStringFunc(template, func(match string) string {
		id := fmt.Sprintf("__RAW_BLOCK_%d__", len(rawBlocks))
		groups := rawRe.FindStringSubmatch(match)
		rawBlocks[id] = groups[1]
		return id
	})

	extendsMatch := extendsRe.FindStringSubmatch(template)
	if len(extendsMatch) > 0 {
		baseFile := extendsMatch[1]
		baseContent, err := e.ParseFile(baseFile)
		if err == nil {
			blocks := map[string]string{}
			blockMatches := blockRe.FindAllStringSubmatch(template, -1)
			for _, m := range blockMatches {
				blocks[m[1]] = m[2]
			}
			baseTpl := blockRe.ReplaceAllStringFunc(baseContent, func(match string) string {
				bm := blockRe.FindStringSubmatch(match)
				if val, ok := blocks[bm[1]]; ok {
					return val
				}
				return bm[2]
			})
			return e.renderInternal(baseTpl, context, rawBlocks)
		}
	}

	template = setRe.ReplaceAllStringFunc(template, func(match string) string {
		groups := setRe.FindStringSubmatch(match)
		context[groups[1]] = strings.TrimSpace(groups[2])
		return ""
	})

	template = includeRe.ReplaceAllStringFunc(template, func(match string) string {
		groups := includeRe.FindStringSubmatch(match)
		file := groups[1]
		localCtx := make(map[string]interface{})
		for k, v := range context {
			localCtx[k] = v
		}
		if len(groups) > 3 && groups[3] != "" {
			params := strings.Split(groups[3], ",")
			for _, p := range params {
				kv := strings.SplitN(strings.TrimSpace(p), ":", 2)
				if len(kv) == 2 {
					localCtx[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), "'\"")
				}
			}
		}
		content, err := e.ParseFile(file)
		if err != nil {
			return fmt.Sprintf("[include error: %s]", file)
		}
		res, _ := e.renderInternal(content, localCtx, rawBlocks)
		return res
	})

	template = ifRe.ReplaceAllStringFunc(template, func(match string) string {
		groups := ifRe.FindStringSubmatch(match)
		condExpr := groups[1]
		ifTrue := groups[2]
		ifFalse := ""
		if len(groups) > 3 {
			ifFalse = groups[3]
		}
		if evalCondition(condExpr, context) {
			res, _ := e.renderInternal(ifTrue, context, rawBlocks)
			return res
		} else {
			res, _ := e.renderInternal(ifFalse, context, rawBlocks)
			return res
		}
	})

	template = forRe.ReplaceAllStringFunc(template, func(match string) string {
		groups := forRe.FindStringSubmatch(match)
		itemVar := groups[1]
		listVar := groups[3]
		block := groups[4]
		listVal, ok := context[listVar]
		if !ok {
			return ""
		}
		var result strings.Builder
		switch items := listVal.(type) {
		case []interface{}:
			for i, v := range items {
				loopCtx := make(map[string]interface{})
				for k, val := range context {
					loopCtx[k] = val
				}
				loopCtx[itemVar] = v
				loopCtx["loop.index"] = i
				loopCtx["loop.first"] = (i == 0)
				loopCtx["loop.last"] = (i == len(items)-1)
				res, _ := e.renderInternal(block, loopCtx, rawBlocks)
				result.WriteString(res)
			}
		}
		return result.String()
	})

	template = hookRe.ReplaceAllStringFunc(template, func(match string) string {
		hookName := hookRe.FindStringSubmatch(match)[1]
		if snippets, ok := e.hooks[hookName]; ok {
			var result strings.Builder
			for _, snip := range snippets {
				content, err := e.ParseFile(snip)
				if err == nil {
					res, _ := e.renderInternal(content, context, rawBlocks)
					result.WriteString(res)
				}
			}
			return result.String()
		}
		return ""
	})

	var result strings.Builder
	lastIdx := 0
	for _, loc := range varRe.FindAllStringSubmatchIndex(template, -1) {
		result.WriteString(template[lastIdx:loc[0]])
		groups := varRe.FindStringSubmatch(template[loc[0]:loc[1]])
		key := strings.TrimSpace(groups[1])
		val, ok := context[key]
		if !ok {
			val = ""
		}
		if len(groups) > 2 && groups[2] != "" {
			if filterFn, ok := e.filters[groups[2]]; ok {
				result.WriteString(filterFn(val))
				lastIdx = loc[1]
				continue
			}
		}
		result.WriteString(fmt.Sprintf("%v", val))
		lastIdx = loc[1]
	}
	result.WriteString(template[lastIdx:])

	for id, raw := range rawBlocks {
		resultStr := result.String()
		result.Reset()
		result.WriteString(strings.ReplaceAll(resultStr, id, raw))
	}

	return result.String(), nil
}

// --- Basit koşul değerlendirme ---
func evalCondition(expr string, context map[string]interface{}) bool {
	expr = strings.TrimSpace(expr)
	parts := regexp.MustCompile(`\s+`).Split(expr, -1)
	if len(parts) == 1 {
		val, ok := context[parts[0]]
		if !ok {
			return false
		}
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v != "" && v != "0" && v != "false"
		case int:
			return v != 0
		}
		return false
	}
	if len(parts) == 3 {
		left := parts[0]
		op := parts[1]
		right := parts[2]
		lval, lok := context[left]
		if !lok {
			lval = left
		}
		rval := right
		if rv, ok := context[right]; ok {
			switch v := rv.(type) {
			case string:
				rval = v
			case int:
				rval = fmt.Sprintf("%d", v)
			case float64:
				rval = fmt.Sprintf("%f", v)
			default:
				rval = fmt.Sprintf("%v", v)
			}
		}
		lstr := fmt.Sprintf("%v", lval)
		rstr := fmt.Sprintf("%v", rval)
		switch op {
		case "==":
			return lstr == rstr
		case "!=":
			return lstr != rstr
		case ">":
			lf, _ := strconv.ParseFloat(lstr, 64)
			rf, _ := strconv.ParseFloat(rstr, 64)
			return lf > rf
		case "<":
			lf, _ := strconv.ParseFloat(lstr, 64)
			rf, _ := strconv.ParseFloat(rstr, 64)
			return lf < rf
		case ">=":
			lf, _ := strconv.ParseFloat(lstr, 64)
			rf, _ := strconv.ParseFloat(rstr, 64)
			return lf >= rf
		case "<=":
			lf, _ := strconv.ParseFloat(lstr, 64)
			rf, _ := strconv.ParseFloat(rstr, 64)
			return lf <= rf
		}
	}
	return false
}
