terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = "darioenv-flyable"
  region  = "europe-west4"
}

resource "google_storage_bucket" "flights_dump" {
  name          = "darioenv-flights-database"
  location      = "EU"
  force_destroy = false

  uniform_bucket_level_access = true

  versioning {
    enabled = false
  }
} 