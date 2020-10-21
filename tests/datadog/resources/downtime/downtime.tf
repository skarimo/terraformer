# Create a new daily 1700-0900 Datadog downtime for all monitors
resource "datadog_downtime" "downtime_example" {
  scope = ["*"]

  recurrence {
    type   = "days"
    period = 1
  }
}
