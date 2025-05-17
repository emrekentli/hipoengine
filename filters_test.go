package hipoengine

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSortFilter(t *testing.T) {
	arr := []interface{}{3, 1, 2}
	out := DefaultFilters["sort"](arr)
	exp := []interface{}{1, 2, 3}
	if !reflect.DeepEqual(out, exp) {
		t.Errorf("sort failed: got %v, want %v", out, exp)
	}
}

func TestUniqFilter(t *testing.T) {
	arr := []interface{}{1, 2, 2, 3, 1}
	out := DefaultFilters["uniq"](arr)
	exp := []interface{}{1, 2, 3}
	if !reflect.DeepEqual(out, exp) {
		t.Errorf("uniq failed: got %v, want %v", out, exp)
	}
}

func TestSlugifyFilter(t *testing.T) {
	s := "Çılgın Türkçe Başlık! 2024"
	out := DefaultFilters["slugify"](s)
	exp := "cilgin-turkce-baslik-2024"
	if out != exp {
		t.Errorf("slugify failed: got %v, want %v", out, exp)
	}
}

func TestHumanizeFilter(t *testing.T) {
	timeAgo := time.Now().Add(-2 * time.Hour)
	out := DefaultFilters["humanize"](timeAgo)
	if !strings.Contains(out.(string), "saat önce") {
		t.Errorf("humanize failed: got %v", out)
	}
}

func TestSplitFilter(t *testing.T) {
	s := "a,b,c"
	out := DefaultFilters["split"](s, ",")
	exp := []interface{}{"a", "b", "c"}
	if !reflect.DeepEqual(out, exp) {
		t.Errorf("split failed: got %v, want %v", out, exp)
	}
}

func TestStartswithFilter(t *testing.T) {
	s := "hello world"
	out := DefaultFilters["startswith"](s, "hello")
	if out != true {
		t.Errorf("startswith failed: got %v, want true", out)
	}
}

func TestEndswithFilter(t *testing.T) {
	s := "hello world"
	out := DefaultFilters["endswith"](s, "world")
	if out != true {
		t.Errorf("endswith failed: got %v, want true", out)
	}
}

func TestPadFilter(t *testing.T) {
	s := "abc"
	out := DefaultFilters["pad"](s, 5)
	exp := "abc  "
	if out != exp {
		t.Errorf("pad failed: got %v, want %v", out, exp)
	}
}

func TestLjustFilter(t *testing.T) {
	s := "abc"
	out := DefaultFilters["ljust"](s, 6)
	exp := "abc   "
	if out != exp {
		t.Errorf("ljust failed: got %v, want %v", out, exp)
	}
}

func TestRjustFilter(t *testing.T) {
	s := "abc"
	out := DefaultFilters["rjust"](s, 6)
	exp := "   abc"
	if out != exp {
		t.Errorf("rjust failed: got %v, want %v", out, exp)
	}
}

func TestRegexReplaceFilter(t *testing.T) {
	s := "abc123def456"
	out := DefaultFilters["regex_replace"](s, "[0-9]+", "X")
	exp := "abcXdefX"
	if out != exp {
		t.Errorf("regex_replace failed: got %v, want %v", out, exp)
	}
}
