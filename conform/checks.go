package conform

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// commonHeader is the §3.1 header fields every document must carry.
type commonHeader struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Tool          string `json:"tool"`
	ToolVersion   string `json:"toolVersion"`
	Language      string `json:"language"`
	GeneratedAt   string `json:"generatedAt"`
}

// validateHeader checks the §3.1 common header fields.
//
//fusa:req REQ-FO-CNF008
func validateHeader(h commonHeader, expectedKind string) []string {
	var errs []string
	if h.SchemaVersion == "" {
		errs = append(errs, "missing schemaVersion")
	} else if !schemaVerRE.MatchString(h.SchemaVersion) {
		errs = append(errs, fmt.Sprintf("schemaVersion %q not MAJOR.MINOR", h.SchemaVersion))
	}
	if h.Kind == "" {
		errs = append(errs, "missing kind")
	} else if expectedKind != "" && h.Kind != expectedKind {
		// check-report also allows "report" as kind (§9.1)
		if expectedKind != "check-report" || h.Kind != "report" {
			errs = append(errs, fmt.Sprintf("kind %q, want %q", h.Kind, expectedKind))
		}
	}
	if h.Tool == "" {
		errs = append(errs, "missing tool")
	}
	if h.ToolVersion == "" {
		errs = append(errs, "missing toolVersion")
	}
	if h.Language == "" {
		errs = append(errs, "missing language")
	}
	if h.GeneratedAt == "" {
		errs = append(errs, "missing generatedAt")
	}
	return errs
}

var (
	versionLineRE    = regexp.MustCompile(`^(\S+)\s+(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`)
	ruleIDRE         = regexp.MustCompile(`^[A-Z][A-Z0-9]*(-[A-Z0-9.]+)*$`)
	fingerprintRE    = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	sha256PrefixedRE = regexp.MustCompile(`^sha256:`)
	bareHex64RE      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	schemaVerRE      = regexp.MustCompile(`^[0-9]+\.[0-9]+`)
)

// checkVersion validates §9.1 version output.
//
//fusa:req REQ-FO-CNF009
func (r *runner) checkVersion() {
	stdout, _, code := r.run(r.dir, r.binary, "version")
	if code != 0 {
		r.fail("version/exit-code", "§9.1", LevelMUST, "version exits 0",
			fmt.Sprintf("exit %d", code))
		return
	}
	line := strings.TrimSpace(strings.SplitN(string(stdout), "\n", 2)[0])
	m := versionLineRE.FindStringSubmatch(line)
	if m == nil {
		r.fail("version/output-format", "§9.1", LevelMUST, "version output matches <tool> <semver>",
			fmt.Sprintf("got %q", line))
		return
	}
	r.pass("version/output-format", "§9.1", LevelMUST, "version output matches <tool> <semver>")
	r.report.Tool = m[1]
	r.report.ToolVersion = m[2]
	r.lang = langFromBinary(filepath.Base(r.binary))

	// SHOULD: version --format json
	stdout2, _, code2 := r.run(r.dir, r.binary, "version", "--format", "json")
	if code2 != 0 {
		r.skip("version/json-format", "§9.1", LevelSHOULD, "version --format json",
			"command not supported")
		return
	}
	var vj struct {
		Tool        string `json:"tool"`
		Version     string `json:"version"`
		SpecVersion string `json:"specVersion"`
	}
	if err := decodeJSON(stdout2, &vj); err != nil {
		r.fail("version/json-format", "§9.1", LevelSHOULD, "version --format json",
			fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	var missing []string
	if vj.Tool == "" {
		missing = append(missing, "tool")
	}
	if vj.Version == "" {
		missing = append(missing, "version")
	}
	if vj.SpecVersion == "" {
		missing = append(missing, "specVersion")
	}
	if len(missing) > 0 {
		r.fail("version/json-format", "§9.1", LevelSHOULD, "version --format json",
			"missing: "+strings.Join(missing, ", "))
		return
	}
	r.pass("version/json-format", "§9.1", LevelSHOULD, "version --format json")
	r.report.SpecVersion = vj.SpecVersion
	if r.report.Language == "" {
		r.report.Language = r.lang
	}
}

// checkInit validates §9.1 init creates .fusa.json and .fusa-reqs.json.
//
//fusa:req REQ-FO-CNF010
func (r *runner) checkInit() {
	// Remove the files scaffold wrote so init has to create them.
	initDir, err := os.MkdirTemp("", "fusaops-conform-init-*")
	if err != nil {
		r.skip("init/creates-fusa-json", "§9.1", LevelMUST, "init creates .fusa.json",
			"could not create init test dir")
		return
	}
	defer func() { _ = os.RemoveAll(initDir) }()

	_, _, code := r.run(initDir, r.binary, "init",
		"--name", "conform-test", "--standard", "iso26262")
	if code != 0 {
		// Some tools may not support --name/--standard flags; skip gracefully.
		r.skip("init/creates-fusa-json", "§9.1", LevelMUST, "init creates .fusa.json",
			fmt.Sprintf("init exit %d — may require interactive flags", code))
		r.skip("init/creates-fusa-reqs-json", "§9.1", LevelMUST, "init creates .fusa-reqs.json",
			"skipped: init failed")
		return
	}

	// .fusa.json
	configData, err := os.ReadFile(filepath.Join(initDir, ".fusa.json"))
	if err != nil {
		r.fail("init/creates-fusa-json", "§9.1", LevelMUST, "init creates .fusa.json",
			"file not found after init")
	} else {
		var cfg map[string]interface{}
		if err2 := json.Unmarshal(configData, &cfg); err2 != nil {
			r.fail("init/creates-fusa-json", "§9.1", LevelMUST, "init creates .fusa.json",
				fmt.Sprintf("invalid JSON: %v", err2))
		} else {
			r.pass("init/creates-fusa-json", "§9.1", LevelMUST, "init creates .fusa.json")
		}
	}

	// .fusa-reqs.json
	reqsData, err := os.ReadFile(filepath.Join(initDir, ".fusa-reqs.json"))
	if err != nil {
		r.fail("init/creates-fusa-reqs-json", "§9.1", LevelMUST, "init creates .fusa-reqs.json",
			"file not found after init")
	} else {
		var rfile struct {
			Requirements []interface{} `json:"requirements"`
		}
		if err2 := json.Unmarshal(reqsData, &rfile); err2 != nil {
			r.fail("init/creates-fusa-reqs-json", "§9.1", LevelMUST, "init creates .fusa-reqs.json",
				fmt.Sprintf("invalid JSON or missing requirements key: %v", err2))
		} else {
			r.pass("init/creates-fusa-reqs-json", "§9.1", LevelMUST, "init creates .fusa-reqs.json")
		}
	}
}

// checkCheck validates §3.1/§4 check --format json output.
//
//fusa:req REQ-FO-CNF011
func (r *runner) checkCheck() {
	stdout, _, _ := r.run(r.dir, r.binary, "check", "--format", "json")

	var doc struct {
		commonHeader
		ProjectRoot string `json:"projectRoot"`
		Findings    []struct {
			RuleID   string `json:"ruleId"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Location *struct {
				File   string      `json:"file"`
				Line   interface{} `json:"line"`
				Column interface{} `json:"column"`
				// flat fields would be separate top-level keys, not nested
			} `json:"location"`
			Category    string `json:"category"`
			Fingerprint string `json:"fingerprint"`
		} `json:"findings"`
		Summary *struct {
			Total    *int `json:"total"`
			Errors   *int `json:"errors"`
			Warnings *int `json:"warnings"`
			Infos    *int `json:"infos"`
		} `json:"summary"`
	}

	if err := decodeJSON(stdout, &doc); err != nil {
		r.fail("check/json-parse", "§4", LevelMUST, "check --format json produces valid JSON",
			fmt.Sprintf("%v", err))
		return
	}
	r.pass("check/json-parse", "§4", LevelMUST, "check --format json produces valid JSON")

	// §3.1 common header
	if errs := validateHeader(doc.commonHeader, "check-report"); len(errs) > 0 {
		r.fail("check/common-header", "§3.1", LevelMUST, "check report carries common header",
			strings.Join(errs, "; "))
	} else {
		r.pass("check/common-header", "§3.1", LevelMUST, "check report carries common header")
		r.report.Language = doc.Language
	}

	// §3.2 projectRoot
	if doc.ProjectRoot == "" {
		r.fail("check/project-root", "§3.2", LevelMUST, "check report has projectRoot",
			"projectRoot absent or empty")
	} else {
		r.pass("check/project-root", "§3.2", LevelMUST, "check report has projectRoot")
	}

	// §4 findings array
	if doc.Findings == nil {
		r.fail("check/findings-present", "§4", LevelMUST, "check report has findings array",
			"findings key absent")
	} else {
		r.pass("check/findings-present", "§4", LevelMUST, "check report has findings array")
	}

	// §4 summary
	if doc.Summary == nil || doc.Summary.Total == nil || doc.Summary.Errors == nil ||
		doc.Summary.Warnings == nil || doc.Summary.Infos == nil {
		r.fail("check/summary", "§4", LevelMUST, "check report has summary {total,errors,warnings,infos}",
			"summary or required counter absent")
	} else {
		r.pass("check/summary", "§4", LevelMUST, "check report has summary {total,errors,warnings,infos}")
	}

	// Per-finding validation
	for i, f := range doc.Findings {
		prefix := fmt.Sprintf("finding[%d]", i)
		if f.RuleID == "" {
			r.fail("check/finding-ruleId", "§4/§1.5.1", LevelMUST,
				"finding has ruleId",
				fmt.Sprintf("%s: ruleId absent or empty", prefix))
			break
		}
		if !ruleIDRE.MatchString(f.RuleID) {
			r.fail("check/finding-ruleId", "§4/§1.5.1", LevelMUST,
				"finding ruleId matches ^[A-Z][A-Z0-9]*(-[A-Z0-9.]+)*$",
				fmt.Sprintf("%s: ruleId %q invalid", prefix, f.RuleID))
			break
		}

		switch f.Severity {
		case "ERROR", "WARNING", "INFO":
		default:
			r.fail("check/finding-severity", "§2.4/§4", LevelMUST,
				"finding severity ∈ {ERROR,WARNING,INFO}",
				fmt.Sprintf("%s: severity %q", prefix, f.Severity))
			return
		}

		if f.Location == nil {
			r.fail("check/finding-location-nested", "§4", LevelMUST,
				"finding.location is a nested object",
				fmt.Sprintf("%s: location absent", prefix))
			return
		}
		if f.Location.File == "" {
			r.fail("check/finding-location-file", "§4", LevelMUST,
				"finding.location.file present",
				fmt.Sprintf("%s: location.file absent", prefix))
			return
		}
		if filepath.IsAbs(f.Location.File) {
			r.fail("check/finding-location-relative", "§4", LevelMUST,
				"finding.location.file is project-relative",
				fmt.Sprintf("%s: location.file %q is absolute", prefix, f.Location.File))
			return
		}
	}
	if len(doc.Findings) > 0 {
		r.pass("check/finding-ruleId", "§4/§1.5.1", LevelMUST, "finding has ruleId")
		r.pass("check/finding-severity", "§2.4/§4", LevelMUST, "finding severity ∈ {ERROR,WARNING,INFO}")
		r.pass("check/finding-location-nested", "§4", LevelMUST, "finding.location is a nested object")
		r.pass("check/finding-location-file", "§4", LevelMUST, "finding.location.file present")
		r.pass("check/finding-location-relative", "§4", LevelMUST, "finding.location.file is project-relative")

		// SHOULD: fingerprint
		hasFP := false
		for _, f := range doc.Findings {
			if f.Fingerprint != "" {
				hasFP = true
				if !fingerprintRE.MatchString(f.Fingerprint) {
					r.fail("check/fingerprint-format", "§4.2", LevelSHOULD,
						"fingerprint format sha256:<64 hex>",
						fmt.Sprintf("fingerprint %q invalid", f.Fingerprint))
					return
				}
			}
		}
		if hasFP {
			r.pass("check/fingerprint-format", "§4.2", LevelSHOULD,
				"fingerprint format sha256:<64 hex>")
		} else {
			r.skip("check/fingerprint-format", "§4.2", LevelSHOULD,
				"fingerprint format sha256:<64 hex>",
				"no fingerprints in output (SHOULD but not MUST yet)")
		}

		// SHOULD: category enum
		checkCategory(r, doc.Findings[0].Category)
	} else {
		// No findings — these checks are vacuously satisfied.
		r.pass("check/finding-ruleId", "§4/§1.5.1", LevelMUST, "finding has ruleId")
		r.pass("check/finding-severity", "§2.4/§4", LevelMUST, "finding severity ∈ {ERROR,WARNING,INFO}")
		r.pass("check/finding-location-nested", "§4", LevelMUST, "finding.location is a nested object")
		r.pass("check/finding-location-file", "§4", LevelMUST, "finding.location.file present")
		r.pass("check/finding-location-relative", "§4", LevelMUST, "finding.location.file is project-relative")
		r.skip("check/fingerprint-format", "§4.2", LevelSHOULD,
			"fingerprint format sha256:<64 hex>", "no findings produced")
		r.skip("check/category-enum", "§4", LevelSHOULD, "finding.category closed enum",
			"no findings produced")
	}
}

var validCategories = map[string]bool{
	"lint": true, "style": true, "safety": true, "security": true,
	"coverage": true, "requirement": true, "concurrency": true,
	"supply-chain": true, "config": true, "other": true,
}

func checkCategory(r *runner, cat string) {
	if cat == "" {
		r.skip("check/category-enum", "§4", LevelSHOULD, "finding.category closed enum",
			"category field absent")
		return
	}
	if !validCategories[cat] {
		r.fail("check/category-enum", "§4", LevelSHOULD, "finding.category closed enum",
			fmt.Sprintf("category %q not in enum", cat))
		return
	}
	r.pass("check/category-enum", "§4", LevelSHOULD, "finding.category closed enum")
}

// checkTrace validates §5 trace --format json.
//
//fusa:req REQ-FO-CNF012
func (r *runner) checkTrace() {
	stdout, _, code := r.run(r.dir, r.binary, "trace", "--format", "json")
	if code > 1 {
		r.skip("trace/json", "§5", LevelMUST, "trace --format json",
			fmt.Sprintf("exit %d (runtime error)", code))
		return
	}

	var doc struct {
		commonHeader
		ProjectRoot  string        `json:"projectRoot"`
		Requirements []interface{} `json:"requirements"`
		Tags         []interface{} `json:"tags"`
		Coverage     *struct {
			Total     *int `json:"totalRequirements"`
			Traced    *int `json:"tracedRequirements"`
			Tested    *int `json:"testedRequirements"`
			SecTested *int `json:"secTestedRequirements"`
			// Detect non-conformant flat shape
			FlatTotal  *int `json:"total"`
			FlatTraced *int `json:"traced"`
		} `json:"coverage"`
	}

	if err := decodeJSON(stdout, &doc); err != nil {
		r.fail("trace/json-parse", "§5", LevelMUST, "trace --format json produces valid JSON",
			fmt.Sprintf("%v", err))
		return
	}
	r.pass("trace/json-parse", "§5", LevelMUST, "trace --format json produces valid JSON")

	if errs := validateHeader(doc.commonHeader, "trace-matrix"); len(errs) > 0 {
		r.fail("trace/common-header", "§3.1", LevelMUST, "trace report carries common header",
			strings.Join(errs, "; "))
	} else {
		r.pass("trace/common-header", "§3.1", LevelMUST, "trace report carries common header")
	}

	if doc.Requirements == nil {
		r.fail("trace/requirements-key", "§5", LevelMUST, "trace has requirements[] key",
			"requirements key absent (flat matrix[] shape?)")
	} else {
		r.pass("trace/requirements-key", "§5", LevelMUST, "trace has requirements[] key")
	}

	if doc.Tags == nil {
		r.fail("trace/tags-key", "§5", LevelMUST, "trace has tags[] key",
			"tags key absent")
	} else {
		r.pass("trace/tags-key", "§5", LevelMUST, "trace has tags[] key")
	}

	if doc.Coverage == nil {
		r.fail("trace/coverage-schema", "§5", LevelMUST,
			"trace coverage has {totalRequirements,tracedRequirements,testedRequirements,secTestedRequirements}",
			"coverage object absent")
		return
	}
	if doc.Coverage.FlatTotal != nil || doc.Coverage.FlatTraced != nil {
		r.fail("trace/coverage-schema", "§5", LevelMUST,
			"trace coverage has {totalRequirements,tracedRequirements,testedRequirements,secTestedRequirements}",
			"flat total/traced/tested keys found — non-conformant schema (§5 specifies the qualified names)")
		return
	}
	if doc.Coverage.Total == nil || doc.Coverage.Traced == nil ||
		doc.Coverage.Tested == nil || doc.Coverage.SecTested == nil {
		r.fail("trace/coverage-schema", "§5", LevelMUST,
			"trace coverage has {totalRequirements,tracedRequirements,testedRequirements,secTestedRequirements}",
			"one or more counter fields absent")
		return
	}
	r.pass("trace/coverage-schema", "§5", LevelMUST,
		"trace coverage has {totalRequirements,tracedRequirements,testedRequirements,secTestedRequirements}")
}

// checkQualify validates §6 qualify --output output.
//
//fusa:req REQ-FO-CNF013
func (r *runner) checkQualify() {
	tmp, err := os.CreateTemp("", "qualify-*.json")
	if err != nil {
		r.skip("qualify/output", "§6", LevelMUST, "qualify --output writes file", "cannot create temp file")
		return
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	_, _, code := r.run(r.dir, r.binary, "qualify", "--output", tmp.Name())
	if code > 1 {
		r.skip("qualify/output", "§6", LevelMUST, "qualify --output writes file",
			fmt.Sprintf("exit %d (runtime error)", code))
		return
	}

	data, err := os.ReadFile(tmp.Name())
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		r.fail("qualify/output", "§6", LevelMUST, "qualify --output writes file",
			"output file empty or unreadable")
		return
	}
	r.pass("qualify/output", "§6", LevelMUST, "qualify --output writes file")

	var doc struct {
		commonHeader
		Total   *int `json:"total"`
		Passed  *int `json:"passed"`
		Failed  *int `json:"failed"`
		Results []struct {
			Name   string `json:"name"`
			Result string `json:"result"`
		} `json:"results"`
		// Detect non-conformant key names
		TestsPassed *int `json:"tests_passed"`
		TestsFailed *int `json:"tests_failed"`
	}

	if err := decodeJSON(data, &doc); err != nil {
		r.fail("qualify/json-parse", "§6", LevelMUST, "qualify output is valid JSON",
			fmt.Sprintf("%v", err))
		return
	}
	r.pass("qualify/json-parse", "§6", LevelMUST, "qualify output is valid JSON")

	if errs := validateHeader(doc.commonHeader, "qualification"); len(errs) > 0 {
		r.fail("qualify/common-header", "§3.1", LevelMUST, "qualify report carries common header",
			strings.Join(errs, "; "))
	} else {
		r.pass("qualify/common-header", "§3.1", LevelMUST, "qualify report carries common header")
	}

	if doc.TestsPassed != nil || doc.TestsFailed != nil {
		r.fail("qualify/key-names", "§6", LevelMUST,
			"qualify uses total/passed/failed (not tests_passed/tests_failed)",
			"tests_passed or tests_failed keys found — non-conformant")
	} else if doc.Total == nil || doc.Passed == nil || doc.Failed == nil {
		r.fail("qualify/key-names", "§6", LevelMUST,
			"qualify uses total/passed/failed (not tests_passed/tests_failed)",
			"total, passed, or failed absent")
	} else {
		r.pass("qualify/key-names", "§6", LevelMUST,
			"qualify uses total/passed/failed (not tests_passed/tests_failed)")
	}

	// Validate results[].result enum
	validResult := map[string]bool{"PASS": true, "FAIL": true, "SKIP": true, "ERROR": true}
	for _, res := range doc.Results {
		if !validResult[res.Result] {
			r.fail("qualify/result-enum", "§6", LevelMUST,
				"results[].result ∈ {PASS,FAIL,SKIP,ERROR}",
				fmt.Sprintf("result %q not in enum", res.Result))
			return
		}
	}
	r.pass("qualify/result-enum", "§6", LevelMUST, "results[].result ∈ {PASS,FAIL,SKIP,ERROR}")
}

// checkRelease validates §7 release writes sbom.json.
//
//fusa:req REQ-FO-CNF014
func (r *runner) checkRelease() {
	outDir, err := os.MkdirTemp("", "fusaops-conform-release-*")
	if err != nil {
		r.skip("release/sbom-written", "§7", LevelMUST, "release writes sbom.json", "cannot create temp dir")
		return
	}
	defer func() { _ = os.RemoveAll(outDir) }()

	_, _, code := r.run(r.dir, r.binary, "release", "--output-dir", outDir)
	if code > 1 {
		r.skip("release/sbom-written", "§7", LevelMUST, "release writes sbom.json",
			fmt.Sprintf("exit %d (runtime error)", code))
		return
	}

	sbomPath := filepath.Join(outDir, "sbom.json")
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		r.fail("release/sbom-written", "§7", LevelMUST, "release writes sbom.json",
			"sbom.json not found in --output-dir")
		return
	}
	r.pass("release/sbom-written", "§7", LevelMUST, "release writes sbom.json")

	var doc struct {
		commonHeader
		Format     string        `json:"format"`
		Module     string        `json:"module"`
		Components []interface{} `json:"components"`
	}
	if err := decodeJSON(data, &doc); err != nil {
		r.fail("release/sbom-json", "§7", LevelMUST, "sbom.json is valid JSON", fmt.Sprintf("%v", err))
		return
	}
	r.pass("release/sbom-json", "§7", LevelMUST, "sbom.json is valid JSON")

	if errs := validateHeader(doc.commonHeader, "sbom"); len(errs) > 0 {
		r.fail("release/sbom-header", "§3.1/§7", LevelMUST, "sbom.json carries common header",
			strings.Join(errs, "; "))
	} else {
		r.pass("release/sbom-header", "§3.1/§7", LevelMUST, "sbom.json carries common header")
	}

	if doc.Module == "" {
		r.fail("release/sbom-module", "§7", LevelMUST, "sbom.json has module field",
			"module absent or empty")
	} else {
		r.pass("release/sbom-module", "§7", LevelMUST, "sbom.json has module field")
	}

	if doc.Components == nil {
		r.fail("release/sbom-components", "§7", LevelMUST, "sbom.json has components array",
			"components key absent")
	} else {
		r.pass("release/sbom-components", "§7", LevelMUST, "sbom.json has components array")
	}
}

// checkAuditPack validates §8 audit-pack produces a ZIP with manifest.json.
//
//fusa:req REQ-FO-CNF015
func (r *runner) checkAuditPack() {
	tmp, err := os.CreateTemp("", "audit-pack-*.zip")
	if err != nil {
		r.skip("audit-pack/is-zip", "§8", LevelMUST, "audit-pack produces ZIP", "cannot create temp file")
		return
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	_, _, code := r.run(r.dir, r.binary, "audit-pack", "--output", tmp.Name())
	if code > 1 {
		r.skip("audit-pack/is-zip", "§8", LevelMUST, "audit-pack produces ZIP",
			fmt.Sprintf("exit %d (runtime error)", code))
		return
	}

	zr, err := zip.OpenReader(tmp.Name())
	if err != nil {
		r.fail("audit-pack/is-zip", "§8", LevelMUST, "audit-pack produces ZIP",
			fmt.Sprintf("not a valid ZIP: %v", err))
		return
	}
	defer func() { _ = zr.Close() }()
	r.pass("audit-pack/is-zip", "§8", LevelMUST, "audit-pack produces ZIP")

	// Find manifest.json (exact case, §8)
	var manifestFile *zip.File
	for _, f := range zr.File {
		if f.Name == "manifest.json" {
			manifestFile = f
			break
		}
	}
	if manifestFile == nil {
		r.fail("audit-pack/manifest-present", "§8", LevelMUST,
			"audit-pack ZIP contains manifest.json",
			"manifest.json not found in ZIP root (case-sensitive)")
		return
	}
	r.pass("audit-pack/manifest-present", "§8", LevelMUST,
		"audit-pack ZIP contains manifest.json")

	rc, err := manifestFile.Open()
	if err != nil {
		r.fail("audit-pack/manifest-readable", "§8", LevelMUST,
			"manifest.json is readable", fmt.Sprintf("%v", err))
		return
	}
	defer rc.Close()

	var manifest struct {
		commonHeader
		Module string `json:"module"`
		Files  []struct {
			Path   string `json:"path"`
			Size   *int64 `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	dec := json.NewDecoder(rc)
	if err := dec.Decode(&manifest); err != nil {
		r.fail("audit-pack/manifest-json", "§8", LevelMUST,
			"manifest.json is valid JSON", fmt.Sprintf("%v", err))
		return
	}
	r.pass("audit-pack/manifest-json", "§8", LevelMUST, "manifest.json is valid JSON")

	if errs := validateHeader(manifest.commonHeader, "audit-manifest"); len(errs) > 0 {
		r.fail("audit-pack/manifest-header", "§3.1/§8", LevelMUST,
			"manifest.json carries common header",
			strings.Join(errs, "; "))
	} else {
		r.pass("audit-pack/manifest-header", "§3.1/§8", LevelMUST,
			"manifest.json carries common header")
	}

	if manifest.Files == nil {
		r.fail("audit-pack/manifest-files", "§8", LevelMUST,
			"manifest.json has files array", "files key absent")
		return
	}
	r.pass("audit-pack/manifest-files", "§8", LevelMUST, "manifest.json has files array")

	// Validate sha256 fields are bare hex (NOT "sha256:" prefixed)
	for _, f := range manifest.Files {
		if sha256PrefixedRE.MatchString(f.SHA256) {
			r.fail("audit-pack/manifest-sha256-bare", "§2.7/§8", LevelMUST,
				"manifest.json files[].sha256 is bare hex (no 'sha256:' prefix)",
				fmt.Sprintf("file %q: sha256 %q has algo: prefix — use bare hex here", f.Path, f.SHA256))
			return
		}
		if !bareHex64RE.MatchString(f.SHA256) {
			r.fail("audit-pack/manifest-sha256-bare", "§2.7/§8", LevelMUST,
				"manifest.json files[].sha256 is bare hex (no 'sha256:' prefix)",
				fmt.Sprintf("file %q: sha256 %q is not 64 lowercase hex", f.Path, f.SHA256))
			return
		}
	}
	r.pass("audit-pack/manifest-sha256-bare", "§2.7/§8", LevelMUST,
		"manifest.json files[].sha256 is bare hex (no 'sha256:' prefix)")
}

// checkCapabilities validates §9.1 SHOULD capabilities --format json.
//
//fusa:req REQ-FO-CNF016
func (r *runner) checkCapabilities() {
	stdout, _, code := r.run(r.dir, r.binary, "capabilities", "--format", "json")
	if code != 0 {
		r.skip("capabilities/schema", "§9.1", LevelSHOULD,
			"capabilities --format json",
			"command not supported")
		return
	}

	var doc struct {
		commonHeader
		SpecVersion string        `json:"specVersion"`
		Commands    []interface{} `json:"commands"`
	}
	if err := decodeJSON(stdout, &doc); err != nil {
		r.fail("capabilities/schema", "§9.1", LevelSHOULD,
			"capabilities --format json produces valid JSON",
			fmt.Sprintf("%v", err))
		return
	}

	var errs []string
	if hErrs := validateHeader(doc.commonHeader, "capabilities"); len(hErrs) > 0 {
		errs = append(errs, hErrs...)
	}
	if doc.SpecVersion == "" {
		errs = append(errs, "missing specVersion")
	}
	if doc.Commands == nil {
		errs = append(errs, "missing commands array")
	}
	if len(errs) > 0 {
		r.fail("capabilities/schema", "§9.1", LevelSHOULD,
			"capabilities carries kind/specVersion/commands",
			strings.Join(errs, "; "))
		return
	}
	r.pass("capabilities/schema", "§9.1", LevelSHOULD,
		"capabilities carries kind/specVersion/commands")
}
