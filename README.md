# File Monitor (FMon)

Filesystem Management / Monitoring Tool.

Originally created for easier cleanup of folders that 'decrease in value'.

Think of this concept as caches filling up, your docker `vhdx` file getting too big or virtual environments becoming unescessary storage space after a while.

FMon allows you to easily bind actions to filesystem events or as a cronjob.

Actions refer to your [shell script](./docs/shell/README.md). This 'shell' script can either be written in javascript (cross platform), sh (linux/darwin) or powershell/batch files (windows), allowing for maximum flexibility.

**Installation:**

```Shell
go install github.com/Nigel2392/fmon@latest
```

## Available Commands

The following commands are available for setting up your service.

### `fmon config`

[Manage the configuration file.](./docs/config.md)

### `fmon service`

[Manage the service.](./docs/service.md)

### `fmon watcher`

[Manage the watcher.](./docs/watcher.md)

## Command Help

Help about any command
