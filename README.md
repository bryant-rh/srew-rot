# What is Srew-rot?

srew-bot is a command line tools for updating plugin manifests in srew Index repo

With the use of srew
https://github.com/bryant-rh/srew


# Usage

## First Step

create .srew.yaml
```Bash
apiVersion: srew.sensors.com/v1alpha2
kind: Plugin
metadata:
  name: resource-view
spec:
  version: {{ .TagName }}
  homepage: https://github.com/bryant-rh/kubectl-resource-view
  shortDescription: Display Resource (CPU/Memory/PodCount) Usage and Request and Limit.
  description: |
    This plugin Display Resource (CPU/Memory/PodCount) Usage and Request and Limit.
    The resource command allows you to see the resource consumption for nodes or pods.
    This command requires Metrics Server to be correctly configured and working on the server.
  caveats: |
    Usage:
      kubectl-resource-view [flags] [options]
      kubectl-resource-view [command]

    Examples:
      node        Display Resource (CPU/Memory/PodCount) usage of nodes
      pod         Display Resource (CPU/Memory)          usage of pods
  platforms:
  - selector:
      matchLabels:
        os: darwin
        arch: amd64
    {{addURIAndSha "https://github.com/bryant-rh/kubectl-resource-view/releases/download/{{ .TagName }}/kubectl-resource-view_{{ .TagName }}_darwin_amd64.tar.gz" .TagName }}
    bin: kubectl-resource-view
  - selector:
      matchLabels:
        os: darwin
        arch: arm64
    {{addURIAndSha "https://github.com/bryant-rh/kubectl-resource-view/releases/download/{{ .TagName }}/kubectl-resource-view_{{ .TagName }}_darwin_arm64.tar.gz" .TagName }}
    bin: kubectl-resource-view
  - selector:
      matchLabels:
        os: linux
        arch: amd64
    {{addURIAndSha "https://github.com/bryant-rh/kubectl-resource-view/releases/download/{{ .TagName }}/kubectl-resource-view_{{ .TagName }}_linux_amd64.tar.gz" .TagName }}
    bin: kubectl-resource-view
  - selector:
      matchLabels:
        os: windows
        arch: amd64
    {{addURIAndSha "https://github.com/bryant-rh/kubectl-resource-view/releases/download/{{ .TagName }}/kubectl-resource-view_{{ .TagName }}_windows_amd64.tar.gz" .TagName }}
    bin: kubectl-resource-view.exe
```

## Second Step

```Bash
export SREW_SERVER_BASEURL="[YOUR SREW-SERVER URL]"
export SREW_SERVER_USERNAME="[YOUR SREW-SERVER USERNAME]"
export SREW_SERVER_PASSWORD="[YOUR SREW-SERVER PASSWORD]"

```

## Third Step

```Bash
# ./srew-rot -h

srew-bot is a command line tools for updating plugin manifests in srew Index repo

Usage:
  srew-bot [command]

Available Commands:
  help        Help about any command
  template    template helps validate the krew index template file without going through github actions workflow

Flags:
  -h, --help   help for srew-bot

Use "srew-bot [command] --help" for more information about a command.
```

example:

```Bash
go run main.go template --tag v0.1.0 --template-file .srew.yaml --debug
```