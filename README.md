# parallel-git-repo

## Build

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