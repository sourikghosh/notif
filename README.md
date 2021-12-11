<img align="right" width="200px" src="https://github.com/sourikghosh/notif/blob/main/notif.png">

# NOTIF

<b>Notif</b> is a distributed notification service. Its build using <em><b>Nats JetStream</b></em> which serves as a distributed <em>workQueue</em> and consumes the event with a <em>pull</em> based consumer.The pull subscriber actually pulls event from stream which means its can be scaled horizontally very easily. The pull-subscriber fetch events in batchs which can be configured.<br>Notif also has instrumentation support for <em>distributed tracing</em> for observability which is key for async transactions using <em><b>Open Telemetry</b></em> and <em><b>Jaeger</b></em> as exporter.

## Prerequisite
- a smtp server / enable smtp service on your email
- docker and go installed

## Installation
- `git@github.com:sourikghosh/notif.git`
- `cd notif`
- `go mod downlaod`
- `make js`
- `make trace`
- `make server`

## Notification
Notif currently only supports email as notification. Email is sent using go standard lib smtp package.<br>Notif does not provide a SMTP server it takes few required credentials to create a secured TLS smtp client connection if possible to send emails.

## Enpoints
Notif currently only support rest endpoint to create notification events.<br>gRPC enpoints are coming soon.

## Jaeger-UI
Then navigate to [Jaeger-UI](http://localhost:16686)
