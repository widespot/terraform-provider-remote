terraform {
  required_providers {
    hashicups = {
      source = "hashicorp.com/edu/hashicups-pf"
    }
  }
}

provider "hashicups" {
  username          = "root"
  host              = "localhost:8022"
  private_key_path  = "../../tests/id_rsa"
}

resource "hashicups_folder" "edu" {
  count      = 1
  path       = "/tmp/tests6"
  owner_name = "root"
  group_name = "root"
}
