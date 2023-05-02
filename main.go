package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

func main() {
	owner := flag.String("owner", "", "repository owner")
	repo := flag.String("repository", "", "repository name")
	workflowFilename := flag.String("workflow", "", "workflow filename (base filename, not path)")
	branch := flag.String("branch", "", "branch name")
	jobName := flag.String("job", "", "job name (within the workflow file)")
	testName := flag.String("test", "", "Go test name. All log data is returned otherwise.")
	token, hasToken := os.LookupEnv("GITHUB_TOKEN")

	flag.Parse()

	if len(*owner) == 0 {
		panic("owner is a required parameter. see usage via --help")
	}
	if len(*repo) == 0 {
		panic("repo is a required parameter. see usage via --help")
	}
	if len(*workflowFilename) == 0 {
		panic("workflowFilename is a required parameter. see usage via --help")
	}
	if len(*branch) == 0 {
		panic("branch is a required parameter. see usage via --help")
	}
	if len(*jobName) == 0 {
		panic("jobName is a required parameter. see usage via --help")
	}

	var gh *github.Client
	if hasToken {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(ctx, ts)
		gh = github.NewClient(tc)
	} else {
		gh = github.NewClient(nil)
	}

	logs, err := getLogs(gh, *owner, *repo, *workflowFilename, *branch, *jobName)
	if err != nil {
		panic(err)
	}

	logs = removeTimestampPrefix(logs)

	if len(*testName) > 0 {
		logs, err = filterLogs(logs, []byte(*testName))
		if err != nil {
			panic(err)
		}
	}

	fmt.Println(string(logs))
}

// Returns new logs.
// Removes the timestamp prefix from each line of the logs.
func removeTimestampPrefix(logs []byte) []byte {
	newLogs := []byte{}
	for i := 0; i < len(logs); {
		endOfTimestampIdx := findNext(logs, i, ' ')
		endOfLineIdx := findNext(logs, endOfTimestampIdx+1, '\n')
		line := logs[endOfTimestampIdx+1 : endOfLineIdx+1]
		newLogs = append(newLogs, line...)
		i = endOfLineIdx + 1
	}
	return newLogs
}

// Returns new logs.
// Includes log lines which begin with the given test name.
// Also includes lines with appear to be part of the given test, but which do not start with the given test name.
func filterLogs(logs []byte, testName []byte) ([]byte, error) {
	filteredLogs := []byte{}

	i := 0
	priorLineMatchedPrefix := false
	for {
		if i >= len(logs) {
			break
		}

		endOfLineIdx := findNext(logs, i, '\n')

		// if the line has the testName prefix, add the line to filteredLogs
		if hasPrefix(logs, i, testName) {
			line := logs[i : endOfLineIdx+1]
			filteredLogs = append(filteredLogs, line...)
			priorLineMatchedPrefix = true
		} else {
			// extend the "selection" to lines that don't have the prefix if we haven't moved to a new test yet
			// Go tests must start with "Test" so we can use this as a filter to know when we moved to a new test
			if priorLineMatchedPrefix {
				if hasPrefix(logs, i, []byte("Test")) {
					priorLineMatchedPrefix = false
				} else {
					line := logs[i : endOfLineIdx+1]
					filteredLogs = append(filteredLogs, line...)
				}
			}
		}

		i = endOfLineIdx + 1 // advance to next line
	}

	return filteredLogs, nil
}

// Returns whether the given string, starting at the given offset, equals the given prefix for the length of the given prefix.
func hasPrefix(str []byte, offset int, prefix []byte) bool {
	for j := 0; j < len(prefix); j++ {
		if str[offset+j] != prefix[j] {
			return false
		}
	}
	return true
}

// Returns the next index of the next given character in the given string, or the last index of the given string.
func findNext(str []byte, offset int, test byte) int {
	for i := offset; i < len(str); i++ {
		if str[i] == test {
			return i
		}
	}
	return len(str) - 1
}

// Returns the content of the log for the most recent job matching the given parameters.
func getLogs(gh *github.Client, owner string, repo string, workflowFilename string, branch string, jobName string) ([]byte, error) {
	runs, _, err := gh.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, workflowFilename, &github.ListWorkflowRunsOptions{Branch: branch})
	if err != nil {
		return nil, err
	}

	latestRunID := runs.WorkflowRuns[0].ID

	jobs, _, err := gh.Actions.ListWorkflowJobs(context.Background(), owner, repo, *latestRunID, &github.ListWorkflowJobsOptions{})
	if err != nil {
		return nil, err
	}

	jobID := int64(-1)
	for _, job := range jobs.Jobs {
		if *job.Name == jobName {
			jobID = *job.ID
			break
		}
	}
	if jobID == -1 {
		return nil, fmt.Errorf("did not find matching job")
	}

	_, logsGHResp, err := gh.Actions.GetWorkflowJobLogs(context.Background(), owner, repo, jobID, false)
	if err != nil {
		return nil, err
	}

	logsResp, err := http.Get(logsGHResp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}
	defer logsResp.Body.Close()

	logsBody, err := io.ReadAll(logsResp.Body)
	if err != nil {
		return nil, err
	}

	return logsBody, nil
}
