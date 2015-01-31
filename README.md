# terraform-provider-openstack

This is an experimental OpenStack provider for Terraform. It is based off of the excellent work already done by haklop which can be found [here](https://github.com/haklop/terraform). The main difference is that it works with the latest version of gophercloud and does not need to be compiled along with the entire Terraform code.

However, it only supports the `openstack_compute` resource at this time. Even more, not all functionality (such as resizing) has been implemented or tested.

Only launching and destroying an instance in a `nova-network` based environment has been tested so far.

## Warning

I have zero knowledge of Go. This work consisted of me reading the Go tutorial and fooling around for a few hours. If things look sloppy and outright wrong, they are.

Also, please be aware that this is just me fooling around over a weekend. I will probably not take this plugin beyond the current state. If you'd like to take over work, by all means, go for it.

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

```
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

```
provier "openstack" { }
```

For more information on OpenStack `openrc` files, see [http://docs.openstack.org/user-guide/content/cli_openrc.html].

### Terraform Configuration

The following examples have been tested:

```
provider "openstack" {}

resource "openstack_compute" "test" {
  name = "jttest"
  image_id = "ecdd59d0-eff5-4d1b-be5e-dde94ffcfdb2"
  # or image_name = "Ubuntu 14.04"
  flavor_ref = "1"
  # or flavor_name = "m1.large"
  key_name = "my_key"
  networks = [ "94e12a2a-d692-4e6f-8e34-560e8a97ead5" ]
  security_groups = [ "default", "my_custom_group" ]
  user_data = "#!/bin/bash\nping -c 10 yahoo.com"
  config_drive = true
  metadata {
    foo = "bar"
    baz = "foo"
  }
}
```

### Launch

```shell
$ terraform plan
$ terraform build
$ terraform destroy
```

## Notes

`image_id`, `flavor_ref`, and `networks` must be the UUIDs and not the canonical names. Also that the networks must be in array/list format.

`networks` is optional if your OpenStack cloud only has one network.

`admin_pass` is enabled, but I haven't verified it yet.

`networks_advanced` is available for advanced networking configuration. One or more of the following can be specified:

```
networks_advanced {
  uuid: "94e12a2a-d692-4e6f-8e34-560e8a97ead5"
  # Not fully tested yet
  port: "(port uuid)"
  # Not fully tested yet
  fixed_ip: "10.1.1.150"
}
```


## Credits

* Eric / haklop for his initial [work](https://github.com/haklop/terraform)
* tkak for their [object storage provider](https://github.com/tkak/terraform-provider-conoha) which I would have been lost without.
