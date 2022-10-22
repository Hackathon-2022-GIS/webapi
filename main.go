package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type bike struct {
	BikeId     uint64 `json:"bike_id"`
	BatteryPct uint8  `json:"battery_pct"`
	Status     string `json:"status"`
	StationId  *int   `json:"station_id"`
}

type bikeResult struct {
	Bikes []bike `json:"bikes"`
	Query string `json:"query"`
}

func fetchBikes(key, val []string) ([]byte, error) {
	if len(key) != len(val) {
		return nil, errors.New("len(key) != len(val)")
	}
	dsn := os.Getenv("TIDB_DSN")
	db, err := sql.Open("mysql", dsn)
	defer db.Close()
	if err != nil {
		return nil, err
	}

	columns := []string{"bike_id", "battery_pct", "status", "station_id"}
	where := make([]string, 0, len(key))
	conds := make([]any, 0, len(val))
	for i, k := range key {
		found := ""
		for _, c := range columns {
			if strings.EqualFold(k, c) {
				found = c
				break
			}
		}
		if found != "" {
			if found == "station_id" {
				if strings.EqualFold(val[i], "nil") || strings.EqualFold(val[i], "null") {
					where = append(where, found+" is null")
					continue
				}
			}
			where = append(where, found+" = ?")
			conds = append(conds, val[i])
		}
	}
	whereStr := ""
	if len(where) > 0 {
		whereStr = " WHERE " + strings.Join(where, " AND ")
	}
	query := `select ` + strings.Join(columns, ",") + ` from bikes` + whereStr + ` limit 1000`
	fmt.Println("Running: " + query)
	rows, err := db.Query(query, conds...)
	if err != nil {
		fmt.Printf("error in SQL: %s\nError: %s", query, err.Error())
		return nil, err
	}
	bikes := make([]bike, 0, 8)
	for rows.Next() {
		var b bike
		//var stationId *int // to handle null and convert to 0?
		err = rows.Scan(&b.BikeId, &b.BatteryPct, &b.Status, &b.StationId)
		if err != nil {
			return nil, err
		}
		bikes = append(bikes, b)
	}
	res := &bikeResult{Bikes: bikes, Query: query}
	return json.Marshal(res)
}

func bikesEndpoint(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Welcome to the bikesEndpoint!")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := r.ParseForm(); err != nil {
		fmt.Printf("bikes form error: %s", err.Error())
		return
	}
	keys := make([]string, 0, len(r.Form))
	vals := make([]string, 0, len(r.Form))
	for k, v := range r.Form {
		keys = append(keys, k)
		vals = append(vals, v[0])
	}
	json, err := fetchBikes(keys, vals)
	if err != nil {
		w.Write([]byte(`{"error":` + strconv.Quote(err.Error())))
		fmt.Printf("bikes error: %s", err.Error())
		return
	}
	w.Write(json)
	fmt.Println("Served one request")
}

type station struct {
	StationId        uint64 `json:"station_id"`
	StationName      string `json:"station_name"`
	StationLocation  string `json:"station_location"`
	StationLongitude string `json:"station_longitute"`
	StationLatitude  string `json:"station_latitute"`
	//StationLocation []byte `json:"station_location"`
}

type stationResult struct {
	Stations []station `json:"stations"`
	Query    string    `json:"query"`
}

var stationColumns = []string{"station_id", "station_name", "station_location"}

func fetchStations(key []string, val [][]string) ([]byte, error) {
	if len(key) != len(val) {
		return nil, errors.New("len(key) != len(val)")
	}
	dsn := os.Getenv("TIDB_DSN")
	db, err := sql.Open("mysql", dsn)
	defer db.Close()
	if err != nil {
		return nil, err
	}

	whereStr, conds, err := getStationWhere(key, val)
	if len(whereStr) > 0 {
		whereStr = " WHERE " + whereStr
	}
	query := `select ` + strings.Join(stationColumns[:len(stationColumns)-1], ",") + `,ST_AsText(` + stationColumns[len(stationColumns)-1] + `) from stations` + whereStr + ` limit 1000`
	fmt.Println("Running: " + query)
	rows, err := db.Query(query, conds...)
	if err != nil {
		fmt.Printf("error in SQL: %s\nError: %s", query, err.Error())
		return nil, err
	}
	stations := make([]station, 0, 8)
	for rows.Next() {
		var s station
		err = rows.Scan(&s.StationId, &s.StationName, &s.StationLocation)
		if err != nil {
			return nil, err
		}
		tmpStr := strings.TrimPrefix(s.StationLocation, "POINT (")
		tmpStr = strings.TrimSuffix(tmpStr, ")")
		strs := strings.Split(tmpStr, " ")
		s.StationLongitude, s.StationLatitude = strs[0], strs[1]
		stations = append(stations, s)
	}
	fmt.Printf("returning %d stations\nquery: %s\n", len(stations), query)
	res := &stationResult{Stations: stations, Query: query}
	return json.Marshal(res)
}

func getStationWhere(key []string, val [][]string) (string, []any, error) {
	if len(key) != len(val) {
		return "", nil, errors.New("len(key) != len(val)")
	}

	where := make([]string, 0, len(key))
	conds := make([]any, 0, len(val))
	dist := make([]string, 0)
	geo := make([]string, 0)
	for i, k := range key {
		fmt.Println("checking key: ", k)
		found := ""
		for _, c := range stationColumns {
			if strings.EqualFold(k, c) {
				found = c
				break
			}
		}
		if found != "" {
			w := make([]string, 0, len(val[i]))
			for j := range val[i] {
				if found == "station_id" {
					if strings.EqualFold(val[i][j], "nil") || strings.EqualFold(val[i][j], "null") {
						w = append(w, found+" is null")
						continue
					}
				}
				w = append(w, found+" = ?")
				conds = append(conds, val[i][j])
			}
			where = append(where, strings.Join(w, " OR "))
			continue
		}
		if strings.EqualFold(k, "distance") {
			dist = append(dist, val[i]...)
			continue
		}
		if strings.EqualFold(k, "geo") {
			geo = append(geo, val[i]...)
			continue
		}
		notIntersects := strings.EqualFold(k, "notintersects")
		if notIntersects || strings.EqualFold(k, "intersects") {
			notStr := "1 = "
			if notIntersects {
				notStr = "0 = "
			}
			w := make([]string, 0, len(val[i]))
			for j := range val[i] {
				w = append(w, notStr+"ST_Intersects(`station_location`,ST_GeomFromText(?))")
				conds = append(conds, val[i][j])
			}
			if notIntersects {
				where = append(where, strings.Join(w, " AND "))
			} else {
				where = append(where, strings.Join(w, " OR "))
			}
		}
	}
	if len(dist) < len(geo) {
		geo = geo[:len(dist)]
	}
	if len(dist) > len(geo) {
		dist = dist[:len(geo)]
	}
	if len(dist) > 0 {
		w := make([]string, 0, len(dist))
		for i := range dist {
			w = append(w, "ST_Distance(`station_location`,ST_GeomFromText(?)) < ?")
			fmt.Println("geo: ", geo[i])
			conds = append(conds, geo[i], dist[i])
		}
		where = append(where, strings.Join(w, " OR "))
	}
	return strings.Join(where, " AND "), conds, nil
}

type jsonAndQuery struct {
	Query string          `json:"query"`
	Json  json.RawMessage `json:"stations"`
}

func fetchStationsAndBikes(key []string, val [][]string) ([]byte, error) {
	dsn := os.Getenv("TIDB_DSN")
	db, err := sql.Open("mysql", dsn)
	defer db.Close()
	if err != nil {
		return nil, err
	}
	whereStr, conds, err := getStationWhere(key, val)
	if whereStr != "" {
		whereStr = " WHERE " + whereStr
	}
	if err != nil {
		fmt.Printf("error in generating WHERE, Error: %s", err.Error())
		return nil, err
	}
	//query := `select ` + strings.Join(stationColumns[:len(stationColumns)-1], ",") + `,ST_AsText(` + stationColumns[len(stationColumns)-1] + `) from stations` + whereStr + ` limit 1000`
	queryStart := `select JSON_ARRAYAGG(o)
from (
  select JSON_OBJECT(
    "station_id",s.station_id,
    "station_name", s.station_name,
    "station_location",ST_AsText(s.station_location),
    "station_longitude",REGEXP_SUBSTR(ST_AsText(s.station_location),'[-.,0-9]+'),
    "station_latitude",REGEXP_SUBSTR(ST_AsText(s.station_location),'[-.,0-9]+',1,2),
    "bikes", JSON_ARRAYAGG(
       JSON_OBJECT(
          "bike_id",b.bike_id,
          "battery_pct",b.battery_pct,
          "status",b.status
       )
    )
  ) as o
  from stations s inner join bikes b on s.station_id = b.station_id`
	queryEnd := ` group by s.station_id ) t;`

	query := queryStart + whereStr + queryEnd

	fmt.Println("Running: " + query)
	rows, err := db.Query(query, conds...)
	if err != nil {
		fmt.Printf("error in SQL: %s\nError: %s", query, err.Error())
		return nil, err
	}
	ret := jsonAndQuery{}
	ret.Query = query
	for rows.Next() {
		retStr := ""
		err = rows.Scan(&retStr)
		if err != nil {
			return nil, err
		}
		ret.Json = []byte(retStr)
		break // Only one line expected
	}
	return json.Marshal(ret)
}

func stationsEndpoint(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Welcome to the stationsEndpoint!")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := r.ParseForm(); err != nil {
		fmt.Printf("stations form error: %s", err.Error())
		return
	}
	keys := make([]string, 0, len(r.Form))
	vals := make([][]string, 0, len(r.Form))
	for k, v := range r.Form {
		keys = append(keys, k)
		vs := make([]string, 0, len(v))
		vs = append(vs, v...)
		vals = append(vals, vs)
	}
	var ret []byte // json data
	var err error
	withBikes := r.Form.Get("bikes")
	if withBikes != "" && withBikes != "0" {
		ret, err = fetchStationsAndBikes(keys, vals)
	} else {
		ret, err = fetchStations(keys, vals)
	}
	if err != nil {
		w.Write([]byte(`{"error":` + strconv.Quote(err.Error())))
		fmt.Printf("stations error: %s", err.Error())
		return
	}
	w.Write(ret)
	fmt.Println("Served one request")
}

func handleRequests() {
	http.HandleFunc("/bikes", bikesEndpoint)
	http.HandleFunc("/stations", stationsEndpoint)
	log.Fatal(http.ListenAndServe(":4001", nil))
}

func main() {
	handleRequests()
}
