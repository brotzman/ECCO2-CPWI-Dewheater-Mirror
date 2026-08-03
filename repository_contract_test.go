package main

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

func mustReadContract(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestRepositoryBuildAndInstallerContracts(t *testing.T) {
	workflow := mustReadContract(t, ".github/workflows/build-release.yml")
	for _, want := range []string{
		"actions/checkout@v7", "actions/setup-go@v7", "actions/setup-dotnet@v6",
		"actions/upload-artifact@v7", "actions/download-artifact@v8",
		"go test -race ./...", "-Mode Setup", "-Mode Msi", "publish_release",
		"GH_REPO: ${{ github.repository }}", "ECCO2-CPWI-Dew-Mirror-1.0.1-Build-Logs", "timeout-minutes", "shell: pwsh",
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("workflow missing %q", want)
		}
	}
	buildRelease := mustReadContract(t, "scripts/Build-Release.ps1")
	if !strings.Contains(buildRelease, "go build -tags installerpayload") {
		t.Error("Build-Release.ps1 must compile the setup with the installerpayload tag")
	}
	setup := mustReadContract(t, "cmd/setup/main_windows.go")
	for _, want := range []string{"//go:build windows && installerpayload", "//go:embed payload/*", "IsUserAnAdmin", "-Verb RunAs", "ECCO2CPWIDewMirrorUninstall.exe", "WScript.Shell", "QuietUninstallString", "if !admin()", "h, err := acquireMutex()", "os.Exit(1)", "taskkill.exe", "argumentClause := \"\"", "if argumentLine != \"\""} {
		if !strings.Contains(setup, want) {
			t.Errorf("native setup missing %q", want)
		}
	}
	if strings.Index(setup, "if !admin()") > strings.Index(setup, "h, err := acquireMutex()") {
		t.Error("native setup must elevate before acquiring the machine-wide setup mutex")
	}
	packageWxs := mustReadContract(t, "installer/wix/Package.wxs")
	var doc any
	if err := xml.Unmarshal([]byte(packageWxs), &doc); err != nil {
		t.Fatalf("Package.wxs is invalid XML: %v", err)
	}
	for _, want := range []string{"ProgramFiles64Folder", "DesktopFolder", "ProgramMenuFolder", "MajorUpgrade", "ECCO2CPWIDewMirror.exe"} {
		if !strings.Contains(packageWxs, want) {
			t.Errorf("Package.wxs missing %q", want)
		}
	}
}
