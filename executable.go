package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func findExecutable(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("未找到命令: %s", name)
	}
	programFiles := os.Getenv("ProgramFiles")
	localAppData := os.Getenv("LOCALAPPDATA")
	systemDrive := os.Getenv("SystemDrive")
	if systemDrive == "" {
		systemDrive = "C:"
	}
	standardProgramFiles := filepath.Join(systemDrive+string(os.PathSeparator), "Program Files")
	var candidates []string
	switch name {
	case "git":
		candidates = []string{
			filepath.Join(programFiles, "Git", "cmd", "git.exe"),
			filepath.Join(standardProgramFiles, "Git", "cmd", "git.exe"),
			filepath.Join(localAppData, "Programs", "Git", "cmd", "git.exe"),
		}
	case "git-lfs":
		candidates = []string{
			filepath.Join(programFiles, "Git", "mingw64", "bin", "git-lfs.exe"),
			filepath.Join(standardProgramFiles, "Git", "mingw64", "bin", "git-lfs.exe"),
			filepath.Join(programFiles, "Git LFS", "git-lfs.exe"),
		}
	case "gh":
		candidates = []string{
			filepath.Join(programFiles, "GitHub CLI", "gh.exe"),
			filepath.Join(standardProgramFiles, "GitHub CLI", "gh.exe"),
			filepath.Join(localAppData, "Programs", "GitHub CLI", "gh.exe"),
		}
	case "glab":
		candidates = []string{
			filepath.Join(programFiles, "glab", "glab.exe"),
			filepath.Join(standardProgramFiles, "glab", "glab.exe"),
			filepath.Join(localAppData, "glab", "glab.exe"),
			filepath.Join(localAppData, "Programs", "glab", "glab.exe"),
		}
	default:
		return "", fmt.Errorf("未找到命令: %s", name)
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("未找到命令: %s", name)
}
