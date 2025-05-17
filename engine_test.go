package hipoengine

import (
	"strings"
	"testing"
)

func TestRenderSimpleText(t *testing.T) {
	e := NewEngine()
	ctx := map[string]interface{}{"name": "Emre"}
	out, err := e.Render("Merhaba {{ name }}!", ctx)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "Merhaba Emre!" {
		t.Errorf("Beklenen: 'Merhaba Emre!', Gerçek: '%s'", out)
	}
}

func TestRenderForLoop(t *testing.T) {
	e := NewEngine()
	ctx := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
	}
	tpl := `{{ for item in items }}{{ item }}{{ endfor }}`
	out, err := e.Render(tpl, ctx)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "a\nb\nc" {
		t.Errorf("Beklenen: 'a\\nb\\nc', Gerçek: '%s'", out)
	}
}

func TestForNodeTypeError(t *testing.T) {
	e := NewEngine()
	ctx := map[string]interface{}{
		"items": "not_a_slice",
	}
	tpl := `{{ for item in items }}{{ item }}{{ endfor }}`
	_, err := e.Render(tpl, ctx)
	if err == nil {
		t.Error("ForNode tip hatası bekleniyordu, hata alınmadı")
	}
	t.Logf("Hata mesajı: %v", err)
	if !strings.Contains(err.Error(), "ForNode") {
		t.Errorf("Hata mesajı beklenen türde değil: %v", err)
	}
}

func TestMissingVariable(t *testing.T) {
	e := NewEngine()
	out, err := e.Render("{{ notfound }}", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "" {
		t.Errorf("Beklenen: '', Gerçek: '%s'", out)
	}
}

func TestUnknownFilter(t *testing.T) {
	e := NewEngine()
	out, err := e.Render("{{ name|notafilter }}", map[string]interface{}{"name": "Emre"})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	// Bilinmeyen filtrede uyarı çıktısı beklenir
	if out != "Emre [filter notafilter not found]" {
		t.Errorf("Beklenen: 'Emre [filter notafilter not found]', Gerçek: '%s'", out)
	}
}

func TestUnknownFunction(t *testing.T) {
	e := NewEngine()
	out, err := e.Render("{{ NotAFunction() }}", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "" {
		t.Errorf("Beklenen: '', Gerçek: '%s'", out)
	}
}

func TestHTMLEscapeDefault(t *testing.T) {
	e := NewEngine()
	out, err := e.Render("{{ html }}", map[string]interface{}{"html": "<b>test</b>"})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "&lt;b&gt;test&lt;/b&gt;" {
		t.Errorf("Beklenen: '&lt;b&gt;test&lt;/b&gt;', Gerçek: '%s'", out)
	}
}

func TestSafeFilter(t *testing.T) {
	e := NewEngine()
	out, err := e.Render("{{ html|safe }}", map[string]interface{}{"html": "<b>test</b>"})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "<b>test</b>" {
		t.Errorf("Beklenen: '<b>test</b>', Gerçek: '%s'", out)
	}
}

func TestI18nTransFunction(t *testing.T) {
	e := NewEngine()
	err := e.SetTranslationsFromDir("locale")
	if err != nil {
		t.Fatalf("Çeviri dosyaları yüklenemedi: %v", err)
	}
	e.SetLang("tr")
	ctx := map[string]interface{}{"name": "Emre"}
	out, err := e.Render(`{{ trans("greeting", ctx) }}`, ctx)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "Merhaba, Emre!" {
		t.Errorf("Beklenen: 'Merhaba, Emre!', Gerçek: '%s'", out)
	}

	// Tek tırnak ile
	out, err = e.Render(`{{ trans('greeting', ctx) }}`, ctx)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "Merhaba, Emre!" {
		t.Errorf("Beklenen: 'Merhaba, Emre!' (tek tırnak), Gerçek: '%s'", out)
	}

	e.SetLang("en")
	out, err = e.Render(`{{ trans("greeting", ctx) }}`, ctx)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "Hello, Emre!" {
		t.Errorf("Beklenen: 'Hello, Emre!', Gerçek: '%s'", out)
	}

	// Tek tırnak ile
	out, err = e.Render(`{{ trans('greeting', ctx) }}`, ctx)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "Hello, Emre!" {
		t.Errorf("Beklenen: 'Hello, Emre!' (tek tırnak), Gerçek: '%s'", out)
	}
}

func TestI18nPluralizationAndFallback(t *testing.T) {
	e := NewEngine()
	e.SetFallbackLang("en")
	e.SetTranslations(map[string]interface{}{
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
	t.Logf("translations: %#v", e.translations)
	e.SetLang("tr")
	ctx := map[string]interface{}{"count": 1}
	out, _ := e.Render(`{{ trans("cart.items", ctx, count) }}`, ctx)
	t.Logf("template render: '%s'", out)
	if strings.TrimSpace(out) != "Sepetinizde 1 ürün var." {
		t.Errorf("Beklenen: 'Sepetinizde 1 ürün var.', Gerçek: '%s'", out)
	}

	ctx = map[string]interface{}{"count": 3}
	out, _ = e.Render(`{{ trans("cart.items", ctx, count) }}`, ctx)
	t.Logf("template render: '%s'", out)
	if strings.TrimSpace(out) != "Sepetinizde 3 ürün var." {
		t.Errorf("Beklenen: 'Sepetinizde 3 ürün var.', Gerçek: '%s'", out)
	}

	// Fallback test (tr'de olmayan anahtar, en'de var)
	ctx = map[string]interface{}{}
	out, _ = e.Render(`{{ trans("user.profile.name", ctx) }}`, ctx)
	if out != "Name (EN)" {
		t.Errorf("Beklenen: 'Name (EN)', Gerçek: '%s'", out)
	}

	// tr'de user.profile.name iç içe map olarak var
	e.SetLang("tr")
	ctx = map[string]interface{}{}
	out, _ = e.Render(`{{ trans("user.profile.name", ctx) }}`, ctx)
	if out != "Ad (TR)" {
		t.Errorf("Beklenen: 'Ad (TR)', Gerçek: '%s'", out)
	}

	// Doğrudan transFuncWithContext fonksiyonunu test et
	ctx = map[string]interface{}{"count": 1}
	out2 := e.transFuncWithContext(ctx, "cart.items", ctx, 1)
	t.Logf("Doğrudan transFuncWithContext: %v", out2)
}

func TestLookupNamespace(t *testing.T) {
	m := map[string]interface{}{
		"cart": map[string]interface{}{
			"items": map[string]interface{}{
				"one": "ok",
			},
		},
	}
	val, found := lookupNamespace(m, "cart.items")
	if !found || val == nil {
		t.Errorf("lookupNamespace 'cart.items' bulamadı")
	}
	if vmap, ok := val.(map[string]interface{}); !ok || vmap["one"] != "ok" {
		t.Errorf("lookupNamespace yanlış değer döndürdü: %v", val)
	}
}

func TestLookupNamespaceWithTestMap(t *testing.T) {
	m := map[string]interface{}{
		"cart": map[string]interface{}{
			"items": map[string]interface{}{
				"one": "ok",
			},
		},
	}
	val, found := lookupNamespace(m, "cart.items")
	t.Logf("found=%v val=%#v", found, val)
	if !found || val == nil {
		t.Errorf("lookupNamespace 'cart.items' bulamadı (test map)")
	}
}
