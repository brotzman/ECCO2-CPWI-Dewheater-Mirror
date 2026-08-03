package main

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsStatusWindowStartupContract(t *testing.T) {
	b, err := os.ReadFile("main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	checks := map[string]string{
		"GUI thread is locked":                            "runtime.LockOSThread()",
		"status window has visible failure diagnostics":   "Statusfenster konnte nicht erstellt werden",
		"startup failures use a native dialog":            "pMessageBoxW.Call",
		"window is shown explicitly":                      "pShowWindow.Call(hwndMain, SW_SHOWNORMAL)",
		"message loop is checked for errors":              "Windows-Nachrichtenschleife fehlgeschlagen",
		"polling starts only after the window is visible": "go pollLoop()",
		"startup log records visible window":              "WINDOW_VISIBLE",
		"configuration is per-user":                       "LOCALAPPDATA",
	}
	for name, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("%s: missing %q", name, want)
		}
	}
	lock := strings.Index(s, "runtime.LockOSThread()")
	create := strings.Index(s, "createdWindow, _, createErr := pCreateWindowExW.Call")
	getMessage := strings.LastIndex(s, "pGetMessageW.Call")
	if lock < 0 || create < 0 || getMessage < 0 || !(lock < create && create < getMessage) {
		t.Fatalf("Win32 owner-thread order is invalid: lock=%d create=%d getMessage=%d", lock, create, getMessage)
	}
	visible := strings.Index(s, "WINDOW_VISIBLE")
	polling := strings.Index(s, "go pollLoop()")
	if visible < 0 || polling < 0 || visible > polling {
		t.Fatalf("network polling must start after the status window is visible")
	}
	if strings.Contains(s, "if hwndMain == 0 {\n\t\treturn") {
		t.Fatal("silent status-window creation failure remains")
	}
}

func TestWindowsMinMaxInfoAvoidsUnsafeUintptrConversion(t *testing.T) {
	b, err := os.ReadFile("main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "(*MINMAXINFO)(unsafe.Pointer(lp))") {
		t.Fatal("WM_GETMINMAXINFO still converts lParam directly to unsafe.Pointer")
	}
	for _, want := range []string{"func setMinimumTrackSize", "pRtlMoveMemory.Call", "setMinimumTrackSize(lp, 850, 720)"} {
		if !strings.Contains(s, want) {
			t.Errorf("safe WM_GETMINMAXINFO handling missing %q", want)
		}
	}
}
