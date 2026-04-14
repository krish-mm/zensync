package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func exportZen(out io.Writer) error {
	printSection(out, "EXPORT")
	printInfo(out, "Checking Zen profile directory")

	supportDir, err := detectZenSupportDir()
	if err != nil {
		return err
	}
	printPath(out, "Source", supportDir)

	printInfo(out, "Ensuring Zen Browser is closed")
	if err := ensureZenClosed(out); err != nil {
		return err
	}

	downloadsDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	downloadsDir = filepath.Join(downloadsDir, "Downloads")
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure Downloads directory: %w", err)
	}

	ts := time.Now().Format("2006-01-02_15-04-05")
	outZip := filepath.Join(downloadsDir, fmt.Sprintf("zen_backup_%s.zip", ts))

	tempDir, err := os.MkdirTemp("", "zensync-export-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	rootName := filepath.Base(supportDir)
	stagedRoot := filepath.Join(tempDir, rootName)
	if err := copyDir(supportDir, stagedRoot); err != nil {
		return fmt.Errorf("failed to stage Zen data: %w", err)
	}

	manifest := filepath.Join(tempDir, "manifest.txt")
	manifestContent := fmt.Sprintf(
		"created_at=%s\nsource_dir=%s\nroot_name=%s\n",
		time.Now().UTC().Format(time.RFC3339),
		supportDir,
		rootName,
	)
	if err := os.WriteFile(manifest, []byte(manifestContent), 0o644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	printInfo(out, "Creating backup zip")
	if err := zipPaths(outZip, []string{stagedRoot, manifest}, tempDir); err != nil {
		return fmt.Errorf("failed to create zip: %w", err)
	}

	printSuccess(out, "Backup created")
	printPath(out, "File", outZip)
	return nil
}

func importZen(out io.Writer, zipArg string) error {
	printSection(out, "IMPORT")

	zipPath, err := expandPath(zipArg)
	if err != nil {
		return err
	}
	printPath(out, "Backup", zipPath)

	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("zip not found: %s", zipPath)
	}

	printInfo(out, "Ensuring Zen Browser is closed")
	if err := ensureZenClosed(out); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "zensync-import-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	printInfo(out, "Extracting backup")
	if err := unzip(zipPath, tempDir); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	restoreRoot, err := findRestoreRoot(tempDir)
	if err != nil {
		return err
	}
	printPath(out, "Restore root", restoreRoot)

	appSupport := filepath.Join(mustHomeDir(), "Library", "Application Support")
	targetDir := filepath.Join(appSupport, filepath.Base(restoreRoot))
	if existingDir, err := detectZenSupportDir(); err == nil {
		targetDir = existingDir
	}
	printPath(out, "Target", targetDir)

	if _, err := os.Stat(targetDir); err == nil {
		ts := time.Now().Format("2006-01-02_15-04-05")
		safetyBackup := targetDir + ".pre_restore_" + ts
		printInfo(out, "Moving existing data to safety backup")
		if err := os.Rename(targetDir, safetyBackup); err != nil {
			return fmt.Errorf("failed to create safety backup: %w", err)
		}
		printPath(out, "Safety backup", safetyBackup)
	}

	printInfo(out, "Restoring profile data")
	if err := copyDir(restoreRoot, targetDir); err != nil {
		return fmt.Errorf("failed to restore Zen data: %w", err)
	}

	printSuccess(out, "Restore completed")
	printPath(out, "Directory", targetDir)
	return nil
}

func detectZenSupportDir() (string, error) {
	home := mustHomeDir()
	candidates := []string{
		filepath.Join(home, "Library", "Application Support", "zen"),
		filepath.Join(home, "Library", "Application Support", "Zen"),
		filepath.Join(home, "Library", "Application Support", "Zen Browser"),
		filepath.Join(home, "Library", "Application Support", "ZenBrowser"),
	}

	for _, candidate := range candidates {
		if hasZenProfileStructure(candidate) {
			return candidate, nil
		}
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", errors.New("could not find Zen support directory under ~/Library/Application Support")
}

func hasZenProfileStructure(dir string) bool {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return false
	}

	if _, err := os.Stat(filepath.Join(dir, "profiles.ini")); err == nil {
		return true
	}
	if info, err := os.Stat(filepath.Join(dir, "Profiles")); err == nil && info.IsDir() {
		return true
	}

	return false
}

func ensureZenClosed(out io.Writer) error {
	if !isZenRunning() {
		printInfo(out, "Zen Browser is already closed")
		return nil
	}

	printInfo(out, "Closing Zen Browser")
	_ = exec.Command("osascript", "-e", `tell application "Zen" to quit`).Run()
	time.Sleep(3 * time.Second)

	if !isZenRunning() {
		return nil
	}

	printInfo(out, "Force closing Zen Browser process")
	_ = exec.Command("pkill", "-x", "Zen").Run()
	time.Sleep(2 * time.Second)

	if isZenRunning() {
		return errors.New("Zen Browser is still running; please close it and retry")
	}

	return nil
}

func isZenRunning() bool {
	if err := exec.Command("pgrep", "-x", "Zen").Run(); err == nil {
		return true
	}
	if err := exec.Command("pgrep", "-f", "/Applications/Zen.app").Run(); err == nil {
		return true
	}
	return false
}
