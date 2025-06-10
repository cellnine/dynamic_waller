provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
}

resource "google_compute_network" "vpc_network" {
  name                    = "heic-maker-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = "heic-maker-subnet"
  ip_cidr_range = "10.10.10.0/24"
  network       = google_compute_network.vpc_network.id
  region        = var.gcp_region
}

resource "google_compute_firewall" "firewall" {
  name    = "heic-maker-firewall"
  network = google_compute_network.vpc_network.id

  allow {
    protocol = "tcp"
    ports    = ["22", "80", "443"]
  }

  source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_address" "static_ip" {
  name   = "heic-maker-static-ip"
  region = var.gcp_region
}

resource "google_compute_instance" "vm_instance" {
  name         = "heic-maker-vm"
  machine_type = var.machine_type
  zone         = var.gcp_zone

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
    }
  }

  network_interface {
    network    = google_compute_network.vpc_network.id
    subnetwork = google_compute_subnetwork.subnet.id
    access_config {
      nat_ip = google_compute_address.static_ip.address
    }
  }

  metadata_startup_script = <<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y git docker.io docker-compose

    usermod -aG docker ${var.ssh_user}

    sudo -u ${var.ssh_user} git clone https://github.com/YOUR_USERNAME/YOUR_REPO.git /home/${var.ssh_user}/heic-wallpaper-maker
  EOF

  # 7. This injects your public SSH key for passwordless access
  metadata = {
    ssh-keys = "${var.ssh_user}:${file(var.ssh_pub_key_path)}"
  }

  # Allow the service account to have access to the instance
  service_account {
    scopes = ["cloud-platform"]
  }
}
