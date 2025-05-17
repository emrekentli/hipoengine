package hipoengine

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ASTNode, tüm node türlerinin implement ettiği arayüzdür.
type ASTNode interface {
	Execute(ctx *Context) (string, error)
	ExecuteRaw(ctx *Context) (interface{}, error)
}

// TextNode, düz metin node'u.
type TextNode struct {
	Text string
}

// Execute, TextNode'u string olarak döndürür.
func (n *TextNode) Execute(ctx *Context) (string, error) {
	return n.Text, nil
}

func (n *TextNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	s := n.Text
	s = strings.TrimSpace(s)
	if len(s) > 1 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1], nil
	}
	return n.Text, nil
}

// VariableNode, değişken ve filtre zinciri node'u.
type VariableNode struct {
	Name    string
	Value   interface{}
	Filters []FilterCall
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

// Execute, VariableNode'u string olarak render eder (HTML escape ve |safe filtresi uygular).
func (n *VariableNode) Execute(ctx *Context) (string, error) {
	val, _ := n.ExecuteRaw(ctx)
	safe := false
	for _, filter := range n.Filters {
		if filter.Name == "safe" {
			safe = true
		}
	}
	if val == nil {
		return "", nil
	}
	if _, ok := val.(map[string]interface{}); ok {
		return "", nil
	}
	str := fmt.Sprintf("%v", val)
	if !safe {
		str = htmlEscape(str)
	}
	return str, nil
}

// parseFilterArgs, filtre argümanlarını uygun tipe çevirir.
func parseFilterArgs(args []string) []interface{} {
	res := make([]interface{}, len(args))
	for i, arg := range args {
		if len(arg) > 1 && arg[0] == '"' && arg[len(arg)-1] == '"' {
			res[i] = arg[1 : len(arg)-1]
		} else if ival, err := strconv.Atoi(arg); err == nil {
			res[i] = ival
		} else if fval, err := strconv.ParseFloat(arg, 64); err == nil {
			res[i] = fval
		} else {
			res[i] = arg
		}
	}
	return res
}

// ExecuteRaw, VariableNode'un değerini filtrelerle birlikte döndürür.
func (n *VariableNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	var val interface{}

	// Eğer Name fonksiyon çağrısı ise (ör: trans("cart.items", count))
	if n.Name != "" && strings.Contains(n.Name, "(") && strings.HasSuffix(n.Name, ")") {
		openIdx := strings.Index(n.Name, "(")
		funcName := strings.TrimSpace(n.Name[:openIdx])
		argsStr := strings.TrimSpace(n.Name[openIdx+1 : len(n.Name)-1])
		var args []interface{}
		if argsStr != "" {
			parts := splitArgs(argsStr)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if (strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"")) || (strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'")) {
					args = append(args, part[1:len(part)-1])
				} else {
					resolved := ctx.Resolve(part)
					args = append(args, resolved)
				}
			}
		}
		if fn, ok := ctx.funcs[funcName]; ok {
			var fnResult interface{}
			if ctx.engine != nil && ctx.engine.Profiler != nil {
				start := time.Now()
				fnResult = fn(args...)
				dur := time.Since(start)
				ctx.engine.Profiler.Add(funcName, "function", dur)
			} else {
				fnResult = fn(args...)
			}
			val = fnResult
		} else {
			return "", nil
		}
	} else if n.Value != nil {
		if s, ok := n.Value.(string); ok && strings.Contains(s, "(") {
			val = ctx.Resolve(s)
		} else {
			val = n.Value
		}
		for _, filter := range n.Filters {
			if fn, ok := ctx.filters[filter.Name]; ok {
				if ctx.engine != nil && ctx.engine.Profiler != nil {
					start := time.Now()
					val = fn(val, parseFilterArgs(filter.Args)...)
					dur := time.Since(start)
					ctx.engine.Profiler.Add(filter.Name, "filter", dur)
				} else {
					val = fn(val, parseFilterArgs(filter.Args)...)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[hipoengine] Uyarı: '%s' isimli filtre bulunamadı.\n", filter.Name)
				val = fmt.Sprintf("%v [filter %s not found]", val, filter.Name)
			}
		}
		if _, ok := val.(map[string]interface{}); ok && strings.Contains(n.Name, "(") {
			return "", nil
		}
		return val, nil
	} else if n.Name == "" && n.Value == nil {
		val = ctx.Resolve("")
	} else {
		val = ctx.Resolve(n.Name)
	}
	for _, filter := range n.Filters {
		if fn, ok := ctx.filters[filter.Name]; ok {
			if ctx.engine != nil && ctx.engine.Profiler != nil {
				start := time.Now()
				val = fn(val, parseFilterArgs(filter.Args)...)
				dur := time.Since(start)
				ctx.engine.Profiler.Add(filter.Name, "filter", dur)
			} else {
				val = fn(val, parseFilterArgs(filter.Args)...)
			}
		} else {
			fmt.Fprintf(os.Stderr, "[hipoengine] Uyarı: '%s' isimli filtre bulunamadı.\n", filter.Name)
			val = fmt.Sprintf("%v [filter %s not found]", val, filter.Name)
		}
	}
	return val, nil
}

// splitArgs, fonksiyon çağrısı argümanlarını virgülden ayırır, tırnak içindeki virgülleri hesaba katar.
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c == '"' || c == '\'') && (i == 0 || s[i-1] != '\\') {
			if inQuotes && c == quoteChar {
				inQuotes = false
			} else if !inQuotes {
				inQuotes = true
				quoteChar = c
			}
			current.WriteByte(c)
		} else if c == ',' && !inQuotes {
			args = append(args, current.String())
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// ListNode, birden fazla node'u sıralı tutar.
type ListNode struct {
	Nodes []ASTNode
}

// Execute, ListNode altındaki tüm node'ları sıralı render eder.
func (n *ListNode) Execute(ctx *Context) (string, error) {
	var sb strings.Builder
	for _, node := range n.Nodes {
		out, err := node.Execute(ctx)
		if err != nil {
			return "", err
		}
		sb.WriteString(out)
	}
	return sb.String(), nil
}

func (n *ListNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	return n.Execute(ctx)
}

// IfBranch, if/elif bloğunun koşulu ve gövdesi.
type IfBranch struct {
	Condition string
	Body      ASTNode
}

// IfNode, if, elif, else bloklarını tutar.
type IfNode struct {
	Branches []IfBranch
	ElseBody ASTNode
}

// Execute, IfNode'un koşullarını değerlendirip uygun gövdeyi render eder.
func (n *IfNode) Execute(ctx *Context) (string, error) {
	for _, branch := range n.Branches {
		if evalBool(branch.Condition, ctx) {
			return branch.Body.Execute(ctx)
		}
	}
	if n.ElseBody != nil {
		return n.ElseBody.Execute(ctx)
	}
	return "", nil
}

func (n *IfNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	return n.Execute(ctx)
}

// ForNode, for döngüsü node'u.
type ForNode struct {
	VarName    string
	Collection string
	Body       ASTNode
}

// Execute, ForNode'un koleksiyonunu döngüyle render eder.
func (n *ForNode) Execute(ctx *Context) (string, error) {
	col := ctx.Resolve(n.Collection)
	arr, ok := col.([]interface{})
	if !ok {
		return "", fmt.Errorf("ForNode: '%s' koleksiyonu []interface{} tipinde değil, değer: %v", n.Collection, col)
	}
	var sb strings.Builder
	for i, item := range arr {
		if m := toStringMap(item); m != nil {
			child := ctx.NewChild(map[string]interface{}{n.VarName: m})
			out, err := n.Body.Execute(child)
			if err != nil {
				return "", err
			}
			sb.WriteString(out)
		} else {
			child := ctx.NewChild(map[string]interface{}{n.VarName: item})
			out, err := n.Body.Execute(child)
			if err != nil {
				return "", err
			}
			sb.WriteString(out)
		}
		if i != len(arr)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

func (n *ForNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	return n.Execute(ctx)
}

// WithNode, with bloğu (alias atanarak yeni context oluşturur).
type WithNode struct {
	Expr  string
	Alias string
	Body  ASTNode
}

// Execute, WithNode'un alias ile yeni context oluşturup gövdeyi render eder.
func (n *WithNode) Execute(ctx *Context) (string, error) {
	val := ctx.Resolve(n.Expr)
	child := ctx.NewChild(map[string]interface{}{n.Alias: val})
	return n.Body.Execute(child)
}

func (n *WithNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	return n.Execute(ctx)
}

// IncludeNode, dosya içeriğini include eder.
type IncludeNode struct {
	File string
}

// Execute, IncludeNode'un dosyasını zincirli context ile render eder.
func (n *IncludeNode) Execute(ctx *Context) (string, error) {
	if ctx.engine == nil {
		return "", fmt.Errorf("engine not set in context for include")
	}
	child := ctx.NewChild(ctx.data)
	return ctx.engine.RenderFileContext(n.File, child)
}

func (n *IncludeNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	return n.Execute(ctx)
}

// BlockNode, override edilebilir blok node'u.
type BlockNode struct {
	Name string
	Body ASTNode
}

// Execute, BlockNode'un gövdesini yeni bir child context ile render eder.
func (n *BlockNode) Execute(ctx *Context) (string, error) {
	return n.Body.Execute(ctx.NewChild(nil))
}

func (n *BlockNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	return n.Execute(ctx)
}

// ExtendsNode, extends mekanizması, base dosyayı ve override blokları tutar.
type ExtendsNode struct {
	BaseFile string
	Blocks   map[string]ASTNode
}

// Execute, ExtendsNode'un base dosyasını ve override bloklarını render eder.
func (n *ExtendsNode) Execute(ctx *Context) (string, error) {
	if ctx.engine == nil {
		return "", fmt.Errorf("engine not set in context for extends")
	}
	baseContent, err := ctx.engine.ReadFileCached(n.BaseFile)
	if err != nil {
		return "", err
	}
	tpl := extractTemplateBlock(baseContent)
	baseParser := NewParser(tpl)
	baseAst, err := baseParser.ParseWithBlocks(n.Blocks)
	if err != nil {
		return "", err
	}
	return baseAst.Execute(ctx)
}

func (n *ExtendsNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	return n.Execute(ctx)
}

// SetNode, template içinde değişken atama node'u.
type SetNode struct {
	VarName string
	Value   ASTNode
}

// Execute, SetNode'un değerini çalıştırıp context'e ekler.
func (n *SetNode) Execute(ctx *Context) (string, error) {
	val, err := n.Value.ExecuteRaw(ctx)
	if err != nil {
		return "", err
	}
	ctx.data[n.VarName] = val
	return "", nil
}

func (n *SetNode) ExecuteRaw(ctx *Context) (interface{}, error) {
	_, err := n.Execute(ctx)
	return nil, err
}
