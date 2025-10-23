
resource "remote_folder" "edu" {
  count      = 1
  path       = "/tmp/tests11"
  owner_name = "raphaeljoie"
  #group_name = "root"
}

resource "remote_file" "edu" {
  count      = 1
  ensure_dir = true
  path       = "${remote_folder.edu[0].path}/blabetiblou/lol/test.txt"
  content    = "blabetiblou"
  owner_name = "raphaeljoie"
  #group_name = "root"
}
