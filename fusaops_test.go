package fusaops

import "testing"

func TestSeverityRank(t *testing.T) {
	if SeverityError.Rank() <= SeverityWarning.Rank() {
		t.Errorf("error rank %d not greater than warning rank %d", SeverityError.Rank(), SeverityWarning.Rank())
	}
	if SeverityWarning.Rank() <= SeverityInfo.Rank() {
		t.Errorf("warning rank %d not greater than info rank %d", SeverityWarning.Rank(), SeverityInfo.Rank())
	}
	if SeverityInfo.Rank() <= Severity("bogus").Rank() {
		t.Errorf("info rank should exceed unknown severity rank")
	}
}

func TestSeverityString(t *testing.T) {
	if SeverityError.String() != "ERROR" {
		t.Errorf("got %q, want ERROR", SeverityError.String())
	}
}

func TestLanguageString(t *testing.T) {
	cases := map[Language]string{LangGo: "go", LangC: "c", LangCpp: "cpp"}
	for l, want := range cases {
		if l.String() != want {
			t.Errorf("language %v: got %q, want %q", l, l.String(), want)
		}
	}
}
