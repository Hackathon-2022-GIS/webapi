Simple webservice for bikeshare data, showing TiDB geo support

Build with:
```shell
go build .
```

The webapi service has a hardcoded port of 4001

Start the service with an environment variable of the database
connection:
```shell
TIDB_DSN=user:pass@tcp(1.2.3.4:4000)/myschema ./
```

Two endpoints:
```
/bikes[?{bike_id|battery_pct|status|station_id}=string[&...]]
```
Where station_id also handles nil/null.
One can also combine the different conditions (which will result
in AND cond = val...)

Example:
```url
http://127.0.0.1:4001/bikes?station_id=32403
```
```json
{
   "bikes":[
      {
         "bike_id":9223372036854776310,
         "battery_pct":9,
         "status":"in_use",
         "station_id":32403
      },
      {
         "bike_id":9223372036854776357,
         "battery_pct":88,
         "status":"in_use",
         "station_id":32403
      }
   ],
   "query":"select bike_id,battery_pct,status,station_id from bikes WHERE station_id = ? limit 1000"
}
```

```
/stations[?{station_id|station_name|station_location|distance|geo|[not]intersects}=string[&...]]
```

One can also combine the different conditions (which will result
in AND cond = val...)
One can also repeat the same condition, which will make the search for
that condition similar to cond IN (val[0],val[1]...)

distance and geo comes in pairs, non pairs will be ignored. (geo must be
given as GeomAsText format)

intersects parameter as GeomAsText that the station_location will intersect
with (multiple conditions works as "OR")

notintersects parameter as GeomAsText that the station cannot intersect
with (multiple conditions works as "AND")

Example:
```url
http://127.0.0.1:4001/stations?distance=0.0009&geo=POINT%20(-77.05332%2038.85725)
```
```json
{
   "stations":[
      {
         "station_id":31001,
         "station_name":"18th St \u0026 S Eads St",
         "station_location":"POINT (-77.05332 38.85725)",
         "station_longitute":"-77.05332",
         "station_latitute":"38.85725"
      }
   ],
   "query":"select station_id,station_name,ST_AsText(station_location) from stations WHERE ST_Distance(`station_location`,ST_GeomFromText(?)) \u003c ? limit 1000"
}
```

```url
http://127.0.0.1:4001/stations?intersects=POINT%20(-77.05332%2038.85725)
```
```json
{
   "stations":[
      {
         "station_id":31001,
         "station_name":"18th St \u0026 S Eads St",
         "station_location":"POINT (-77.05332 38.85725)",
         "station_longitute":"-77.05332",
         "station_latitute":"38.85725"
      }
   ],
   "query":"select station_id,station_name,ST_AsText(station_location) from stations WHERE 1 = ST_Intersects(`station_location`,ST_GeomFromText(?)) limit 1000"
}
```

The stations endpoint also has an option to get all bikes in each station as well, with the 'bikes=1' url argument:
```url
http://example.com:4001/stations?bikes=1&distance=0.009&geo=POINT%20(-77.052808%2038.814577)
```
```json
{
   "query":"select JSON_ARRAYAGG(o)\nfrom (\n  select JSON_OBJECT(\n    \"station_id\",s.station_id,\n    \"station_name\", s.station_name,\n    \"station_location\",ST_AsText(s.station_location),\n    \"station_longitude\",REGEXP_SUBSTR(ST_AsText(s.station_location),'[-.,0-9]+'),\n    \"station_latitude\",REGEXP_SUBSTR(ST_AsText(s.station_location),'[-.,0-9]+',1,2),\n    \"bikes\", JSON_ARRAYAGG(\n       JSON_OBJECT(\n          \"bike_id\",b.bike_id,\n          \"battery_pct\",b.battery_pct,\n          \"status\",b.status\n       )\n    )\n  ) as o\n  from stations s inner join bikes b on s.station_id = b.station_id WHERE ST_Distance(`station_location`,ST_GeomFromText(?)) \u003c ? group by s.station_id ) t;",
   "stations":[
      {
         "bikes":[
            {
               "battery_pct":59,
               "bike_id":14987979559889011483,
               "status":"docked"
            }
         ],
         "station_id":31097,
         "station_latitude":"38.812718",
         "station_location":"POINT (-77.044097 38.812718)",
         "station_longitude":"-77.044097",
         "station_name":"Saint Asaph St \u0026 Madison St"
      },
      {
         "bikes":[
            {
               "battery_pct":51,
               "bike_id":3458764513820541129,
               "status":"docked"
            },
            {
               "battery_pct":73,
               "bike_id":3458764513820541202,
               "status":"docked"
            }
         ],
         "station_id":31915,
         "station_latitude":"38.818748",
         "station_location":"POINT (-77.047783 38.818748)",
         "station_longitude":"-77.047783",
         "station_name":"Powhatan St \u0026 Bashford Ln"
      },
      {
         "bikes":[
            {
               "battery_pct":92,
               "bike_id":3458764513820541261,
               "status":"reserved"
            },
            {
               "battery_pct":85,
               "bike_id":3458764513820541290,
               "status":"docked"
            }
         ],
         "station_id":31099,
         "station_latitude":"38.813485",
         "station_location":"POINT (-77.049468 38.813485)",
         "station_longitude":"-77.049468",
         "station_name":"Madison St \u0026 N Henry St"
      },
      {
         "bikes":[
            {
               "battery_pct":1,
               "bike_id":9223372036854776378,
               "status":"docked"
            },
            {
               "battery_pct":48,
               "bike_id":14987979559889011417,
               "status":"in_use"
            }
         ],
         "station_id":31045,
         "station_latitude":"38.805648",
         "station_location":"POINT (-77.05293 38.805648)",
         "station_longitude":"-77.05293",
         "station_name":"Commerce St \u0026 Fayette St"
      },
      {
         "bikes":[
            {
               "battery_pct":9,
               "bike_id":10952754293765046326,
               "status":"docked"
            },
            {
               "battery_pct":14,
               "bike_id":12105675798371894238,
               "status":"docked"
            },
            {
               "battery_pct":32,
               "bike_id":3458764513820541390,
               "status":"docked"
            }
         ],
         "station_id":31047,
         "station_latitude":"38.814577",
         "station_location":"POINT (-77.052808 38.814577)",
         "station_longitude":"-77.052808",
         "station_name":"Braddock Rd Metro"
      },
      {
         "bikes":[
            {
               "battery_pct":94,
               "bike_id":9223372036854776370,
               "status":"docked"
            },
            {
               "battery_pct":45,
               "bike_id":12105675798371894207,
               "status":"docked"
            }
         ],
         "station_id":31085,
         "station_latitude":"38.820064",
         "station_location":"POINT (-77.057619 38.820064)",
         "station_longitude":"-77.057619",
         "station_name":"Mount Vernon Ave \u0026 E Nelson Ave"
      },
      {
         "bikes":[
            {
               "battery_pct":57,
               "bike_id":14987979559889011345,
               "status":"docked"
            },
            {
               "battery_pct":71,
               "bike_id":14987979559889011357,
               "status":"docked"
            }
         ],
         "station_id":31046,
         "station_latitude":"38.811456",
         "station_location":"POINT (-77.050276 38.811456)",
         "station_longitude":"-77.050276",
         "station_name":"Henry St \u0026 Pendleton St"
      },
      {
         "bikes":[
            {
               "battery_pct":32,
               "bike_id":14987979559889011393,
               "status":"reserved"
            }
         ],
         "station_id":31087,
         "station_latitude":"38.820932",
         "station_location":"POINT (-77.053096 38.820932)",
         "station_longitude":"-77.053096",
         "station_name":"Monroe Ave \u0026 Leslie Ave"
      }
   ]
}
```
