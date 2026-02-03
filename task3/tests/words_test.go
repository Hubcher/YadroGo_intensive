package words_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const address = "http://localhost:28080"

var client = http.Client{
	Timeout: 10 * time.Second,
}

func TestPreflight(t *testing.T) {
	require.Equal(t, true, true)
}

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func TestPing(t *testing.T) {
	resp, err := client.Get(address + "/ping")
	require.NoError(t, err, "cannot ping")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "wrong status")

	var reply PingResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reply))
	require.Equal(t, "ok", reply.Replies["words"])
}

type WordsResponse struct {
	Words []string `json:"words"`
	Total int      `json:"total"`
}

func TestWords(t *testing.T) {

	testCases := []struct {
		desc     string
		given    string
		expected []string
	}{
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
			desc:     "weird",
			given:    "Moscow!123'check-it'or   123, man,that,difficult:heck",
			expected: []string{"moscow", "check", "123", "man", "difficult", "heck"},
		},
		{
			desc:     "large",
			given:    strings.Repeat("1234", 1<<10),
			expected: []string{strings.Repeat("1234", 1<<10)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			query := url.Values{}
			query.Add("phrase", tc.given)
			req, err := http.NewRequestWithContext(
				ctx, http.MethodGet, address+"/api/words?"+query.Encode(), nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var reply WordsResponse
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&reply))
			require.ElementsMatch(t, tc.expected, reply.Words)
			require.Equal(t, len(tc.expected), reply.Total)
		})
	}
}

func TestPhraseTooLarge(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := url.Values{}
	query.Add("phrase", "0"+strings.Repeat("1234", 1<<10))
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, address+"/api/words?"+query.Encode(), nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPhraseMissing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, address+"/api/words", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
