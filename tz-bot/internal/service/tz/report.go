package tzservice

import (
	"strconv"
	"strings"
)

type ReportEntry struct {
	ErrorID          string
	GroupID          string
	Code             string
	ErrType          string
	Snippet          string
	LineStart        *int
	LineEnd          *int
	CandidateIDs     []string
	ElementID        string   // где найдено/пытались ставить
	Status           string   // found|not-found|skipped
	Reason           string   // пояснение (контейнер, мульти-узел, пересечения и т.п.)
	CandidatePreview []string // id::preview
}

func summarizeText(s string) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > 80 {
		s = string([]rune(s))[:80] + "…"
	}
	return s
}

func (r ReportEntry) String() string {
	// компактная строка (TSV). Snippet можно усечь до 120 символов.
	snip := r.Snippet
	if len([]rune(snip)) > 120 {
		snip = string([]rune(snip)[:120]) + "…"
	}
	ls, le := "", ""
	if r.LineStart != nil {
		ls = strconv.Itoa(*r.LineStart)
	}
	if r.LineEnd != nil {
		le = strconv.Itoa(*r.LineEnd)
	}
	if r.CandidatePreview != nil {
		strings.Join(r.CandidatePreview, " | ")
	}
	return strings.Join([]string{
		r.ErrorID, r.ErrType, r.Code, r.GroupID, ls, le,
		strings.Join(r.CandidateIDs, ","), r.ElementID, r.Status, r.Reason, snip,
	}, "\t")
}

func buildReport(report []ReportEntry) string {
	var b strings.Builder
	b.WriteString("error_id\terr_type\tcode\tgroup\tline_start\tline_end\tcandidates\tchosen_element\tstatus\treason\tsnippet\n")
	for _, e := range report {
		b.WriteString(e.String())
		b.WriteByte('\n')
	}
	return b.String()
}
