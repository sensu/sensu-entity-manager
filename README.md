[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/calebhailey/sensu-entity-manager)
![goreleaser](https://github.com/calebhailey/sensu-entity-manager/workflows/goreleaser/badge.svg)
[![Go Test](https://github.com/calebhailey/sensu-entity-manager/workflows/Go%20Test/badge.svg)](https://github.com/calebhailey/sensu-entity-manager/actions?query=workflow%3A%22Go+Test%22)
[![goreleaser](https://github.com/calebhailey/sensu-entity-manager/workflows/goreleaser/badge.svg)](https://github.com/calebhailey/sensu-entity-manager/actions?query=workflow%3Agoreleaser)

# Sensu Entity Manager

## Table of Contents
- [Overview](#overview)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Handler definition](#handler-definition)
  - [Supported Annotations](#supported-annotations)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

Event-based Sensu entity management for service-discovery (add/remove 
subscriptions) and other automation workflows.                     

## Usage examples

```
$ sensu-entity-manager --help
Event-based Sensu entity management for service-discovery (add/remove subscriptions) and other automation workflows.

Usage:
  sensu-entity-manager [flags]
  sensu-entity-manager [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -t, --access-token string      Sensu Access Token
      --add-subscriptions        Checks event.Check.Output for a newline-separated list of subscriptions to add
  -k, --api-key string           Sensu API Key
  -a, --api-url string           Sensu API URL (default "http://127.0.0.1:8080")
  -h, --help                     help for sensu-entity-manager
  -c, --trusted-ca-file string   Sensu Trusted Certificate Authority file

Use "sensu-entity-manager [command] --help" for more information about a command.
```

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add calebhailey/sensu-entity-manager
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/calebhailey/sensu-entity-manager].

### Handler definition

```yml
---
type: Handler
api_version: core/v2
metadata:
  name: sensu-entity-manager
spec:
  type: pipe
  command: >- 
    sensu-entity-manager 
    --add-subscriptions
  timeout: 5
  runtime_assets:
  - calebhailey/sensu-entity-manager:0.1.1
  secrets:
  - name: SENSU_API_KEY
    secret: entity-manager-api-key
---
type: Secret
api_version: secrets/v1
metadata:
  name: entity-manager-api-key
spec: 
  provider: env
  id: SENSU_ENTITY_MANAGER_API_KEY
```

#### Proxy Support

This handler supports the use of the environment variables HTTP_PROXY,
HTTPS_PROXY, and NO_PROXY (or the lowercase versions thereof). HTTPS_PROXY takes
precedence over HTTP_PROXY for https requests.  The environment values may be
either a complete URL or a "host[:port]", in which case the "http" scheme is assumed.

### Supported Annotations 

The following _event-scoped annotations_ are supported. 

- `sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions`

  Comma-separated list of subscriptions to add (e.g. `nginx,http-service`). 

- `sensu.io/plugins/sensu-entity-manager/config/patch/labels`

  Comma-separated list of key=value pairs to add (e.g. `region=us-west-1,app=example`).

- `sensu.io/plugins/sensu-entity-manager/config/patch/annotations`

  Semicolon-separated list of key=value pairs to add (e.g. 
  `scrape_config="{\"ports\": [9091,9093]}";service_account=sensu`). 

> _NOTE: event-scoped annotations are set at the root-level of the event 
> (i.e. `event.Annotations`). Entity-scoped (`event.Entity.Annotations`) and 
> Check-scoped (`event.Check.Annotations`) annotations are currently not 
> supported._


#### Examples

To change the example argument for a particular check, for that checks's metadata add the following:

```yml
type: CheckConfig
api_version: core/v2
metadata:
  annotations:
    sensu.io/plugins/sensu-entity-manager/config/example-argument: "Example change"
[...]
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-entity-manager repository:

```
go build
```

## Roadmap

- [x] Add support for adding/modifying entity subscriptions
- [ ] Add support for adding/modifying entity labels  
- [ ] Add support for adding/modifying entity annotations  
- [ ] Add support for modifying other [entity-patchable fields][11] (e.g. 
      `created_by`, `entity_class`, `deregister`, etc).

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://github.com/sensu-community/sensu-plugin-sdk
[3]: https://github.com/sensu-plugins/community/blob/master/PLUGIN_STYLEGUIDE.md
[4]: https://github.com/sensu-community/handler-plugin-template/blob/master/.github/workflows/release.yml
[5]: https://github.com/sensu-community/handler-plugin-template/actions
[6]: https://docs.sensu.io/sensu-go/latest/reference/handlers/
[7]: https://github.com/sensu-community/handler-plugin-template/blob/master/main.go
[8]: https://bonsai.sensu.io/
[9]: https://github.com/sensu-community/sensu-plugin-tool
[10]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[11]: https://docs.sensu.io/sensu-go/latest/api/entities/#update-an-entity-with-patch 