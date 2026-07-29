// Package req manages the FuSaOps requirement registry (.fusa-reqs.json).
//
// It provides load/save operations, CSV import/export, and XML import/export
// for DOORS ReqIF, Polarion, Codebeamer, and Jama Connect formats.
package req

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReqsFile is the default requirement registry filename.
const ReqsFile = ".fusa-reqs.json"

// Entry is a single requirement in the FuSaOps registry.
//
//fusa:req REQ-FO-REQ001
type Entry struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Text        string `json:"text,omitempty"`
	Description string `json:"description,omitempty"`
	Standard    string `json:"standard,omitempty"`
	Level       string `json:"level,omitempty"`
	Parent      string `json:"parent,omitempty"` // ID of parent HLR; set for LLR requirements
	Priority    string `json:"priority,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
}

// LoadRegistry reads requirements from .fusa-reqs.json in dir.
//
//fusa:req REQ-FO-REQ001
func LoadRegistry(dir string) ([]Entry, error) {
	path := filepath.Join(dir, ReqsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("req: read %s: %w", ReqsFile, err)
	}
	var payload struct {
		Requirements []Entry `json:"requirements"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("req: parse %s: %w", ReqsFile, err)
	}
	return payload.Requirements, nil
}

// FindDuplicateIDs returns every id that appears more than once in entries,
// sorted for deterministic output. Per x-FuSa spec §1.2.2, an id MUST be
// unique within .fusa-reqs.json; a tool MUST validate this whenever it reads
// the file and MUST NOT silently merge or drop duplicates.
//
//fusa:req REQ-FO-REQ004
func FindDuplicateIDs(entries []Entry) []string {
	counts := make(map[string]int, len(entries))
	for _, e := range entries {
		counts[e.ID]++
	}
	var dupes []string
	for id, n := range counts {
		if n > 1 {
			dupes = append(dupes, id)
		}
	}
	sort.Strings(dupes)
	return dupes
}

// SaveRegistry writes entries as .fusa-reqs.json in dir.
//
//fusa:req REQ-FO-REQ001
func SaveRegistry(dir string, entries []Entry) error {
	payload := struct {
		Requirements []Entry `json:"requirements"`
	}{Requirements: entries}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("req: marshal: %w", err)
	}
	path := filepath.Join(dir, ReqsFile)
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("req: write %s: %w", ReqsFile, err)
	}
	return nil
}

// ─── CSV ─────────────────────────────────────────────────────────────────────

// ParseCSV reads requirements from a CSV reader.
// Expected header: id, title, text, standard, level, parent, priority, rationale
//
//fusa:req REQ-FO-REQ002
func ParseCSV(r io.Reader) ([]Entry, error) {
	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("req: read csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("req: CSV file is empty")
	}
	header := records[0]
	if len(header) < 2 || strings.ToLower(header[0]) != "id" {
		return nil, fmt.Errorf("req: CSV must have a header row starting with id,title")
	}
	idx := make(map[string]int)
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(row []string, col string) string {
		i, ok := idx[col]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	var entries []Entry
	for _, row := range records[1:] {
		id := get(row, "id")
		if id == "" {
			continue
		}
		entries = append(entries, Entry{
			ID:          id,
			Title:       get(row, "title"),
			Text:        get(row, "text"),
			Description: get(row, "description"),
			Standard:    get(row, "standard"),
			Level:       get(row, "level"),
			Parent:      get(row, "parent"),
			Priority:    get(row, "priority"),
			Rationale:   get(row, "rationale"),
		})
	}
	return entries, nil
}

// RenderCSV writes entries as CSV with a header row.
//
//fusa:req REQ-FO-REQ003
func RenderCSV(w io.Writer, entries []Entry) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"id", "title", "text", "description", "standard", "level", "parent", "priority", "rationale"}); err != nil {
		return fmt.Errorf("req: write csv header: %w", err)
	}
	for _, e := range entries {
		if err := cw.Write([]string{e.ID, e.Title, e.Text, e.Description, e.Standard, e.Level, e.Parent, e.Priority, e.Rationale}); err != nil {
			return fmt.Errorf("req: write csv row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// ─── DOORS ReqIF ─────────────────────────────────────────────────────────────

type reqifRoot struct {
	XMLName     xml.Name  `xml:"REQ-IF"`
	CoreContent reqifCore `xml:"CORE-CONTENT"`
}

type reqifCore struct {
	SpecObjects reqifSpecObjects `xml:"SPEC-OBJECTS"`
}

type reqifSpecObjects struct {
	Objects []reqifSpecObject `xml:"SPEC-OBJECT"`
}

type reqifSpecObject struct {
	Values reqifValues `xml:"VALUES"`
}

type reqifValues struct {
	Attrs []reqifAttrValue `xml:"ATTRIBUTE-VALUE-STRING"`
}

type reqifAttrValue struct {
	TheValue string `xml:"THE-VALUE,attr"`
}

// ParseDOORS parses a DOORS ReqIF XML byte slice.
//
//fusa:req REQ-FO-REQ002
func ParseDOORS(data []byte) ([]Entry, error) {
	var root reqifRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("req: parse DOORS ReqIF: %w", err)
	}
	var entries []Entry
	for _, obj := range root.CoreContent.SpecObjects.Objects {
		attrs := obj.Values.Attrs
		if len(attrs) == 0 {
			continue
		}
		e := Entry{}
		if len(attrs) >= 1 {
			e.ID = attrs[0].TheValue
		}
		if len(attrs) >= 2 {
			e.Title = attrs[1].TheValue
		}
		if len(attrs) >= 3 {
			e.Text = attrs[2].TheValue
		}
		if e.ID == "" {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ExportDOORS serialises entries as minimal DOORS ReqIF XML.
//
//fusa:req REQ-FO-REQ003
func ExportDOORS(entries []Entry) ([]byte, error) {
	type attrDef struct {
		Ref string `xml:"ATTRIBUTE-DEFINITION-STRING-REF"`
	}
	type attrValStr struct {
		TheValue string  `xml:"THE-VALUE,attr"`
		Def      attrDef `xml:"DEFINITION"`
	}
	type values struct {
		Attrs []attrValStr `xml:"ATTRIBUTE-VALUE-STRING"`
	}
	type specObj struct {
		Vals values `xml:"VALUES"`
	}
	type specObjs struct {
		Objects []specObj `xml:"SPEC-OBJECT"`
	}
	type coreContent struct {
		SpecObjects specObjs `xml:"SPEC-OBJECTS"`
	}
	type reqif struct {
		XMLName     xml.Name    `xml:"REQ-IF"`
		CoreContent coreContent `xml:"CORE-CONTENT"`
	}

	var objs []specObj
	for _, e := range entries {
		attrs := []attrValStr{
			{TheValue: e.ID, Def: attrDef{Ref: "attr-id"}},
			{TheValue: e.Title, Def: attrDef{Ref: "attr-title"}},
		}
		text := e.Text
		if text == "" {
			text = e.Description
		}
		if text != "" {
			attrs = append(attrs, attrValStr{TheValue: text, Def: attrDef{Ref: "attr-text"}})
		}
		objs = append(objs, specObj{Vals: values{Attrs: attrs}})
	}
	root := reqif{CoreContent: coreContent{SpecObjects: specObjs{Objects: objs}}}
	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("req: export DOORS ReqIF: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(out)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// ─── Codebeamer XML ──────────────────────────────────────────────────────────

type codebeamerTracker struct {
	XMLName xml.Name         `xml:"tracker"`
	Items   []codebeamerItem `xml:"item"`
}

type codebeamerItem struct {
	ID          string                  `xml:"id,attr"`
	Name        string                  `xml:"name"`
	Summary     string                  `xml:"summary"`
	Description string                  `xml:"description,omitempty"`
	Fields      *codebeamerCustomFields `xml:"customFields,omitempty"`
}

type codebeamerCustomFields struct {
	Fields []codebeamerField `xml:"field"`
}

type codebeamerField struct {
	ID    string `xml:"id,attr"`
	Value string `xml:",chardata"`
}

// ParseCodebeamer parses a Codebeamer tracker XML export.
//
//fusa:req REQ-FO-REQ002
func ParseCodebeamer(data []byte) ([]Entry, error) {
	var root codebeamerTracker
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("req: parse Codebeamer XML: %w", err)
	}
	var entries []Entry
	for _, item := range root.Items {
		id := item.ID
		if id == "" {
			id = item.Name
		}
		if id == "" {
			continue
		}
		e := Entry{ID: id, Title: item.Summary, Text: item.Description}
		if item.Fields != nil {
			for _, f := range item.Fields.Fields {
				if f.ID == "level" {
					e.Level = f.Value
				}
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ExportCodebeamer serialises entries as Codebeamer tracker XML.
//
//fusa:req REQ-FO-REQ003
func ExportCodebeamer(entries []Entry) ([]byte, error) {
	var items []codebeamerItem
	for _, e := range entries {
		text := e.Text
		if text == "" {
			text = e.Description
		}
		item := codebeamerItem{ID: e.ID, Name: e.ID, Summary: e.Title, Description: text}
		if e.Level != "" {
			item.Fields = &codebeamerCustomFields{Fields: []codebeamerField{{ID: "level", Value: e.Level}}}
		}
		items = append(items, item)
	}
	root := codebeamerTracker{Items: items}
	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("req: export Codebeamer XML: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(out)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// ─── Jama Connect XML ────────────────────────────────────────────────────────

type jamaItems struct {
	XMLName xml.Name   `xml:"items"`
	Items   []jamaItem `xml:"item"`
}

type jamaItem struct {
	ID     string      `xml:"id,attr"`
	Name   string      `xml:"name"`
	Desc   string      `xml:"description,omitempty"`
	Fields *jamaFields `xml:"fields,omitempty"`
}

type jamaFields struct {
	Fields []jamaField `xml:"field"`
}

type jamaField struct {
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

// ParseJama parses a Jama Connect XML export.
//
//fusa:req REQ-FO-REQ002
func ParseJama(data []byte) ([]Entry, error) {
	var root jamaItems
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("req: parse Jama XML: %w", err)
	}
	var entries []Entry
	for _, item := range root.Items {
		id := item.ID
		if id == "" {
			id = item.Name
		}
		if id == "" {
			continue
		}
		e := Entry{ID: id, Title: item.Name, Text: item.Desc}
		if item.Fields != nil {
			for _, f := range item.Fields.Fields {
				if f.ID == "level" {
					e.Level = f.Value
				}
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ExportJama serialises entries as Jama Connect XML.
//
//fusa:req REQ-FO-REQ003
func ExportJama(entries []Entry) ([]byte, error) {
	var items []jamaItem
	for _, e := range entries {
		text := e.Text
		if text == "" {
			text = e.Description
		}
		item := jamaItem{ID: e.ID, Name: e.Title, Desc: text}
		if e.Level != "" {
			item.Fields = &jamaFields{Fields: []jamaField{{ID: "level", Value: e.Level}}}
		}
		items = append(items, item)
	}
	root := jamaItems{Items: items}
	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("req: export Jama XML: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(out)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// ─── Polarion XML ─────────────────────────────────────────────────────────────

type polarionWorkitems struct {
	XMLName   xml.Name           `xml:"workitems"`
	Workitems []polarionWorkitem `xml:"workitem"`
}

type polarionWorkitem struct {
	ID          string                `xml:"id,attr"`
	Title       string                `xml:"title"`
	Description string                `xml:"description,omitempty"`
	Fields      *polarionCustomFields `xml:"customFields,omitempty"`
}

type polarionCustomFields struct {
	Fields []polarionCustomField `xml:"customField"`
}

type polarionCustomField struct {
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

// ParsePolarion parses a Polarion workitems XML export.
//
//fusa:req REQ-FO-REQ002
func ParsePolarion(data []byte) ([]Entry, error) {
	var root polarionWorkitems
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("req: parse Polarion XML: %w", err)
	}
	var entries []Entry
	for _, wi := range root.Workitems {
		if wi.ID == "" {
			continue
		}
		e := Entry{ID: wi.ID, Title: wi.Title, Text: wi.Description}
		if wi.Fields != nil {
			for _, cf := range wi.Fields.Fields {
				if cf.ID == "level" {
					e.Level = cf.Value
				}
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ExportPolarion serialises entries as Polarion workitems XML.
//
//fusa:req REQ-FO-REQ003
func ExportPolarion(entries []Entry) ([]byte, error) {
	var items []polarionWorkitem
	for _, e := range entries {
		text := e.Text
		if text == "" {
			text = e.Description
		}
		wi := polarionWorkitem{ID: e.ID, Title: e.Title, Description: text}
		if e.Level != "" {
			wi.Fields = &polarionCustomFields{Fields: []polarionCustomField{{ID: "level", Value: e.Level}}}
		}
		items = append(items, wi)
	}
	root := polarionWorkitems{Workitems: items}
	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("req: export Polarion XML: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(out)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
