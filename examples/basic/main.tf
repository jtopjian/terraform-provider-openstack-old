resource "openstack_keypair" "demo_key" {
  name = "demo_key"
  public_key = "${file("key/id_rsa.pub")}"
  region = "${var.region}"
}

resource "openstack_secgroup" "demo_group" {
  name = "demo_group"
  description = "Demo Group"
  region = "${var.region}"

  rule {
    protocol = "tcp"
    from_port = "22"
    to_port = "22"
    cidr = "0.0.0.0/0"
  }

  rule {
    protocol = "tcp"
    from_port = "22"
    to_port = "22"
    cidr = "::/0"
  }
}

resource "openstack_instance" "demo_instance" {
  name = "demo_instance"
  image_name = "${var.image_name}"
  flavor_name = "${var.flavor_name}"
  key_name = "${openstack_keypair.demo_key.name}"
  security_groups = [ "default", "${openstack_secgroup.demo_group.name}" ]
  region = "${var.region}"

  connection {
    user = "ubuntu"
    key_file = "key/id_rsa"
    host = "${openstack_instance.demo_instance.network_info.cybera_ipv6}"
  }

  provisioner file {
    source = "variables.tf"
    destination = "variables.tf"
  }
}

