package builder

import "strings"

// Vercmp compares two pacman package versions ([epoch:]pkgver[-pkgrel]) like pacman's vercmp:
// -1 if a is older than b, 0 if equal, 1 if newer. A port of alpm_pkg_vercmp (libalpm/version.c),
// so the builder agrees with pacman on what "newer" means without calling vercmp per comparison.
func Vercmp(a, b string) int {
	if a == b {
		return 0
	}
	epoch1, ver1, rel1 := parseEVR(a)
	epoch2, ver2, rel2 := parseEVR(b)
	if c := rpmvercmp(epoch1, epoch2); c != 0 {
		return c
	}
	if c := rpmvercmp(ver1, ver2); c != 0 {
		return c
	}
	// A missing pkgrel matches any pkgrel.
	if rel1 != "" && rel2 != "" {
		return rpmvercmp(rel1, rel2)
	}
	return 0
}

// parseEVR splits [epoch:]version[-release]; the epoch defaults to "0".
func parseEVR(evr string) (epoch, version, release string) {
	i := 0
	for i < len(evr) && isDigit(evr[i]) {
		i++
	}
	epoch, version = "0", evr
	if i < len(evr) && evr[i] == ':' {
		if i > 0 {
			epoch = evr[:i]
		}
		version = evr[i+1:]
	}
	if j := strings.LastIndexByte(version, '-'); j >= 0 {
		version, release = version[:j], version[j+1:]
	}
	return epoch, version, release
}

// rpmvercmp compares alternating numeric and alphabetic segments; separators only count by length.
func rpmvercmp(a, b string) int {
	if a == b {
		return 0
	}
	one, two := 0, 0 // read positions in a and b
	p1, p2 := 0, 0   // ends of the previous segments
	for one < len(a) && two < len(b) {
		for one < len(a) && !isAlnum(a[one]) {
			one++
		}
		for two < len(b) && !isAlnum(b[two]) {
			two++
		}
		if one >= len(a) || two >= len(b) {
			break
		}
		// Different separator lengths decide on their own.
		if one-p1 != two-p2 {
			if one-p1 < two-p2 {
				return -1
			}
			return 1
		}
		p1, p2 = one, two

		isNum := isDigit(a[p1])
		segment := isAlpha
		if isNum {
			segment = isDigit
		}
		for p1 < len(a) && segment(a[p1]) {
			p1++
		}
		for p2 < len(b) && segment(b[p2]) {
			p2++
		}
		// b has a segment of the other kind here: numeric segments are newer than alphabetic ones.
		if p2 == two {
			if isNum {
				return 1
			}
			return -1
		}

		s1, s2 := a[one:p1], b[two:p2]
		if isNum {
			s1, s2 = strings.TrimLeft(s1, "0"), strings.TrimLeft(s2, "0")
			if len(s1) != len(s2) {
				if len(s1) > len(s2) {
					return 1
				}
				return -1
			}
		}
		if c := strings.Compare(s1, s2); c != 0 {
			return c
		}
		one, two = p1, p2
	}

	if one >= len(a) && two >= len(b) {
		return 0
	}
	// A remaining alphabetic segment never beats an empty string ("1.0alpha" < "1.0"), anything else does.
	if (one >= len(a) && !isAlpha(b[two])) || (one < len(a) && isAlpha(a[one])) {
		return -1
	}
	return 1
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isAlnum(c byte) bool { return isDigit(c) || isAlpha(c) }
