// functions.go
// Built-in fonksiyonlar ve fonksiyon yönetimi
package hipoengine

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Function func(args ...interface{}) interface{}

// Built-in fonksiyonlar için örnekler (engine.go'da RegisterFunction ile ekleniyor)
// Burada fonksiyonları merkezi olarak yönetebilirsin.

// getCategories: TheMealDB API'den kategorileri çeker ve slice olarak döndürür
func GetCategories(args ...interface{}) interface{} {
	resp, err := http.Get("https://www.themealdb.com/api/json/v1/1/categories.php")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}
	cats, ok := result["categories"].([]interface{})
	if !ok {
		return nil
	}
	return cats
}
