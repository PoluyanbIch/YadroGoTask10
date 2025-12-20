package words

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNorm(t *testing.T) {
	testCases := []struct {
		desc     string
		given    string
		expected []string
	}{
		{
			desc:     "empty",
			given:    "",
			expected: []string{},
		},
		{
			desc:     "simple",
			given:    "simple",
			expected: []string{"simpl"},
		},
		{
			desc:     "followers",
			given:    "I follow followers",
			expected: []string{"follow"},
		},
		{
			desc:     "punctuation",
			given:    "I shouted: 'give me your car!!!",
			expected: []string{"shout", "give", "car"},
		},
		{
			desc:     "stop words only",
			given:    "I and you or me or them, who will?",
			expected: []string{},
		},
		{
			desc:     "not latin letters",
			given:    "русский язык",
			expected: []string{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			res := Norm(tc.given)
			require.ElementsMatch(t, tc.expected, res)
		})
	}
}

func TestIsEnglishWord(t *testing.T) {
	testCases := []struct {
		desc     string
		given    string
		expected bool
	}{
		{
			desc:     "Latin",
			given:    "word",
			expected: true,
		},
		{
			desc:     "empty",
			given:    "",
			expected: false,
		},
		{
			desc:     "with numbers",
			given:    "word123",
			expected: false,
		},
		{
			desc:     "Russian",
			given:    "русский",
			expected: false,
		},
		{
			desc:     "symbols",
			given:    "word-",
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			res := isEnglishWord(tc.given)
			require.Equal(t, tc.expected, res)
		})
	}
}
