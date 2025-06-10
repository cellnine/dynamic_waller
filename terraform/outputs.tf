output "vm_name" {
  description = "The name of the created VM instance."
  value       = google_compute_instance.vm_instance.name
}

output "vm_public_ip" {
  description = "The public IP address of the VM instance."
  value       = google_compute_address.static_ip.address
}
