
resource "remote_folder" "edu" {
  count      = 1
  path       = "/tmp/tests11"
  owner_name = "root"
  group_name = "root"
}

resource "remote_file" "edu" {
  count      = 24

  ensure_dir = true
  path       = "${remote_folder.edu[0].path}/blabetiblou/lol/test.${count.index}.txt"
  content    = "blabetiblou"
  owner_name = "root"
  group_name = "root"
}
