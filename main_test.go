package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterLogs1(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nTestB 1\nTestA 2\nTestB 2\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	require.NoError(t, err)
	assert.Equal(t, []byte("TestA 1\nTestA 2\n"), filteredLogs)
}

func TestFilterLogs2(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nTestB 1\nTestA 2\nTestB 2\n"
	testName := "TestB"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	require.NoError(t, err)
	assert.Equal(t, []byte("TestB 1\nTestB 2\n"), filteredLogs)
}

func TestFilterLogsNoNewlineAtEnd(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nTestB 1\nTestA 2\nTestB 2"
	testName := "TestB"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	require.NoError(t, err)
	assert.Equal(t, []byte("TestB 1\nTestB 2"), filteredLogs)
}

// when a line has no prefix, it should be included if the line before it has the desired prefix, or if the line
// before it recursively matches this condition
func TestFilterLogsNoPrefixContinuation1(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nno prefix\nTestB 1\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	require.NoError(t, err)
	assert.Equal(t, []byte("TestA 1\nno prefix\n"), filteredLogs)
}

func TestFilterLogsNoPrefixContinuation2(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nno prefix 1\nTestA 2\nTestB 1\nno prefix 2\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	require.NoError(t, err)
	assert.Equal(t, []byte("TestA 1\nno prefix 1\nTestA 2\n"), filteredLogs)
}

func TestFilterLogsNoPrefixContinuation3(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nno prefix 1\nTestA 2\nTestB 1\nno prefix 2\n"
	testName := "TestB"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	require.NoError(t, err)
	assert.Equal(t, []byte("TestB 1\nno prefix 2\n"), filteredLogs)
}

func TestFilterLogsNoMatchingLines(t *testing.T) {
	t.Parallel()
	logs := "TestB 1\nno prefix\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	require.NoError(t, err)
	assert.Equal(t, []byte(""), filteredLogs)
}

func TestRemoveTimestampPrefix1(t *testing.T) {
	logs := []byte("2023-05-02T19:31:15.2539162Z Done in 219ms.")
	actual := removeTimestampPrefix(logs)
	assert.Equal(t, []byte("Done in 219ms."), actual)
}
