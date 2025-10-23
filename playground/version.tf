terraform {
  required_providers {
    remote = {
      source = "registry.terraform.io/widespot/remote"
      #version = "99.0.0"
    }
  }
}

provider "remote" {
  username          = "root"
  host              = "localhost:8022"
  private_key_path  = "./id_rsa"
}
