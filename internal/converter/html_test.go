package converter

import (
	"errors"
	"strings"
	"testing"
)

func TestConvertFragmentTransformsNarouHTML(t *testing.T) {
	got, err := ConvertFragment(`<p id="L1">本文<b>太字</b><i>斜体</i><s>取消</s><a href="https://example.com">リンク</a></p>`, Options{})
	if err != nil {
		t.Fatalf("ConvertFragment() error = %v", err)
	}

	want := `<p>本文<strong>太字</strong><em>斜体</em><span class="strikethrough">取消</span>リンク</p>`
	if got != want {
		t.Fatalf("ConvertFragment() = %q, want %q", got, want)
	}
}

func TestConvertFragmentKeepsRubyAndEmphasisDots(t *testing.T) {
	got, err := ConvertFragment(`<p id="L2"><ruby><rb>漢字</rb><rt>かんじ</rt></ruby><em class="emphasisDots">強調</em></p>`, Options{})
	if err != nil {
		t.Fatalf("ConvertFragment() error = %v", err)
	}

	want := `<p><ruby><rb>漢字</rb><rt>かんじ</rt></ruby><em class="emphasisDots">強調</em></p>`
	if got != want {
		t.Fatalf("ConvertFragment() = %q, want %q", got, want)
	}
}

func TestConvertFragmentResolvesImages(t *testing.T) {
	got, err := ConvertFragment(`<p id="L3"><img src="https://example.mitemin.net/userpageimage/viewimage/icode/514881/" /></p>`, Options{
		ImageResolver: func(src string) (string, error) {
			if !strings.Contains(src, "514881") {
				t.Fatalf("src = %q, want image URL", src)
			}
			return "../images/i514881.jpg", nil
		},
	})
	if err != nil {
		t.Fatalf("ConvertFragment() error = %v", err)
	}

	want := `<p><img src="../images/i514881.jpg" /></p>`
	if got != want {
		t.Fatalf("ConvertFragment() = %q, want %q", got, want)
	}
}

func TestConvertFragmentReturnsImageResolverError(t *testing.T) {
	wantErr := errors.New("missing image")
	_, err := ConvertFragment(`<img src="missing.jpg">`, Options{
		ImageResolver: func(string) (string, error) {
			return "", wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("ConvertFragment() error = %v, want %v", err, wantErr)
	}
}

func TestConvertFragmentAppliesTypography(t *testing.T) {
	got, err := ConvertFragment(`<p id="L4">「12！！？」</p>`, Options{})
	if err != nil {
		t.Fatalf("ConvertFragment() error = %v", err)
	}

	want := `<p class="half-indent">「<span class="tcy">12</span><span class="tcy">!!?</span>」</p>`
	if got != want {
		t.Fatalf("ConvertFragment() = %q, want %q", got, want)
	}
}
