# terraform-provider-openstack

This is an experimental OpenStack provider for Terraform. It is based off of the excellent work already done by haklop which can be found [here](https://github.com/haklop/terraform). The main difference is that it works with the latest version of gophercloud and does not need to be compiled along with the entire Terraform code.

## Warning

I have almost no knowledge of Go. This work consists of me copying copying other examples and being amazed that any of this actually works. If things look sloppy and outright wrong, they are.

Also, please be aware that this is just a fun side project for me. If you'd like to take over work in a more serious manner, by all means, go for it.

## Installation

Download the provider:

```shell
$ go get github.com/jtopjian/terraform-provider-openstack
```

Download and install the dependencies:

```shell
$ cd $GOPATH/src/github.com/jtopjian/terraform-provider-openstack
$ godep restore
```

Compile it:

```shell
$ go build -o terraform-provider-openstack
```

Copy it to the directory you keep Terraform:

```shell
$ sudo cp terraform-provider-openstack /usr/local/bin/terraform
```

## Usage

### Provider Authentication

You can authenticate with the OpenStack cloud by either explicitly setting parameters or using an `openrc`-style file.

#### Explicit Parameters

```ruby
provider "openstack" {
  identity_endpoint = "http://example.com:5000/v2.0"
  username = "jdoe"
  tenant_name = "jdoe"
  password = "password"
}
```

#### openrc-style

First, source your `openrc` file:

```shell
$ source openrc
```

Next, configure the provider in the `*.tf` file:

```ruby
provider "openstack" { }
```

For more information on OpenStack `openrc` files, see [here](http://docs.openstack.org/user-guide/content/cli_openrc.html]).

### Terraform Configuration

This example does the following:

* Creates a security group that allows SSH access from anywhere.
* Imports a public key.
* Creates an instance and allows access via the imported key.
* Allocates a floating IP.
* Associates the floating IP to the new instance.

```ruby
provider "openstack" {}

resource "openstack_secgroup" "test" {
  name = "MyTestGroup"
  description = "Testing"

  rule {
    protocol = "tcp"
    from_port = "22"
    to_port = "22"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_keypair" "mykey" {
  name = "mykey"
  public_key = "(contents of id_rsa.pub or similar)"
}

resource "openstack_compute" "test" {
  name = "test"
  image_name = "Ubuntu 14.04"
  flavor_name = "m1.large"
  key_name = "${openstack_keypair.mykey.name}"
  networks = [ "94e12a2a-d692-4e6f-8e34-560e8a97ead5" ]
  security_groups = [ "default", "${openstack_secgroup.test.name}" ]
}

resource "openstack_floating_ip" "test" {
  pool = "nova"
  network_service = "nova-network"
  instance_id = "${openstack_compute.test.id}"
}

```

### Launch

```shell
$ terraform plan
$ terraform build
$ terraform destroy
```

## Reference and Notes

### openstack_compute

#### Notes:

* Modifications to launched instances hasn't been tested yet.
* Either an `image_id` or an `image_name` is required.
* Either a `flavor_id` or a `flavor_name` is required.

#### Parameters

* `name`: the name of the instance. Required.
* `image_id`: the UUID of the image.
* `image_name`: the canonical name of the image.
* `flavor_id`: the UUID of the flavor.
* `flavor_name`: the canonical name of the flavor.
* `key_name`: the ssh keypair name.
* `networks`: an array of network UUIDs that the instance will be attached to.
* `security_groups`: an array of security group names to apply to the instance.
* `config_drive`: boolean to enable config drive.
* `admin_pass`: a login password to the instance. NOT TESTED.
* `metadata`: a set of key/value pairs to apply to the instance:
* `region`: Which region to , for multi-region clouds.

```ruby
metadata {
  foo = "bar"
  baz =" foo"
}
```

* `network`: configure a network with specific details. May be specified multiple times for multiple networks:

```ruby
network {
  UUID = "94e12a2a-d692-4e6f-8e34-560e8a97ead5"
  port_id = "(neutron port-id)" # NOT TESTED
  fixed_ip = "192.168.255.20" # NOT TESTED
}
```

### openstack_keypair

#### Notes

* Only importing an existing key is supported. Generating a new key and downloading the private key is not supported yet.

#### Parameters

* `name`: the name of the keypair. Required.
* `public_key`: the contents of an `id_rsa.pub` or similar public key file. Required.
* `region`: Which region to send the key to, for multi-region clouds.


### openstack_floating_ip

#### Notes

* Only `nova-network`-based clouds work at this time.

#### Parameters

* `pool`: the floating IP pool to pull an address from. Required.
* `network_service`: Either `nova-network` or `neutron`. Defaults to Neutron.
* `instance_id`: the UUID of the instance to associate the floating IP with.
* `region`: Which region to pull an IP from, for multi-region clouds.

### openstack_secgroup

#### Notes

* Either a `cidr` or `source_group` is required for each rule.

#### Parameters

* `name`: the name of the security group. Required.
* `description`: a description of the security group. Required.
* `rule`: One or more rule blocks consisting of the following:
  * `from_port`: Beginning of a port range. Required.
  * `to_port`: End of a port range. Required.
  * `protocol`: A protocol such as tcp, udp, icmp, etc. Required.
  * `cidr`: A network cidr to grant access. `0.0.0.0/0` for all IPv4 addresses and `::/0` for all IPv6 addresses.
  * `source_group`: Use another security group as the allowed access list.

## Credits

* Eric / haklop for his initial [work](https://github.com/haklop/terraform)
* tkak for their [object storage provider](https://github.com/tkak/terraform-provider-conoha) which I would have been lost without.
