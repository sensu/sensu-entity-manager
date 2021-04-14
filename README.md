[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-entity-manager)
![goreleaser](https://github.com/sensu/sensu-entity-manager/workflows/goreleaser/badge.svg)
[![Go Test](https://github.com/sensu/sensu-entity-manager/workflows/Go%20Test/badge.svg)](https://github.com/sensu/sensu-entity-manager/actions?query=workflow%3A%22Go+Test%22)
[![goreleaser](https://github.com/sensu/sensu-entity-manager/workflows/goreleaser/badge.svg)](https://github.com/sensu/sensu-entity-manager/actions?query=workflow%3Agoreleaser)

# Sensu Entity Manager

## Table of Contents
- [Overview](#overview)
- [Usage examples](#usage-examples)
  - [Help output](#help-output)
  - [Environment variables](#environment-variables)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Handler definition](#handler-definition)
  - [Supported Annotations](#supported-annotations)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

Event-based Sensu entity management for automated service-discovery (add/remove subscriptions) and other automation workflows.
The Sensu Entity Manager works with any check plugin or event producer that generates one instruction per line in any of the following formats:

- **Subscriptions (one string per line):**

  ```
  system/linux
  postgres
  ```

- **Labels and Annotations (one `key=value` pair per line):**

  ```
  region=us-west-2
  application_id=1001
  ```

- **Commands (one space-separated `command argument` pair per line):**

  ```
  add-subscription system/linux
  add-subscription postgres
  add-label region=us-west-2
  add-annotation application_id=1001
  ```

## Usage examples

### Help output

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
      --add-all                  Checks event.Check.Output for a newline-separated list of entity management commands to execute
      --add-annotations          Checks event.Check.Output for a newline-separated list of annotation key=value pairs to add
      --add-labels               Checks event.Check.Output for a newline-separated list of label key=value pairs to add
      --add-subscriptions        Checks event.Check.Output for a newline-separated list of subscriptions to add
  -k, --api-key string           Sensu API Key
  -a, --api-url string           Sensu API URL (default "http://127.0.0.1:8080")
  -h, --help                     help for sensu-entity-manager
  -c, --trusted-ca-file string   Sensu Trusted Certificate Authority file

Use "sensu-entity-manager [command] --help" for more information about a command.
```

### Environment variables

| Argument          | Environment Variable  |
|-------------------|-----------------------|
| --api-url         | SENSU_API_URL         |
| --api-key         | SENSU_API_KEY         |
| --access-token    | SENSU_ACCESS_TOKEN    |
| --trusted-ca-file | SENSU_TRUSTED_CA_FILE |

**Security Note:** Care should be taken to not expose the API key or access token for this handler by explicitly specifying either on the command line or by directly setting the environment variable(s) in the handler definition.
It is suggested to make use of [secrets management][3] to provide the API key or access token as environment variables.
The [handler definition shown below](#handler-definition) references the API Key as a secret using the built-in [env secrets provider][4].

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin.
If you're not using an asset, please consider doing so!
If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the following command to add the asset:

```
sensuctl asset add sensu/sensu-entity-manager
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][2].

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
    --add-all
  timeout: 5
  runtime_assets:
  - sensu/sensu-entity-manager:0.1.1
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

This handler supports the use of the environment variables HTTP_PROXY, HTTPS_PROXY, and NO_PROXY (or the lowercase versions thereof).
HTTPS_PROXY takes precedence over HTTP_PROXY for https requests.
The environment values may be either a complete URL or a "host[:port]", in which case the "http" scheme is assumed.

### Supported Annotations

The following _event-scoped annotations_ are supported.

- `sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions`

  Comma-separated list of subscriptions to add (e.g. `nginx,http-service`).

- `sensu.io/plugins/sensu-entity-manager/config/patch/labels`

  Comma-separated list of key=value pairs to add (e.g. `region=us-west-1,app=example`).

- `sensu.io/plugins/sensu-entity-manager/config/patch/annotations`

  Semicolon-separated list of key=value pairs to add (e.g. `scrape_config="{\"ports\": [9091,9093]}";service_account=sensu`).

> _NOTE: event-scoped annotations are set at the root-level of the event (i.e. `event.Annotations`).
> Entity-scoped (`event.Entity.Annotations`) and Check-scoped (`event.Check.Annotations`) annotations are currently not supported._


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

The preferred way of installing and deploying this plugin is to use it as an Asset.
If you would like to compile and install the plugin from source or contribute to it, download the latest version or create an executable from this source.

From the local path of the sensu-entity-manager repository:

```
go build
```

## Roadmap

- [x] Add support for adding/modifying entity subscriptions
- [x] Add support for adding/modifying entity labels
- [x] Add support for adding/modifying entity annotations
- [ ] Add support for modifying other [entity-patchable fields][11] (e.g.
      `created_by`, `entity_class`, `deregister`, etc).

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://bonsai.sensu.io/assets/sensu/sensu-entity-manager
[3]: https://docs.sensu.io/sensu-go/latest/guides/secrets-management/
[4]: https://docs.sensu.io/sensu-go/latest/guides/secrets-management/#use-env-for-secrets-management
[10]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[11]: https://docs.sensu.io/sensu-go/latest/api/entities/#update-an-entity-with-patch
