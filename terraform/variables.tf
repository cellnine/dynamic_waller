variable "gcp_project_id" {
  description = "The GCP Project ID to deploy resources into."
  type        = string
}

variable "gcp_region" {
  description = "The GCP region to deploy resources into."
  type        = string
  default     = "europe-west1"
}

variable "gcp_zone" {
  description = "The GCP zone to deploy the VM into."
  type        = string
  default     = "europe-west1-b"
}

variable "machine_type" {
  description = "The machine type for the VM."
  type        = string
  default     = "e2-medium"
}

variable "ssh_user" {
  description = "The username for SSH access."
  type        = string
  default     = "deployer"
}

variable "ssh_pub_key_path" {
  description = "The path to the public SSH key for the user."
  type        = string
  default     = "~/Documents/endeavour/dynamicwall.pub"
}

variable "github_repo_ssh_url" {
  description = "The SSH URL of the GitHub repository to clone."
  type        = string
}
