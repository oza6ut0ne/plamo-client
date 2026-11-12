package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/koron-go/jsonhttpc"
	"github.com/mattn/go-isatty"
)

const (
	Model_0_8B = "hf.co/pfnet/plamo-3-translate-0.8b-GGUF"
	Model_2B   = "hf.co/pfnet/plamo-3-translate-2b-GGUF"
)

type Request struct {
	Prompt string `json:"prompt"`

	NPredict      int     `json:"n_predict"`
	RepeatPenalty float64 `json:"repeat_penalty"`
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"top_p"`
	Model         string  `json:"model"`
}

type Response struct {
	Content string `json:"content"`

	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`

	GenerationSettings struct {
		DynatempExponent       float64       `json:"dynatemp_exponent"`
		DynatempRange          float64       `json:"dynatemp_range"`
		FrequencyPenalty       float64       `json:"frequency_penalty"`
		Grammar                string        `json:"grammar"`
		IgnoreEOS              bool          `json:"ignore_eos"`
		LogitBias              []interface{} `json:"logit_bias"`
		MinKeep                float64       `json:"min_keep"`
		MinP                   float64       `json:"min_p"`
		Mirostat               float64       `json:"mirostat"`
		MirostatETA            float64       `json:"mirostat_eta"`
		MirostatTAU            float64       `json:"mirostat_tau"`
		Model                  string        `json:"model"`
		NCtx                   int           `json:"n_ctx"`
		NDiscard               int           `json:"n_discard"`
		NKeep                  int           `json:"n_keep"`
		NPredict               int           `json:"n_predict"`
		NProbs                 int           `json:"n_probs"`
		PenalizeNL             bool          `json:"penalize_nl"`
		PenaltyPromptTokens    []interface{} `json:"penalty_prompt_tokens"`
		PresensePenalty        float64       `json:"presence_penalty"`
		RepeatLastN            int64         `json:"repeat_last_n"`
		RepeatPenalty          float64       `json:"repeat_penalty"`
		Samplers               []string      `json:"samplers"`
		Seed                   int64         `json:"seed"`
		Stop                   []string      `json:"stop"`
		Stream                 bool          `json:"stream"`
		Temperature            float64       `json:"temperature"`
		TfsZ                   float64       `json:"tfs_z"`
		TopK                   float64       `json:"top_k"`
		TopP                   float64       `json:"top_p"`
		TypicalP               float64       `json:"typical_p"`
		UsePenaltyPromptTokens bool          `json:"use_penalty_prompt_tokens"`
	} `json:"generation_settings"`

	IDSlot int    `json:"id_slot"`
	Model  string `json:"model"`
	Prompt string `json:"prompt"`

	Stop         bool   `json:"stop"`
	StoppedEOS   bool   `json:"stopped_eos"`
	StoppedLimit bool   `json:"stopped_limit"`
	StoppedWord  bool   `json:"stopped_word"`
	StoppingWord string `json:"stopping_word"`

	Timings struct {
		PredictedMS         float64 `json:"predicted_ms"`
		PredictedN          int     `json:"predicted_n"`
		PredictedPerSecond  float64 `json:"predicted_per_second"`
		PredictedPerTokenMS float64 `json:"predicted_per_token_ms"`

		PromptMS         float64 `json:"prompt_ms"`
		PromptN          int     `json:"prompt_n"`
		PromptPerSecond  float64 `json:"prompt_per_second"`
		PromptPerTokenMS float64 `json:"prompt_per_token_ms"`
	} `json:"timiings"`

	TokensCached    int `json:"tokens_cached"`
	TokensEvaluated int `json:"tokens_evaluated"`
	TokensPredicted int `json:"tokens_predicted"`

	Truncated bool `json:"truncated"`
}

var (
	verbose bool
	host    string
	port    int
	twoB    bool

	entrypoint string
	reqTmpl    Request
)

func translate(text string, outputLanguage string, form string) (string, error) {
	param := NewPromptParam(outputLanguage, form, text)
	prompt, err := param.Generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate prompt: %w", err)
	}

	if verbose {
		log.Printf("outputLanguage is %q", param.OutputLanguage)
		log.Printf("prompt is... %q", prompt)
	}

	req := reqTmpl
	req.Prompt = prompt
	var res Response
	err = jsonhttpc.Do(context.Background(), "POST", entrypoint, &req, &res)
	if err != nil {
		return "", fmt.Errorf("failed to request: %w", err)
	}
	return res.Choices[0].Text, nil
}

func reverseMode(mode string) string {
	switch mode {
	case Japanese:
		return English
	case English:
		return Japanese
	default:
		return Japanese
	}
}

func isTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func canBeInteractive() bool {
	return isTerminal(os.Stdin.Fd()) && isTerminal(os.Stdout.Fd())
}

// readAllTrimmed reads all of r and returns the content with leading and
// trailing spaces trimmed.
func readAllTrimmed(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// normalizeInteractiveInput trims leading and trailing spaces from a line of
// interactive input. Internal line breaks are kept intact so that pasted
// multi-line text can be translated as a single block. The second return value
// is false when the input is empty after trimming, in which case it should be
// ignored.
func normalizeInteractiveInput(s string) (string, bool) {
	text := strings.TrimSpace(s)
	return text, text != ""
}

func main() {
	var (
		outputLanguage string
		form           string
		iteration      int
	)

	// Core options.
	flag.BoolVar(&verbose, "verbose", false, `verbose messages`)
	flag.StringVar(&host, "host", "127.0.0.1", `Ollama host`)
	flag.IntVar(&port, "port", 11434, `Ollama port`)
	flag.StringVar(&outputLanguage, "lang", "", `output language`)
	flag.StringVar(&form, "form", Polite, `writing style`)
	flag.BoolVar(&twoB, "2b", false, `use 2b model`)

	// Request options.
	flag.StringVar(&reqTmpl.Model, "model", Model_0_8B, `model name`)
	flag.IntVar(&reqTmpl.NPredict, "n_predict", -1, `number of predict`)
	flag.Float64Var(&reqTmpl.RepeatPenalty, "repeat_penalty", 1.0, `repeat penalty`)
	flag.Float64Var(&reqTmpl.Temperature, "temperature", 0.0, `temperature`)
	flag.Float64Var(&reqTmpl.TopP, "top_p", 0.0, `top P`)

	// Operation mode options.
	flag.IntVar(&iteration, "iteration", 0, "number of times to repeat the reverse translation. -1 means to repeat until the translation matches the translation history.")

	flag.Parse()

	entrypoint = fmt.Sprintf("http://%s:%d/v1/completions", host, port)

	if twoB && strings.EqualFold(reqTmpl.Model, Model_0_8B) {
		reqTmpl.Model = Model_2B
	}

	// Validate option values.
	if !IsValidForm(form) {
		log.Fatalf("unknown %q form. please choose one from following: %s",
			form, strings.Join(ValidForm, ", "))
	}

	// Translate the text given by the argument, if any. The argument takes
	// precedence over standard input.
	if flag.NArg() > 0 {
		err := singleSentenceMode(flag.Arg(0), outputLanguage, form, iteration)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Translate the text piped or redirected to standard input as a single
	// block.
	if !isTerminal(os.Stdin.Fd()) {
		text, err := readAllTrimmed(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		err = singleSentenceMode(text, outputLanguage, form, iteration)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Start interactive translation mode, if the os.Stdin and os.Stdout are
	// connected a TTY.
	if !canBeInteractive() {
		log.Fatal("cannot start interactive mode: standard output is not a terminal")
	}
	if err := interactiveMode(outputLanguage, form); err != nil {
		log.Fatal(err)
	}
}

// Operation mode implementations

func singleSentenceMode(textOrig, outputLanguageOrig, form string, iteration int) error {
	text := strings.TrimSpace(textOrig)
	if text == "" {
		return errors.New("no text to translate")
	}
	outputLanguage, err := regulateMode(outputLanguageOrig, text)
	if err != nil {
		return err
	}

	// One-shot translation.
	if iteration == 0 {
		translation, err := translate(text, outputLanguage, form)
		if err != nil {
			return err
		}
		fmt.Println(translation)
		return nil
	}

	// Repeat the translation with specifying the number of times.
	if iteration > 0 {
		for i := range iteration + 1 {
			translation, err := translate(text, outputLanguage, form)
			if err != nil {
				return fmt.Errorf("failed at #%d: %w", i, err)
			}
			fmt.Printf("#%d\t%s\n", i, translation)
			text = translation
			outputLanguage = reverseMode(outputLanguage)
		}
		return nil
	}

	// Repeat the translation until the same result is obtained.
	seen := map[string]struct{}{}
	for i := 0; ; i++ {
		if _, ok := seen[text]; ok {
			break
		}
		seen[text] = struct{}{}
		translation, err := translate(text, outputLanguage, form)
		if err != nil {
			log.Fatalf("failed at #%d: %s", i, err)
		}
		fmt.Printf("#%d\t%s\n", i, translation)
		text = translation
		outputLanguage = reverseMode(outputLanguage)
	}
	return nil
}

func interactiveMode(outputLanguageOrig, form string) error {
	reader, err := newInteractiveReader(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	defer reader.Close()
	for {
		line, err := reader.ReadBlock()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, errInterrupt) {
				return nil
			}
			return err
		}
		text, ok := normalizeInteractiveInput(line)
		if !ok {
			continue
		}
		outputLanguage, err := regulateMode(outputLanguageOrig, text)
		if err != nil {
			return err
		}
		translation, err := translate(text, outputLanguage, form)
		if err != nil {
			return err
		}
		writeRawLine(os.Stdout, translation)
	}
}
