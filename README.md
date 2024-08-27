# helm-clean
A helm plugin to clean release by date

# Installation

```
helm plugin install https://github.com/joelee2012/helm-clean/releases/download/v0.1.0/helm-clean_0.1.0_linux_amd64.tar.gz
```

# Usage

```
helm clean -h
Clean/List the release which was updated before duration

Examples:
        # List all release which was updated before 240h
        helm clean -A -b 240h

        # List release was create by chart-1
        helm clean -A -b 240h -f chart-1

        # Exclude namespace match pattern
        helm clean -A -b 240h -e kube-system

        # Exclude release match pattern
        helm clean -A -b 240h -e ':release-1'

        # Exclude release and namespace match pattern
        helm clean -A -b 240h -e '.*-namespace:.*-release'

Usage:
  clean [flags]

Flags:
  -A, --all-namespaces     Check releases across all namespaces
  -b, --before helm list   The last updated time before now, eg: 8h, (default 0) equal helm list
  -d, --dry-run            Dry run mode only print the release info (default true)
  -e, --exclude strings    Regular expression '<namespace>:<release>', the matched 
                           release and namespace will be excluded from the result (can specify multiple)
  -f, --filter strings     Regular expression, the chart of releases that matched the 
                           expression will be included in the result only (can specify multiple)
  -h, --help               help for clean
  -v, --version            version for clean
```

