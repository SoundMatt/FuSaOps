package report

import (
	"encoding/xml"
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// JUnit XML minimal model — one testsuite per component so findings are
// attributable to the language toolchain that produced them.

type junitTestsuites struct {
	XMLName  xml.Name     `xml:"testsuites"`
	Name     string       `xml:"name,attr"`
	Tests    int          `xml:"tests,attr"`
	Failures int          `xml:"failures,attr"`
	Errors   int          `xml:"errors,attr"`
	Time     string       `xml:"time,attr"`
	Suites   []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	XMLName  xml.Name    `xml:"testsuite"`
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Errors   int         `xml:"errors,attr"`
	Time     string      `xml:"time,attr"`
	Cases    []junitCase `xml:"testcase"`
}

type junitCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// junitCaseName formats the testcase name for a finding.
func junitCaseName(f fusaops.Finding) string {
	name := f.QualifiedRuleID()
	if f.Location.File != "" {
		if f.Location.Line > 0 {
			name = fmt.Sprintf("%s (%s:%d)", f.QualifiedRuleID(), f.Location.File, f.Location.Line)
		} else {
			name = fmt.Sprintf("%s (%s)", f.QualifiedRuleID(), f.Location.File)
		}
	}
	return name
}

// renderJUnit writes the aggregate report as JUnit XML for CI test result
// consumers such as Jenkins, Azure DevOps, and CircleCI.
//
// Each component (language × tool) maps to a <testsuite>. Each finding maps to
// a <testcase>; ERROR and WARNING findings carry a <failure> element. Components
// with no findings emit a synthetic passing <testcase name="(no findings)"/>.
// Skipped components emit a <testcase><skipped/></testcase>.
//
//fusa:req REQ-FO-RPT013
func renderJUnit(w io.Writer, r *AggregateReport) error {
	root := junitTestsuites{
		Name:   "FuSaOps",
		Time:   "0.000",
		Errors: 0,
	}

	for _, c := range r.Components {
		classname := fmt.Sprintf("%s.%s", c.Language, c.Tool)
		suite := junitSuite{
			Name:   fmt.Sprintf("%s/%s", c.Language, c.Tool),
			Time:   "0.000",
			Errors: 0,
		}

		if c.Skipped != "" {
			suite.Tests = 1
			suite.Cases = append(suite.Cases, junitCase{
				Name:      "(skipped)",
				Classname: classname,
				Time:      "0.000",
				Skipped:   &junitSkipped{Message: c.Skipped},
			})
		} else if len(c.Findings) == 0 {
			suite.Tests = 1
			suite.Cases = append(suite.Cases, junitCase{
				Name:      "(no findings)",
				Classname: classname,
				Time:      "0.000",
			})
		} else {
			suite.Tests = len(c.Findings)
			for _, f := range c.Findings {
				tc := junitCase{
					Name:      junitCaseName(f),
					Classname: classname,
					Time:      "0.000",
				}
				if f.Severity == fusaops.SeverityError || f.Severity == fusaops.SeverityWarning {
					body := f.Message
					if f.Location.File != "" {
						body = fmt.Sprintf("%s:%d: %s", f.Location.File, f.Location.Line, f.Message)
					}
					tc.Failure = &junitFailure{
						Message: f.Message,
						Type:    string(f.Severity),
						Body:    body,
					}
					suite.Failures++
				}
				suite.Cases = append(suite.Cases, tc)
			}
		}

		root.Tests += suite.Tests
		root.Failures += suite.Failures
		root.Suites = append(root.Suites, suite)
	}

	if _, err := fmt.Fprintf(w, "%s\n", xml.Header); err != nil {
		return fmt.Errorf("report: junit header: %w", err)
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return fmt.Errorf("report: junit encode: %w", err)
	}
	_, err := fmt.Fprintln(w)
	return err
}
