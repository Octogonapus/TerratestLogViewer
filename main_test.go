package main

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v52/github"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// basic test to check it does not crash. hard to test much else
func TestGetLogs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")})
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)
	logs, err := getLogs(gh, "Octogonapus", "TerratestLogViewer", "test.yml", "main", "test")
	assert.NotEmpty(t, logs)
	assert.NoError(t, err)
}

func TestFilterLogs1(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nTestB 1\nTestA 2\nTestB 2\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "TestA 1\nTestA 2\n", string(filteredLogs))
}

func TestFilterLogs2(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nTestB 1\nTestA 2\nTestB 2\n"
	testName := "TestB"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "TestB 1\nTestB 2\n", string(filteredLogs))
}

func TestFilterLogsNoNewlineAtEnd(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nTestB 1\nTestA 2\nTestB 2"
	testName := "TestB"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "TestB 1\nTestB 2", string(filteredLogs))
}

// when a line has no prefix, it should be included if the line before it has the desired prefix, or if the line
// before it recursively matches this condition
func TestFilterLogsNoPrefixContinuation1(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nno prefix\nTestB 1\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "TestA 1\nno prefix\n", string(filteredLogs))
}

func TestFilterLogsNoPrefixContinuation2(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nno prefix 1\nTestA 2\nTestB 1\nno prefix 2\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "TestA 1\nno prefix 1\nTestA 2\n", string(filteredLogs))
}

func TestFilterLogsNoPrefixContinuation3(t *testing.T) {
	t.Parallel()
	logs := "TestA 1\nno prefix 1\nTestA 2\nTestB 1\nno prefix 2\n"
	testName := "TestB"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "TestB 1\nno prefix 2\n", string(filteredLogs))
}

func TestFilterLogsNoMatchingLines(t *testing.T) {
	t.Parallel()
	logs := "TestB 1\nno prefix\n"
	testName := "TestA"
	filteredLogs, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "", string(filteredLogs))
}

func TestRemoveTimestampPrefix1(t *testing.T) {
	t.Parallel()
	logs := []byte("2023-05-02T19:31:15.2539162Z Done in 219ms.")
	actual := removeTimestampPrefix(logs)
	assert.Equal(t, "Done in 219ms.", string(actual))
}

func TestRemoveTestNamePrefix(t *testing.T) {
	t.Parallel()
	logs := []byte("TestFoo 1\nno prefix 2\nTestFoo 3\nno prefix 4\n")
	actual := removeTestNamePrefix(logs, []byte("TestFoo"))
	assert.Equal(t, "1\nno prefix 2\n3\nno prefix 4\n", string(actual))
}

// A test's failure should be included when filtering for a specific test, even when another test's output precedes it
func TestTestFailureIncluded(t *testing.T) {
	t.Parallel()
	logs := []byte("TestFoo 1\nTestBar 1\n=== NAME  TestFoo\n    foo.go:123:\n") // a real example would have many more lines without a prefix but this should be enough
	testName := "TestFoo"
	actual, err := filterLogs([]byte(logs), []byte(testName))
	assert.NoError(t, err)
	assert.Equal(t, "TestFoo 1\n=== NAME  TestFoo\n    foo.go:123:\n", string(actual))
}

func TestHasTestFailurePrefix(t *testing.T) {
	t.Parallel()
	logs := []byte("=== NAME  TestFoo\n")
	testName := "TestFoo"
	actual := hasTestFailurePrefix(logs, 0, []byte(testName))
	assert.True(t, actual)
}

func TestParseSummary(t *testing.T) {
	t.Parallel()
	logs := "--- PASS: TestAll (2788.26s)\nksjdfks\n--- FAIL: Bar"
	actual := parseSummary([]byte(logs))
	assert.Equal(t, "--- PASS: TestAll (2788.26s)\n--- FAIL: Bar", string(actual))
}

func TestParseSummaryForSubtests(t *testing.T) {
	t.Parallel()
	logs := "--- PASS: TestAll (2788.26s)\n    --- FAIL: TestAll/foo (10.11s)\n    --- PASS: TestAll/bar (10.12s)"
	actual := parseSummary([]byte(logs))
	assert.Equal(t, logs, string(actual))
}

func TestParseRemoteOwnerAndRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cmd := exec.Command("git", "init", ".")
	cmd.Dir = dir
	assert.NoError(t, cmd.Run())

	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/Octogonapus/TerratestLogViewer.git")
	cmd.Dir = dir
	assert.NoError(t, cmd.Run())

	r, err := git.PlainOpen(dir)
	assert.NoError(t, err)

	owner, repo, err := parseRemoteOwnerAndRepo(r)
	assert.NoError(t, err)
	assert.Equal(t, "Octogonapus", owner)
	assert.Equal(t, "TerratestLogViewer", repo)
}

func TestParseBranch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cmd := exec.Command("git", "init", ".")
	cmd.Dir = dir
	assert.NoError(t, cmd.Run())

	cmd = exec.Command("git", "checkout", "-b", "myBranchName")
	cmd.Dir = dir
	assert.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "--allow-empty", "-m", "msg")
	cmd.Dir = dir
	assert.NoError(t, cmd.Run())

	r, err := git.PlainOpen(dir)
	assert.NoError(t, err)

	branch, err := parseBranch(r)
	assert.NoError(t, err)
	assert.Equal(t, "myBranchName", branch)
}
