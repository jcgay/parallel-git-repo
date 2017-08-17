# parallel-git-repo

Run command on git repositories in parallel.

## Installation

### From source, clone the repository, then

    go install

## Configuration

Configure the repositories list where command will be run in `$HOME/.parallel-git-repositories`:

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

This is a [`TOML`](https://github.com/toml-lang/toml) file.

## Usage

### List available commands:

    parallel-git-repo -h

Example when running `pull` command:

```
> parallel-git-repo pull
maven-color: ✔
maven-notifier: ✔
```

### Run an arbitrary command:

    parallel-git-repo run git remote -v

### Run command for a specific group

    parallel-git-repo -g=notifier status

## Build

### Status

[![Build Status](https://travis-ci.org/jcgay/parallel-git-repo.svg?branch=master)](https://travis-ci.org/jcgay/parallel-git-repo)
[![Code Report](https://goreportcard.com/badge/github.com/jcgay/parallel-git-repo)](https://goreportcard.com/report/github.com/jcgay/parallel-git-repo)
[![Coverage Status](https://coveralls.io/repos/github/jcgay/parallel-git-repo/badge.svg?branch=master)](https://coveralls.io/github/jcgay/parallel-git-repo?branch=master)

### Release

    make release

### List available tasks

    make help