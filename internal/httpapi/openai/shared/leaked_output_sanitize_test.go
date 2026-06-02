package shared

import "testing"

func TestLeakedToolResultSectionFilterRemovesAcrossChunks(t *testing.T) {
	var filter LeakedToolResultSectionFilter
	got := ""
	for _, chunk := range []string{
		"before <|Tool|>{\"content\":\"secret\"",
		",\"tool_call_id\":\"call_1\"}",
		"<|end_of_toolresults|> after",
	} {
		got += filter.Apply(chunk)
	}
	if got != "before  after" {
		t.Fatalf("unexpected filtered output: %q", got)
	}
}

func TestLeakedToolResultSectionFilterHandlesFullwidthMarkers(t *testing.T) {
	var filter LeakedToolResultSectionFilter
	got := filter.Apply("A<｜Tool｜>{\"content\":\"secret\"}") +
		filter.Apply("<｜end▁of▁toolresults｜>B")
	if got != "AB" {
		t.Fatalf("unexpected filtered output for fullwidth markers: %q", got)
	}
}
