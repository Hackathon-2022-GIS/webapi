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

Two endpoints:
/bikes[?{bike_id|battery_pct|status|station_id}=string[&...]]
Where station_id also handles nil/null.
One can also combine the different conditions (which will result
in AND cond = val...)

Example:
http://127.0.0.1:4001/bikes?station_id=32403
{"bikes":[{"bike_id":9223372036854776310,"battery_pct":9,"status":"in_use","station_id":32403},{"bike_id":9223372036854776357,"battery_pct":88,"status":"in_use","station_id":32403}],"query":"select bike_id,battery_pct,status,station_id from bikes WHERE station_id = ? limit 1000"}

/stations[?{station_id|station_name|station_location|distance|geo|[not]intersects}=string[&...]]

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
http://127.0.0.1:4001/stations?distance=0.0009&geo=POINT%20(-77.05332%2038.85725)
{"stations":[{"station_id":31001,"station_name":"18th St \u0026 S Eads St","station_location":"POINT (-77.05332 38.85725)"}],"query":"select station_id,station_name,ST_AsText(station_location) from stations WHERE ST_Distance(`station_location`,ST_GeomFromText(?)) \u003c ? limit 1000"}

http://127.0.0.1:4001/stations?intersects=POINT%20(-77.05332%2038.85725)
{"stations":[{"station_id":31001,"station_name":"18th St \u0026 S Eads St","station_location":"POINT (-77.05332 38.85725)"}],"query":"select station_id,station_name,ST_AsText(station_location) from stations WHERE 1 = ST_Intersects(`station_location`,ST_GeomFromText(?)) limit 1000"}


