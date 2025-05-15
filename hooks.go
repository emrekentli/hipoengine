package hipoengine

// Package hipoengine, template engine için hook yönetimi sağlar.

// HookFiles, her bir hook ismine karşılık gelen dosya listesini tutar.
type HookFiles map[string][]string

// Example: 
// engine.RegisterHook("footer", []string{"footer.tpl"}) 