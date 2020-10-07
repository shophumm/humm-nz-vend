
# Humm Vend Application Proxy


## Integrate your Vend Terminal with Humm

## Development

This assumes you have the repo cloned to $GOPATH/src/github.com/shophumm/humm-vend.

### Dependencies
* Go (tested with version 1.10)
* A MariaDB or MySQL Database. Other db's can be supported easily however MariaDB is fast and easy to replicate. A docker-compose file exists which can be used for testing. 


The application requires a configuration file. A default config file is located in configs/vendproxy.json however by default the application will look in /etc/vendproxy/vendproxy.json. 

In order to change this behaviour to make development more convenient set the environment variable DEV to true

```$ export DEV=true```


### Build 

```$ cd cmd; go build ./vendproxy.go ```

### Docker Build

* Assumes you have the AWS-CLI installed and configured
* Assumes you have docker-compose installed


#### Get the login command for docker to register with ECS
``` $(aws ecr get-login --no-include-email --region <ecs region>) ```

* Execute the returned command (as long as it looks sane)

#### Build

```
    $ cd docker
    $ docker-compose build
    $ docker-compose push

```


## Executing

```$ ./vendproxy ```

```
$:~/go/src/github.com/vend/peg/cmd$ ./vendproxy 
{"level":"info","msg":"Attempting to connect to database user:password@tcp(172.18.0.2)/vend?parseTime=true\u0026loc=Local \n","time":"2018-08-23T17:10:55+09:30"}
{"level":"info","msg":"Starting webserver on port 5000 \n","time":"2018-08-23T17:10:55+09:30"}

```


### Docker

A development ```docker-compose.yml``` is provided which will allow a local setup of the proxy. You will need to provide the following environment files

* maridb.env

```
MYSQL_ROOT_PASSWORD=<root_password>
MYSQL_USER=vendproxy
MYSQL_PASSWORD=<vendproxy_db_password>
MYSQL_DATABASE=vend
```

* vendproxy_au.env

** Note these are intentionally lowercase

```
database_username=vendproxy
database_password=<vendproxy_db_password>
database_host=database-vend
session_secret=<session_secret>
humm_gatewayurl=https://integration-pos.shophumm.co.nz/webapi/v1/

```

* nginx.env

```
HTTP_PORT=80
TLS_PORT=443


NZ_SITE_URL=vend.shophumm.co.nz
NZ_SITE_HOME=/srv/www/vendproxy
NZ_SSL_CRT=/run/secrets/wildcard.shophumm.co.nz.crt
NZ_SSL_KEY=/run/secrets/wildcard.shophumm.co.nz.key
NZ_PROXY_TO=http://vendproxy_nz:5000
```

* TLS Certificates

You will need to create a directory structure similar to ```/etc/ssl```. 

```
$ cd docker
$ mkdir -p ./ssl/certs && mkdir ./ssl/private
$ cp my.crt ./ssl/certs
$ cp my.key ./ssl/private
````

Your docker/docker-compose.yml will also need to be updated to reference these files. @todo .env file


#### Database

Ensure that the database and database tables that are required are executed against the environment 

``` mysql -u root -p<rootpw> -h <host> vend < ./scripts/init.sql ```

This will likely change in the near future to use a sqitch.org plan / container 

### Production Setup 

#### Configuration

Ensure that the following settings are changed in your production configuration or in a vendproxy.env file if deploying using docker.

* database.username
* database.password
* session.secret (used to encrypt session info)
* humm.gatewayurl (should be set to the prod end point)



### Deployment with Docker


## Licenses
- [MIT License](https://github.com/vend/peg/blob/master/LICENSE)
- [Google Open Source Font Attribution](https://fonts.google.com/attribution)
