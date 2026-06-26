package service

import (
	"bytes"
	"strings"
	"text/template"
)

type customTemplateData struct {
	APIHost          string
	APIToken         string
	InstanceID       int64
	UserPlaybookYAML string
}

func renderCustomCloudInit(apiHost, apiToken string, instanceID int64, playbookYAML string) (string, error) {
	tmpl, err := template.New("custom").Funcs(template.FuncMap{
		"indent": func(spaces int, s string) string {
			prefix := strings.Repeat(" ", spaces)
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = prefix + line
				}
			}
			return strings.Join(lines, "\n")
		},
	}).ParseFiles("ansible/custom-template.yaml.j2")
	if err != nil {
		return "", err
	}

	data := customTemplateData{
		APIHost:          apiHost,
		APIToken:         apiToken,
		InstanceID:       instanceID,
		UserPlaybookYAML: playbookYAML,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "custom-template.yaml.j2", data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
