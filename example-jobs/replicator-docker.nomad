job "replicator" {
  datacenters = ["dc1"]
  region      = "global"
  type        = "service"

  update {
    stagger      = "10s"
    max_parallel = 1
  }

  meta {
    VERSION = "v1.0.0"
  }

  group "replicator" {
    count = 2

    task "replicator" {
      driver = "docker"

      config {
        image        = "elsce/replicator:${NOMAD_META_VERSION}"
        network_mode = "host"
        args         = [
          "agent",
        ]
      }

      resources {
        cpu    = 250
        memory = 60

        network {
          mbits = 5
        }
      }
    }
  }
}
