package hipoengine

import (
	"fmt"
	"strings"
)

// ParseBlocks, template içerisindeki tüm {{ block name }}...{{ endblock }} bloklarını parse eder
func ParseBlocks(tpl string) (map[string]ASTNode, error) {
	blocks := make(map[string]ASTNode)
	for {
		tpl = strings.TrimLeft(tpl, " \t\r\n") // baştaki boşlukları ve satır sonlarını atla
		if len(tpl) == 0 {
			break
		}
		start := strings.Index(tpl, "{{ block ")
		if start == -1 {
			break
		}
		if start > 0 {
			// Block'tan önce kalan gereksiz içeriği atla
			tpl = tpl[start:]
			start = 0
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
		blockBody := after[:endIdx]
		ast, err := NewParser(blockBody).Parse()
		if err != nil {
			return nil, err
		}
		blocks[name] = ast
		tpl = after[endIdx+len(endBlock):]
	}
	return blocks, nil
}

// extractExtendsFileName, {{ extends "filename" }} tagından dosya ismini temiz olarak çıkarır
func extractExtendsFileName(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "{{ extends ")
	tag = strings.TrimSuffix(tag, "}}")
	tag = strings.Trim(tag, " \t\r\n\"'")
	return tag
}
