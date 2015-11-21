# parallel-git-repo

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