# Universal exporter for Prometheus

This is universal (HTTP client) exporter for Prometheus.
It allows to fetch data from external service and process them into metrics, based on declarative configuration.

- ### Example: export current weather data from open-meteo.com

<details>
<summary>Response body from API</summary>

```json
{
  "latitude": 48.2,
  "longitude": 16.38,
  "generationtime_ms": 0.10216236114501953,
  "utc_offset_seconds": 0,
  "timezone": "GMT",
  "timezone_abbreviation": "GMT",
  "elevation": 179.0,
  "current_units": {
    "time": "iso8601",
    "interval": "seconds",
    "temperature_2m": "°C",
    "relative_humidity_2m": "%",
    "apparent_temperature": "°C",
    "is_day": "",
    "precipitation": "mm",
    "rain": "mm",
    "showers": "mm",
    "snowfall": "cm",
    "weather_code": "wmo code",
    "cloud_cover": "%",
    "pressure_msl": "hPa",
    "surface_pressure": "hPa",
    "wind_speed_10m": "km/h",
    "wind_direction_10m": "°",
    "wind_gusts_10m": "km/h"
  },
  "current": {
    "time": "2025-05-23T05:45",
    "interval": 900,
    "temperature_2m": 10.5,
    "relative_humidity_2m": 52,
    "apparent_temperature": 6.7,
    "is_day": 1,
    "precipitation": 0.00,
    "rain": 0.00,
    "showers": 0.00,
    "snowfall": 0.00,
    "weather_code": 2,
    "cloud_cover": 48,
    "pressure_msl": 1014.7,
    "surface_pressure": 993.1,
    "wind_speed_10m": 12.7,
    "wind_direction_10m": 331,
    "wind_gusts_10m": 30.2
  }
}
```
</details>

<details>
<summary>Fully functional configuration</summary>

[see also](examples/config-openmeteo.yaml)

```yaml
---
metrics:
  openmeteo_current_apparent_temperature:
    help: Current temperature feel like
    labels:
      - location
  openmeteo_current_cloud_cover:
    help: Total cloud cover as an area fraction
    labels:
      - location
  openmeteo_current_precipitation:
    help: Probability of precipitation
    labels:
      - location
  openmeteo_current_pressure_msl:
    help: Atmospheric air pressure reduced to mean sea level
    labels:
      - location
  openmeteo_current_rain:
    help: Rain from large scale weather systems
    labels:
      - location
  openmeteo_current_relative_humidity:
    help: Relative humidity
    labels:
      - location
  openmeteo_current_showers:
    help: Showers from convective precipitation
    labels:
      - location
  openmeteo_current_snowfall:
    help: Snowfall
    labels:
      - location
  openmeteo_current_surface_pressure:
    help: Atmospheric air pressure at surface
    labels:
      - location
  openmeteo_current_temperature:
    help: Current temperature
    labels:
      - location
  openmeteo_current_wind_dir:
    help: current wind direction
    labels:
      - location
  openmeteo_current_wind_gusts:
    help: Wind gusts at 10 meters above ground
    labels:
      - location
  openmeteo_current_wind_speed:
    help: Current wind speed.
    labels:
      - location
targets:
  openmeteo_vienna:
    steps:
      001-set:
        order: 1
        set:
          data:
            openmeteo:
              mapping:
                - path: current.apparent_temperature
                  metric: openmeteo_current_apparent_temperature
                - path: current.cloud_cover
                  metric: openmeteo_current_cloud_cover
                - path: current.precipitation
                  metric: openmeteo_current_precipitation
                - path: current.pressure_msl
                  metric: openmeteo_current_pressure_msl
                - path: current.rain
                  metric: openmeteo_current_rain
                - path: current.relative_humidity_2m
                  metric: openmeteo_current_relative_humidity
                - path: current.showers
                  metric: openmeteo_current_showers
                - path: current.snowfall
                  metric: openmeteo_current_snowfall
                - path: current.pressure_msl
                  metric: openmeteo_current_surface_pressure
                - path: current.temperature_2m
                  metric: openmeteo_current_temperature
                - path: current.wind_direction_10m
                  metric: openmeteo_current_wind_dir
                - path: current.wind_gusts_10m
                  metric: openmeteo_current_wind_gusts
                - path: current.wind_speed_10m
                  metric: openmeteo_current_wind_speed
      002-fetch:
        order: 2
        ext:
          function: http_fetch
          args:
            url: https://api.open-meteo.com/v1/forecast?latitude={{ .vars.latitude }}&longitude={{ .vars.longitude }}&current=temperature_2m,relative_humidity_2m,apparent_temperature,is_day,precipitation,rain,showers,snowfall,weather_code,cloud_cover,pressure_msl,surface_pressure,wind_speed_10m,wind_direction_10m,wind_gusts_10m
            headers:
              accept: application/json
            storeTo: openmeteo.Response
            parseJson: true
      003-process:
        order: 3
        forEach:
          query: openmeteo.mapping
          action:
            steps:
              01-set-metric:
                order: 1
                ext:
                  function: prom_gauge
                  args:
                    value: '{{ printf "{{ .openmeteo.Response.json.%s }}" .forEach.path }}'
                    ref: '{{ .forEach.metric }}'
                    labels:
                      - '{{ .vars.locationLabel }}'

vars:
  locationLabel: Vienna
  latitude: 48.20
  longitude: 16.37

```

</details>

<details>
<summary>metric output (trimmed to relevant part)</summary>

```openmetric
# HELP openmeteo_current_apparent_temperature Current temperature feel like
# TYPE openmeteo_current_apparent_temperature gauge
openmeteo_current_apparent_temperature{location="Vienna"} 12.3
# HELP openmeteo_current_cloud_cover Total cloud cover as an area fraction
# TYPE openmeteo_current_cloud_cover gauge
openmeteo_current_cloud_cover{location="Vienna"} 100
# HELP openmeteo_current_precipitation Probability of precipitation
# TYPE openmeteo_current_precipitation gauge
openmeteo_current_precipitation{location="Vienna"} 0
# HELP openmeteo_current_pressure_msl Atmospheric air pressure reduced to mean sea level
# TYPE openmeteo_current_pressure_msl gauge
openmeteo_current_pressure_msl{location="Vienna"} 1014
# HELP openmeteo_current_rain Rain from large scale weather systems
# TYPE openmeteo_current_rain gauge
openmeteo_current_rain{location="Vienna"} 0
# HELP openmeteo_current_relative_humidity Relative humidity
# TYPE openmeteo_current_relative_humidity gauge
openmeteo_current_relative_humidity{location="Vienna"} 33
# HELP openmeteo_current_showers Showers from convective precipitation
# TYPE openmeteo_current_showers gauge
openmeteo_current_showers{location="Vienna"} 0
# HELP openmeteo_current_snowfall Snowfall
# TYPE openmeteo_current_snowfall gauge
openmeteo_current_snowfall{location="Vienna"} 0
# HELP openmeteo_current_surface_pressure Atmospheric air pressure at surface
# TYPE openmeteo_current_surface_pressure gauge
openmeteo_current_surface_pressure{location="Vienna"} 1014
# HELP openmeteo_current_temperature Current temperature
# TYPE openmeteo_current_temperature gauge
openmeteo_current_temperature{location="Vienna"} 16
# HELP openmeteo_current_wind_dir current wind direction
# TYPE openmeteo_current_wind_dir gauge
openmeteo_current_wind_dir{location="Vienna"} 319
# HELP openmeteo_current_wind_gusts Wind gusts at 10 meters above ground
# TYPE openmeteo_current_wind_gusts gauge
openmeteo_current_wind_gusts{location="Vienna"} 23.8
# HELP openmeteo_current_wind_speed Current wind speed.
# TYPE openmeteo_current_wind_speed gauge
openmeteo_current_wind_speed{location="Vienna"} 10.5
```

</details>

**Explanation**

open-meteo.com API provides current weather data for given GPS coordinates at following URL:

`https://api.open-meteo.com/v1/forecast?latitude=48.2082&longitude=16.3738&current=temperature_2m,relative_humidity_2m,apparent_temperature,is_day,precipitation,rain,showers,snowfall,weather_code,cloud_cover,pressure_msl,surface_pressure,wind_speed_10m,wind_direction_10m,wind_gusts_10m`

Configuration above defines one target `openmeteo_vienna` with few variables and steps necessary to extract metrics.
Most notably there is

1. `001-set` which defines mapping between JSON body response path and metric name
2. `002-fetch` send HTTP request to API endpoint and parses body as JSON into data tree at location `.openmeteo.Response.json`
3. `003-process` iterates over each item in list at location `.openmeteo.mapping` and for every item it gets data from json body (`.path`) and set it to associated gauge metric (`.metric`)


- ### Example: export current weather data from openweathermap.org

<details>
<summary>Response body from API</summary>

```json
{
  "coord": {
    "lon": 16.3738,
    "lat": 48.2082
  },
  "weather": [
    {
      "id": 804,
      "main": "Clouds",
      "description": "overcast clouds",
      "icon": "04n"
    }
  ],
  "base": "stations",
  "main": {
    "temp": 14.27,
    "feels_like": 13.79,
    "temp_min": 12.21,
    "temp_max": 14.96,
    "pressure": 1008,
    "humidity": 78,
    "sea_level": 1008,
    "grnd_level": 982
  },
  "visibility": 10000,
  "wind": {
    "speed": 2.06,
    "deg": 10
  },
  "clouds": {
    "all": 100
  },
  "dt": 1746390531,
  "sys": {
    "type": 2,
    "id": 2037452,
    "country": "AT",
    "sunrise": 1746329452,
    "sunset": 1746382296
  },
  "timezone": 7200,
  "id": 2761369,
  "name": "Vienna",
  "cod": 200
}

```
</details>

<details>
<summary>Fully functional configuration</summary>

[see also](examples/config-owm.yaml)

_Note you need to define environment variable `OWM_API_KEY` with actual API key_

```yaml
---
metrics:
  owm_current_humidity:
    help: Current humidity
    labels:
      - location
  owm_current_pressure:
    help: Current atmospheric pressure
    labels:
      - location
  owm_current_temperature:
    help: Current temperature
    labels:
      - location
  owm_current_temperature_feel:
    help: Current temperature feel like
    labels:
      - location
  owm_current_temperature_min:
    help: Minimal currently observed temperature
    labels:
      - location
  owm_current_temperature_max:
    help: Maximal currently observed temperature
    labels:
      - location
  owm_current_wind_direction:
    help:
    labels:
      - location
  owm_current_wind_speed:
    labels:
      - location
  owm_exporter_api_requests:
    labels:
      - location
  owm_exporter_scrapes_total:
    help: Total number of times OWM was scraped for metrics.
    labels:
      - location
targets:
  owm_vienna:
    vars:
      locationLabel: Vienna
      latitude: 48.20
      longitude: 16.37

    steps:
      001-set:
        order: 1
        set:
          data:
            OWM_Mapping:
              - path: main.temp
                metric: owm_current_temperature
              - path: main.feels_like
                metric: owm_current_temperature_feel
              - path: main.temp_min
                metric: owm_current_temperature_min
              - path: main.temp_max
                metric: owm_current_temperature_max
              - path: main.pressure
                metric: owm_current_pressure
              - path: main.humidity
                metric: owm_current_humidity
              - path: wind.speed
                metric: owm_current_wind_speed
              - path: wind.deg
                metric: owm_current_wind_direction
      002-import-env:
        order: 2
        env:
          include: OWM_API_KEY
      003-fetch:
        order: 3
        ext:
          function: http_fetch
          args:
            url: https://api.openweathermap.org/data/2.5/weather?lat={{ .vars.latitude }}&lon={{ .vars.longitude }}&units=metric&appid={{ .Env.OWM_API_KEY }}
            headers:
              accept: application/json
            storeTo: Result.OWM.RAW
            parseJson: true
      004-process:
        order: 4
        forEach:
          query: OWM_Mapping
          action:
            steps:
              01-set-metric:
                order: 1
                ext:
                  function: prom_gauge
                  args:
                    value: '{{ printf "{{ .Result.OWM.RAW.json.%s }}" .forEach.path }}'
                    ref: '{{ .forEach.metric }}'
                    labels:
                      - '{{ .vars.locationLabel }}'

```

</details>

<details>
<summary>metric output (trimmed to relevant part)</summary>

```openmetric
# HELP owm_current_humidity Current humidity
# TYPE owm_current_humidity gauge
owm_current_humidity{location="Vienna"} 90
# HELP owm_current_pressure Current atmospheric pressure
# TYPE owm_current_pressure gauge
owm_current_pressure{location="Vienna"} 1007
# HELP owm_current_temperature Current temperature
# TYPE owm_current_temperature gauge
owm_current_temperature{location="Vienna"} 12.61
# HELP owm_current_temperature_feel Current temperature feel like
# TYPE owm_current_temperature_feel gauge
owm_current_temperature_feel{location="Vienna"} 12.28
# HELP owm_current_temperature_max Maximal currently observed temperature
# TYPE owm_current_temperature_max gauge
owm_current_temperature_max{location="Vienna"} 14.34
# HELP owm_current_temperature_min Minimal currently observed temperature
# TYPE owm_current_temperature_min gauge
owm_current_temperature_min{location="Vienna"} 11
# HELP owm_current_wind_direction
# TYPE owm_current_wind_direction gauge
owm_current_wind_direction{location="Vienna"} 170
# HELP owm_current_wind_speed
# TYPE owm_current_wind_speed gauge
owm_current_wind_speed{location="Vienna"} 2.57
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
```

</details>

**Explanation**

openweathermap.org API provides current weather data for given GPS coordinates at following URL:

`https://api.openweathermap.org/data/2.5/weather?lat=48.2&lon=16.38&units=metric&appid=API_KEY`

Configuration above defines one target `owm_vienna` with few variables and steps necessary to extract metrics.
Most notably there is

1. `001-set` which defines mapping between JSON body response path and metric name
2. `002-import-env` takes API key from environment variable `OWM_API_KEY` and set it into data tree at location `.Env.OWM_API_KEY`
3. `003-fetch` send HTTP request to API endpoint and parses body as JSON into data tree at location `.Result.OWM.RAW.json`
4. `004-process` iterates over each item in list at location `.OWM_Mapping` and for every item it gets data from json body and set it to associated gauge metric
