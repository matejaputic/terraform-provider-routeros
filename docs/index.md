# RouterOS Provider (preview)

Development address: `registry.terraform.io/matejaputic/routeros`; Registry publication is not configured.

Sixteen resources are exposed: IP address, bridge/port, VLAN, interface list/member, IP pool, DHCP network/server/static lease, static route, permanent IPv4 firewall address-list entries and reviewed ordered filter/NAT/mangle/raw subsets. See [coverage](development/coverage.md) for reviewed fields, live evidence and limitations. Configure `hosturl` and `username` plus sensitive `password`, or use their ROS_/MIKROTIK_ environment fallbacks. Optional `ca_certificate` is a PEM file path; `insecure` disables TLS verification explicitly. `rest_timeout` is seconds (default 59, minimum 5). Native API transport is not supported.

See the repository README for security constraints and the verified local CHR acceptance scope. This preview does not claim migration compatibility.
