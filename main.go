package main

import (
	"fmt"
	"github.com/seninrepo/hipoengine"
	"strconv"
)

func main() {
	engine := hipoengine.New()

	// Custom filter ekle
	engine.RegisterFilter("money", func(val interface{}) string {
		f, _ := strconv.ParseFloat(fmt.Sprintf("%v", val), 64)
		return fmt.Sprintf("%.2f TL", f)
	})

	// Template dosyasını oku ve parse et
	tpl, err := engine.ParseFile("templates/main.tpl")
	if err != nil {
		panic(err)
	}

	// Render et
	output, err := engine.Render(tpl, map[string]interface{}{
		"name":       "Ahmet",
		"price":      123.45,
		"products":   []interface{}{"elma", "armut", "muz"},
		"isLoggedIn": true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(output)
} 