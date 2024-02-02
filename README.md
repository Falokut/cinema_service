# Content

+ [Configuration](#configuration)
    + [Params info](#configuration-params-info)
        + [time.Duration](#timeduration-yaml-supported-values)
        + [Database config](#database-config)
        + [Jaeger config](#jaeger-config)
        + [Prometheus config](#prometheus-config)
+ [Metrics](#metrics)
+ [Docs](#docs)
+ [Author](#author)
+ [License](#license)
---------

# Configuration

1. [Configure movies_db](movies_db/README.md#Configuration)
2. Create .env on project root dir  
Example env:
```env
DB_PASSWORD=Password
```
3. Create a configuration file or change the config.yml file in docker\containers-configs.
If you are creating a new configuration file, specify the path to it in docker-compose volume section (your-path/config.yml:configs/)

## Configuration params info
if supported values is empty, then any type values are supported

| yml name | yml section | env name | param type| description | supported values |
|-|-|-|-|-|-|
| log_level   |      | LOG_LEVEL  |   string   |      logging level        | panic, fatal, error, warning, warn, info, debug, trace|
| host   |  listen    | HOST  |   string   |  ip address or host to listen   |  |
| port   |  listen    | PORT  |   string   |  port to listen   | The string should not contain delimiters, only the port number|
| server_mode   |  listen    | SERVER_MODE  |   string   | Server listen mode, Rest API, gRPC or both | GRPC, REST, BOTH|
| healthcheck_port   |      | HEALTHCHECK_PORT  |   string   |     port for healthcheck| any valid port that is not occupied by other services. The string should not contain delimiters, only the port number|
|service_name|  prometheus    | PROMETHEUS_SERVICE_NAME | string |  service name, thats will show in prometheus  ||
|server_config|  prometheus    |   | nested yml configuration  [metrics server config](#prometheus-config) | |
|db_config|||nested yml configuration  [database config](#database-config) || configuration for database connection | |
|jaeger|||nested yml configuration  [jaeger config](#jaeger-config)|configuration for jaeger connection ||
| network   | cinemas_cache     | CINEMAS_CACHE_NETWORK  |   string   |     network type       | tcp or udp|
| addr   |   cinemas_cache   | CINEMAS_CACHE_ADDR  |   string   |   ip address(or host) with port of redis| all valid addresses formatted like host:port or ip-address:port |
|password| cinemas_cache|CINEMAS_CACHE_PASSWORD|string|password for connection to the redis||
|db| cinemas_cache|CINEMAS_CACHE_DB|string|the number of the database in the redis||
| ttl   | cinemas_cache     |  |  time.Duration with positive duration | the time that cinemas will be stored in the cache|[supported values](#timeduration-yaml-supported-values)|
| network   | cities_cinemas_cache     | CITIES_CINEMA_CACHE_NETWORK  |   string   |     network type       | tcp or udp|
| addr   |   cities_cinemas_cache   | CITIES_CINEMA_CACHE_ADDR  |   string   |   ip address(or host) with port of redis| all valid addresses formatted like host:port or ip-address:port |
|password| cities_cinemas_cache|CITIES_CINEMA_CACHE_PASSWORD|string|password for connection to the redis||
|db| cities_cinemas_cache|CITIES_CINEMA_CACHE_DB|string|the number of the database in the redis||
| ttl   | cities_cinemas_cache     |  |  time.Duration with positive duration | the time that cities will be stored in the cache|[supported values](#timeduration-yaml-supported-values)|
| network   | cities_cache     | CITIES_CACHE_NETWORK  |   string   |     network type       | tcp or udp|
| addr   |   cities_cache   | CITIES_CACHE_ADDR  |   string   |   ip address(or host) with port of redis| all valid addresses formatted like host:port or ip-address:port |
|password| cities_cache|CITIES_CACHE_PASSWORD|string|password for connection to the redis||
|db| cities_cache|CITIES_CACHE_DB|string|the number of the database in the redis||
| ttl   | cities_cache     |  |  time.Duration with positive duration | the time that cities will be stored in the cache|[supported values](#timeduration-yaml-supported-values)|
| network   | halls_cache     | HALLS_CACHE_NETWORK  |   string   |     network type       | tcp or udp|
| addr   |   halls_cache   | HALLS_CACHE_ADDR  |   string   |   ip address(or host) with port of redis| all valid addresses formatted like host:port or ip-address:port |
|password| halls_cache|HALLS_CACHE_PASSWORD|string|password for connection to the redis||
|db| halls_cache|HALLS_CACHE_DB|string|the number of the database in the redis||
| ttl   | halls_cache     |  |  time.Duration with positive duration | the time that halls configuration will be stored in the cache|[supported values](#timeduration-yaml-supported-values)|

### time.Duration yaml supported values
A Duration value can be expressed in various formats, such as in seconds, minutes, hours, or even in nanoseconds. Here are some examples of valid Duration values:
- 5s represents a duration of 5 seconds.
- 1m30s represents a duration of 1 minute and 30 seconds.
- 2h represents a duration of 2 hours.
- 500ms represents a duration of 500 milliseconds.
- 100µs represents a duration of 100 microseconds.
- 10ns represents a duration of 10 nanoseconds.


### Database config
|yml name| env name|param type| description | supported values |
|-|-|-|-|-|
|host|DB_HOST|string|host or ip address of database| |
|port|DB_PORT|string|port of database| any valid port that is not occupied by other services. The string should not contain delimiters, only the port number|
|username|DB_USERNAME|string|username(role) in database||
|password|DB_PASSWORD|string|password for role in database||
|db_name|DB_NAME|string|database name (database instance)||
|ssl_mode|DB_SSL_MODE|string|enable or disable ssl mode for database connection|disabled or enabled|

### Jaeger config
|yml name| env name|param type| description | supported values |
|-|-|-|-|-|
|address|JAEGER_ADDRESS|string|ip address(or host) with port of jaeger service| all valid addresses formatted like host:port or ip-address:port |
|service_name|JAEGER_SERVICE_NAME|string|service name, thats will show in jaeger in traces||
|log_spans|JAEGER_LOG_SPANS|bool|whether to enable log scans in jaeger for this service or not||

### Prometheus config
|yml name| env name|param type| description | supported values |
|-|-|-|-|-|
|host|METRIC_HOST|string|ip address or host to listen for prometheus service||
|port|METRIC_PORT|string|port to listen for  of prometheus service| any valid port that is not occupied by other services. The string should not contain delimiters, only the port number|



# Metrics
The service uses Prometheus and Jaeger and supports distribution tracing

# Docs
[Swagger docs](swagger/docs/cinema_service_v1.swagger.json)

# Author

- [@Falokut](https://github.com/Falokut) - Primary author of the project

# License

This project is licensed under the terms of the [MIT License](https://opensource.org/licenses/MIT).

---
