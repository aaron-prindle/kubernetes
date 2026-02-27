set terminal pngcairo size 800,600 enhanced font 'Arial,12'
set output 'docs/contention_scaling_plot.png'

set title 'API Server Parallel LIST Contention: CPU Thrashing' font ',14'
set xlabel 'Number of Duplicated Pods (Pending)'
set ylabel 'Total API Server CPU Time (Seconds)'

set grid
set key top left
set xrange [4000:11000]
set yrange [0:1500]

# Add labels to points
set label "238.4s" at 5000, 290 center textcolor rgb "red"
set label "1336.8s" at 10000, 1380 center textcolor rgb "red"
set label "240.2s" at 5000, 190 center textcolor rgb "dark-green"
set label "678.4s" at 10000, 620 center textcolor rgb "dark-green"

# Define data points inline
$data << EOD
# Pods  Master_s  Experimental_s
5000    238.4      240.2
10000   1336.8     678.4
EOD

plot $data using 1:2 with linespoints lw 3 pt 7 ps 1.5 lc rgb "red" title 'master (Baseline []byte)', \
     $data using 1:3 with linespoints lw 3 pt 5 ps 1.5 lc rgb "dark-green" title 'experimental (stringhandle)'
