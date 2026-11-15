package shape

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCutProblemsPreservesWhy(t *testing.T) {
	parallel(t)
	fragment := "- Importing an account export creates a new account with everything | expectation: features/steps/rails-ft-data-export-and-import__driver/export_import_test.rb: get \"#{@slug}/account/imports/new\" | contract: config/environments/test.rb: config.action_mailer.default_url_options = { host: \"example.com "
	contract := fragment + strings.Repeat("界", 247) + " | why: importing must preserve the exported account"
	decision := "statement: " + strings.Repeat("界", 574) + " | why: because"
	cases := []struct{ name, input, want string }{
		{"synthetic contract", contract, string([]rune(contract)[:243]) + " ...  | why: importing must preserve the exported account"},
		{"decision", decision, string([]rune(decision)[:280]) + " ...  | why: because"},
	}
	for _, n := range []int{149, 150, 151} {
		why := " | why: " + strings.Repeat("理", n-8)
		cases = append(cases, struct{ name, input, want string }{fmt.Sprintf("why %d runes", n), strings.Repeat("頭", 600-n) + why, strings.Repeat("頭", 295-min(n, 150)) + " ... " + string([]rune(why)[:min(n, 150)])})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cutProblems([]string{tc.input})
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if !utf8.ValidString(got[0]) || utf8.RuneCountInString(got[0]) != 300 {
				t.Fatalf("invalid bounded output: %q", got[0])
			}
		})
	}
}

func TestCutProblemsThreshold(t *testing.T) {
	parallel(t)
	for _, n := range []int{299, 300, 301} {
		for _, why := range []string{"", " | why: w"} {
			t.Run(fmt.Sprintf("%d/why=%t", n, why != ""), func(t *testing.T) {
				input := strings.Repeat("頭", n-utf8.RuneCountInString(why)) + why
				want := input
				if n > 300 {
					want = strings.Repeat("頭", 300)
					if why != "" {
						want = strings.Repeat("頭", 286) + " ... " + why
					}
				}
				if got := cutProblems([]string{input}); !slices.Equal(got, []string{want}) {
					t.Fatalf("got %q, want %q", got, want)
				}
			})
		}
	}
}

func TestCutProblemsLineLimit(t *testing.T) {
	parallel(t)
	for _, n := range []int{19, 20, 21} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			input := make([]string, n)
			for i := range input {
				input[i] = fmt.Sprintf("problem %d", i)
			}
			want := input[:min(n, 20)]
			if got := cutProblems(input); !slices.Equal(got, want) {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}
