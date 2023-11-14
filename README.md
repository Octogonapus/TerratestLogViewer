# TerratestLogViewer

A commandline utility to retrieve and filter logs from Terratest.
Filters out a single test's logs from interleaved parallel log output.
Automatically downloads logs from GitHub Actions.

```sh
export GITHUB_TOKEN="my token"

# Git state can be pulled from your shell context
TerratestLogViewer ---workflow my_workflow.yml --job my_job --test TestSomething | less

# Print a summary of test results with no test logs
TerratestLogViewer ---workflow my_workflow.yml --job my_job --summary

# Fully specified
TerratestLogViewer --owner MyOrg --repository myRepo --workflow my_workflow.yml --branch my_branch --job my_job --test TestSomething | less
```

## Install

### Binary Installation

Download an appropriate binary from the [latest release](https://github.com/Octogonapus/TerratestLogViewer/releases/latest).

### Manual Installation

```sh
git clone https://github.com/Octogonapus/TerratestLogViewer
cd TerratestLogViewer
go build
go install
```
