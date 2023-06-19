provider "remote" {
  host     = "localhost:8022"
  username = "root"
  password = file(".PASSWORD")
}
