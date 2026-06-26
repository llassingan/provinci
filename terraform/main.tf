terraform {
  required_providers {
    oci = {
      source  = "hashicorp/oci"
      version = "~> 6.0"
    }
  }
}

provider "oci" {
  region = var.region
}

resource "oci_core_vcn" "vps_store" {
  compartment_id = var.compartment_ocid
  cidr_blocks    = ["10.0.0.0/16"]
  display_name   = "vps-store-vcn"
  dns_label      = "vpsstore"
}

resource "oci_core_internet_gateway" "igw" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.vps_store.id
  display_name   = "vps-store-igw"
}

resource "oci_core_default_route_table" "default" {
  manage_default_resource_id = oci_core_vcn.vps_store.default_route_table_id
  route_rules {
    destination       = "0.0.0.0/0"
    network_entity_id = oci_core_internet_gateway.igw.id
  }
}

resource "oci_core_subnet" "public" {
  compartment_id    = var.compartment_ocid
  vcn_id            = oci_core_vcn.vps_store.id
  cidr_block        = "10.0.1.0/24"
  display_name      = "vps-store-public-subnet"
  dns_label         = "public"
  security_list_ids = [oci_core_security_list.public.id]
}

resource "oci_core_security_list" "public" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.vps_store.id
  display_name   = "vps-store-public-sl"

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 80
      max = 80
    }
  }
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 443
      max = 443
    }
  }
  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }
}
