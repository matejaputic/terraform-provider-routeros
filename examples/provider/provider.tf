terraform {
  required_providers {
    routeros = {
      source = "matejaputic/routeros"
    }
  }
}

# Supply ROS_HOSTURL, ROS_USERNAME and ROS_PASSWORD in the environment.
provider "routeros" {}
