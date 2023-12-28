- Create an ProtoProxy which can listen Nats and push to db
	1. Need to create store library
	2. need to select db
	3. need to deploy db
	4. Need to create the deserialize library
		1. use unit test to validate
		2. transform proto encoding to tsdb query	
	5. Need to deploy NatsIO
	6. Need to create a NatsIO library 
	7. Need to create to pipe the natsio telemetry to the db through protoproxy