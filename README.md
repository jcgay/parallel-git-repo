# parallel-git-repo

Run command on git repositories in parallel.

## Installation

### Homebrew

    brew install jcgay/jcgay/parallel-git-repo

### Binaries

Download the archive for your platform from the [latest release](https://github.com/jcgay/parallel-git-repo/releases/latest).

### Go style

    go install github.com/jcgay/parallel-git-repo@latest

## Configuration

Configure the repositories list where command will be run in `$HOME/.parallel-git-repositories`.

Point to a different file with the `-c` flag or the `PARALLEL_GIT_REPO_CONFIG` environment variable (flag wins, then env var, then the default above). This lets a team version its config inside a repository, or keep separate work and personal setups:

```
$ parallel-git-repo -c ./team-repos.toml -g all status
$ PARALLEL_GIT_REPO_CONFIG=~/work.toml parallel-git-repo fetch
```

```
[repositories]
  default = [
    "/Users/jcgay/dev/maven-notifier",
    "/Users/jcgay/dev/maven-color"
  ]
```

The group `default` is mandatory.

You can create groups of repositories to separate them by concern:

```
[repositories]
  default = [
    "/Users/jcgay/dev/maven-color",
    "/Users/jcgay/dev/buildplan-maven-plugin"
  ]
  notifier = [
    "/Users/jcgay/dev/maven-notifier",
    "/Users/jcgay/dev/gradle-notifier"
  ]
```

Also define commands that you want to run on these repositories:

```
[commands]
  fetch = "git fetch -p"
  status = "git status"
  pull = "git pull"
  push = "git push $@"
  checkout = "git checkout $@"
  current-branch = "git rev-parse --abbrev-ref HEAD"
  merge = "git merge --log --no-ff $@"
  set-version = "mvn versions:set -DnewVersion=$1"
  ismerged = "git branch --list $1 -r --merged"
  contains = "git branch -r --contains $1"
```

This is a [`TOML`](https://github.com/toml-lang/toml) file. Instead of editing it by hand you can register a repository with the `add` command:

```
$> cd ~/dev/new-project && parallel-git-repo add
Added /Users/jcgay/dev/new-project to group "default"

$> parallel-git-repo add -g notifier ~/dev/gradle-notifier
```

The path defaults to the current directory and the group to `default`; a missing group is created.

A command that uses shell features — quoted arguments, pipes, chaining (`&&`, `;`) or redirection — is run through `/bin/sh`, so it behaves as you would type it in a terminal:

```
[commands]
  gone = "git branch --merged | grep -v master"
  sync = "git fetch -p && git pull"
```

Plain commands (no shell metacharacters) are executed directly, without a shell. In both cases the `$@` and `$1`, `$2`… placeholders are replaced with the arguments you pass on the command line.

## Usage

### List available commands:

    parallel-git-repo -h

Example when running `pull` command:

```
$> parallel-git-repo pull
maven-color: ✔
maven-notifier: ✔
```

### Run an arbitrary command:

    parallel-git-repo run git remote -v

### Run command for a specific group

    parallel-git-repo -g=notifier status

Pass several comma-separated groups, or `all` to run over every group (repositories shared by several groups are only run once):

    parallel-git-repo -g=notifier,maven status
    parallel-git-repo -g=all fetch

### Limit how many commands run in parallel

By default at most 8 commands run at once. Use `-j` to change the limit (`-j 1` runs sequentially):

    parallel-git-repo -j 4 pull

## Build

### Status

[![Build Status](https://github.com/jcgay/parallel-git-repo/actions/workflows/go.yml/badge.svg)](https://github.com/jcgay/parallel-git-repo/actions/workflows/go.yml)
[![Code Report](https://goreportcard.com/badge/github.com/jcgay/parallel-git-repo)](https://goreportcard.com/report/github.com/jcgay/parallel-git-repo)

### Release

Push a tag; the release workflow builds the binaries with [GoReleaser](https://goreleaser.com) and publishes them to GitHub Releases:

    make tag
    git push origin <version>

### List available tasks

    make help
