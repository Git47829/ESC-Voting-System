variable "proxmox_url" {
  description = "Proxmox API endpoint"
  type        = string
}

variable "proxmox_api_token" {
  description = "API token - root@pam!terraform=..."
  type        = string
  sensitive   = true
}

variable "proxmox_node" {
  description = "Target Proxmox node name"
  type        = string
  default     = "pve"
}

variable "ssh_public_key" {
  description = "SSH public key for VM access"
  type        = string
}

variable "k3s_nodes" {
  description = "Map of k3s nodes to provision"
  type = map(object({
    role   = string
    cores  = number
    memory = number
    disk   = number
    ip     = string
  }))
  default = {
    "k3s-server-01" = { role = "server", cores = 2, memory = 4096, disk = 20, ip = "192.168.178.110" }
    "k3s-agent-01"  = { role = "agent", cores = 4, memory = 8192, disk = 40, ip = "192.168.178.111" }
    "k3s-agent-02"  = { role = "agent", cores = 4, memory = 8192, disk = 40, ip = "192.168.178.112" }
    "k3s-agent-03"  = { role = "agent", cores = 4, memory = 8192, disk = 40, ip = "192.168.178.113" }
  }
}

variable "proxmox_host" {
  description = "Proxmox host IP for SSH"
  type        = string
}

variable "gateway" {
  type    = string
  default = "192.168.178.1"
}

variable "template_vm_id" {
  description = "VM ID of cloud init template"
  type        = number
  default     = 9000
}

variable "k3s_token" {
  description = "Pre-shared token for k3s cluster"
  type        = string
  sensitive   = true
}
