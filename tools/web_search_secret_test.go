package tools

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The SerpAPI request carries the operator's key in its URL. These tests pin
// two properties without touching the network (http.DefaultTransport is
// replaced for the duration of each test):
//
//  1. caller text is escaped into one query value and cannot add or override
//     parameters, and
//  2. no error returned to the caller contains the request URL, key included.

const searchKeyCanary = "canary-search-key-7f3a91"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubTransport(t *testing.T, f roundTripFunc) {
	t.Helper()
	orig := http.DefaultTransport
	http.DefaultTransport = f
	t.Cleanup(func() { http.DefaultTransport = orig })
}

func TestWebSearch_QueryIsEscapedIntoOneValue(t *testing.T) {
	queries := []string{
		"plain words",
		"x&api_key=attacker",
		"x\n",
		"x\x00y",
		"a=b#frag?z",
		"100% sure + more",
	}
	for _, engine := range []string{"serpapi", "duckduckgo"} {
		for _, q := range queries {
			t.Run(engine+"/"+strings.NewReplacer("\n", `\n`, "\x00", `\0`).Replace(q), func(t *testing.T) {
				t.Setenv("SERPAPI_KEY", searchKeyCanary)
				var got *http.Request
				stubTransport(t, func(r *http.Request) (*http.Response, error) {
					got = r
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
						Header:     http.Header{},
						Request:    r,
					}, nil
				})

				_, err := WebSearch(context.Background(), map[string]interface{}{
					"query": q, "search_engine": engine,
				})
				require.NoError(t, err)
				require.NotNil(t, got, "the request must be built and sent for any query text")

				vals := got.URL.Query()
				assert.Equal(t, q, vals.Get("q"), "the query must round-trip as exactly one value")
				assert.Len(t, vals["q"], 1)
				if engine == "serpapi" {
					assert.Equal(t, []string{searchKeyCanary}, vals["api_key"], "caller text must not add or replace api_key")
				} else {
					assert.Empty(t, vals["api_key"], "the key is sent to SerpAPI only")
				}
			})
		}
	}
}

func TestWebSearch_ErrorsNeverCarryTheRequestURL(t *testing.T) {
	for _, engine := range []string{"serpapi", "duckduckgo"} {
		t.Run(engine+"/transport error", func(t *testing.T) {
			t.Setenv("SERPAPI_KEY", searchKeyCanary)
			// net/http wraps this in a *url.Error whose text is the full URL.
			stubTransport(t, func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("dial refused")
			})

			res, err := WebSearch(context.Background(), map[string]interface{}{
				"query": "anything", "search_engine": engine,
			})
			require.NoError(t, err)
			assert.Equal(t, false, res["success"])
			msg, _ := res["error"].(string)
			assert.Equal(t, "search request failed", msg)
			assert.NotContains(t, msg, searchKeyCanary)
		})
	}

	t.Run("serpapi/provider error body is not echoed", func(t *testing.T) {
		t.Setenv("SERPAPI_KEY", searchKeyCanary)
		stubTransport(t, func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader("bad request " + r.URL.String())),
				Header:     http.Header{},
				Request:    r,
			}, nil
		})
		res, err := WebSearch(context.Background(), map[string]interface{}{"query": "q"})
		require.NoError(t, err)
		msg, _ := res["error"].(string)
		assert.NotContains(t, msg, searchKeyCanary)
		assert.Contains(t, msg, "401")
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Setenv("SERPAPI_KEY", searchKeyCanary)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res, err := WebSearch(ctx, map[string]interface{}{"query": "q"})
		require.NoError(t, err)
		msg, _ := res["error"].(string)
		assert.NotContains(t, msg, searchKeyCanary)
	})
}

func TestRequestErrorDetail_StripsTheURL(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.invalid/?api_key="+searchKeyCanary, nil)
	require.NoError(t, err)
	resp, doErr := (&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}).Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, doErr)
	require.Contains(t, doErr.Error(), searchKeyCanary, "precondition: the raw error does carry the URL")

	detail := requestErrorDetail(doErr)
	assert.NotContains(t, detail, searchKeyCanary)
	assert.Contains(t, detail, "boom")
}
