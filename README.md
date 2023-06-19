# Terraform provider for remote files and folders

Are you looking for a `.tf` provider to manage your files on remote linux hosts? 
Leverage an `ssh` connectivity to track
- [x] file content
- [x] file & folder permissions
- [x] file & folder ownership
- [x] file & folder group

## Contribution
```
export TF_CLI_CONFIG_FILE=terraform.tfrc
docker-compose up -d -f tests/docker-compose.yml
ssh root@localhost -p 8022
root@127.0.0.1's password: password
```
