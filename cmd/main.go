package main

import (
	"fmt"
	"hipoengine"
	"os"
	"time"
)

func printTest(title string, fn func() string) {
	fmt.Println("==============================")
	fmt.Println(title)
	fmt.Println("------------------------------")
	fmt.Println(fn())
	fmt.Println("==============================")
}

func main() {
	engine := hipoengine.NewEngine()

	// Örnek özel filtre (DefaultFilters zaten dahil)
	engine.RegisterFilter("exclaim", func(val interface{}, args ...interface{}) interface{} {
		return fmt.Sprintf("%v!", val)
	})

	// Örnek fonksiyonlar
	engine.RegisterFunction("GetX", func(args ...interface{}) interface{} {
		return map[string]interface{}{"name": "Cemal"}
	})
	engine.RegisterFunction("GetProduct", func(args ...interface{}) interface{} {
		return []interface{}{
			map[string]interface{}{"name": "Ürün 1", "price": 100},
			map[string]interface{}{"name": "Ürün 2", "price": 200},
		}
	})
	// TheMealDB kategorileri fonksiyonunu kaydet
	engine.RegisterFunction("getCategories", hipoengine.GetCategories)

	// Eksiksiz context
	fullContext := map[string]interface{}{
		"user":  map[string]interface{}{"name": "emre", "age": 21},
		"val":   "Merhaba Dünya",
		"items": []interface{}{1, 2, 3},
		"now":   time.Date(2024, 6, 13, 15, 4, 5, 0, time.UTC),
		"html":  "<script>alert('xss')</script>",
	}

	// Örnek layout ve view dosya yolları
	layoutFile := "testdata/layouts/layout.hipo"
	viewFile := "testdata/views/child.hipo"

	printTest("1. Layout ile Render (RenderWithLayout)", func() string {
		result, err := engine.RenderWithLayout(viewFile, layoutFile, fullContext)
		if err != nil {
			return "Render hatası: " + err.Error()
		}
		return result
	})

	// child.hipo'yu layout ile render edip HTML çıktısını dosyaya yaz
	outputPath := "testdata/outputs/child.html"
	var result string
	var err error
	result, err = engine.RenderWithLayout(viewFile, layoutFile, fullContext)
	if err != nil {
		fmt.Println("RenderWithLayout hata:", err)
	} else {
		err = os.MkdirAll("testdata/outputs", 0755)
		if err != nil {
			fmt.Println("Klasör oluşturulamadı:", err)
		} else {
			err = os.WriteFile(outputPath, []byte(result), 0644)
			if err != nil {
				fmt.Println("Dosyaya yazılamadı:", err)
			} else {
				fmt.Println("HTML çıktı dosyaya yazıldı:", outputPath)
			}
		}
	}

	printTest("2. Default Filtreli Render", func() string {
		out, err := engine.Render("Merhaba {{ name|default:\"Anonim\"|upper }}!", map[string]interface{}{})
		if err != nil {
			return "Hata: " + err.Error()
		}
		return out
	})

	printTest("3. HTML Escape ve |safe Filtresi", func() string {
		out1, _ := engine.Render("<b>{{ html }}</b>", fullContext)
		out2, _ := engine.Render("<b>{{ html|safe }}</b>", fullContext)
		return "Escape edilmiş çıktı: " + out1 + "\n|safe ile escape edilmemiş çıktı: " + out2
	})

	printTest("4. Zengin Filtreler", func() string {
		// now := fullContext["now"] // Artık kullanılmıyor, kaldırıldı
		out := ""
		add := func(s string) { out += s + "\n" }
		res, _ := engine.Render("Bugün: {{ now|date:\"02.01.2006\" }}", fullContext)
		add("date filtresi: " + res)
		res, _ = engine.Render("Liste: {{ items|join:, }}", fullContext)
		add("join filtresi: " + res)
		res, _ = engine.Render("Toplam: {{ val|add:7 }}", fullContext)
		add("add filtresi: " + res)
		res, _ = engine.Render("Para: {{ val|money:\"₺\" }}", fullContext)
		add("money filtresi: " + res)
		res, _ = engine.Render("Kısa: {{ val|truncate:7 }}", fullContext)
		add("truncate filtresi: " + res)
		res, _ = engine.Render("Dilim: {{ val|slice:2,4 }}", fullContext)
		add("slice filtresi: " + res)
		res, _ = engine.Render("Değiştir: {{ val|replace:'a','e' }}", fullContext)
		add("replace filtresi: " + res)
		res, _ = engine.Render("Mutlak: {{ val|abs }}", map[string]interface{}{"val": -42})
		add("abs filtresi: " + res)
		res, _ = engine.Render("Evet/Hayır: {{ val|yesno:'Evet,Hayır' }}", map[string]interface{}{"val": 1})
		add("yesno filtresi: " + res)
		return out
	})

	printTest("5. Set/Assign Özelliği", func() string {
		out := ""
		add := func(s string) { out += s + "\n" }
		res, _ := engine.Render(`{{ set x = 42 }}x: {{ x }}`, fullContext)
		add("Literal atama: " + res)
		res, _ = engine.Render(`{{ set y = "merhaba"|upper }}Y: {{ y }}`, fullContext)
		add("Filtreli atama: " + res)
		res, _ = engine.Render(`{{ set z = user.name|title }}Z: {{ z }}`, fullContext)
		add("Context'ten atama: " + res)
		return out
	})

	printTest("6. Set ile Fonksiyon Çağrısı", func() string {
		out := ""
		add := func(s string) { out += s + "\n" }
		res, _ := engine.Render(`{{ set products = GetProduct() }}İlk ürün: {{ products[0].name }} - {{ products[0].price }}`, fullContext)
		add("Fonksiyon set ile atama: " + res)
		return out
	})

	printTest("7. Set + For ile Fonksiyon Çağrısı", func() string {
		out := ""
		add := func(s string) { out += s + "\n" }
		res, _ := engine.Render(`{{ set products = GetProduct() }}Tüm ürünler:\n{{ for urun in products }}- {{ urun.name }}: {{ urun.price }}\n{{ endfor }}`, fullContext)
		add("Set + for ile ürünler:\n" + res)
		return out
	})

	printTest("8. Gelişmiş Dizi/Map Erişimi", func() string {
		out := ""
		add := func(s string) { out += s + "\n" }
		context := map[string]interface{}{
			"user": map[string]interface{}{"first_name": "Ali", "last_name": "Veli"},
			"products": []interface{}{
				map[string]interface{}{"name": "Ürün 1", "features": []interface{}{"A", "B"}},
				map[string]interface{}{"name": "Ürün 2", "features": []interface{}{"C", "D"}},
			},
		}
		res, _ := engine.Render(`İlk ürünün ikinci özelliği: {{ products[0].features[1] }}\nKullanıcı adı: {{ user["first_name"] }}\nSon ürün: {{ products[-1].name }}`, context)
		add("Gelişmiş erişim örneği:\n" + res)
		return out
	})

	engine.SetTranslationsFromDir("locale")
	engine.SetLang("tr")
	printTest("i18n Türkçe (çift tırnak)", func() string {
		ctx := map[string]interface{}{"name": "Emre"}
		out, _ := engine.Render("{{ trans(\"welcome\", ctx) }} - {{ trans(\"greeting\", ctx) }}", ctx)
		return out
	})
	printTest("i18n Türkçe (tek tırnak)", func() string {
		ctx := map[string]interface{}{"name": "Emre"}
		out, _ := engine.Render("{{ trans('welcome', ctx) }} - {{ trans('greeting', ctx) }}", ctx)
		return out
	})
	engine.SetLang("en")
	printTest("i18n İngilizce (çift tırnak)", func() string {
		ctx := map[string]interface{}{"name": "Emre"}
		out, _ := engine.Render("{{ trans(\"welcome\", ctx) }} - {{ trans(\"greeting\", ctx) }}", ctx)
		return out
	})
	printTest("i18n İngilizce (tek tırnak)", func() string {
		ctx := map[string]interface{}{"name": "Emre"}
		out, _ := engine.Render("{{ trans('welcome', ctx) }} - {{ trans('greeting', ctx) }}", ctx)
		return out
	})

	// --- i18n pluralization ve fallback demo ---
	engine.SetTranslations(map[string]interface{}{
		"tr": map[string]interface{}{
			"cart": map[string]interface{}{
				"items": map[string]interface{}{
					"one":   "Sepetinizde 1 ürün var.",
					"other": "Sepetinizde {{ count }} ürün var.",
				},
			},
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"name": "Ad (TR)",
				},
			},
		},
		"en": map[string]interface{}{
			"cart": map[string]interface{}{
				"items": map[string]interface{}{
					"one":   "You have 1 item in your cart.",
					"other": "You have {{ count }} items in your cart.",
				},
			},
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"name": "Name (EN)",
				},
			},
		},
	})
	engine.SetLang("tr")
	ctx := map[string]interface{}{"count": 1}
	out, _ := engine.Render(`{{ trans("cart.items", ctx, count) }}`, ctx)
	fmt.Println("TR, count=1:", out)
	ctx["count"] = 3
	out, _ = engine.Render(`{{ trans("cart.items", ctx, count) }}`, ctx)
	fmt.Println("TR, count=3:", out)
	engine.SetLang("en")
	ctx = map[string]interface{}{"count": 1}
	out, _ = engine.Render(`{{ trans("cart.items", ctx, count) }}`, ctx)
	fmt.Println("EN, count=1:", out)
	ctx["count"] = 3
	out, _ = engine.Render(`{{ trans("cart.items", ctx, count) }}`, ctx)
	fmt.Println("EN, count=3:", out)
	// Fallback ve namespace
	ctx = map[string]interface{}{}
	engine.SetLang("tr")
	out, _ = engine.Render(`{{ trans("user.profile.name", ctx) }}`, ctx)
	fmt.Println("TR, user.profile.name:", out)
	engine.SetLang("en")
	out, _ = engine.Render(`{{ trans("user.profile.name", ctx) }}`, ctx)
	fmt.Println("EN, user.profile.name:", out)

	fmt.Println("\n--- TheMealDB Kategorileri (template ile) ---")
	out, err = engine.Render(`{{ set categories = getCategories() }}{{ for cat in categories }}- {{ cat.strCategory }}\n{{ endfor }}`, nil)
	if err != nil {
		fmt.Println("getCategories hata:", err)
	} else {
		fmt.Println(out)
	}

	printTest("child.hipo doğrudan çıktı", func() string {
		out, err := engine.RenderFile("testdata/views/child.hipo", fullContext)
		if err != nil {
			return "Render hatası: " + err.Error()
		}
		return out
	})
}
