# parallel-git-repo

Run command on git repositories in parallel.

## Installation

### From source, clone the repository, then

    go install

## Usage

Configure the repositories list where command will be run in `$HOME/.parallel-git-repositories`:

```yaml
repositories:
  - /Users/jcgay/dev/maven-notifier
  - /Users/jcgay/dev/maven-color
```

This is a [`YAML`](http://www.yaml.org) file.

List available commands:

    parallel-git-repo -h

Example when running `pull` command:

```
> parallel-git-repo pull
maven-color: ✔
maven-notifier: ✔
```

## Build

### Status

[![Build Status](https://travis-ci.org/jcgay/parallel-git-repo.svg?branch=master)](https://travis-ci.org/jcgay/parallel-git-repo)

### Release

- Configure Bintray deployment in `.goxc.local.json`:

```json
{
    "ConfigVersion": "0.9",
    "TaskSettings": {
        "bintray": {
            "apikey": "12d312314235afe56090932ea13234"
        }
    }
}
```

- run `goxc default bintray`

### List available tasks

    goxc -h tasks