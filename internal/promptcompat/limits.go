package promptcompat

import "fmt"

const MaxPromptRunes = 131072

func ValidatePromptLength(prompt string) error {
	if len([]rune(prompt)) <= MaxPromptRunes {
		return nil
	}
	return fmt.Errorf("context too long: maximum 128k characters allowed")
}
