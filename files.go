package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func zipPaths(zipFile string, sources []string, baseDir string) error {
	out, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, source := range sources {
		source = filepath.Clean(source)
		info, err := os.Stat(source)
		if err != nil {
			return err
		}

		if info.IsDir() {
			err = filepath.WalkDir(source, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}

				rel, err := filepath.Rel(baseDir, path)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)

				if d.IsDir() {
					if rel == "." {
						return nil
					}
					_, err := zw.Create(rel + "/")
					return err
				}

				return addFileToZip(zw, path, rel)
			})
			if err != nil {
				return err
			}
			continue
		}

		rel, err := filepath.Rel(baseDir, source)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if err := addFileToZip(zw, source, rel); err != nil {
			return err
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, filePath, zipPath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, f)
	return err
}

func unzip(srcZip, dest string) error {
	zr, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		outPath := filepath.Join(dest, f.Name)
		cleanDest := filepath.Clean(dest) + string(os.PathSeparator)
		cleanOut := filepath.Clean(outPath)
		if !strings.HasPrefix(cleanOut, cleanDest) && cleanOut != filepath.Clean(dest) {
			return fmt.Errorf("invalid zip entry path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			// ZIPs created on some platforms (for example FAT metadata) can
			// report directory mode without execute bits, which makes nested
			// extraction fail with permission denied.
			if err := os.MkdirAll(outPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		fileMode := f.Mode()
		if fileMode.Perm() == 0 {
			fileMode = 0o644
		}

		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(outFile, rc)
		closeErr1 := outFile.Close()
		closeErr2 := rc.Close()

		if copyErr != nil {
			return copyErr
		}
		if closeErr1 != nil {
			return closeErr1
		}
		if closeErr2 != nil {
			return closeErr2
		}
	}

	return nil
}

func findRestoreRoot(extracted string) (string, error) {
	candidates := []string{
		filepath.Join(extracted, "zen"),
		filepath.Join(extracted, "Zen"),
		filepath.Join(extracted, "Zen Browser"),
		filepath.Join(extracted, "ZenBrowser"),
	}

	for _, candidate := range candidates {
		if hasZenProfileStructure(candidate) {
			return candidate, nil
		}
	}

	var matches []string
	err := filepath.WalkDir(extracted, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if hasZenProfileStructure(path) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		closest := matches[0]
		closestDepth := pathDepth(extracted, closest)
		for _, match := range matches[1:] {
			depth := pathDepth(extracted, match)
			if depth < closestDepth {
				closest = match
				closestDepth = depth
			}
		}
		return closest, nil
	}

	return "", errors.New("could not find Zen data inside backup zip")
}

func pathDepth(base, path string) int {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(os.PathSeparator))
}

func copyDir(src, dst string) error {
	src = filepath.Clean(src)
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			_ = os.RemoveAll(target)
			return os.Symlink(linkTarget, target)
		}

		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}

func expandPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("path cannot be empty")
	}

	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(p, "~/")), nil
	}

	return filepath.Abs(p)
}

func mustHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return home
}
