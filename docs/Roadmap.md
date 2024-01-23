# Roadmap

- Create an ProtoProxy which can listen Nats and push to db
  1. [x] Need to create store library
     - [x] Create store server
  2. [x] need to select db [questdb]
  3. [x] need to deploy db [docker compose]
  4. [x] Need to create the deserialize library to line protocol
  5. [x] use unit test to validate
  6. [x] Need to deploy NatsIO [docker compose]
  7. [x] Need to create a NatsIO library
  8. [x] Need to create to pipe the natsio telemetry to the db through protoproxy
  9. [x] Deploy Questdb and connect
  10. [ ] Add metrics to protoproxy
  11. [ ] Add dashboard for protoproxy
  12. [ ] Add dashboard for natsio
  13. [ ] Add dashboard for Questdb
  14. [ ] Add timeseries field to proto libraryd
  15. [x] Add metrics endpoint for prometheus, nodeexporter, natsio, questdb
  16. [x] Configure a grafana with questdb and prometheus data source
