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
Notif currently only supports <b>email</b> as notification. Email is sent using go standard lib <em>smtp package</em>.<br>Notif does not provide a <b>SMTP server</b> it takes few required credentials to create a <em>secured TLS</em> smtp client connection if possible to send emails.<br>Email accept all possible content-type so you can send from beautiful htmls to basic plain/text. For examples refer to [example](https://github.com/sourikghosh/notif/blob/main/examples/main.go)  

## Enpoints
Notif currently only support rest endpoint to create notification events.<br>gRPC enpoints are coming soon.
```bash
curl --request POST \
  --url http://localhost:6969/notif-svc/v1/create \
  --header 'Content-Type: application/json' \
  --data '{
	"fromName":"Sourik Ghosh",
	"toList":[
		{
			"emailAddr":"someemail@example.com",
			"userName":"some one"
		}
	],
	"subject":"This is the subject",
	"body":"Hi someone, Have a great day !!!"
}'
```

## Jaeger-UI
Then navigate to [Jaeger-UI](http://localhost:16686)
