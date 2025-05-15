# HipoEngine

Go için modern, hızlı ve genişletilebilir bir template engine.

## Özellikler
- Extends, block, include, for, if, set, filter, raw, comment desteği
- Custom filter ve hook ekleme
- Template dosyası cache'leme
- Thread-safe kullanım

## Kurulum

```sh
go get github.com/emrekentli/hipoengine
```

## Kullanım

```go
import "github.com/emrekentli/hipoengine"

engine := hipoengine.New()

// Custom filter ekle
engine.RegisterFilter("money", func(val interface{}) string {
    f, _ := strconv.ParseFloat(fmt.Sprintf("%v", val), 64)
    return fmt.Sprintf("%.2f TL", f)
})

// Hook ekle (ör: footer)
engine.RegisterHook("footer", []string{"templates/footer.tpl"})

tpl, err := engine.ParseFile("templates/main.tpl")
if err != nil {
    panic(err)
}

output, err := engine.Render(tpl, map[string]interface{}{
    "name": "Ahmet",
    "price": 123.45,
    "products": []interface{}{ "Elma", "Armut", "Muz" },
    "isLoggedIn": true,
})
if err != nil {
    panic(err)
}
fmt.Println(output)
```

### Template içinde hook kullanımı

```tpl
... diğer template kodları ...
{% hook "footer" %}
```

## Katkı
Katkıda bulunmak isterseniz lütfen bir issue açın veya doğrudan pull request gönderin.

## Destek
Sorunlarınız veya önerileriniz için GitHub Issues bölümünü kullanabilirsiniz.

## Lisans
MIT 