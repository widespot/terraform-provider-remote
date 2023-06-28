# Terraform provider for remote files and folders

Are you looking for a `.tf` provider to manage your files on remote linux hosts? 
Leverage an `ssh` connectivity to track
- [x] file content
- [x] file & folder permissions
- [x] file & folder ownership
- [x] file & folder group

## Usage
```terraform
provider "remote" {
  username          = "root"
  host              = "localhost:8022"
  private_key_path  = "./id_rsa"
}

resource "remote_folder" "folder" {
  path       = "/tmp/tests11"
  owner_name = "root"
  group_name = "root"
}

resource "remote_file" "file" {
  path       = "${remote_folder.folder.path}/test.txt"
  content    = "blabetiblou"
  owner_name = "root"
  group_name = "root"
}
```

## Development
```shell
# Build last version (99.0.0) in playground directory
make install
# Deploy 
make playground
# Try connectivity to remote, using password
ssh root@localhost -p 8022
> root@127.0.0.1\'s password: password
# Try connectivity to remote, using identity file
ssh root@localhost -p 8022 -i playground/id_rsa
# Play with playground/main.tf
terraform -chdir=playground apply
```
