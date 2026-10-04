# Resource reference

The provider exposes **152 canonical resources** and **zero data sources**. These pages describe the reviewed field subsets, not every RouterOS or reference-provider option.

Schemas/models are produced by the official generators; lifecycle code remains maintained separately. Publication and SDK-state migration are not claimed. See [configuration](../index.md), [coverage and evidence](../development/coverage.md), and [runtime safety](../development/runtime-policy.md).

| Resource | Lifecycle | Required package |
| --- | --- | --- |
| [`routeros_capsman_aaa`](capsman_aaa.md) | Settings; destroy unmanages | `wireless` |
| [`routeros_capsman_manager`](capsman_manager.md) | Settings; destroy unmanages | `wireless` |
| [`routeros_container_config`](container_config.md) | Settings; destroy unmanages | `container` |
| [`routeros_disk_settings`](disk_settings.md) | Settings; destroy unmanages | — |
| [`routeros_interface_6to4`](interface_6to4.md) | Collection CRUD | — |
| [`routeros_interface_bonding`](interface_bonding.md) | Collection CRUD | — |
| [`routeros_interface_bridge`](interface_bridge.md) | Collection CRUD | — |
| [`routeros_interface_bridge_port`](interface_bridge_port.md) | Collection CRUD | — |
| [`routeros_interface_bridge_settings`](interface_bridge_settings.md) | Settings; destroy unmanages | — |
| [`routeros_interface_bridge_vlan`](interface_bridge_vlan.md) | Collection CRUD | — |
| [`routeros_interface_detect_internet`](interface_detect_internet.md) | Settings; destroy unmanages | — |
| [`routeros_interface_dot1x_client`](interface_dot1x_client.md) | Collection CRUD | — |
| [`routeros_interface_dot1x_server`](interface_dot1x_server.md) | Collection CRUD | — |
| [`routeros_interface_gre6`](interface_gre6.md) | Collection CRUD | — |
| [`routeros_interface_l2tp_client`](interface_l2tp_client.md) | Collection CRUD | — |
| [`routeros_interface_l2tp_server`](interface_l2tp_server.md) | Settings; destroy unmanages | — |
| [`routeros_interface_list`](interface_list.md) | Collection CRUD | — |
| [`routeros_interface_list_member`](interface_list_member.md) | Collection CRUD | — |
| [`routeros_interface_macvlan`](interface_macvlan.md) | Collection CRUD | — |
| [`routeros_interface_ovpn_client`](interface_ovpn_client.md) | Collection CRUD | — |
| [`routeros_interface_ovpn_server`](interface_ovpn_server.md) | Collection CRUD | — |
| [`routeros_interface_pppoe_client`](interface_pppoe_client.md) | Collection CRUD | — |
| [`routeros_interface_pppoe_server`](interface_pppoe_server.md) | Collection CRUD | — |
| [`routeros_interface_sstp_client`](interface_sstp_client.md) | Collection CRUD | — |
| [`routeros_interface_sstp_server`](interface_sstp_server.md) | Settings; destroy unmanages | — |
| [`routeros_interface_veth`](interface_veth.md) | Collection CRUD | `container` |
| [`routeros_interface_vlan`](interface_vlan.md) | Collection CRUD | — |
| [`routeros_interface_vxlan`](interface_vxlan.md) | Collection CRUD | — |
| [`routeros_interface_vxlan_vteps`](interface_vxlan_vteps.md) | Collection CRUD | — |
| [`routeros_interface_wireguard`](interface_wireguard.md) | Collection CRUD | — |
| [`routeros_interface_wireguard_peer`](interface_wireguard_peer.md) | Collection CRUD | — |
| [`routeros_interface_wireless_cap`](interface_wireless_cap.md) | Settings; destroy unmanages | `wireless` |
| [`routeros_ip_address`](ip_address.md) | Collection CRUD | — |
| [`routeros_ip_cloud`](ip_cloud.md) | Settings; destroy unmanages | — |
| [`routeros_ip_cloud_advanced`](ip_cloud_advanced.md) | Settings; destroy unmanages | — |
| [`routeros_ip_dhcp_client`](ip_dhcp_client.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_client_option`](ip_dhcp_client_option.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_relay`](ip_dhcp_relay.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_server`](ip_dhcp_server.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_server_config`](ip_dhcp_server_config.md) | Settings; destroy unmanages | — |
| [`routeros_ip_dhcp_server_lease`](ip_dhcp_server_lease.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_server_network`](ip_dhcp_server_network.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_server_option`](ip_dhcp_server_option.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_server_option_matcher`](ip_dhcp_server_option_matcher.md) | Collection CRUD | — |
| [`routeros_ip_dhcp_server_option_set`](ip_dhcp_server_option_set.md) | Collection CRUD | — |
| [`routeros_ip_dns`](ip_dns.md) | Settings; destroy unmanages | — |
| [`routeros_ip_dns_adlist`](ip_dns_adlist.md) | Collection CRUD | — |
| [`routeros_ip_dns_forwarders`](ip_dns_forwarders.md) | Collection CRUD | — |
| [`routeros_ip_dns_record`](ip_dns_record.md) | Collection CRUD | — |
| [`routeros_ip_firewall_addr_list`](ip_firewall_addr_list.md) | Collection CRUD | — |
| [`routeros_ip_firewall_connection_tracking`](ip_firewall_connection_tracking.md) | Settings; destroy unmanages | — |
| [`routeros_ip_firewall_filter`](ip_firewall_filter.md) | Collection CRUD | — |
| [`routeros_ip_firewall_layer7_protocol`](ip_firewall_layer7_protocol.md) | Collection CRUD | — |
| [`routeros_ip_firewall_mangle`](ip_firewall_mangle.md) | Collection CRUD | — |
| [`routeros_ip_firewall_nat`](ip_firewall_nat.md) | Collection CRUD | — |
| [`routeros_ip_firewall_raw`](ip_firewall_raw.md) | Collection CRUD | — |
| [`routeros_ip_hotspot`](ip_hotspot.md) | Collection CRUD | — |
| [`routeros_ip_hotspot_ip_binding`](ip_hotspot_ip_binding.md) | Collection CRUD | — |
| [`routeros_ip_hotspot_profile`](ip_hotspot_profile.md) | Collection CRUD | — |
| [`routeros_ip_hotspot_user`](ip_hotspot_user.md) | Collection CRUD | — |
| [`routeros_ip_hotspot_user_profile`](ip_hotspot_user_profile.md) | Collection CRUD | — |
| [`routeros_ip_hotspot_walled_garden`](ip_hotspot_walled_garden.md) | Collection CRUD | — |
| [`routeros_ip_hotspot_walled_garden_ip`](ip_hotspot_walled_garden_ip.md) | Collection CRUD | — |
| [`routeros_ip_ipsec_identity`](ip_ipsec_identity.md) | Collection CRUD | — |
| [`routeros_ip_ipsec_mode_config`](ip_ipsec_mode_config.md) | Collection CRUD | — |
| [`routeros_ip_ipsec_peer`](ip_ipsec_peer.md) | Collection CRUD | — |
| [`routeros_ip_ipsec_policy`](ip_ipsec_policy.md) | Collection CRUD | — |
| [`routeros_ip_ipsec_policy_group`](ip_ipsec_policy_group.md) | Replacement-only collection | — |
| [`routeros_ip_ipsec_profile`](ip_ipsec_profile.md) | Collection CRUD | — |
| [`routeros_ip_ipsec_proposal`](ip_ipsec_proposal.md) | Collection CRUD | — |
| [`routeros_ip_ipsec_settings`](ip_ipsec_settings.md) | Settings; destroy unmanages | — |
| [`routeros_ip_nat_pmp`](ip_nat_pmp.md) | Settings; destroy unmanages | — |
| [`routeros_ip_nat_pmp_interfaces`](ip_nat_pmp_interfaces.md) | Collection CRUD | — |
| [`routeros_ip_neighbor_discovery_settings`](ip_neighbor_discovery_settings.md) | Settings; destroy unmanages | — |
| [`routeros_ip_pool`](ip_pool.md) | Collection CRUD | — |
| [`routeros_ip_route`](ip_route.md) | Collection CRUD | — |
| [`routeros_ip_settings`](ip_settings.md) | Settings; destroy unmanages | — |
| [`routeros_ip_smb`](ip_smb.md) | Settings; destroy unmanages | — |
| [`routeros_ip_ssh_server`](ip_ssh_server.md) | Settings; destroy unmanages | — |
| [`routeros_ip_tftp`](ip_tftp.md) | Collection CRUD | — |
| [`routeros_ip_tftp_settings`](ip_tftp_settings.md) | Settings; destroy unmanages | — |
| [`routeros_ip_traffic_flow`](ip_traffic_flow.md) | Settings; destroy unmanages | — |
| [`routeros_ip_traffic_flow_ipfix`](ip_traffic_flow_ipfix.md) | Settings; destroy unmanages | — |
| [`routeros_ip_traffic_flow_target`](ip_traffic_flow_target.md) | Collection CRUD | — |
| [`routeros_ip_upnp`](ip_upnp.md) | Settings; destroy unmanages | — |
| [`routeros_ip_upnp_interfaces`](ip_upnp_interfaces.md) | Collection CRUD | — |
| [`routeros_ip_vrf`](ip_vrf.md) | Collection CRUD | — |
| [`routeros_ipv6_address`](ipv6_address.md) | Collection CRUD | — |
| [`routeros_ipv6_dhcp_client`](ipv6_dhcp_client.md) | Collection CRUD | — |
| [`routeros_ipv6_dhcp_client_option`](ipv6_dhcp_client_option.md) | Collection CRUD | — |
| [`routeros_ipv6_dhcp_server`](ipv6_dhcp_server.md) | Collection CRUD | — |
| [`routeros_ipv6_dhcp_server_option`](ipv6_dhcp_server_option.md) | Collection CRUD | — |
| [`routeros_ipv6_dhcp_server_option_sets`](ipv6_dhcp_server_option_sets.md) | Collection CRUD | — |
| [`routeros_ipv6_firewall_addr_list`](ipv6_firewall_addr_list.md) | Collection CRUD | — |
| [`routeros_ipv6_firewall_filter`](ipv6_firewall_filter.md) | Collection CRUD | — |
| [`routeros_ipv6_nd_prefix`](ipv6_nd_prefix.md) | Collection CRUD | — |
| [`routeros_ipv6_neighbor_discovery`](ipv6_neighbor_discovery.md) | Collection CRUD | — |
| [`routeros_ipv6_pool`](ipv6_pool.md) | Collection CRUD | — |
| [`routeros_ipv6_route`](ipv6_route.md) | Collection CRUD | — |
| [`routeros_ipv6_settings`](ipv6_settings.md) | Settings; destroy unmanages | — |
| [`routeros_ppp_aaa`](ppp_aaa.md) | Settings; destroy unmanages | — |
| [`routeros_ppp_profile`](ppp_profile.md) | Collection CRUD | — |
| [`routeros_ppp_secret`](ppp_secret.md) | Collection CRUD | — |
| [`routeros_queue_simple`](queue_simple.md) | Collection CRUD | — |
| [`routeros_queue_tree`](queue_tree.md) | Collection CRUD | — |
| [`routeros_queue_type`](queue_type.md) | Collection CRUD | — |
| [`routeros_radius`](radius.md) | Collection CRUD | — |
| [`routeros_radius_incoming`](radius_incoming.md) | Settings; destroy unmanages | — |
| [`routeros_routing_bfd_configuration`](routing_bfd_configuration.md) | Collection CRUD | — |
| [`routeros_routing_bgp_connection`](routing_bgp_connection.md) | Collection CRUD | — |
| [`routeros_routing_bgp_evpn`](routing_bgp_evpn.md) | Collection CRUD | — |
| [`routeros_routing_bgp_instance`](routing_bgp_instance.md) | Collection CRUD | — |
| [`routeros_routing_bgp_template`](routing_bgp_template.md) | Collection CRUD | — |
| [`routeros_routing_bgp_vpn`](routing_bgp_vpn.md) | Collection CRUD | — |
| [`routeros_routing_filter_rule`](routing_filter_rule.md) | Collection CRUD | — |
| [`routeros_routing_id`](routing_id.md) | Collection CRUD | — |
| [`routeros_routing_igmp_proxy_interface`](routing_igmp_proxy_interface.md) | Collection CRUD | — |
| [`routeros_routing_ospf_area`](routing_ospf_area.md) | Collection CRUD | — |
| [`routeros_routing_ospf_area_range`](routing_ospf_area_range.md) | Collection CRUD | — |
| [`routeros_routing_ospf_instance`](routing_ospf_instance.md) | Collection CRUD | — |
| [`routeros_routing_ospf_interface_template`](routing_ospf_interface_template.md) | Collection CRUD | — |
| [`routeros_routing_rule`](routing_rule.md) | Collection CRUD | — |
| [`routeros_snmp`](snmp.md) | Settings; destroy unmanages | — |
| [`routeros_snmp_community`](snmp_community.md) | Collection CRUD | — |
| [`routeros_system_certificate_scep_server`](system_certificate_scep_server.md) | Collection CRUD | — |
| [`routeros_system_clock`](system_clock.md) | Settings; destroy unmanages | — |
| [`routeros_system_identity`](system_identity.md) | Settings; destroy unmanages | — |
| [`routeros_system_led_settings`](system_led_settings.md) | Settings; destroy unmanages | — |
| [`routeros_system_logging`](system_logging.md) | Collection CRUD | — |
| [`routeros_system_note`](system_note.md) | Settings; destroy unmanages | — |
| [`routeros_system_ntp_client`](system_ntp_client.md) | Settings; destroy unmanages | — |
| [`routeros_system_ntp_server`](system_ntp_server.md) | Settings; destroy unmanages | — |
| [`routeros_system_scheduler`](system_scheduler.md) | Collection CRUD | — |
| [`routeros_system_user`](system_user.md) | Collection CRUD | — |
| [`routeros_system_user_aaa`](system_user_aaa.md) | Settings; destroy unmanages | — |
| [`routeros_system_user_group`](system_user_group.md) | Collection CRUD | — |
| [`routeros_system_user_settings`](system_user_settings.md) | Settings; destroy unmanages | — |
| [`routeros_system_user_sshkeys`](system_user_sshkeys.md) | Replacement-only collection | — |
| [`routeros_tool_bandwidth_server`](tool_bandwidth_server.md) | Settings; destroy unmanages | — |
| [`routeros_tool_email`](tool_email.md) | Settings; destroy unmanages | — |
| [`routeros_tool_graphing_interface`](tool_graphing_interface.md) | Collection CRUD | — |
| [`routeros_tool_graphing_queue`](tool_graphing_queue.md) | Collection CRUD | — |
| [`routeros_tool_graphing_resource`](tool_graphing_resource.md) | Collection CRUD | — |
| [`routeros_tool_mac_server`](tool_mac_server.md) | Settings; destroy unmanages | — |
| [`routeros_tool_mac_server_ping`](tool_mac_server_ping.md) | Settings; destroy unmanages | — |
| [`routeros_tool_mac_server_winbox`](tool_mac_server_winbox.md) | Settings; destroy unmanages | — |
| [`routeros_tool_netwatch`](tool_netwatch.md) | Collection CRUD | — |
| [`routeros_user_manager_advanced`](user_manager_advanced.md) | Settings; destroy unmanages | `user-manager` |
| [`routeros_user_manager_database`](user_manager_database.md) | Settings; destroy unmanages | `user-manager` |
| [`routeros_user_manager_settings`](user_manager_settings.md) | Settings; destroy unmanages | `user-manager` |
| [`routeros_wifi_cap`](wifi_cap.md) | Settings; destroy unmanages | — |
| [`routeros_wifi_capsman`](wifi_capsman.md) | Settings; destroy unmanages | — |
