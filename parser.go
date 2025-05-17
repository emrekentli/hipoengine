package hipoengine

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser, template stringini ve opsiyonel dosya adını tutar.
type Parser struct {
	template string
	filename string // opsiyonel, hata mesajı için
}

// NewParser, template stringiyle yeni bir parser oluşturur.
func NewParser(template string) *Parser {
	return &Parser{template: template}
}

// NewParserWithFile, template ve dosya adı ile yeni bir parser oluşturur.
func NewParserWithFile(template, filename string) *Parser {
	return &Parser{template: template, filename: filename}
}

// Filtre ve fonksiyon argümanlarını destekleyen yapı
// ör: {{ name|default:"Anonim"|upper }}
type FilterCall struct {
	Name string
	Args []string
}

// VariableNode: {{ variable | filter1 | filter2 }}
// ... existing code ...

// SetNode: {{ set foo = ... }}

// TemplateError, parse hatalarında satır/sütun/dosya adı ve mesajı tutar.
type TemplateError struct {
	File    string
	Line    int
	Column  int
	Message string
}

func (e *TemplateError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("Parse error in %s at line %d, col %d: %s", e.File, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("Parse error at line %d, col %d: %s", e.Line, e.Column, e.Message)
}

// Parse, template'i AST'ye dönüştürür. Hatalı durumda TemplateError döner.
func (p *Parser) Parse() (ASTNode, error) {
	tpl := p.template
	nodes := []ASTNode{}

	trimmed := strings.TrimSpace(tpl)
	// Extends kontrolü (ilk satır)
	if strings.HasPrefix(trimmed, "{{ extends ") {
		endIdx := strings.Index(trimmed, "}}")
		if endIdx == -1 {
			return nil, &TemplateError{File: p.filename, Line: 1, Column: 1, Message: "unclosed extends tag"}
		}
		tag := trimmed[:endIdx+2]
		baseFile := extractExtendsFileName(tag)
		remain := trimmed[endIdx+2:]

		// Child bloklarını parse et
		childBlocks, err := ParseBlocks(remain)
		if err != nil {
			return nil, err
		}

		return &ExtendsNode{BaseFile: baseFile, Blocks: childBlocks}, nil
	}

	// Template içeriğinde {{ ... }} bloklarını ayrıştır
	for len(tpl) > 0 {
		start := strings.Index(tpl, "{{")
		if start == -1 {
			if tpl != "" {
				nodes = append(nodes, &TextNode{Text: tpl})
			}
			break
		}
		// Öncesindeki metni TextNode olarak ekle
		if start > 0 {
			text := tpl[:start]
			if strings.TrimSpace(text) != "" {
				nodes = append(nodes, &TextNode{Text: text})
			}
		}
		end := strings.Index(tpl[start:], "}}")
		if end == -1 {
			line, col := getLineCol(p.template, len(p.template)-len(tpl)+start)
			msg := "unclosed variable or block"
			if p.filename != "" {
				return nil, &TemplateError{File: p.filename, Line: line, Column: col, Message: msg}
			}
			return nil, &TemplateError{File: "", Line: line, Column: col, Message: msg}
		}
		tag := strings.TrimSpace(tpl[start+2 : start+end])

		// SET: {{ set foo = ... }}
		if strings.HasPrefix(tag, "set ") {
			setExpr := strings.TrimSpace(tag[4:])
			eqIdx := strings.Index(setExpr, "=")
			if eqIdx == -1 {
				return nil, fmt.Errorf("set ifadesinde '=' eksik")
			}
			varName := strings.TrimSpace(setExpr[:eqIdx])
			rhs := strings.TrimSpace(setExpr[eqIdx+1:])
			// Sağ tarafı yeni bir parser ile parse et (tek bir node beklenir)
			valAst, err := NewParser(rhs).Parse()
			if err != nil {
				return nil, fmt.Errorf("set ifadesi değeri parse edilemedi: %w", err)
			}
			// Eğer tek bir node ise, doğrudan onu ata
			if list, ok := valAst.(*ListNode); ok && len(list.Nodes) == 1 {
				valAst = list.Nodes[0]
			}
			// Eğer node bir TextNode ise ve içeriği değişken/filtre zinciri içeriyorsa tekrar parse et
			if txt, ok := valAst.(*TextNode); ok {
				content := strings.TrimSpace(txt.Text)
				if content != "" {
					// Eğer content bir literal değilse veya filtre içeriyorsa tekrar parse et
					if strings.Contains(content, "|") || (!((len(content) > 1 && ((content[0] == '"' && content[len(content)-1] == '"') || (content[0] == '\'' && content[len(content)-1] == '\''))) && !isNumeric(content))) {
						vnParser := NewParser("{{ " + content + " }}")
						vnAst, err := vnParser.Parse()
						if err == nil {
							if list, ok := vnAst.(*ListNode); ok && len(list.Nodes) == 1 {
								valAst = list.Nodes[0]
							} else {
								valAst = vnAst
							}
						}
					}
				}
			}
			nodes = append(nodes, &SetNode{VarName: varName, Value: valAst})
			tpl = tpl[start+end+2:]
			continue
		}

		// INCLUDE: {{ include "file" }}
		if strings.HasPrefix(tag, "include ") {
			fname := strings.TrimSpace(strings.Trim(tag[8:], `"'`))
			nodes = append(nodes, &IncludeNode{File: fname})
			tpl = tpl[start+end+2:]
			continue
		}

		// IF BLOCK
		if strings.HasPrefix(tag, "if ") {
			branches := []IfBranch{}
			var elseBody ASTNode
			remain := tpl[start+end+2:]
			cond := strings.TrimSpace(tag[3:])
			var ifBody string

			for {
				elifIdx := strings.Index(remain, "{{ elif ")
				elseIdx := strings.Index(remain, "{{ else }}")
				endifIdx := strings.Index(remain, "{{ endif }}")

				minIdx := -1
				if endifIdx != -1 {
					minIdx = endifIdx
				}
				if elifIdx != -1 && (minIdx == -1 || elifIdx < minIdx) {
					minIdx = elifIdx
				}
				if elseIdx != -1 && (minIdx == -1 || elseIdx < minIdx) {
					minIdx = elseIdx
				}
				if minIdx == -1 {
					return nil, fmt.Errorf("unclosed if/elif/else/endif block")
				}
				ifBody = remain[:minIdx]
				bodyNode, err := NewParser(ifBody).Parse()
				if err != nil {
					return nil, err
				}
				branches = append(branches, IfBranch{Condition: cond, Body: bodyNode})

				if minIdx == elifIdx {
					remain = remain[elifIdx+len("{{ elif "):]
					endElif := strings.Index(remain, "}}")
					if endElif == -1 {
						return nil, fmt.Errorf("unclosed elif tag")
					}
					cond = strings.TrimSpace(remain[:endElif])
					remain = remain[endElif+2:]
					continue
				}

				if minIdx == elseIdx {
					remain = remain[elseIdx+len("{{ else }}"):]
					endifIdx2 := strings.Index(remain, "{{ endif }}")
					if endifIdx2 == -1 {
						return nil, fmt.Errorf("unclosed endif after else")
					}
					elseBodyStr := remain[:endifIdx2]
					elseBody, err = NewParser(strings.TrimSpace(elseBodyStr)).Parse()
					if err != nil {
						return nil, err
					}
					remain = remain[endifIdx2+len("{{ endif }}"):]
					tpl = remain
					nodes = append(nodes, &IfNode{Branches: branches, ElseBody: elseBody})
					break
				}

				if minIdx == endifIdx {
					remain = remain[endifIdx+len("{{ endif }}"):]
					tpl = remain
					nodes = append(nodes, &IfNode{Branches: branches, ElseBody: nil})
					break
				}
			}
			continue
		}

		// FOR BLOCK
		if strings.HasPrefix(tag, "for ") {
			inner := strings.TrimSpace(tag[4:])
			parts := strings.Fields(inner)
			var varName, colName string
			if len(parts) == 2 {
				colName = parts[0]
				varName = parts[1]
			} else if len(parts) == 3 && parts[1] == "in" {
				varName = parts[0]
				colName = parts[2]
			} else {
				return nil, fmt.Errorf("invalid for syntax")
			}
			endforIdx := strings.Index(tpl[start+end+2:], "{{ endfor }}")
			if endforIdx == -1 {
				return nil, fmt.Errorf("unclosed for block")
			}
			bodyTpl := tpl[start+end+2 : start+end+2+endforIdx]
			bodyNode, err := NewParser(bodyTpl).Parse()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &ForNode{VarName: varName, Collection: colName, Body: bodyNode})
			tpl = tpl[start+end+2+endforIdx+len("{{ endfor }}"):]
			continue
		}

		// WITH BLOCK
		if strings.HasPrefix(tag, "with ") {
			inner := strings.TrimSpace(tag[5:])
			parts := strings.Fields(inner)
			var expr, alias string
			if len(parts) == 2 {
				expr = parts[0]
				alias = parts[1]
			} else if len(parts) == 3 && parts[1] == "as" {
				expr = parts[0]
				alias = parts[2]
			} else {
				return nil, fmt.Errorf("invalid with syntax")
			}
			endwithIdx := strings.Index(tpl[start+end+2:], "{{ endwith }}")
			if endwithIdx == -1 {
				return nil, fmt.Errorf("unclosed with block")
			}
			bodyTpl := tpl[start+end+2 : start+end+2+endwithIdx]
			bodyNode, err := NewParser(bodyTpl).Parse()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &WithNode{Expr: expr, Alias: alias, Body: bodyNode})
			tpl = tpl[start+end+2+endwithIdx+len("{{ endwith }}"):]
			continue
		}

		// VARIABLE with filters
		parts := strings.Split(tag, "|")
		varName := strings.TrimSpace(parts[0])
		filters := []FilterCall{}
		for _, f := range parts[1:] {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			fName := f
			fArgs := []string{}
			if idx := strings.Index(f, ":"); idx != -1 {
				fName = strings.TrimSpace(f[:idx])
				argsStr := strings.TrimSpace(f[idx+1:])
				for _, arg := range strings.Split(argsStr, ",") {
					arg = strings.TrimSpace(arg)
					if arg != "" {
						fArgs = append(fArgs, arg)
					}
				}
			}
			filters = append(filters, FilterCall{Name: fName, Args: fArgs})
		}
		// Fonksiyon çağrısı ise Name'e fonksiyon çağrısı stringini ata, Value nil olsun
		varName = strings.TrimSpace(varName)
		if strings.Contains(varName, "(") && strings.HasSuffix(varName, ")") {
			nodes = append(nodes, &VariableNode{Name: varName, Value: nil, Filters: filters})
		} else if len(filters) > 0 {
			var value interface{} = nil
			if len(varName) > 1 && ((varName[0] == '"' && varName[len(varName)-1] == '"') || (varName[0] == '\'' && varName[len(varName)-1] == '\'')) {
				value = varName[1 : len(varName)-1]
				varName = ""
			} else if ival, err := strconv.Atoi(varName); err == nil {
				value = ival
				varName = ""
			} else if fval, err := strconv.ParseFloat(varName, 64); err == nil {
				value = fval
				varName = ""
			}
			nodes = append(nodes, &VariableNode{Name: varName, Value: value, Filters: filters})
		} else {
			// Sadece değişken veya literal
			var value interface{} = nil
			if len(varName) > 1 && ((varName[0] == '"' && varName[len(varName)-1] == '"') || (varName[0] == '\'' && varName[len(varName)-1] == '\'')) {
				value = varName[1 : len(varName)-1]
				varName = ""
			} else if ival, err := strconv.Atoi(varName); err == nil {
				value = ival
				varName = ""
			} else if fval, err := strconv.ParseFloat(varName, 64); err == nil {
				value = fval
				varName = ""
			}
			if value != nil {
				nodes = append(nodes, &VariableNode{Name: "", Value: value, Filters: nil})
			} else {
				nodes = append(nodes, &VariableNode{Name: varName, Value: nil, Filters: nil})
			}
		}
		tpl = tpl[start+end+2:]
	}

	return &ListNode{Nodes: nodes}, nil
}

// ParseWithBlocks, override edilen bloklarla birlikte template'i AST'ye dönüştürür.
func (p *Parser) ParseWithBlocks(override map[string]ASTNode) (ASTNode, error) {
	tpl := p.template
	nodes := []ASTNode{}
	for len(tpl) > 0 {
		start := strings.Index(tpl, "{{ block ")
		if start == -1 {
			if tpl != "" {
				nodes = append(nodes, &TextNode{Text: tpl})
			}
			break
		}
		if start > 0 {
			nodes = append(nodes, &TextNode{Text: tpl[:start]})
		}
		endBlockName := strings.Index(tpl[start:], "}}")
		if endBlockName == -1 {
			return nil, fmt.Errorf("unclosed block tag")
		}
		name := strings.TrimSpace(tpl[start+len("{{ block ") : start+endBlockName])
		after := tpl[start+endBlockName+2:]
		endBlock := "{{ endblock }}"
		endIdx := strings.Index(after, endBlock)
		if endIdx == -1 {
			return nil, fmt.Errorf("unclosed endblock for block: %s", name)
		}
		if override != nil {
			if overrideAst, ok := override[name]; ok {
				nodes = append(nodes, &BlockNode{Name: name, Body: overrideAst})
			} else {
				ast, err := NewParser(after[:endIdx]).Parse()
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, &BlockNode{Name: name, Body: ast})
			}
		}
		tpl = after[endIdx+len(endBlock):]
	}
	return &ListNode{Nodes: nodes}, nil
}

// getLineCol, tpl içindeki offset'i satır/sütun olarak bulur.
func getLineCol(tpl string, offset int) (int, int) {
	line, col := 1, 1
	for _, c := range tpl[:offset] {
		if c == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

// isNumeric, bir string'in sayısal olup olmadığını kontrol eder.
func isNumeric(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}
