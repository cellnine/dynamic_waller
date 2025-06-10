terraform {
  backend "gcs" {
    bucket = "wallman"
    prefix = "tf/state"
  }
}
