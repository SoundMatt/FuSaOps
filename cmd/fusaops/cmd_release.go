package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/release"
	"github.com/SoundMatt/FuSaOps/sbom"
)

// runRelease generates the multi-language SBOM, build provenance, and artifact manifest.
//
//fusa:req REQ-FO-CLI065
//fusa:req REQ-FO-CLI081
func runRelease(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops release", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops release [flags]\n\n")
		fmt.Fprintf(stderr, "Generate the cross-language SBOM, build provenance, and artifact manifest.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir       = fs.String("dir", "", "project root directory (default: current directory)")
		outputDir = fs.String("output-dir", "", "output directory for generated files (default: project root)")
		builder   = fs.String("builder", "", "builder identifier (e.g. 'github-actions'); auto-detected from CI env vars when empty")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops release: get working directory: %v\n", err)
			return 1
		}
	}

	outDir := *outputDir
	if outDir == "" {
		outDir = projectRoot
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		fmt.Fprintf(stderr, "fusaops release: create output directory: %v\n", err)
		return 1
	}

	// Step 1: Cross-language SBOM.
	_, opts, _, err := loadOptions(projectRoot, "", stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops release: load options: %v\n", err)
		return 1
	}
	agg, err := orchestrator.New(nil).RunSBOM(context.Background(), projectRoot, opts)
	if err != nil && !errors.Is(err, fusaops.ErrNoAdapters) {
		fmt.Fprintf(stderr, "fusaops release: build SBOM: %v\n", err)
		return 1
	}
	sbomPath := filepath.Join(outDir, release.SBOMFile)
	if agg != nil {
		if sbomErr := sbom.RenderToFile(agg, "json", sbomPath); sbomErr != nil {
			fmt.Fprintf(stderr, "fusaops release: save SBOM: %v\n", sbomErr)
			return 1
		}
		fmt.Fprintf(stdout, "SBOM written to %s (%d packages)\n", sbomPath, agg.TotalPackages)
	} else {
		fmt.Fprintf(stdout, "SBOM: no adapters detected, skipping\n")
		sbomPath = ""
	}

	// Step 2: Build provenance.
	prov, err := release.BuildProvenance(context.Background(), projectRoot, *builder)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops release: build provenance: %v\n", err)
		return 1
	}
	provPath := filepath.Join(outDir, release.ProvenanceFile)
	if saveErr := release.SaveJSON(provPath, prov); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops release: save provenance: %v\n", saveErr)
		return 1
	}
	if renderErr := release.RenderProvenance(stdout, prov, "text"); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops release: render provenance: %v\n", renderErr)
		return 1
	}
	fmt.Fprintf(stdout, "Provenance written to %s\n", provPath)

	// Step 3: Artifact manifest.
	var manifestPaths []string
	if sbomPath != "" {
		manifestPaths = append(manifestPaths, sbomPath)
	}
	manifestPaths = append(manifestPaths, provPath)

	manifest, err := release.BuildManifest(manifestPaths)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops release: build manifest: %v\n", err)
		return 1
	}
	manifestPath := filepath.Join(outDir, release.ManifestFile)
	if saveErr := release.SaveJSON(manifestPath, manifest); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops release: save manifest: %v\n", saveErr)
		return 1
	}
	if renderErr := release.RenderManifest(stdout, manifest, "text"); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops release: render manifest: %v\n", renderErr)
		return 1
	}
	fmt.Fprintf(stdout, "Artifact manifest written to %s\n", manifestPath)

	return 0
}
