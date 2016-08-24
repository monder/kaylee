# Kaylee

Kaylee is a container management system built on top of [fleet] and [etcd] as a lightweight alternative to [kubernetes] or [nomad].

## Overview

Kaylee is written in go an consists of single binary for both cli and server. 

Server reads a configuration from etcd and schedules/destroys units in fleet based on changes.
Units are scheduled with prefix `k2:` and wont affect other units, so it could be used safely with regular systemd ones.

## Getting started

### Requirements
- [fleet]
- [etcd]
- [flannel]
- [rkt]

### Example unit

Unit file is a `yaml` file describing the job.

Sample unit file:
```yaml
# vim: ts=2:sw=2
name: countly
replicas: 4
maxReplicasPerHost: 2
env:
- name: MONGO_URL
  value: mongodb://mongo-1.lan,mongo-2.lan,mongo-3.lan/countly?replicaSet=main
machine:
- ssd=false
args:
- --dns=10.0.0.2
- --net=flannel,default
apps:
- image: docker://monder/countly-docker:latest
- image: monder.cc/rkt-sidekick:v0.0.3
  args:
  - --cidr 10.5.0.0/16
  - --interval 10m
  - --expireDir /services
  - --format '$$ip:80'
  - /services/countly/${KAYLEE_ID}
```

- The above file will scheule 4 fleet units (`replicas: 4`).
- Fleet units will have 2 unique hashes in the name as well as a `Conflicts` section to allow maximum 2 (`maxReplicasPerHost: 2`) units per host.
- The `ExecStart` of the unit will have a `rkt run` command with `--dns=` and `--net=` arguments as well as two applications - `docker://monder/countly-docker:latest` and `monder.cc/rkt-sidekick:v0.0.3` with cpecified arguments.
- Additionally it will pass environment variables specified in `env` as `-e` docker arguments.
`ssd=false` will be added to `X-Fleet` as `MachineMetadata`.

Please see the [Units](#units) for full specification.

### Run a server
```
kaylee server 
```
Optional arguments:
```
OPTIONS:
   --fleet-prefix "/_coreos.com/fleet/"	keyspace for fleet data in etcd
   --unit-prefix "k2"			prefix for units in fleet
GLOBAL OPTIONS:
   --etcd-endpoints "http://127.0.0.1:4001,http://127.0.0.1:2379"	a comma-delimited list of etcd endpoints
   --etcd-prefix "/kaylee2"					a keyspace for unit data in etcd
   --help, -h								show help
   --version, -v							print the version
```
Server connects and interacts with fleet via etcd api.

If mutiple instances of server are running the leader election will take place and only one will be active. At least 2 servers should be used in production to provide high-availability.

### CLI

When the server is running, the same application could be used to schedule the units.
`--etcd-endpoints` and `--etcd-prefix` should be specified to connect to the same cluster.

- `kaylee run <unit file>` schedules the unit or updates an existing one with the same name. File name does not matter.
- `kaylee ls` lists scheduled unit names
- `kaylee restart <unit name>` reschedules the unit without any changes

## Units

Each unit consists of one `yaml` file with `name` and other options.
```yaml
name: api
replicas: 2                 # Number of instances to scheule. Default: 1
maxReplicasPerHost: 1       # Maximum number of instances on this unit on the host. Optional.
envFiles:                   # Files to load environment variables from. Optional.
- /etc/environment
env:                        # Array of environment variables. Optional
- name: SOME_VAR
  value: some_value
machine:                    # Array of machine fleet constraints.
- ssd=true
- region=eu-west-1
machineId: 'b15ba...'       # Id of the machine. If specified - other constrains and number of replicas wont work.
args:                       # Additional arguments to pass to rkt command
- --dns=10.0.0.2
global: false               # Make this unit global. Number of replicas wont work in this configuration
conflicts:                  # Additional fleet constraints if any. Added to X-Fleet/Conflicts with kaylee unit naming.
- 'mongo-n*'                # Will add k2:mongo-n*:*.service to X-Fleet/Conflicts
apps:                       # Array of applications to lauch as part of a single pod
- image: docker://mongo:3.2 # Image name/location/hash
- image: monder.cc/rkt-sidekick:v0.0.3
  cmd: /bin/rkt-sidekick    # Override command (--exec)
  args:                     # Arguments to pass to this particular application
  - --cidr 10.5.0.0/16
  - --interval 10m
  - --expireDir /services
  - --format '$$ip:80'      # Note the escaping of $
  - /services/mongo/${KAYLEE_ID} # KAYLEE_ID environment variable is unique for each run
volumes:  
- source: /mnt/efs/aaa      # Path on host
  path: /etc/confd          # Path in container
```

## Scheduling a unit

For CI purposes it is possible to schedule units without using CLI.
Scheduling or updating a unit coul be one by creating/updating an entry under `/kaylee2/units/` in etcd with content of the unit file in JSON format.
For example the [sample above](#example-unit) is the entry `/kaylee2/units/countly` with content below:
```json
{"name":"countly","replicas":4,"maxReplicasPerHost":2,"env":[{"name":"MONGO_URL","value":"mongodb://mongo-1.lan,mongo-2.lan,mongo-3.lan/countly?replicaSet=main"}],"apps":[{"image":"docker://monder/countly-docker:latest"},{"image":"monder.cc/rkt-sidekick:v0.0.3","args":["--cidr 10.5.0.0/16","--interval 10m","--expireDir /services","--format '$$ip:80'","/services/countly/${KAYLEE_ID}"]}],"args":["--dns=10.0.0.2","--net=flannel,default"],"machine":["ssd=false"]}
```


## License
MIT

[fleet]: https://github.com/coreos/fleet
[etcd]: https://github.com/coreos/etcd
[kubernetes]: http://kubernetes.io
[nomad]: https://www.nomadproject.io
[flannel]: https://github.com/coreos/flannel
[rkt]: https://github.com/coreos/rkt
