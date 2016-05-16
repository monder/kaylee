# Kaylee

Kaylee is a container management system built on top of [fleet] and [etcd] as a lightweight alternative to [kubernetes] or [nomad].

## Overview

Kaylee is written in go an consists of single binary for both cli and server. 

Server reads a configuration from etcd and schedules/destroys units in fleet based on changes.
Units are scheduled with prefix `k:` and wont affect other units, so it could be used safely with regular systemd ones.

## Getting started

### Requirements
- [fleet]
- [etcd]
- [flannel]
- [docker]

### Example unit

Unit file is a `yaml` file describing the job.

Sample unit file:
```yaml
name: countly
spec:
  replicas: 4
  maxReplicasPerHost: 2
  image: monder/countly-docker:latest
  env:
  - name: MONGO_URL
    value: mongodb://mongo-1.k.lan,mongo-2.k.lan,mongo-3.k.lan/countly?replicaSet=main
  machine:
  - 'ssd=false'    
```

- The above file will scheule 4 fleet units (`replicas: 4`).
- Fleet units will have 2 unique hashes in the name as well as a `Conflicts` section to allow maximum 2 (`maxReplicasPerHost: 2`) units per host.
- The `ExecStart` of the unit will have a `docker run` command with `monder/countly-docker:latest` as image name.
- Additionally it will pass environment variables specified in `env` as `-e` docker arguments.
`ssd=false` will be added to `X-Fleet` as `MachineMetadata`.
- `name` will be used in [Container registration](#container-registration)

Please see the [Units](#units) for full specification.

### Run a server
```
kaylee server 
```
Optional arguments:
```
OPTIONS:
   --fleet-prefix "/_coreos.com/fleet/"	keyspace for fleet data in etcd
   --unit-prefix "k"			prefix for units in fleet
GLOBAL OPTIONS:
   --etcd-endpoints "http://127.0.0.1:4001,http://127.0.0.1:2379"	a comma-delimited list of etcd endpoints
   --etcd-prefix "/kaylee"						a keyspace for unit data in etcd
   --help, -h								show help
   --version, -v							print the version
```
Server connects and interacts with fleet via etcd.
`DOCKER_HOST` env variable should be defined to connect to docker instance.

If mutiple instances of server are running the leader election will take place and only one will be active. At least 2 servers should be used in production to provide high-availability.

### CLI

When the server is running, the same application could be used to schedule the units.
`--etcd-endpoints` and `--etcd-prefix` should be specified to connect to the same cluster.

- `kaylee run <unit file>` schedules the unit or updates an existing one with the same name. File name does not matter.
- `kaylee ls` lists scheduled unit names
- `kaylee restart <unit name>` reschedules the unit without any changes

## Container registration

Each `kaylee` server started listens to a local machine's docker `start` and `die` events and exposes container IP addresses in `etcd`

For example when the unit named `api` is started, the `etcd` entry
`/kaylee/instances/9ac71e49-4c93-465a-9571-2c0175f73449/api/k_api_19c656_dc608f_1` will have docker IP address of the container, which is reachable via `flannel`. That is why no port specification needed for the unit.

It could be easily combined with moudules like [monder/route53-etcd] to add the container addresses to route53 using `/kaylee/instances/*/api/*` as a config.

## Units

Each unit consists of one `yaml` file with `name` and `spec` at the root level.
```yaml
name: 'api'
spec:
  replicas: 2           # Number of instances to scheule. Default: 1
  maxReplicasPerHost: 1 # Maximum number of instances on this unit on the host. Optional.
  image: mongo:3.2      # Docker image to run. Required.
  cmd: --quiet          # Command to pass to docker to lauch this image. Optional.
  envFiles:             # Files to load environment variables from. Optional.
  - '/etc/environment'
  env:                  # Array of environment variables. Optional
  - name: SOME_VAR
    value: some_value
  volumes:              # Volumes for docker to mount. It will run dokcer volume create before the actual command. Optional
  - id: 'vol-88bc607a'  # name of the volume
    driver: 'blocker'   # driver to pass as --driver
    path: '/data'       # mount path
    options: 'a=1'      # options to pass to the driver. Default ''
  machine:              # Array of machine fleet constraints.
  - 'ssd=true'
  - 'region=eu-west-1'
  machineId: 'b15ba...' # Id of the machine. If specified - other constrains and number of replicas wont work.
  dockerArgs:           # Additional arguments to pass to docker command
  - '--security-opt=label:disable'
  global: false         # Make this unit global. Number of replicas wont work in this configuration
  conflicts:            # Additional fleet constraints if any. Added to X-Fleet/Conflicts as is.
  - 'mongo-1'
```

## Scheduling a unit

For CI purposes it is possible to schedule units without using CLI.
Scheduling or updating a unit coul be one by creating/updating an entry under `/kaylee/units/` in etcd with content of the unit file in JSON format.
For example the [sample above](#example-unit) is the entry `/kaylee/units/countly` with content below:
```json
{"name":"countly","spec":{"replicas":4,"maxReplicasPerHost":2,"image":"monder/countly-docker:latest","env":[{"name":"MONGO_URL","value":"mongodb://mongo-1.k.lan,mongo-2.k.lan,mongo-3.k.lan/countly?replicaSet=main"}],"machine":["ssd=false"]}}
```


## License
MIT

[fleet]: https://github.com/coreos/fleet
[etcd]: https://github.com/coreos/etcd
[kubernetes]: http://kubernetes.io
[nomad]: https://www.nomadproject.io
[monder/route53-etcd]: https://github.com/monder/route53-etcd
[flannel]: https://github.com/coreos/flannel
[docker]: https://www.docker.com
