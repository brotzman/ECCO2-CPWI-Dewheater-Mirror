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
		"Test-ReleaseArtifacts.ps1", "setup-test-summary.log", "msi-test-summary.log", "WORKFLOW FAILURE:",
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("workflow missing %q", want)
		}
	}
	buildRelease := mustReadContract(t, "scripts/Build-Release.ps1")
	if !strings.Contains(buildRelease, "go build -tags installerpayload") {
		t.Error("Build-Release.ps1 must compile the setup with the installerpayload tag")
	}
	for _, want := range []string{"'artifacts'", "'ci-input'", "ZipFile]::CreateFromDirectory", "(Join-Path $root 'LICENSE')", "assets\\ECCO2CPWIDewMirror.ico", "ECCO2CPWIDewMirror.ico"} {
		if !strings.Contains(buildRelease, want) {
			t.Errorf("source packaging contract missing %q", want)
		}
	}
	installerTest := mustReadContract(t, "scripts/Test-Installer.ps1")
	for _, want := range []string{"Start-menu shortcut missing", "Uninstall registration missing", "Installed document missing", "Installed icon missing", "FAIL: $Mode installer test", "Matching uninstall registrations", "DiagnosticDirectory $logs", "Verify-Uninstalled", "Remove-Item $exe -Force", "Get-SafePropertyValue", "PSObject.Properties[$PropertyName]"} {
		if !strings.Contains(installerTest, want) {
			t.Errorf("installer test contract missing %q", want)
		}
	}
	for _, forbidden := range []string{"$_.DisplayName", "$_.DisplayVersion"} {
		if strings.Contains(installerTest, forbidden) {
			t.Errorf("installer registry scan must not access optional properties directly: %q", forbidden)
		}
	}
	releaseTest := mustReadContract(t, "scripts/Test-ReleaseArtifacts.ps1")
	for _, want := range []string{".gitignore", "LICENSE", "artifacts/", "Checksum mismatch", "ECCO2CPWIDewMirror.ico", "assets/ECCO2CPWIDewMirror.png"} {
		if !strings.Contains(releaseTest, want) {
			t.Errorf("release artifact test missing %q", want)
		}
	}
	setup := mustReadContract(t, "cmd/setup/main_windows.go")
	for _, want := range []string{"//go:build windows && installerpayload", "//go:embed payload/*", "IsUserAnAdmin", "-Verb RunAs", "ECCO2CPWIDewMirrorUninstall.exe", "WScript.Shell", "QuietUninstallString", "ECCO2CPWIDewMirror.ico", "$s.IconLocation", "if !admin()", "h, err := acquireMutex()", "os.Exit(1)", "taskkill.exe", "argumentClause := \"\"", "if argumentLine != \"\""} {
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
	for _, want := range []string{"ProgramFiles64Folder", "DesktopFolder", "ProgramMenuFolder", "MajorUpgrade", "ECCO2CPWIDewMirror.exe", "ECCO2CPWIDewMirror.ico", "CmpAppIcon", "ARPPRODUCTICON", "Icon=\"AppShortcutIcon\"", "CmpLicense", "LICENSE"} {
		if !strings.Contains(packageWxs, want) {
			t.Errorf("Package.wxs missing %q", want)
		}
	}
}
