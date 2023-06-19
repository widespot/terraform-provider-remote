terraform {
  required_providers {
    remote = {
      source = "registry.terraform.io/widespot/remote"
    }
  }
}

provider "remote" {
  username          = "root"
  host              = "localhost:8022"
  private_key_path  = "./id_rsa"
}

resource "remote_folder" "edu" {
  count      = 1
  path       = "/tmp/tests6"
  owner_name = "root"
  group_name = "root"
}
