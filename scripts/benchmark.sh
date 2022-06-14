hyperfine -w 2 --export-markdown benchmarks.md \
'OCTOSQL_NO_TELEMETRY=1 octosql "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count"' \
"q -d ',' -H \"SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count\"" \
"q -C -d ',' -H \"SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count\"" \
'dsq taxi.csv "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM {} GROUP BY passenger_count"'
'dsq -C taxi.csv "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM {} GROUP BY passenger_count"'
