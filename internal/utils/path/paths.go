package pathutil

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var windowsUnixDrivePathRe = regexp.MustCompile(`^/([a-zA-Z])/(.*)$`)
var unixUserHomePathRe = regexp.MustCompile(`^/(Users|home)/[^/]+(?:/(.*))?$`)

func NormalizeHostPath(input string) string {
	pathValue := strings.TrimSpace(input)
	if pathValue == "" {
		if wd, err := os.Getwd(); err == nil {
			pathValue = wd
		}
	}

	pathValue = os.ExpandEnv(pathValue)
	pathValue = expandHome(pathValue)

	if runtime.GOOS == "windows" {
		pathValue = normalizeWindowsPath(pathValue)
	} else {
		pathValue = normalizeUnixPath(pathValue)
	}

	if !filepath.IsAbs(pathValue) {
		if wd, err := os.Getwd(); err == nil {
			pathValue = filepath.Join(wd, pathValue)
		}
	}

	return filepath.Clean(pathValue)
}

func NormalizeContainerPath(input string) string {
	pathValue := strings.TrimSpace(input)
	if pathValue == "" {
		return "/"
	}

	pathValue = strings.ReplaceAll(pathValue, `\`, "/")
	return path.Clean("/" + strings.TrimLeft(pathValue, "/"))
}

func normalizeWindowsPath(input string) string {
	pathValue := input
	if mapped, ok := mapUnixUserHomeToWindowsHome(pathValue); ok {
		return filepath.Clean(mapped)
	}

	if matches := windowsUnixDrivePathRe.FindStringSubmatch(pathValue); len(matches) == 3 {
		drive := strings.ToUpper(matches[1])
		rest := strings.TrimLeft(strings.ReplaceAll(matches[2], "/", `\`), `\`)
		pathValue = filepath.Clean(drive + `:\` + rest)
		return remapMismatchedWindowsUserHome(pathValue)
	}

	pathValue = filepath.FromSlash(pathValue)

	if strings.HasPrefix(pathValue, `\`) && filepath.VolumeName(pathValue) == "" {
		drive := "C:"
		if wd, err := os.Getwd(); err == nil {
			if volume := filepath.VolumeName(wd); volume != "" {
				drive = volume
			}
		}
		pathValue = drive + `\` + strings.TrimLeft(pathValue, `\`)
	}

	return remapMismatchedWindowsUserHome(filepath.Clean(pathValue))
}

func normalizeUnixPath(input string) string {
	pathValue := strings.ReplaceAll(input, `\`, "/")
	return filepath.Clean(pathValue)
}

func expandHome(input string) string {
	if input == "~" || strings.HasPrefix(input, "~/") || strings.HasPrefix(input, `~\`) {
		if homeDir, err := os.UserHomeDir(); err == nil {
			if input == "~" {
				return homeDir
			}
			return filepath.Join(homeDir, strings.TrimLeft(input[1:], `/\`))
		}
	}
	return input
}

func mapUnixUserHomeToWindowsHome(input string) (string, bool) {
	matches := unixUserHomePathRe.FindStringSubmatch(input)
	if len(matches) == 0 {
		return "", false
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}

	if len(matches) < 3 || strings.TrimSpace(matches[2]) == "" {
		return homeDir, true
	}

	return filepath.Join(homeDir, filepath.FromSlash(matches[2])), true
}

func remapMismatchedWindowsUserHome(input string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return input
	}

	cleanInput := filepath.Clean(input)
	cleanHome := filepath.Clean(homeDir)

	inputParts := splitWindowsPathParts(cleanInput)
	homeParts := splitWindowsPathParts(cleanHome)

	if len(inputParts) < 3 || len(homeParts) < 3 {
		return cleanInput
	}

	if !strings.EqualFold(inputParts[1], "Users") || !strings.EqualFold(homeParts[1], "Users") {
		return cleanInput
	}

	if strings.EqualFold(inputParts[2], homeParts[2]) {
		return cleanInput
	}

	rest := []string{}
	if len(inputParts) > 3 {
		rest = inputParts[3:]
	}

	remapped := filepath.Join(cleanHome, filepath.Join(rest...))
	return filepath.Clean(remapped)
}

func splitWindowsPathParts(pathValue string) []string {
	p := strings.ReplaceAll(pathValue, "/", `\`)
	p = strings.Trim(p, `\`)
	if p == "" {
		return nil
	}
	return strings.Split(p, `\`)
}
