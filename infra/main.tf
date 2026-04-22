resource "proxmox_virtual_environment_file" "cloud_init_server" {
  content_type = "snippets"
  datastore_id = "local"
  node_name    = var.proxmox_node

  source_raw {
    data      = <<-EOF
      #cloud-config
      disable_root: false
      users:
        - name: ${var.node_login_user}
          groups: [sudo]
          shell: /bin/bash
          sudo: ALL=(ALL) NOPASSWD:ALL
          ssh_authorized_keys:
            - ${var.ssh_public_key}
        - name: root
          shell: /bin/bash
          ssh_authorized_keys:
            - ${var.ssh_public_key}
      package_update: true
      packages:
        - curl
      runcmd:
        - curl -sfL https://get.k3s.io | K3S_TOKEN=${var.k3s_token} sh -s - server --tls-san ${var.k3s_nodes["k3s-server-01"].ip}
    EOF
    file_name = "k3s-server-cloud-init.yaml"
  }
}

resource "proxmox_virtual_environment_file" "cloud_init_agent" {
  content_type = "snippets"
  datastore_id = "local"
  node_name    = var.proxmox_node

  source_raw {
    data      = <<-EOF
      #cloud-config
      disable_root: false
      users:
        - name: ${var.node_login_user}
          groups: [sudo]
          shell: /bin/bash
          sudo: ALL=(ALL) NOPASSWD:ALL
          ssh_authorized_keys:
            - ${var.ssh_public_key}
        - name: root
          shell: /bin/bash
          ssh_authorized_keys:
            - ${var.ssh_public_key}
      package_update: true
      packages:
        - curl
      runcmd:
        - curl -sfL https://get.k3s.io | K3S_URL=https://${var.k3s_nodes["k3s-server-01"].ip}:6443 K3S_TOKEN=${var.k3s_token} sh -
    EOF
    file_name = "k3s-agent-cloud-init.yaml"
  }
}

resource "proxmox_virtual_environment_vm" "k3s_nodes" {
  for_each  = var.k3s_nodes
  name      = each.key
  node_name = var.proxmox_node
  tags      = [each.value.role, "k3s"]

  clone {
    vm_id = var.template_vm_id
    full  = true
  }

  cpu {
    cores = each.value.cores
    type  = "x86-64-v3"
  }

  memory {
    dedicated = each.value.memory
  }

  disk {
    datastore_id = "local-lvm"
    size         = each.value.disk
    interface    = "scsi0"
    discard      = "on"
  }

  network_device {
    bridge = "vmbr0"
    model  = "virtio"
  }

  initialization {
    user_data_file_id = each.value.role == "server" ? proxmox_virtual_environment_file.cloud_init_server.id : proxmox_virtual_environment_file.cloud_init_agent.id
    ip_config {
      ipv4 {
        address = "${each.value.ip}/24"
        gateway = var.gateway
      }
    }
  }

  operating_system {
    type = "l26"
  }

  started = true
}
