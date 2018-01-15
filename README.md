### Service-Center Client for Go-Chassis
This is a service-center client which helps the microservice to interact with Service-Center
for service-registration, discovery, instance registration and dependency management.

This client implements all the [api's](https://rawcdn.githack.com/ServiceComb/service-center/master/docs/api-docs.html) of Service-Center.

This client needs the configuration of the Service-Center which can be added in chassis.yaml of the microservice.
```
cse:
  service:
    registry:
      address: https://cse.cn-north-1.myhwclouds.com:443
      scope: full #set full to enable discovery of other app's services
      watch: false  # set if you want to watch instance change event
      autoIPIndex: true # set to true if you want to resolve IP to microservice.
      api:
        version:v4
```

 
