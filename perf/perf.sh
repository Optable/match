#! /bin/bash

#set variables
CUR_DIR=$(pwd)
BASE_DIR=`dirname $CUR_DIR`

GOLD_STANDARD_PERF_FILE="$BASE_DIR/perf/gold_standard_bench.txt"
LATEST_PERF_FILE="$BASE_DIR/perf/latest_bench.txt"
TEMP_FILE="$BASE_DIR/perf/temp_bench.txt"

BENCH_DIRS=($BASE_DIR/"internal/crypto")

TEST_COUNT=2

echo "Match library performance" > $LATEST_PERF_FILE
echo "------------------------" >> $LATEST_PERF_FILE

PRINT_SYS_INFO=1
for dir in ${BENCH_DIRS[@]}; do
    # build now
    go build $dir
    
    echo "Running performance tests in $dir"
    go test $dir -bench=. -count=$TEST_COUNT > $TEMP_FILE

    # Take environment info from benchmark output
    if [ $PRINT_SYS_INFO -eq 1 ]; then
        cat $TEMP_FILE | head -n 4 >> $LATEST_PERF_FILE
        PRINT_SYS_INFO=0
        echo "------------------------" >> $LATEST_PERF_FILE
    fi
    
    # produce results with benchstat
    echo "Benchmarking $dir" >> $LATEST_PERF_FILE
    benchstat $TEMP_FILE | tail -n+2 >> $LATEST_PERF_FILE

    echo "------------------------" >> $LATEST_PERF_FILE
done

#cleanup
rm $TEMP_FILE