// profiler.go
// Profiling, trace ve audit log yönetimi
package hipoengine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ProfileEntry struct {
	Name      string
	Type      string // template, function, filter
	Count     int
	TotalTime time.Duration
	LastTime  time.Duration
}

type Profiler struct {
	Entries map[string]*ProfileEntry // key: name:type
}

func NewProfiler() *Profiler {
	return &Profiler{Entries: make(map[string]*ProfileEntry)}
}

func (p *Profiler) Add(name, typ string, dur time.Duration) {
	key := name + ":" + typ
	entry, ok := p.Entries[key]
	if !ok {
		entry = &ProfileEntry{Name: name, Type: typ}
		p.Entries[key] = entry
	}
	entry.Count++
	entry.TotalTime += dur
	entry.LastTime = dur
}

func (p *Profiler) ToJSON() string {
	data := make([]*ProfileEntry, 0, len(p.Entries))
	for _, entry := range p.Entries {
		data = append(data, entry)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (p *Profiler) Report() string {
	type reportRow struct {
		Name      string
		Type      string
		Count     int
		TotalTime time.Duration
		AvgTime   time.Duration
		LastTime  time.Duration
	}
	rows := make([]reportRow, 0, len(p.Entries))
	for _, entry := range p.Entries {
		avg := time.Duration(0)
		if entry.Count > 0 {
			avg = entry.TotalTime / time.Duration(entry.Count)
		}
		rows = append(rows, reportRow{
			Name:      entry.Name,
			Type:      entry.Type,
			Count:     entry.Count,
			TotalTime: entry.TotalTime,
			AvgTime:   avg,
			LastTime:  entry.LastTime,
		})
	}
	// En yavaştan en hızlıya sırala (ortalama süreye göre)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].AvgTime > rows[j].AvgTime
	})
	var sb strings.Builder
	sb.WriteString("Profiling Raporu (En yavaştan en hızlıya):\n")
	sb.WriteString("Name\tType\tCount\tTotal\tAvg\tLast\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s\t%s\t%d\t%v\t%v\t%v\n", r.Name, r.Type, r.Count, r.TotalTime, r.AvgTime, r.LastTime))
	}
	return sb.String()
}

type RenderTrace struct {
	Templates      []string
	ContextSummary string
	StartTime      time.Time
	EndTime        time.Time
}

type AuditLogFunc func(user, template string, contextSummary string, duration time.Duration, success bool, err error)
