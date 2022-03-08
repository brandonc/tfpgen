package naming

import (
	"testing"
)

func Test_FindPrefix(t *testing.T) {
	cases := map[string]string{
		"#/components/v": FindPrefix([]string{"#/components/vapid", "#/components/vacant", "#/components/verily"}),
		"answer":         FindPrefix([]string{"answerA", "answera", "answerå"}),
		"...":            FindPrefix([]string{"...three", "...four", "...fove"}),
	}

	for expected, actual := range cases {
		if expected != actual {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	}
}

func Test_ToHCLName(t *testing.T) {
	cases := map[string]string{
		"ThisIsCamelCase":    "this_is_camel_case",
		"1PasswordCamelCase": "1_password_camel_case",
		"mixedCASETYPING":    "mixed_casetyping",
		"star*Patrol":        "star_patrol",
		"ELSTUPIDO":          "elstupido",
		"APIHowdy":           "api_howdy",
		"kebab-phrase-2":     "kebab_phrase_2",
	}

	for before, expected := range cases {
		actual := ToHCLName(before)
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	}
}

func Test_ToTitleName(t *testing.T) {
	cases := map[string]string{
		"thisIsCamelCase":    "ThisIsCamelCase",
		"1PasswordCamelCase": "1PasswordCamelCase",
		"mixedCASETYPING":    "MixedCASETYPING",
		"star*Patrol":        "StarPatrol",
		"APIHowdy":           "APIHowdy",
		"2fast2furious":      "2Fast2Furious",
	}

	for before, expected := range cases {
		actual := ToTitleName(before)
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	}
}
