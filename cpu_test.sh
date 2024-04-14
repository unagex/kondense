# List of bash scripts to stress test cpu.

# use ~10m of cpu
while true; do
    for ((i=0; i<100; i++)); do
        x=$((x * 2 + 1))
    done
    sleep 0.1
done

# use ~100m of cpu
while true; do
    for ((i=0; i<100; i++)); do
        x=$((x * 2 + 1))
    done
    sleep 0.01
done