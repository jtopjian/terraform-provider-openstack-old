# terraform-provider-openstack

This is an experimental OpenStack provider for OpenStack. It is based off of the excellent work already done by haklop which can be found [here](https://github.com/haklop/terraform). The main difference is that it works with the latest version of gophercloud and does not need to be compiled along with the entire Terraform code.

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

### openrc file

All authentication can be done by sourcing an `openrc`-style file.

### Terraform Configuration

I have tested this provider with this extremely simple example:

```
provider "openstack" {}

resource "openstack_compute" "test" {
  name = "jttest"
  image_ref = "ecdd59d0-eff5-4d1b-be5e-dde94ffcfdb2"
  flavor_ref = "1"
  key_name = "my_key"
}
```

### Launch

```shell
$ terraform plan
$ terraform build
$ terraform destroy
```

Note that the `image_ref` and the `flavor_ref` must be the UUIDs and not the canonical names. I've seen how to make this more user friendly from code within gophercloud as well as some of the Packer source.

## Credits

* Eric / haklop for his initial [work](https://github.com/haklop/terraform)
* tkak for their [object storage provider](https://github.com/tkak/terraform-provider-conoha) which I would have been lost without.
