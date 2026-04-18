output "server_ips" {
  value = {
    for k, v in proxmox_virtual_environment_vm.k3s_nodes :
    k => var.k3s_nodes[k].ip
    if var.k3s_nodes[k].role == "server"
  }
}

output "agent_ips" {
  value = {
    for k, v in proxmox_virtual_environment_vm.k3s_nodes :
    k => var.k3s_nodes[k].ip
    if var.k3s_nodes[k].role == "agent"
  }
}
