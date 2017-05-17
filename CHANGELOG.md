## 0.0.2 (Unreleased)

IMPROVEMENTS:

* Allow Dry Run Operations Against Job Scaling Documents. [GH-53]
* Add the `scaling_interval config` parameter to replicator agent. [GH-49]
* Increase Default Cluster Scaling Cooldown Period to 600s. [GH-56]
* Addition of CLI flags for all configuration parameters. [GH-61]
* Add support for `dev` CLI flag [GH-63]
* AWS 'TerminateInstance' function now verifies instance terminated state. [GH-80]

BUG FIXES:

* Prevent Divide By Zero Panic When Replicator Detects No Healthy Worker Nodes. [GH-55]

## 0.0.1 (02 May 2017)

- Initial release.
