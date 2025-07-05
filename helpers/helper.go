package helpers

import (
	"strings"
	"time"
	"unicode/utf8"
)

func IsPasswordStrong(password string) bool {

	const SPECIAL_CHARS = "!@#$%^&*()_+-=[]{}|;:',.<>/?`~\\"
	const NUMERICAL_CHARS = "123456790"
	const UPPERCASE_CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const LOWERCASE_CHARS = "abcdefghijkmlnopqrstuvwxyz"
	hasSpecialChar := false
	hasNumericalChar := false
	hasUpperCaseChar := false
	hasLowerCaseChar := false

	if utf8.RuneCountInString(password) < 6 {
		return false
	}

	for _, passwordChar := range password {

		if hasSpecialChar && hasNumericalChar && hasLowerCaseChar && hasUpperCaseChar {
			break
		}

		if !hasSpecialChar && strings.Contains(SPECIAL_CHARS, string(passwordChar)) {
			hasSpecialChar = true
		}

		if !hasNumericalChar && strings.Contains(NUMERICAL_CHARS, string(passwordChar)) {
			hasNumericalChar = true
		}

		if !hasUpperCaseChar && strings.Contains(UPPERCASE_CHARS, string(passwordChar)) {
			hasUpperCaseChar = true
		}

		if !hasLowerCaseChar && strings.Contains(LOWERCASE_CHARS, string(passwordChar)) {
			hasLowerCaseChar = true
		}
	}

	if hasSpecialChar && hasNumericalChar && hasLowerCaseChar && hasUpperCaseChar {
		return true
	} else {
		return false
	}
}

func IsEmailValid(email string) bool {

	if email == "" {
		return false
	}

	if !strings.Contains(email, "@") {
		return false
	}

	emailParts := strings.Split(email, "@")
	firstPart, secondPart := emailParts[0], emailParts[1]

	if firstPart == "" || secondPart == "" {
		return false
	}

	if strings.Contains(secondPart, ".") && len(strings.Split(secondPart, ".")) == 2 {
		return true
	} else {
		return false
	}
}

func CalculateAgeFromTime(userDateOfBirthTime time.Time) int {
	var age int

	dateOfBirthYear := userDateOfBirthTime.Year()
	dateOfBirthMonth := userDateOfBirthTime.Month()
	dateOfBirthDay := userDateOfBirthTime.Day()

	currentYear := time.Now().Year()
	currentMonth := time.Now().Month()
	currentDay := time.Now().Day()

	if currentYear < dateOfBirthYear {
		return -1
	} else if currentYear == dateOfBirthYear {
		return 0
	} else {

		age = currentYear - dateOfBirthYear

		if currentMonth < dateOfBirthMonth {
			age = age - 1
		} else if currentMonth == dateOfBirthMonth {
			if currentDay < dateOfBirthDay {
				age = age - 1
			}
		}
	}

	return age

}
