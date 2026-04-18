terraform {
  required_version = ">=1.6"
  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = "~> 0.66"
    }
  }
}

provider "proxmox" {
  endpoint  = var.proxmox_url
  api_token = var.proxmox_api_token
  insecure  = true

  ssh {
    agent    = true
    username = "root"
    node {
      name    = var.proxmox_node
      address = var.proxmox_host
    }
  }
}
