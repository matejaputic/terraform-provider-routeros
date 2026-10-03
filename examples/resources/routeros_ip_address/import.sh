#!/bin/sh
# Single quotes prevent shell expansion of the RouterOS ID.
terraform import routeros_ip_address.example '*2'
