# HipoEngine

Go ile yazılmış, büyük ölçekli projeler için modern, güvenli ve genişletilebilir bir template engine.

---

## 🚀 Özellikler

- **Extends, block, include, for, if, set, filter, raw, comment** desteği
- **Argümanlı filter/fonksiyon zinciri**: `{{ name|default:"Anonim"|upper }}`
- **Otomatik HTML escaping** ve `|safe` filtresi
- **Zengin built-in filtreler**: date, join, add, money, truncate, slice, replace, abs, yesno, sort, uniq, slugify, split, pad, ljust, rjust, regex_replace, humanize, vs.
- **Custom filter/fonksiyon ekleme** (her tenant için izole)
- **API'den veri çeken fonksiyonlar** (timeout, cache, rate limit, hata yönetimi ile)
- **Thread-safe, cache'li, yüksek performanslı**
- **Çoklu template arama yolu, alias, context processor, global context**
- **Gelişmiş hata yönetimi** (satır/kolon, error mode, TemplateError struct)
- **set/assign ile template içinde değişken atama**
- **i18n (Çeviri) desteği**: locale/ klasöründen çoklu dil JSON yükleme, pluralization, fallback, namespace, contextli çeviri, dinamik dil değiştirme
- **Profiler, audit log, debug log, sandbox (timeout, step limit, fonksiyon whitelist) desteği**
- **Tenant separation**: Her tenant için izole engine/context/fonksiyon
- **Hot reload, canlı playground, gelişmiş test altyapısı**
- **Modüler dosya yapısı**: engine.go, context.go, nodes.go, filters.go, functions.go, profiler.go, sandbox.go, utils.go

---

## 🏗️ Kurulum

```sh
go get github.com/emrekentli/hipoengine
```

---

## ⚡ Production Kullanımı

### Tenant Bazlı Engine ve Fonksiyonlar
```go
engine := hipoengine.NewEngine()
engine.RegisterFunction("getProducts", func(args ...interface{}) interface{} {
    // ...
})
```

### Fonksiyonlarda Güvenlik ve Performans
- **Timeout**: Fonksiyonlar belirli sürede tamamlanmazsa iptal edilir.
- **Rate Limit**: Sık çağrılan fonksiyonlar için limit.
- **Cache**: Sık yapılan API çağrıları için cache/memoization.
- **Hata Yönetimi**: API hatası durumunda fallback değer.

### Fonksiyon Whitelist/Sandbox
```go
engine.SetAllowedFunctions([]string{"getProducts", "getCategories"})
```

---

## 🧩 Template Söz Dizimi

### Değişkenler ve Filtreler
```jinja
{{ name|default:"Anonim"|upper }}
{{ price|money }}
```

### Fonksiyon Çağrısı
```jinja
{{ getCategories() }}
{{ set categories = getCategories() }}
{{ for cat in categories }}- {{ cat.strCategory }}\n{{ endfor }}
```

### Bloklar ve Layout
```jinja
{{ block title }}Başlık{{ endblock }}
{{ block content }}İçerik{{ endblock }}
```

### Koşullar
```jinja
{{ if user.age >= 18 }}Yetişkin{{ elif user.age >= 13 }}Ergen{{ else }}Çocuk{{ endif }}
```

### Döngü
```jinja
{{ for item in items }}- {{ item }}\n{{ endfor }}
```

### With ve Set
```jinja
{{ with getUser() as user }}Kullanıcı: {{ user.name }}{{ endwith }}
{{ set x = 42 }}
```

### Include
```jinja
{{ include "partials/footer.hipo" }}
```

---

## 🌍 i18n (Çoklu Dil) Kullanımı

- `locale/` klasörüne `en.json`, `tr.json` gibi dosyalar koyun.
- Template içinde:
```jinja
{{ trans("welcome", ctx) }}
{{ trans("cart.items", ctx, count) }}
```
- Pluralization, fallback, contextli çeviri desteklenir.

---

## 🛡️ Güvenlik ve Sandboxing
- **Fonksiyon/filtre whitelist**
- **Timeout ve step limit**
- **Her tenant için context/fonksiyon izolasyonu**
- **Varsayılan HTML escaping**
- **Audit log ve profiler**

---

## 📈 Profiler & Audit Log
- Render sürelerini ve fonksiyon çağrılarını izleyin.
- Hangi tenant, hangi fonksiyonu, hangi parametrelerle çağırdı, ne kadar sürdü, hata oldu mu?

---

## 🧪 Test ve Demo
- `cmd/main.go` içinde kapsamlı demo ve testler mevcut.
- `testdata/views/child.hipo` ve `testdata/layouts/layout.hipo` ile örnek template kullanımı görebilirsiniz.
- Çıktı dosyası: `testdata/outputs/child.html`

---

## 📚 Geliştirici Deneyimi
- Kolay fonksiyon ve filter kayıt API'si
- Gelişmiş hata mesajları ve debug/log modları
- Test ortamı ve sandboxed preview

---

## 🤝 Katkı ve Destek
- Katkıda bulunmak için issue açabilir veya pull request gönderebilirsiniz.
- Sorunlarınız için GitHub Issues bölümünü kullanabilirsiniz.

---

## 📄 Lisans

MIT 