package naming

import (
	"strings"
	"unicode"
)

// FindPrefix returns the string prefix that all words have in common, or empty string ("") if there is none
func FindPrefix(words []string) string {
	if len(words) <= 1 || len(words[0]) == 0 {
		return ""
	}

	test := words[0][0:0]

	for {
		for windex := 0; windex < len(words); windex++ {
			if !strings.HasPrefix(words[windex], test) {
				return test[0 : len(test)-1]
			}
		}

		if len(test) == len(words[0]) {
			break
		}
		test = words[0][0 : len(test)+1]
	}
	return test
}

// ToTitleName converts snake_case or kebab-case to TitleCase
func ToTitleName(s string) string {
	sb := strings.Builder{}
	upperNext := true
	for _, c := range s {
		if !unicode.IsLetter(c) {
			upperNext = true
			if unicode.IsDigit(c) {
				sb.WriteByte(byte(c))
			}
			continue
		}

		if upperNext {
			sb.WriteByte(byte(unicode.ToUpper(c)))
			upperNext = false
		} else {
			sb.WriteByte(byte(c))
		}
	}
	return sb.String()
}

// ValidHCLIdentifier checks to ensure whether an identifier is a valid HCL identifier. That is,
// they contain letters, digits, underscores (_), and dashes (-). The first character must not be a digit.
func ValidHCLIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	for index, c := range s {
		if unicode.IsDigit(c) && index == 0 {
			return false
		}

		if !unicode.IsDigit(c) && !unicode.IsLetter(c) && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

// ToHCLName converts TitleCase or camelCase to snake_case
func ToHCLName(s string) string {
	sb := strings.Builder{}
	preventUnderscore := true

	for index, c := range s {
		isLetter := unicode.IsLetter(c)
		isDigit := unicode.IsDigit(c)
		isUpper := unicode.IsUpper(c)

		if index > 0 && (isUpper || (!isDigit && !isLetter)) {
			if !preventUnderscore {
				sb.WriteByte('_')
				preventUnderscore = true
			}
		}

		if isLetter || isDigit {
			// When combined with isUpper, if two capital letters  are followed by a lowercase,
			// like "HIthere" it should not prevent the next underscore.
			lowerFollowsNextUpper := index < len(s)-2 && unicode.IsLetter(rune(s[index+2])) && !unicode.IsUpper(rune(s[index+2]))
			b := byte(unicode.ToLower(c))
			sb.WriteByte(b)
			preventUnderscore = false
			// Don't let CONSECUTIVECAPS create an underscore between each character
			if isUpper && !lowerFollowsNextUpper {
				preventUnderscore = true
			}
		}
	}

	return sb.String()
}
