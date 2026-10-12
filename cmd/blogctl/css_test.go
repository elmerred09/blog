package main

import (
	"strings"
	"testing"
)

func TestRunCSS(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	if err := run(t.Context(), []string{"css"}, &out); err != nil {
		t.Fatalf("run css: %v", err)
	}
	if !strings.HasPrefix(out.String(), cssHeader) {
		t.Errorf("output doesn't start with the generated-file header:\n%.200s", out.String())
	}
	if !strings.Contains(out.String(), ".chroma {") {
		t.Errorf("output has no .chroma rule:\n%.200s", out.String())
	}
}

func TestRunCSSRejectsArguments(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	err := run(t.Context(), []string{"css", "web/app/chroma.css"}, &out)
	if err == nil || !strings.Contains(err.Error(), "takes no arguments") {
		t.Fatalf("err = %v, want a takes-no-arguments error", err)
	}
	if out.Len() != 0 {
		t.Errorf("wrote output despite the error:\n%s", out.String())
	}
}
