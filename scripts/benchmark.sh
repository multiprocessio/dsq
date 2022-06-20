# Extract if doesn't exist
7z e -aos testdata/taxi.csv.7z

hyperfine --min-runs 10 -w 2 --export-markdown benchmarks.md \
"sqlite3 :memory: -cmd '.mode csv' -cmd '.import taxi.csv taxi' 'SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi GROUP BY passenger_count'" \
"duckdb -c \"SELECT passenger_count, COUNT(*), AVG(total_amount) FROM 'taxi.csv' GROUP BY passenger_count\"" \
'trdsql -ih "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count"' \
'OCTOSQL_NO_TELEMETRY=1 octosql "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count"' \
"q -d ',' -H \"SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count\"" \
"q -d ',' -H -C readwrite \"SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count\"" \
'dsq taxi.csv "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM {} GROUP BY passenger_count"' \
'dsq -C taxi.csv "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM {} GROUP BY passenger_count"' \
'textql -header -sql "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi GROUP BY passenger_count" taxi.csv' \
'spyql "SELECT passenger_count, count_agg(*), avg_agg(total_amount) FROM csv GROUP BY passenger_count" < taxi.csv'
