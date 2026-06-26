output "vcn_ocid" {
  value = oci_core_vcn.vps_store.id
}

output "subnet_ocid" {
  value = oci_core_subnet.public.id
}
