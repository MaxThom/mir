
api:
	buf generate --config api/buf.gen.yaml

rabbit:
	docker stop rabbitmq || true
	sleep 1
	docker run --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3.12-management

db:
	docker stop surrealdb || true
	sleep 1
	docker run --rm --pull always --name surrealdb -p 80:8000 -v ./surrealdb:/opt/surrealdb/ surrealdb/surrealdb:1.0.0-beta.11 start --log info --user root --pass root file:/opt/surrealdb/iot.db

ts:
	docker stop questdb || true
	sleep 1
	docker run --rm --name questdb -p 9000:9000 -p 9009:9009 -p 8812:8812 -p 9003:9003 questdb/questdb:7.3.3
