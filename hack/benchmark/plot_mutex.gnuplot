set terminal pngcairo size 800,400 enhanced font 'Arial,12'
set output 'docs/mutex_contention_plot.png'

set title 'API Server Mutex Contention During Parallel LIST (30s Window)' font ',14'
set ylabel 'Significant Mutex Delays (Count)'
set style data histograms
set style histogram cluster gap 1
set style fill solid 1.0 border -1
set boxwidth 0.9

set yrange [0:5]
set grid y

# Define colors
set style line 1 lc rgb "red"
set style line 2 lc rgb "dark-green"

$data << EOD
Load_Size Master Experimental
5k_Pods   0      0
10k_Pods  0      0
EOD

plot $data using 2:xtic(1) title 'master (Baseline []byte)' ls 1, \
     $data using 3 title 'experimental (unique.Make)' ls 2
