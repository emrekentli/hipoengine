// sandbox.go
// Sandboxing, render timeout ve adım limiti yönetimi
package hipoengine

import (
	"context"
	"errors"
	"time"
)

// RenderOptions: render işlemi için güvenlik ve limit ayarları
// (Engine veya Context'e entegre edilebilir)
type RenderOptions struct {
	Timeout  time.Duration // Maksimum render süresi
	MaxSteps int           // Maksimum node/fonksiyon/filtre adımı
}

// Render adım sayacı (Context'e entegre edilebilir)
type RenderStepCounter struct {
	Steps int
	Limit int
}

func (c *RenderStepCounter) Inc() error {
	c.Steps++
	if c.Limit > 0 && c.Steps > c.Limit {
		return errors.New("Render adım limiti aşıldı (sonsuz döngü koruması)")
	}
	return nil
}

// Render işlemini timeout ile başlat
func RenderWithTimeout(ctx context.Context, renderFunc func() (string, error), timeout time.Duration) (string, error) {
	if timeout <= 0 {
		return renderFunc()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ch := make(chan struct {
		result string
		err    error
	}, 1)
	go func() {
		res, err := renderFunc()
		ch <- struct {
			result string
			err    error
		}{res, err}
	}()
	select {
	case <-ctx.Done():
		return "", errors.New("Render timeout: süre limiti aşıldı")
	case out := <-ch:
		return out.result, out.err
	}
}

// İzinli fonksiyon/filtre/variable kontrolü (SafeMode için)
func IsAllowed(name string, allowed map[string]bool) bool {
	if allowed == nil {
		return true // Sınırsız mod
	}
	return allowed[name]
}
