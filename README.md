# Prometheus LDAP Service Discover Server

## Description

This application allows auto-service discovery of instrumented endpoints for which the hosts are registered in LDAP.  The resulting endpoints are exposed in the format required by [Prometheus HTTP SD](https://prometheus.io/docs/prometheus/latest/http_sd/)

Another common usecase is to use this to allow for discovery of Windows hosts which are listed and managed in ActiveDirectory.  In a corporate settings, this removes the necessity for to register these windows-based metrics endpoints in other solutions such as Consul to gain automatic discovery.

## Usage

Running the server:
```
./prometheus-ldap-sd-server -conf /path/to/config.yaml [-debug] [-version] [-validate]
```

## Command Flags

`-conf` : The path to the configuration file to be used
`-validate` : Validate configuration and exit.
`-debug` : Enable debug mode
`-version` : Show version and exit

## Configuration Options

- `host` : The host on which to listen (default is 127.0.0.1)
- `port`: The port on which to listen (default is 80)
- `ldap_config.server`:  The address of the LDAP/ActiveDirectory server
- `ldap_config.authenticated`: Enable connecting with authentication
- `ldap_config.unsecured`: Allow unsecured connections
- `ldap_config.bind_dn`: The bind DN to use for the authentication user
- `ldap_config.base_dn_mappings`: A map of base DNs in the format of <GROUP_NAME> -> <BASE_DN_LIST>
- `ldap_config.base_dn_mappings.[X].base_dn_list` : List of 
- `ldap_config.base_dn_mappings.[X].exporter_port` : The port on which the prometheux exporter is exposing metrics on the discovered host
- `ldap_config.base_dn_mappings.[X].attributes` : The attributes to include for the list of labels exposed for the list of discovered targets
- `ldap_config.base_dn_mappings.[X].filter` : The filter to be used to limit the list of discovered targets.  Specifying this one will ignore the top level - `ldap_config.filter` option.
- `ldap_config.group_exporter_port_mapping`: A mapping of exporter port to include for each <GROUP_NAME>
- `ldap_config.filter`: The filter to use when querying AD.  Note: This generally shouldn't be modified.
- `ldap_config.attributes`: The list of attributes to fetch from each LDAP object.  
- `ldap_config.cache_dir`: The directory in which the cache is stroed.
- `ldap_config.cache_ttl`: The, ttl in seconds, of the cached results
- `ldap_config.password_env_var`: The environment variable in which the LDAP password is set.

A sample configuration can be found in the `_samples/` directory. 

## Available endpoints

* **GET /targets?targetGroup=<GROUP_NAME>**
    * Return the list of targets (formated in expected HTTP SD format)
* **GET /metrics**
    * Return the list of prometheus metrics for the exporter
* **GET /healthz**
    *  Return the current health status of the exporter
* **GET /config**
    * Return the current config which has been used to start the exporter
* **GET /debug/profile**
    * Generate a debugging profile.  See [here](https://go.dev/blog/pprof) for more details.


## Reference of ActiveDirectory and LDAP attributes

You can find a list of ActiveDirectory attributes here:
https://docs.microsoft.com/en-us/windows/win32/adschema/attributes-all

And the list of LDAP attributes:
http://www.phpldaptools.com/reference/Default-Schema-Attributes/#ad-computer-types



## Building

### 1. Checkout required code version

First, ensure you have checked out the proper release tag in order to get all files/dependencies corresponding to that version. 

### 2. Build Go binary

Run `make build` to build the the binary for the current operatory system or run `make build-all` to build for both Linux and OSX.   Refer to the makefile for additional options.

### 3. Build Docker container
Run the following docker command to build the image
```
docker build -t prometheus-ldap-sd:$(cat VERSION.txt) --build-arg VERSION=$(cat VERSION.txt) .
```


## License

Covered under the [MIT license](LICENSE.md).

## Author

Alain Lefebvre <hartfordfive 'at' gmail.com>
