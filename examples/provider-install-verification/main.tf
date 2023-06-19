terraform {
  required_providers {
    hashicups = {
      source = "hashicorp.com/edu/hashicups-pf"
    }
  }
}

provider "hashicups" {
    username = "root"
    password = "password"
    host = "localhost:8022"
}

resource "hashicups_folder" "edu" {
  count = 1
  path = "/tmp/tests6"
  owner_name = "root"
  group_name = "root"
}

data "hashicups_coffees" "example" {}
