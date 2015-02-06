# Basic OpenStack Example

This example does the following:

* Imports a keypair stored under `key` (not provided).
* Creates a security group that allows access from any IPv4 or IPv6 address to port 22.
* Launches an instance that has that uses the security group and key
* Configures the SSH connection to use IPv6 (you will need to change the variable name).
* Uploads the `variables.tf` file.

## Requirements

* Modify `variables.tf` as appropriate.
* Generate an SSH key and place both the public and private key under `key`.
* Know the network name ahead of time and modify the `connection` info accordingly.

## Usage

* `terraform apply`
* `terraform show`
* `terraform destroy`

Prefix each command with `TF_LOG=1` to see debug output.
