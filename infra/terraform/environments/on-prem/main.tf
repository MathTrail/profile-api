terraform {
  required_version = ">= 1.5"
}

module "database" {
  source = "../../modules/database"

  mode         = "k8s"
  service_name = "mathtrail-profile"
  db_name      = "profile"
  namespace    = "mathtrail"
}

output "db_host" {
  value = module.database.db_host
}

output "db_port" {
  value = module.database.db_port
}
