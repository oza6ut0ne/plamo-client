package main

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
)

const (
	Japanese = "ja"
	English  = "en"
)

func regulateMode(mode string, content string) (string, error) {
	if strings.EqualFold(mode, Japanese) {
		return Japanese, nil
	}
	if strings.EqualFold(mode, English) {
		return English, nil
	}
	if mode != "" && !strings.EqualFold(mode, "auto") {
		return mode, nil
	}

	var total, ascii int64
	for _, r := range content {
		if r >= ' ' {
			total++
		}
		if r >= ' ' && r < 128 {
			ascii++
		}
	}

	//log.Printf("ascii=%d total=%d\n", ascii, total)
	if ascii < 3*total/4 {
		return English, nil
	}
	return Japanese, nil
}

// Available styles.
// See also https://huggingface.co/webbigdata/C3TR-Adapter/discussions/1#669e6ef419d0f96d8a77128b
const (
	None         = "none"
	Plain        = "plain"
	Polite       = "polite"
	NonePrompt   = ""
	PlainPrompt  = "<|plamo:key|>form<|plamo:val|>plain"
	PolitePrompt = "<|plamo:key|>form<|plamo:val|>polite"
)

var FormPrompt = map[string]string{
	None:   NonePrompt,
	Plain:  PlainPrompt,
	Polite: PolitePrompt,
}

var ValidForm = []string{
	None,
	Plain,
	Polite,
}

func IsValidForm(s string) bool {
	for _, valid := range ValidForm {
		if s == valid {
			return true
		}
	}
	return false
}

type PromptParam struct {
	OutputLanguage string
	Form           string
	Content        string
}

func NewPromptParam(outputLanguage string, form string, content string) *PromptParam {
	return &PromptParam{
		OutputLanguage: outputLanguage,
		Form:           FormPrompt[form],
		Content:        content,
	}
}

//go:embed prompt.tmpl
var promptTmplRaw string

var promptTmpl = template.Must(template.New("prompt").Parse(promptTmplRaw))

func (pp PromptParam) Generate() (string, error) {
	bb := &bytes.Buffer{}
	err := promptTmpl.Execute(bb, pp)
	if err != nil {
		return "", err
	}
	return bb.String(), nil
}
