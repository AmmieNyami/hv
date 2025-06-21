package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func removeCommentsFromJson(json string) (string, error) {
	var builder strings.Builder
	inString := false
	escaped := false

	for i := 0; i < len(json); {
		c := json[i]

		if c == '"' && !escaped {
			inString = !inString
		}

		if c == '\\' && inString {
			escaped = !escaped
		} else {
			escaped = false
		}

		if !inString && c == '/' && i+1 < len(json) {
			next := json[i+1]

			if next == '/' {
				i += 2
				for i < len(json) && json[i] != '\n' && json[i] != '\r' {
					i++
				}
				continue
			}

			if next == '*' {
				i += 2
				for {
					if i+1 >= len(json) {
						return "", fmt.Errorf("Unclosed comment")
					}
					if json[i] == '*' && json[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}

		builder.WriteByte(c)
		i++
	}

	return builder.String(), nil
}

func UnmarshalJsonWithComments(s string, v any) error {
	sWithoutComments, err := removeCommentsFromJson(s)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(sWithoutComments), v)
}
