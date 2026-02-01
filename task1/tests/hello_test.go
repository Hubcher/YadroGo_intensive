package hello_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const helloAddress = "http://localhost:28080"

var helloClient = http.Client{
	Timeout: 10 * time.Second,
}

func TestPreflight(t *testing.T) {
	require.Equal(t, true, true)
}

func TestPing(t *testing.T) {
	resp, err := helloClient.Get(helloAddress + "/ping")
	require.NoError(t, err, "cannot ping")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "wrong status")
	msg, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "cannot read reply")
	require.Equal(t, "pong\n", string(msg))
}

func TestHello(t *testing.T) {
	resp, err := helloClient.Get(helloAddress + "/hello?name=Misha")
	require.NoError(t, err, "cannot hello")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "wrong status")
	msg, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "cannot read reply")
	require.Equal(t, "Hello, Misha!\n", string(msg))
}

func TestHelloBadQuery(t *testing.T) {
	resp, err := helloClient.Get(helloAddress + "/hello")
	require.NoError(t, err, "cannot hello")
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "wrong status")
	msg, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "cannot read reply")
	require.Equal(t, "empty name\n", string(msg))
}

func TestHelloBadMethod(t *testing.T) {
	resp, err := helloClient.Post(helloAddress+"/hello", "application/json", nil)
	require.NoError(t, err, "cannot hello")
	defer resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "wrong status")
}
