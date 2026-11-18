// Disposable native package tests. Existing VMs are never overwritten or pruned.
packer {
  required_version = "= 1.16.2"
  required_plugins {
    tart = {
      source  = "github.com/cirruslabs/tart"
      version = "= 1.21.0"
    }
  }
}

variable "vm_name" {
  type        = string
  description = "A new, absent name for this package-test VM."
  validation {
    condition     = can(regex("^patronus-package-[a-z0-9-]+$", var.vm_name))
    error_message = "Use a fresh name beginning patronus-package-."
  }
}

variable "check_script" {
  type        = string
  default     = ""
  description = "Optional host-side shell script to upload and run after setup."
}

source "tart-cli" "ubuntu" {
  vm_base_name = "ghcr.io/cirruslabs/ubuntu@sha256:92f2ea7463d8ae86f061d11d69696ad0b1053a59a958ecf941eebfa83534e700"
  vm_name      = var.vm_name
  cpu_count    = 2
  memory_gb    = 4
  disk_size_gb = 30
  headless     = true
  disable_vnc  = true
  ssh_username = "admin"
  ssh_password = "admin" // Public stock-image credentials; no host credentials.
  ssh_timeout  = "10m"
}

build {
  sources = ["source.tart-cli.ubuntu"]

  provisioner "shell" {
    script = "${path.root}/bootstrap-ubuntu.sh"
  }
  provisioner "shell" {
    script = var.check_script == "" ? "${path.root}/probe.sh" : var.check_script
  }
}
