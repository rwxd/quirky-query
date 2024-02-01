package internal

import (
	"bytes"
	"html/template"
)

func inputToSafeHTML(input string) (string, error) {
	tmpl, err := template.New("input").Parse(input)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer

	err = tmpl.Execute(&buffer, nil)
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}
