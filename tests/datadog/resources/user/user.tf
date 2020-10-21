# Create a new Datadog user
resource "datadog_user" "user_example" {
  email  = "new@example.com"
  handle = "new@example.com"
  name   = "New User"
}
