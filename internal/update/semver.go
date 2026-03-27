package update

import (
	"fmt"
	"strconv"
	"strings"
)

type parsedSemver struct {
	major      int
	minor      int
	patch      int
	prerelease []string
}

func isNewerVersion(current, latest string) bool {
	c, err := parseSemver(current)
	if err != nil {
		return false
	}
	l, err := parseSemver(latest)
	if err != nil {
		return false
	}
	return compareSemver(c, l) < 0
}

func normalizeVersion(v string) string {
	trimmed := strings.TrimSpace(v)
	trimmed = strings.TrimPrefix(trimmed, "v")
	return trimmed
}

func parseSemver(v string) (parsedSemver, error) {
	clean := normalizeVersion(v)
	if clean == "" {
		return parsedSemver{}, fmt.Errorf("version is empty")
	}

	partsPlus := strings.SplitN(clean, "+", 2)
	core := partsPlus[0]

	prerelease := []string{}
	partsPre := strings.SplitN(core, "-", 2)
	core = partsPre[0]
	if len(partsPre) == 2 && partsPre[1] != "" {
		prerelease = strings.Split(partsPre[1], ".")
	}

	coreParts := strings.Split(core, ".")
	if len(coreParts) != 3 {
		return parsedSemver{}, fmt.Errorf("invalid semantic version: %q", v)
	}

	major, err := strconv.Atoi(coreParts[0])
	if err != nil {
		return parsedSemver{}, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(coreParts[1])
	if err != nil {
		return parsedSemver{}, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := strconv.Atoi(coreParts[2])
	if err != nil {
		return parsedSemver{}, fmt.Errorf("invalid patch version: %w", err)
	}

	return parsedSemver{
		major:      major,
		minor:      minor,
		patch:      patch,
		prerelease: prerelease,
	}, nil
}

func compareSemver(a, b parsedSemver) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}

	return comparePrerelease(a.prerelease, b.prerelease)
}

func comparePrerelease(a, b []string) int {
	// Stable versions have higher precedence than prerelease versions.
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return 1
	}
	if len(b) == 0 {
		return -1
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(a) {
			return -1
		}
		if i >= len(b) {
			return 1
		}

		av := a[i]
		bv := b[i]
		an, aErr := strconv.Atoi(av)
		bn, bErr := strconv.Atoi(bv)

		switch {
		case aErr == nil && bErr == nil:
			if an < bn {
				return -1
			}
			if an > bn {
				return 1
			}
		case aErr == nil && bErr != nil:
			return -1
		case aErr != nil && bErr == nil:
			return 1
		default:
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
		}
	}

	return 0
}
