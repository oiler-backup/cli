# Oiler CLI

This package contains source code for Command-Line Interface to [oiler-backup Kubernetes Operator](https://github.com/oiler-backup/core). This utility allows you to perform basic actions with the operator, the list of actions is listed below, and is also available through the 'help' parameter.

## Available Commands

|Command|Purpose|Flags|Usage|
|-------|-------|-----|-----|
| adapter | Manage adapters | - | oiler-cli adapter [command] |
| adapter add | Add an adapter to the ConfigMap | - | oiler-cli adapter add \<name>=\<url> |
| adapter delete | Delete an adapter from the ConfigMap | - | oiler-cli adapter delete \<name> |
| adapter list | List all adapters from the ConfigMap | - | oiler-cli adapter list |
| backup | Manage BackupRequests | - | oiler-cli backup [command] |
| backup list | List all BackupRequest resources in the cluster. | - | oiler-cli backup list |
| backup delete | Delete a BackupRequest | - | oiler-cli backup delete \<name> |
| backup update | Update a field in a BackupRequest in the specified namespace. | - | oiler-cli backup update \<name> \<field>=\<value> |
| backup create | Create a BackupRequest | --db - DB specification in the format dbType@dbUri:dbPort/dbName (default "") | oiler-cli backup create [flags] |
| |  | --db-user - Database User (default "") | |
| |  | --db-pass - Database Pass (default "") | |
| |  | --db-user-stdin - Read user from terminal (Recommended) | |
| |  | --db-pass-stdin - Read password from terminal (Recommended) | |
| |  | --s3 - S3 specification in the format endpoint:port/bucket (default "") | |
| |  | --s3-access-key - S3 access key (default "") | |
| |  | --s3-secret-key - S3 secret key (default "") | |
| |  | --s3-access-key-stdin - Read access-key from terminal (Recommended) | |
| |  | --s3-secret-key-stdin - Read secret-key from terminal (Recommended) | |
| |  | --schedule - Cron schedule for backups (default "*/1 * * * *") | |
| |  | --max-backup-count - Maximum number of backups to retain (default 2) | |
| |  | --name - Name of the BackupRequest (default "") | |
| config | Display the current configuration | - | oiler-cli config [command] |
| config get | Display the current configuration | - | oiler-cli config get |
| config set | Set a configuration parameter | - | oiler-cli config set \<parameter>=\<value> |
| help | Help about any command | - | oiler-cli help [command] |

## Installation

1. Run `make build`
2. Move generated binary from `./bin` somewhere to `$PATH`
3. Try `oiler --help`
4. To cleanup run `make clean`

## Configuration

Configuration file is stored at `/home/${whoami}/.oiler/.config.json`.
It must contain two records:
- kube_config_path - Path to kubeconfig to login to cluster
- namespace - System namespace, where oiler-backup Kubernetes Operator is deployed to