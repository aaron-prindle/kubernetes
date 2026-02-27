set terminal pngcairo size 800,600 enhanced font 'Arial,12'
set output 'docs/memory_scaling_plot.png'

set title 'API Server WatchCache Memory: metav1.FieldsV1 Allocations' font ',14'
set xlabel 'Number of Duplicated Pods (Running)'
set ylabel 'Memory Allocated (MB)'

set grid
set key top left
set xrange [-1000:55000]
set yrange [0:150]

# Add labels to points
set label "10.0 MB" at 5000, 15 center textcolor rgb "red"
set label "130.6 MB" at 50000, 138 center textcolor rgb "red"
set label "2.5 MB" at 5000, -5 center textcolor rgb "dark-green"
set label "27.5 MB" at 50000, 20 center textcolor rgb "dark-green"

# Define data points inline
$data << EOD
# Pods  Master_MB  Experimental_MB
50      0.0        0.0
5000    10.01      2.50
50000   130.59     27.52
EOD

plot $data using 1:2 with linespoints lw 3 pt 7 ps 1.5 lc rgb "red" title 'master (Baseline []byte)', \
     $data using 1:3 with linespoints lw 3 pt 5 ps 1.5 lc rgb "dark-green" title 'experimental (stringhandle)'
