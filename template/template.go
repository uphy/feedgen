package template

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode"
)

var globalFuncs = map[string]interface{}{
	"Env": func(name string) string {
		return os.Getenv(name)
	},
	"Default": func(defaultValue string, value string) string {
		if value == "" {
			return defaultValue
		}
		return value
	},
	"SliceOf": func(values ...string) []string {
		return values
	},
	"Trim": func(s string) string {
		return strings.Trim(s, " 　\t\r\n")
	},
	"Truncate": func(length int, s string) string {
		runes := []rune(s)
		if len(runes) > length {
			return string(runes[0:length]) + "..."
		}
		return s
	},
	"Match": func(pattern string, s string) (string, error) {
		r, err := regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
		matches := r.FindStringSubmatch(s)
		switch len(matches) {
		case 0, 1:
			return "<no match>", nil
		case 2:
			return matches[1], nil
		default:
			return fmt.Sprintf("<multiple matches:%v>", matches), nil
		}
	},
	"Contains": func(substring, s string) bool {
		return strings.Contains(s, substring)
	},
	"FormatEpochMillis": func(epochMillis float64) string {
		n := int64(epochMillis) * 1000000
		return time.Unix(0, n).Format("2006/01/02 15:04")
	},
	"Capitalize": func(s string) string {
		r := []rune(s)
		if len(r) == 0 {
			return s
		}
		r[0] = unicode.ToUpper(r[0])
		return string(r)
	},
}

func evaluate(templateStr string, ctx *TemplateContext) (string, error) {
	contextVars, contextFuncs := ctx.flatten()

	parsed, err := template.New("template-string").Funcs(globalFuncs).Funcs(contextFuncs).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("cannot parse template: %w", err)
	}

	var buf = new(bytes.Buffer)
	if err := parsed.Execute(buf, contextVars); err != nil {
		return "", fmt.Errorf("cannot evaluate: %w", err)
	}
	s := buf.String()
	s = strings.TrimSpace(s)
	return s, nil
}
